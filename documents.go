package paperless

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// documentListEntry mirrors the checksum-bearing shape of Paperless-ngx's
// document objects; only the checksum is needed for ledger reconciliation.
// Since paperless-ngx 3.x the flat checksum is gone from the serializer and
// the versions array carries it instead.
type documentListEntry struct {
	Checksum string                   `json:"checksum"`
	Versions []documentVersionPayload `json:"versions"`
}

// effectiveChecksum returns the document's content checksum across API
// shapes: the flat field served by older paperless-ngx versions first, then
// the root version's checksum (3.x), then any non-empty version checksum.
func (e documentListEntry) effectiveChecksum() string {
	//nolint:branching-flow // wire strings at the parse boundary; branded slots are the v2 typed-API scope (ROADMAP)
	return checksumFrom(e.Checksum, e.Versions)
}

// checksumFrom resolves a document checksum across API shapes; shared by
// the verify (documentListEntry) and backfill (documentMetaPayload) paths.
func checksumFrom(flat string, versions []documentVersionPayload) string {
	if flat != "" {
		return flat
	}

	for _, version := range versions {
		if version.IsRoot && version.Checksum != "" {
			return version.Checksum
		}
	}

	for _, version := range versions {
		if version.Checksum != "" {
			return version.Checksum
		}
	}

	return ""
}

// fetchAllPages walks a paginated DRF list endpoint (page/page_size query
// parameters, `{"results": [...]}` envelope) up to maxDocumentListPages and
// returns the concatenated entries. A short page ends the scan; the page cap
// bounds it defensively so a server serving full pages forever cannot hang
// callers. resource is the singular object name, used to build error messages
// and error codes ("storage path" → "list storage paths (page 2)").
func fetchAllPages[T any](
	ctx context.Context,
	c *Client,
	path, resource string,
) ([]T, error) {
	items := []T{}

	for page := 1; page <= maxDocumentListPages; page++ {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("page_size", strconv.Itoa(documentListPageSize))

		raw, err := c.doRequest(ctx, http.MethodGet, path, query.Encode(), nil, "")
		if err != nil {
			return nil, fmt.Errorf(
				"list %ss (page %d): %w",
				resource,
				page,
				err,
			) //nolint:erraudit // keeps inner code+family
		}

		list := struct {
			Results []T `json:"results"`
		}{}

		if unmarshalErr := json.Unmarshal(raw, &list); unmarshalErr != nil {
			code := "paperless.decode_" + strings.ReplaceAll(resource, " ", "_") + "s"

			return nil, errorfamily.WrapCorruption(unmarshalErr, code,
				"could not decode "+resource+"s list").
				WithContext("page", strconv.Itoa(page))
		}

		items = append(items, list.Results...)

		if len(list.Results) < documentListPageSize {
			break
		}
	}

	return items, nil
}

// ListDocumentChecksums returns the SHA-256 checksum of every document
// currently stored in Paperless-ngx. Used to reconcile the local upload
// ledger against reality: ledger entries whose checksum disappeared from
// Paperless-ngx (documents deleted there) are candidates for re-upload.
func (c *Client) ListDocumentChecksums(ctx context.Context) (map[string]struct{}, error) {
	entries, err := fetchAllPages[documentListEntry](ctx, c, pathDocuments, "document")
	if err != nil {
		return nil, err
	}

	checksums := map[string]struct{}{}

	for _, entry := range entries {
		if checksum := entry.effectiveChecksum(); checksum != "" {
			checksums[checksum] = struct{}{}
		}
	}

	return checksums, nil
}

// documentMetaPayload mirrors the subset of Paperless-ngx's document object
// the metadata backfill needs. correspondent and tags decode to IDs; a null
// correspondent decodes as 0. created stays a string so both the datetime
// form Paperless-ngx serves and legacy date-only values parse.
//
// SDK↔fake wire-shape mirror is deliberate: the fake must serialize exactly the SDK's parse shape.
type documentMetaPayload struct {
	ID            DocumentID                `json:"id"`
	Title         string                    `json:"title"`
	Correspondent CorrespondentID           `json:"correspondent"`
	Created       string                    `json:"created"`
	Tags          []TagID                   `json:"tags"`
	DocumentType  DocumentTypeID            `json:"document_type"`
	CustomFields  []customFieldValuePayload `json:"custom_fields"`
	Checksum      string                    `json:"checksum"`
	Versions      []documentVersionPayload  `json:"versions"`
	// Content is the OCR text Paperless-ngx serves on every document
	// serialization (list and detail alike — DocumentSerializer serves
	// effective_content, "" when the document has none).
	Content string `json:"content"`
}

