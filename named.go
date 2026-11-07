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

	id "github.com/larsartmann/go-branded-id"
	errorfamily "github.com/larsartmann/go-error-family"
)

// EnsureTag returns the ID of the tag with the given name, creating it when
// it does not exist yet.
//
// Tags are provenance metadata applied deterministically at upload time, so
// they are created with the "none" matching algorithm: a "gmail" tag that
// Paperless-ngx's classifier learns to predict would silently leak onto
// manually scanned documents that merely look similar. An existing tag still
// carrying "auto" (created by older InboxClean versions) is self-healed to
// "none" — the classifier then no longer trains against it. Self-healing is
// tag-specific on purpose: correspondents deliberately keep "auto".
func (c *Client) EnsureTag(ctx context.Context, name string) (TagID, error) {
	existing, found, findErr := findNamed[TagBrand](c, ctx, pathTags, "tag", name)
	if findErr != nil {
		return TagID{}, fmt.Errorf(
			"find tag %q: %w",
			name,
			findErr,
		) //nolint:erraudit // keeps inner code+family
	}

	if found {
		if existing.MatchingAlgorithm == int(matchingAlgorithmAuto) {
			if err := updateMatchingAlgorithm(
				c,
				ctx,
				pathTags,
				"tag",
				name,
				existing.ID,
				matchingAlgorithmNone,
			); err != nil {
				return TagID{}, fmt.Errorf(
					"demote auto tag %q: %w",
					name,
					err,
				) //nolint:erraudit // keeps inner code+family
			}
		}

		return existing.ID, nil
	}

	return createNamed[TagBrand](c, ctx, pathTags, "tag", name, matchingAlgorithmNone)
}

// Tag is one tag definition on the server: the ID consumers resolve document
// tag ID lists against (the document list endpoint returns tag IDs, not
// names, so rendering a tag name means joining against this list).
type Tag struct {
	ID   TagID
	Name string
}

// tagPayload mirrors Paperless-ngx's tag serializer fields the SDK relies on.
//
// Unexported wire mirror of the public struct; the JSON tags belong here until v2 unifies the API.
type tagPayload struct {
	ID   TagID  `json:"id"`
	Name string `json:"name"`
}

// ListTags returns every tag definition on the server, paginating the same
// bounded way as the other list endpoints. Callers needing name filters on
// documents (tags arrive as IDs there) resolve them through this list.
func (c *Client) ListTags(ctx context.Context) ([]Tag, error) {
	payloads, listErr := fetchAllPages[tagPayload](ctx, c, pathTags, "tag")
	if listErr != nil {
		return nil, listErr
	}

	tags := make([]Tag, 0, len(payloads))
	for _, payload := range payloads {
		tags = append(tags, Tag(payload))
	}

	return tags, nil
}

// updateMatchingAlgorithm PATCHes an existing object's matching algorithm
// (used to self-heal legacy auto tags down to none). The brand parameter
// keeps the PATCHed ID within the caller's entity family; it is a free
// function because the format toolchain (golines) does not parse generic
// methods even though the Go 1.27 compiler accepts them.
func updateMatchingAlgorithm[
	B any,
](
	client *Client,
	ctx context.Context,
	endpoint,
	kind,
	name string,
	entityID id.ID[B, int],
	algorithm matchingAlgorithm,
) error {
	payload, err := json.Marshal(matchingAlgorithmPatch{MatchingAlgorithm: int(algorithm)})
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"paperless.marshal_"+kind+"_update",
			"could not encode "+kind+" update payload",
		).WithContext(kind, name)
	}

	if _, reqErr := client.doRequest(
		ctx,
		http.MethodPatch,
		endpoint+strconv.Itoa(entityID.Get())+"/",
		"",
		bytes.NewReader(payload),
		"application/json",
	); reqErr != nil {
		return fmt.Errorf(
			"update %s %q: %w",
			kind,
			name,
			reqErr,
		) //nolint:erraudit // keeps inner code+family
	}

	return nil
}

// namedPayload mirrors the create/list subset shared by Paperless-ngx's tag and
// correspondent objects (id + name + matching algorithm). The brand
// parameter binds the decoded ID to one entity family per call site.
//
// Unexported wire mirror; the fake re-declares it because unexported types cannot be reused across packages.
type namedPayload[B any] struct {
	ID                id.ID[B, int] `json:"id"`
	Name              string        `json:"name"`
	MatchingAlgorithm int           `json:"matching_algorithm"`
}

