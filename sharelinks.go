package paperless

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strconv"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ShareLinkFileVersion names which rendition of a document a share link
// exposes. The server's default (and the zero-value fallback it applies
// when the field is omitted) is the archive version.
type ShareLinkFileVersion string

const (
	// ShareLinkFileVersionArchive exposes the OCR'd archive rendition.
	ShareLinkFileVersionArchive ShareLinkFileVersion = "archive"
	// ShareLinkFileVersionOriginal exposes the untouched original file.
	ShareLinkFileVersionOriginal ShareLinkFileVersion = "original"
)

// ShareLink is one public download link for a document. Slug is
// server-generated; the full share URL is <base>/share/<slug>.
//
// Embedding the shared fields would change the public struct's composite-literal surface; unify them in the
// ROADMAP v2 typed-API scope instead.
type ShareLink struct {
	ID      ShareLinkID
	Created time.Time
	// Expiration is when the link stops working; zero means it never
	// expires.
	Expiration time.Time
	Slug       string
	DocumentID DocumentID
	// FileVersion says which rendition the link serves (archive or
	// original).
	FileVersion ShareLinkFileVersion
}

// shareLinkPayload mirrors the ShareLink serializer fields the SDK relies
// on (id, created, expiration, slug, document, file_version).
//
// SDK↔fake wire-shape mirror is deliberate: the fake must serialize exactly the SDK's parse shape.
type shareLinkPayload struct {
	ID          ShareLinkID `json:"id"`
	Created     time.Time   `json:"created"`
	Expiration  *time.Time  `json:"expiration"`
	Slug        string      `json:"slug"`
	Document    DocumentID  `json:"document"`
	FileVersion string      `json:"file_version"`
}

func (p shareLinkPayload) shareLink() ShareLink {
	link := ShareLink{
		ID:          p.ID,
		Created:     p.Created,
		Slug:        p.Slug,
		DocumentID:  p.Document,
		FileVersion: ShareLinkFileVersion(p.FileVersion),
	}

	if p.Expiration != nil {
		link.Expiration = *p.Expiration
	}

	return link
}

// CreateShareLinkRequest asks for one share link. A zero FileVersion is
// omitted from the POST so the server applies its default (archive); a
// nil Expiration means the link never expires.
type CreateShareLinkRequest struct {
	DocumentID  DocumentID
	FileVersion ShareLinkFileVersion
	Expiration  *time.Time
}

// ListShareLinks returns every share link on the server, paginating the
// same bounded way as the document listings.
func (c *Client) ListShareLinks(ctx context.Context) ([]ShareLink, error) {
	payloads, listErr := fetchAllPages[shareLinkPayload](ctx, c, pathShareLinks, "share link")
	if listErr != nil {
		return nil, listErr
	}

	links := make([]ShareLink, 0, len(payloads))

	for _, payload := range payloads {
		links = append(links, payload.shareLink())
	}

	return links, nil
}

// CreateShareLink makes one public download link for a document. The
// server generates the slug — the POST deliberately sends only the
// document, the optional file version, and the optional expiration.
func (c *Client) CreateShareLink(
	ctx context.Context,
	req CreateShareLinkRequest,
) (ShareLink, error) {
	if req.DocumentID.IsZero() {
		return ShareLink{}, errorfamily.NewRejection("paperless.empty_document_id",
			"document ID is required to create a share link")
	}

	payload, err := json.Marshal(struct {
		Document    DocumentID           `json:"document"`
		FileVersion ShareLinkFileVersion `json:"file_version,omitempty"`
		Expiration  *time.Time           `json:"expiration,omitempty"`
	}{
		Document:    req.DocumentID,
		FileVersion: req.FileVersion,
		Expiration:  req.Expiration,
	})
	if err != nil {
		return ShareLink{}, errorfamily.WrapInfrastructure(err, "paperless.marshal_share_link",
			"could not encode the share link payload").
			WithContext("document", strconv.FormatInt(req.DocumentID.Get(), 10))
	}

	raw, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathShareLinks,
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return ShareLink{}, fmt.Errorf(
			"create share link (document %d): %w", //nolint:erraudit // keeps inner code+family
			req.DocumentID.Get(),
			err,
		)
	}

	var created shareLinkPayload
	if err := json.Unmarshal(raw, &created); err != nil {
		return ShareLink{}, errorfamily.WrapCorruption(err, "paperless.decode_share_link",
			"could not decode the created share link").
			WithContext("document", strconv.FormatInt(req.DocumentID.Get(), 10))
	}

	return created.shareLink(), nil
}

// DeleteShareLink revokes one public link; every consumer of the slug
// loses access immediately.
func (c *Client) DeleteShareLink(ctx context.Context, linkID ShareLinkID) error {
	if linkID.IsZero() {
		return errorfamily.NewRejection("paperless.empty_share_link_id",
			"share link ID is required to delete a share link")
	}

	if _, err := c.doRequest(
		ctx,
		http.MethodDelete,
		pathShareLinks+strconv.Itoa(linkID.Get())+"/",
		"",
		nil,
		"",
	); err != nil {
		return fmt.Errorf(
			"delete share link %d: %w",
			linkID.Get(),
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	return nil
}