// documentVersionPayload mirrors one entry of Paperless-ngx's document
// versions array. Since paperless-ngx 3.x the flat document checksum is no
// longer part of the serializer fields; the checksum moved into the
// versions array (is_root marks the current original).
//
// SDK↔fake wire-shape mirror is deliberate: the fake must serialize exactly the SDK's parse shape.
type documentVersionPayload struct {
	ID       DocumentVersionID `json:"id"`
	Checksum string            `json:"checksum"`
	IsRoot   bool              `json:"is_root"`
}

// effectiveChecksum returns the document's content checksum across API
// shapes: the flat field served by older paperless-ngx versions first, then
// the root version's checksum (3.x), then any non-empty version checksum.
func (p documentMetaPayload) effectiveChecksum() string {
	//nolint:branching-flow // wire strings at the parse boundary; branded slots are the v2 typed-API scope (ROADMAP)
	return checksumFrom(p.Checksum, p.Versions)
}

// parseDocumentCreated parses the created value a stored document serves
// back: full RFC 3339 datetimes, timezone-less datetimes, and bare dates
// (uploads send date-only created fields). Timezone-less values are read
// as UTC (time.Parse's zero-offset default); Paperless-ngx stores its
// dates in UTC. Unparseable values yield the zero time.
func parseDocumentCreated(raw string) time.Time {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		time.DateOnly,
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed
		}
	}

	return time.Time{}
}

// DocumentMeta is the metadata snapshot of one stored Paperless-ngx document.
//
// Embedding the shared fields would change the public struct's composite-literal surface; unify them in the
// ROADMAP v2 typed-API scope instead.
type DocumentMeta struct {
	ID             DocumentID
	Title          string
	Correspondent  CorrespondentID
	Created        time.Time
	TagIDs         []TagID
	DocumentTypeID DocumentTypeID
	CustomFields   []CustomFieldValue
	Checksum       string
	// Content is the document's OCR text as served (effective_content in
	// paperless-ngx: the current version's text, "" when not OCR'd).
	Content string
}

// ListDocumentMetas returns id, title, correspondent, created date, tags,
// checksum, and OCR content of every stored document. The backfill matches
// ledger records against server documents (by checksum) and detects
// duplicates (same checksum stored more than once).
func (c *Client) ListDocumentMetas(ctx context.Context) ([]DocumentMeta, error) {
	entries, err := fetchAllPages[documentMetaPayload](ctx, c, pathDocuments, "document meta")
	if err != nil {
		return nil, err
	}

	metas := make([]DocumentMeta, 0, len(entries))
	for i := range entries {
		entry := &entries[i]

		metas = append(metas, DocumentMeta{
			ID:            entry.ID,
			Title:         entry.Title,
			Correspondent: entry.Correspondent,
			//nolint:branching-flow // RFC3339 wire string parsed at the boundary; branded slots are the v2 typed-API scope
			Created:        parseDocumentCreated(entry.Created),
			TagIDs:         entry.Tags,
			DocumentTypeID: entry.DocumentType,
			CustomFields:   customFieldsFromPayload(entry.CustomFields),
			Checksum:       entry.effectiveChecksum(),
			Content:        entry.Content,
		})
	}

	return metas, nil
}

// UpdateDocumentRequest carries the fields a metadata backfill may change.
// Nil pointers leave the field untouched; TagIDs replaces the full tag set
// (Paperless-ngx PATCH semantics for many-to-many fields).
//
// Embedding the shared fields would change the public struct's composite-literal surface; unify them in the
// ROADMAP v2 typed-API scope instead.
type UpdateDocumentRequest struct {
	Title           *string
	Created         *time.Time
	CorrespondentID *CorrespondentID
	TagIDs          []TagID
	DocumentTypeID  *DocumentTypeID
	CustomFields    []CustomFieldValue
}