// matchingAlgorithmPatch is the updateMatchingAlgorithm request body. It
// carries ONLY the matching algorithm: Paperless-ngx's DRF serializer rejects
// a PATCH that re-sends a blank required name ("name":"") with 400 Bad
// Request, which silently broke the legacy auto-tag self-heal on every sync
// tick (live 2026-09-17: "Bad Request: /api/tags/1/" once per 30 min).
type matchingAlgorithmPatch struct {
	MatchingAlgorithm int `json:"matching_algorithm"`
}

// findByName is the shared query+doRequest+decode skeleton behind the
// by-name finders (tags, correspondents, document types, custom fields,
// storage paths): exact case-insensitive name match with page_size=1,
// returning the first result. Request errors return unwrapped so callers
// can add their own context; decode failures are wrapped as corruption with
// a kind-derived code ("paperless.decode_<kind with underscores>s").
func findByName[T any](
	c *Client,
	ctx context.Context,
	endpoint, kind, name string,
) (T, bool, error) {
	var zero T

	query := url.Values{}
	query.Set("name__iexact", name)
	query.Set("page_size", "1")

	raw, reqErr := c.doRequest(ctx, http.MethodGet, endpoint, query.Encode(), nil, "")
	if reqErr != nil {
		return zero, false, reqErr
	}

	list := struct {
		Results []T `json:"results"`
	}{}

	if unmarshalErr := json.Unmarshal(raw, &list); unmarshalErr != nil {
		return zero, false, errorfamily.WrapCorruption(
			unmarshalErr,
			"paperless.decode_"+strings.ReplaceAll(kind, " ", "_")+"s",
			"could not decode "+kind+" search result",
		).WithContext("name", name)
	}

	if len(list.Results) == 0 {
		return zero, false, nil
	}

	return list.Results[0], true, nil
}

// findNamed looks up a Paperless-ngx named object (tag, correspondent) by
// exact (case-insensitive) name. found is false when no match exists.
// Free generic function: the format toolchain does not parse generic
// methods.
func findNamed[B any](
	client *Client,
	ctx context.Context,
	endpoint, kind, name string,
) (namedPayload[B], bool, error) {
	return findByName[namedPayload[B]](client, ctx, endpoint, kind, name)
}

// EnsureCorrespondent returns the ID of the correspondent with the given
// name, creating it when it does not exist yet.
//
// Correspondents keep the "auto" matching algorithm (deliberately, unlike
// tags): learning "documents from sender X look like this" is exactly the
// value auto matching adds for documents NOT uploaded by this pipeline
// (manual scans), and correspondents carry no provenance semantics that
// auto prediction could corrupt.
func (c *Client) EnsureCorrespondent(ctx context.Context, name string) (CorrespondentID, error) {
	return ensureNamed[CorrespondentBrand](c, ctx, pathCorrespondents, "correspondent", name)
}

// EnsureDocumentType returns the ID of the document type with the given
// name, creating it with the "none" matching algorithm when missing. A
// document type applied deterministically at upload time is provenance
// metadata: classifier-learned matching would leak the type onto manual
// scans, mirroring the tags rationale. An EXISTING type keeps its
// configured algorithm — unlike tags there is no legacy-release shape to
// heal, and overriding a deliberate user config would be wrong.
func (c *Client) EnsureDocumentType(ctx context.Context, name string) (DocumentTypeID, error) {
	existing, found, findErr := findNamed[DocumentTypeBrand](
		c,
		ctx,
		pathDocumentTypes,
		"document type",
		name,
	)
	if findErr != nil {
		return DocumentTypeID{}, fmt.Errorf(
			"find document type %q: %w",
			name,
			findErr,
		) //nolint:erraudit // keeps inner code+family
	}

	if found {
		return existing.ID, nil
	}

	return createNamed[DocumentTypeBrand](
		c,
		ctx,
		pathDocumentTypes,
		"document type",
		name,
		matchingAlgorithmNone,
	)
}

// customFieldPayload mirrors one custom field definition.
type customFieldPayload struct {
	ID       CustomFieldID `json:"id"`
	Name     string        `json:"name"`
	DataType string        `json:"data_type"`
}

