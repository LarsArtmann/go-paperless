package paperlesstest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CustomFieldValue is one custom-field value on a document fixture: the
// custom-field definition ID plus the value assigned to it. It mirrors the
// wire shape ({"field": N, "value": "..."}).
type CustomFieldValue struct {
	Field int    `json:"field"`
	Value string `json:"value"`
}

// Document is a stored-document fixture. Seed documents via WithDocuments
// or AddDocument; uploads that succeed through the fake's natural (unscripted)
// consumption path create documents of this shape too.
//
// An empty Checksum is derived from Content (SHA-256 hex, the derivation
// the real server applies) when the document is stored; a Checksum set by
// the test is kept verbatim.
type Document struct {
	ID int
	// Title is the stored document title (the real server derives it from
	// the upload filename when the upload sends no title).
	Title string
	// Content is the stored original file bytes, served by the download
	// endpoint.
	Content []byte
	// Checksum is the document's content checksum. Empty means "derive
	// from Content".
	Checksum string
	// Created is the document's creation date.
	Created time.Time
	// Correspondent is the assigned correspondent ID (0 = none).
	Correspondent int
	// TagIDs are the assigned tag IDs.
	TagIDs []int
	// DocumentTypeID is the assigned document type ID (0 = none).
	DocumentTypeID int
	// CustomFields are the assigned custom-field values.
	CustomFields []CustomFieldValue
}

// ChecksumOf derives the content checksum the real Paperless-ngx assigns on
// consumption (SHA-256 hex).
func ChecksumOf(content []byte) string {
	sum := sha256.Sum256(content)

	return hex.EncodeToString(sum[:])
}

// Documents returns a copy of every stored document fixture, oldest first.
// Mutating the copies does not affect the fake.
func (s *Server) Documents() []Document {
	s.mu.Lock()
	defer s.mu.Unlock()

	documents := make([]Document, 0, len(s.documents))
	for _, doc := range s.documents {
		copied := *doc
		copied.Content = slices.Clone(doc.Content)
		copied.TagIDs = slices.Clone(doc.TagIDs)
		copied.CustomFields = slices.Clone(doc.CustomFields)
		documents = append(documents, copied)
	}

	return documents
}

// addDocument stores one document fixture, assigning an ID and a derived
// checksum when unset. It is the locking entry point for tests and options.
func (s *Server) addDocument(doc Document) *Document {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.appendDocumentLocked(doc)
}

// appendDocumentLocked stores one document fixture. The caller must hold
// s.mu.
func (s *Server) appendDocumentLocked(doc Document) *Document {
	if doc.ID <= 0 {
		s.nextDocumentID++
		doc.ID = s.nextDocumentID
	} else if doc.ID > s.nextDocumentID {
		s.nextDocumentID = doc.ID
	}

	if doc.Checksum == "" && doc.Content != nil {
		doc.Checksum = ChecksumOf(doc.Content)
	}

	stored := doc
	s.documents = append(s.documents, &stored)

	return &stored
}

// documentWire is the serialized document shape served on the wire. The
// checksum fields follow the server's ChecksumShape: flat field
// (pre-3.x) or versions[] array (3.x) — never both.
type documentWire struct {
	ID            int                `json:"id"`
	Title         string             `json:"title"`
	Correspondent int                `json:"correspondent"`
	Created       string             `json:"created"`
	Tags          []int              `json:"tags"`
	DocumentType  int                `json:"document_type"`
	CustomFields  []CustomFieldValue `json:"custom_fields"`
	// Checksum is the flat legacy checksum field; omitted in the versions[]
	// shape (paperless-ngx 3.x removed it from the serializer).
	Checksum string `json:"checksum,omitempty"`
	// Versions carries the checksum inside the document's version history;
	// omitted in the flat shape.
	Versions []versionWire `json:"versions,omitempty"`
}

// versionWire is one entry of the document's versions[] array.
type versionWire struct {
	ID       int    `json:"id"`
	Checksum string `json:"checksum"`
	IsRoot   bool   `json:"is_root"`
}

// documentWireFor serializes one stored document under the server's
// checksum shape. The caller must hold s.mu.
func (s *Server) documentWireFor(doc *Document) documentWire {
	wire := documentWire{
		ID:            doc.ID,
		Title:         doc.Title,
		Correspondent: doc.Correspondent,
		Created:       doc.Created.Format(time.RFC3339),
		Tags:          doc.TagIDs,
		DocumentType:  doc.DocumentTypeID,
		CustomFields:  doc.CustomFields,
	}

	if wire.Tags == nil {
		wire.Tags = []int{}
	}

	if wire.CustomFields == nil {
		wire.CustomFields = []CustomFieldValue{}
	}

	switch s.shape {
	case ChecksumVersions:
		if doc.Checksum != "" {
			wire.Versions = []versionWire{{
				ID:       doc.ID,
				Checksum: doc.Checksum,
				IsRoot:   true,
			}}
		}
	case ChecksumFlat:
		wire.Checksum = doc.Checksum
	}

	return wire
}

// documentWires serializes the documents slice as wire payloads. The caller
// must hold s.mu.
func (s *Server) documentWires(documents []*Document) []documentWire {
	wires := make([]documentWire, 0, len(documents))
	for _, doc := range documents {
		wires = append(wires, s.documentWireFor(doc))
	}

	return wires
}

// envelope is the DRF pagination envelope every paginated list endpoint
// serves. Next/previous stay null: the SDK walks pages via the page query
// parameter and terminates on the short page, never on the links.
type envelope struct {
	Count    int   `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  any   `json:"results"`
}

// defaultPageSize mirrors the SDK's list scan page size (100); DRF also
// defaults to 100 when the client omits page_size.
const defaultPageSize = 100

// pageBounds resolves the DRF page/page_size query parameters into slice
// bounds for a collection of the given size.
func pageBounds(r *http.Request, total int) (start, end int) {
	query := r.URL.Query()

	page := 1
	if parsed, err := strconv.Atoi(query.Get("page")); err == nil && parsed > 0 {
		page = parsed
	}

	size := defaultPageSize
	if parsed, err := strconv.Atoi(query.Get("page_size")); err == nil && parsed > 0 {
		size = parsed
	}

	start = (page - 1) * size
	if start > total {
		start = total
	}

	end = start + size
	if end > total {
		end = total
	}

	return start, end
}

// writePage serves one paginated envelope; results returns the page's
// entries for the resolved bounds. The caller must hold s.mu when the
// closure reads server state.
func (s *Server) writePage(w http.ResponseWriter, r *http.Request, total int, results func(start, end int) any) {
	start, end := pageBounds(r, total)
	s.writeJSON(w, http.StatusOK, envelope{
		Count:   total,
		Results: results(start, end),
	})
}

// routeDocuments serves the document list endpoint. It reports whether the
// request matched one of the document routes.
func (s *Server) routeDocuments(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/api/documents/" {
		return false
	}

	if r.Method != http.MethodGet {
		s.methodNotAllowed(w, r, http.MethodGet)

		return true
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.writePage(w, r, len(s.documents), func(start, end int) any {
		return s.documentWires(s.documents[start:end])
	})

	return true
}

// methodNotAllowed rejects a wrong method on a known path: the test code
// under development called the fake incorrectly, so the test fails.
func (s *Server) methodNotAllowed(w http.ResponseWriter, r *http.Request, allowed ...string) {
	s.t.Errorf("paperlesstest: unexpected method %s on %s (allowed: %v)", r.Method, r.URL.Path, allowed)
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": fmt.Sprintf("Method %q not allowed.", r.Method)})
}