// UpdateDocument PATCHes one document's metadata. Empty requests are
// rejected so callers never burn a round trip on a no-op.
func (c *Client) UpdateDocument(
	ctx context.Context,
	documentID DocumentID,
	req UpdateDocumentRequest,
) error {
	if req.Title == nil && req.Created == nil && req.CorrespondentID == nil && req.TagIDs == nil &&
		req.DocumentTypeID == nil && req.CustomFields == nil {
		return errorfamily.NewRejection("paperless.empty_update", "no metadata fields to update").
			WithContext("document_id", strconv.FormatInt(documentID.Get(), 10))
	}

	body, err := json.Marshal(buildDocumentUpdatePayload(req))
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "paperless.marshal_document_update", "could not encode document update").
			WithContext("document_id", strconv.FormatInt(documentID.Get(), 10))
	}

	if _, err := c.doRequest(
		ctx,
		http.MethodPatch,
		fmt.Sprintf(pathDocumentDetail, documentID.Get()),
		"",
		bytes.NewReader(body),
		"application/json",
	); err != nil {
		return fmt.Errorf(
			"update document %d: %w",
			documentID.Get(),
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	return nil
}

// documentUpdatePayload mirrors the writable subset of Paperless-ngx's
// document serializer. Pointer fields are emitted only when set; an empty
// TagIDs slice is omitted so other-field updates never touch tags.
//
// The ID pointer fields carry NO omitempty — deliberately. A pointer to a
// zero branded ID marshals as null (the FK-clearing contract), and under
// encoding/json/v2 an omitempty field whose encoding is null is dropped
// entirely, which would silently turn a clearing update into a no-op. A
// nil pointer is omitted by the standard rules anyway; the pointer-to-zero
// null must survive.
//
// SDK↔fake wire-shape mirror is deliberate: the fake must serialize exactly the SDK's parse shape.
type documentUpdatePayload struct {
	Title         *string                   `json:"title,omitempty"`
	Created       *string                   `json:"created,omitempty"`
	Correspondent *CorrespondentID          `json:"correspondent"`
	TagIDs        []TagID                   `json:"tags,omitempty"`
	DocumentType  *DocumentTypeID           `json:"document_type"`
	CustomFields  []customFieldValuePayload `json:"custom_fields,omitempty"`
}

// customFieldValuePayload is the wire shape of one document custom-field
// value (paperless-ngx 2.x+): the field definition ID plus the value.
//
// Unexported wire mirror of the public struct; the JSON tags belong here until v2 unifies the API.
type customFieldValuePayload struct {
	Field CustomFieldID `json:"field"`
	Value string        `json:"value"`
}

// customFieldsFromPayload converts the wire shape to the public type.
func customFieldsFromPayload(fields []customFieldValuePayload) []CustomFieldValue {
	if len(fields) == 0 {
		return nil
	}

	result := make([]CustomFieldValue, 0, len(fields))
	for _, field := range fields {
		result = append(result, CustomFieldValue(field))
	}

	return result
}

// buildDocumentUpdatePayload converts the caller-facing update request into
// its wire shape; only the fields the caller set are included.
func buildDocumentUpdatePayload(req UpdateDocumentRequest) documentUpdatePayload {
	payload := documentUpdatePayload{}

	if req.Title != nil {
		payload.Title = req.Title
	}

	if req.Created != nil {
		formatted := req.Created.Format(time.DateOnly)
		payload.Created = &formatted
	}

	if req.CorrespondentID != nil {
		id := *req.CorrespondentID
		payload.Correspondent = &id
	}

	if req.DocumentTypeID != nil {
		id := *req.DocumentTypeID
		payload.DocumentType = &id
	}

	if req.CustomFields != nil {
		payload.CustomFields = make([]customFieldValuePayload, 0, len(req.CustomFields))
		for _, field := range req.CustomFields {
			payload.CustomFields = append(payload.CustomFields, customFieldValuePayload(field))
		}
	}

	payload.TagIDs = req.TagIDs

	return payload
}

// DeleteDocument permanently removes one document from Paperless-ngx.
// Used by the backfill's --prune to drop duplicate server-side copies;
// callers own the destructive-action decision.
func (c *Client) DeleteDocument(ctx context.Context, documentID DocumentID) error {
	if _, err := c.doRequest(
		ctx,
		http.MethodDelete,
		fmt.Sprintf(pathDocumentDetail, documentID.Get()),
		"",
		nil,
		"",
	); err != nil {
		return fmt.Errorf(
			"delete document %d: %w",
			documentID.Get(),
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	return nil
}

// DownloadDocument fetches one document's stored ORIGINAL file bytes
// (GET /api/documents/{id}/download/ — not the archive rendition). The
// decrypt-repair scans them for the /Encrypt trailer before replacing.
func (c *Client) DownloadDocument(ctx context.Context, documentID DocumentID) ([]byte, error) {
	raw, err := c.doRequest(
		ctx,
		http.MethodGet,
		fmt.Sprintf(pathDocumentDownload, documentID.Get()),
		"",
		nil,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"download document %d: %w",
			documentID.Get(),
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	return raw, nil
}