// FindCustomField looks up a custom field definition by name WITHOUT
// creating it — the read-only half of EnsureCustomField, used by dry-run
// backfills to report what they would record without mutating the server.
func (c *Client) FindCustomField(ctx context.Context, name string) (CustomFieldID, bool, error) {
	field, found, err := findByName[customFieldPayload](
		c,
		ctx,
		pathCustomFields,
		"custom field",
		name,
	)
	if err != nil {
		return CustomFieldID{}, false, fmt.Errorf(
			"find custom field %q: %w",
			name,
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	if !found {
		return CustomFieldID{}, false, nil
	}

	return field.ID, true, nil //art-dupl:accept parallel tail of the shared findByName skeleton
}

// EnsureCustomField returns the ID of the custom field definition with the
// given name, creating it as a string field when missing (paperless-ngx 2.x+
// custom fields). Used for the provenance field carrying the Gmail message
// ID. An existing definition of any data type is kept — the caller decides
// whether a non-string field is acceptable by the value it assigns.
func (c *Client) EnsureCustomField(ctx context.Context, name string) (CustomFieldID, error) {
	fieldID, found, findErr := c.FindCustomField(ctx, name)
	if findErr != nil {
		return CustomFieldID{}, findErr
	}

	if found {
		return fieldID, nil
	}

	payload, err := json.Marshal(customFieldPayload{Name: name, DataType: "string"})
	if err != nil {
		return CustomFieldID{}, errorfamily.WrapInfrastructure(
			err,
			"paperless.marshal_custom_field",
			"could not encode custom field payload",
		).WithContext("name", name)
	}

	raw, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathCustomFields,
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return CustomFieldID{}, fmt.Errorf(
			"create custom field %q: %w",
			name,
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	created := customFieldPayload{}
	if unmarshalErr := json.Unmarshal(raw, &created); unmarshalErr != nil {
		return CustomFieldID{}, errorfamily.WrapCorruption(
			unmarshalErr,
			"paperless.decode_custom_field",
			"could not decode created custom field",
		).WithContext("name", name)
	}

	return created.ID, nil
}

// StoragePath is one Paperless-ngx storage path definition: the server-side
// directory template (e.g. "{created_year}/{correspondent}") documents are
// filed into, addressed by name.
//
// The public API struct stays tag-free (the exported JSON shape is a compatibility contract); unification is
// the ROADMAP v2 typed-API scope.
type StoragePath struct {
	ID   StoragePathID
	Slug string
	Name string
	Path string
}

// storagePathPayload mirrors Paperless-ngx's storage path serializer fields
// the SDK relies on.
//
// Unexported wire mirror of the public struct; the JSON tags belong here until v2 unifies the API.
type storagePathPayload struct {
	ID   StoragePathID `json:"id"`
	Slug string        `json:"slug,omitempty"`
	Name string        `json:"name"`
	Path string        `json:"path"`
}

// FindStoragePath looks up a storage path by exact (case-insensitive) name
// WITHOUT creating it — the read-only lookup for dry-run reporting.
func (c *Client) FindStoragePath(ctx context.Context, name string) (StoragePathID, bool, error) {
	path, found, err := findByName[storagePathPayload](
		c,
		ctx,
		pathStoragePaths,
		"storage path",
		name,
	)
	if err != nil {
		return StoragePathID{}, false, fmt.Errorf(
			"find storage path %q: %w",
			name,
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	if !found {
		return StoragePathID{}, false, nil
	}

	return path.ID, true, nil //art-dupl:accept parallel tail of the shared findByName skeleton
}

// EnsureStoragePath returns the ID of the storage path with the given name,
// creating it with the given directory template when missing. An EXISTING
// path keeps its configured template — mirroring the document type policy:
// the template is deliberate user config, and overriding it from a sync
// pipeline would be wrong.

//nolint:branching-flow // public API; label vs template are distinct concepts; brands are ROADMAP v2 scope
func (c *Client) EnsureStoragePath(ctx context.Context, name, path string) (StoragePathID, error) {
	if name == "" || path == "" {
		return StoragePathID{}, errorfamily.NewRejection("paperless.empty_storage_path",
			"storage path name and directory template are required")
	}

	existing, found, findErr := c.FindStoragePath(ctx, name)
	if findErr != nil {
		return StoragePathID{}, fmt.Errorf(
			"find storage path %q: %w",
			name,
			findErr,
		) //nolint:erraudit // keeps inner code+family
	}

	if found {
		return existing, nil
	}

	payload, err := json.Marshal(storagePathPayload{Name: name, Path: path})
	if err != nil {
		return StoragePathID{}, errorfamily.WrapInfrastructure(
			err,
			"paperless.marshal_storage_path",
			"could not encode storage path payload",
		).WithContext("name", name)
	}

	raw, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathStoragePaths,
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return StoragePathID{}, fmt.Errorf(
			"create storage path %q: %w",
			name,
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	created := storagePathPayload{}
	if unmarshalErr := json.Unmarshal(raw, &created); unmarshalErr != nil {
		return StoragePathID{}, errorfamily.WrapCorruption(
			unmarshalErr,
			"paperless.decode_storage_path",
			"could not decode created storage path",
		).WithContext("name", name)
	}

	return created.ID, nil
}

// ListStoragePaths returns every storage path definition on the server,
// paginating the same bounded way as the document listings.
func (c *Client) ListStoragePaths(ctx context.Context) ([]StoragePath, error) {
	payloads, listErr := fetchAllPages[storagePathPayload](ctx, c, pathStoragePaths, "storage path")
	if listErr != nil {
		return nil, listErr
	}

	paths := make([]StoragePath, 0, len(payloads))
	for _, payload := range payloads {
		paths = append(paths, StoragePath(payload))
	}

	return paths, nil
}

// GetCorrespondentName resolves a correspondent ID to its display name.
// The backfill needs it to decide whether a document's existing non-null
// correspondent actually matches the derived sender name — an ID alone
// cannot answer that question.
func (c *Client) GetCorrespondentName(
	ctx context.Context,
	entityID CorrespondentID,
) (string, error) {
	return getNamedDetail[CorrespondentBrand](c, ctx, pathCorrespondents, "correspondent", entityID)
}

// GetDocumentTypeName resolves a document type ID to its display name, the
// document-type sibling of GetCorrespondentName: the backfill compares the
// stored name against the configured type to repair drifted assignments.
func (c *Client) GetDocumentTypeName(ctx context.Context, entityID DocumentTypeID) (string, error) {
	return getNamedDetail[DocumentTypeBrand](c, ctx, pathDocumentTypes, "document type", entityID)
}

// getNamedDetail fetches one named Paperless-ngx object (correspondent,
// document type) by ID and returns its display name. The brand parameter
// keeps the looked-up ID within the caller's entity family; free function
// because the format toolchain does not parse generic methods.
func getNamedDetail[B any](
	client *Client,
	ctx context.Context,
	endpoint, kind string,
	entityID id.ID[B, int],
) (string, error) {
	raw, err := client.doRequest(
		ctx,
		http.MethodGet,
		endpoint+strconv.Itoa(entityID.Get())+"/",
		"",
		nil,
		"",
	)
	if err != nil {
		return "", fmt.Errorf(
			"get %s %d: %w",
			kind,
			entityID.Get(),
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	detail := namedPayload[B]{}
	if unmarshalErr := json.Unmarshal(raw, &detail); unmarshalErr != nil {
		return "", errorfamily.WrapCorruption(
			unmarshalErr,
			"paperless.decode_"+strings.ReplaceAll(kind, " ", "_"),
			"could not decode "+kind,
		).WithContext("id", strconv.Itoa(entityID.Get()))
	}

	return detail.Name, nil
}

// ensureNamed returns the ID of the named Paperless-ngx object (tag,
// correspondent), creating it with the "auto" matching algorithm when it
// does not exist yet. kind names the object family for error codes,
// messages, and log context. Free generic function: the format toolchain
// does not parse generic methods.
func ensureNamed[B any](
	client *Client,
	ctx context.Context,
	endpoint, kind, name string,
) (id.ID[B, int], error) {
	existing, found, findErr := findNamed[B](client, ctx, endpoint, kind, name)
	if findErr != nil {
		return id.ID[B, int]{}, fmt.Errorf(
			"find %s %q: %w",
			kind,
			name,
			findErr,
		) //nolint:erraudit // keeps inner code+family
	}

	if found {
		return existing.ID, nil
	}

	return createNamed[B](client, ctx, endpoint, kind, name, matchingAlgorithmAuto)
}

// createNamed POSTs a new named object with the requested matching
// algorithm and returns its ID. Free generic function: the format
// toolchain does not parse generic methods.
func createNamed[
	B any,
](
	client *Client,
	ctx context.Context,
	endpoint,
	kind,
	name string,
	algorithm matchingAlgorithm,
) (id.ID[B, int], error) {
	payload, err := json.Marshal(namedPayload[B]{Name: name, MatchingAlgorithm: int(algorithm)})
	if err != nil {
		return id.ID[B, int]{}, errorfamily.WrapInfrastructure(
			err,
			"paperless.marshal_"+kind,
			"could not encode "+kind+" payload",
		).WithContext(kind, name)
	}

	raw, err := client.doRequest(
		ctx,
		http.MethodPost,
		endpoint,
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return id.ID[B, int]{}, fmt.Errorf(
			"create %s %q: %w",
			kind,
			name,
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	created := namedPayload[B]{}
	if err := json.Unmarshal(raw, &created); err != nil {
		return id.ID[B, int]{}, errorfamily.WrapCorruption(
			err,
			"paperless.decode_"+kind,
			"could not decode created "+kind,
		).WithContext(kind, name)
	}

	return created.ID, nil
}
