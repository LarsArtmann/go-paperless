package paperlesstest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"slices"
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

// AddDocument stores one document fixture at runtime (the locking
// equivalent of WithDocuments for mid-test seeding) and returns the stored
// copy with its assigned ID and derived checksum.
func (s *Server) AddDocument(doc Document) Document {
	s.mu.Lock()
	defer s.mu.Unlock()

	return *s.appendDocumentLocked(doc)
}

// EditDocument applies edit to one stored document fixture in place and
// reports whether the document exists. Use it to reshape server state
// mid-test (aging metadata, corrupting titles). The edit function runs
// under the fake's lock — it must not call back into the server.
func (s *Server) EditDocument(id int, edit func(doc *Document)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc := s.findDocumentLocked(id)
	if doc == nil {
		return false
	}

	edit(doc)

	return true
}

// addDocument stores one document fixture, assigning an ID and a derived
// checksum when unset. It is the locking entry point for tests and options.
func (s *Server) addDocument(doc Document) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.appendDocumentLocked(doc)
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
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  any     `json:"results"`
}

// defaultPageSize mirrors the SDK's list scan page size (100); DRF also
// defaults to 100 when the client omits page_size.
const defaultPageSize = 100

// pageBounds resolves the DRF page/page_size query parameters into slice
// bounds for a collection of the given size.
func pageBounds(r *http.Request, total int) (int, int) {
	query := r.URL.Query()

	page := 1
	if parsed, err := strconv.Atoi(query.Get("page")); err == nil && parsed > 0 {
		page = parsed
	}

	size := defaultPageSize
	if parsed, err := strconv.Atoi(query.Get("page_size")); err == nil && parsed > 0 {
		size = parsed
	}

	start := min((page-1)*size, total)

	return start, min(start+size, total)
}

// writePage serves one paginated envelope; results returns the page's
// entries for the resolved bounds. The caller must hold s.mu when the
// closure reads server state.
func (s *Server) writePage(
	w http.ResponseWriter,
	r *http.Request,
	total int,
	results func(start, end int) any,
) {
	start, end := pageBounds(r, total)
	s.writeJSON(w, http.StatusOK, envelope{
		Count:   total,
		Results: results(start, end),
	})
}

// routeDocuments serves the document list and document detail endpoints.
// It reports whether the request matched one of the document routes.
func (s *Server) routeDocuments(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/api/documents/" {
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

	if strings.HasPrefix(r.URL.Path, "/api/documents/") {
		return s.routeDocumentDetail(w, r)
	}

	return false
}

// DocumentPatch records one PATCH a test made against a document detail
// route.
type DocumentPatch struct {
	// DocumentID is the patched document.
	DocumentID int
	// Body is the raw JSON patch payload as received.
	Body []byte
}

// Patches returns a copy of every recorded document patch, oldest first.
func (s *Server) Patches() []DocumentPatch {
	s.mu.Lock()
	defer s.mu.Unlock()

	patches := make([]DocumentPatch, 0, len(s.patches))
	for _, patch := range s.patches {
		copied := patch
		copied.Body = slices.Clone(patch.Body)
		patches = append(patches, copied)
	}

	return patches
}

// DeletedDocuments returns the IDs of every document the fake has deleted
// via the detail route, oldest first.
func (s *Server) DeletedDocuments() []int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.deletedDocuments)
}

// documentUpdateWire mirrors the writable subset of Paperless-ngx's
// document serializer: pointer fields apply only when the patch carries
// them, and TagIDs replaces the full tag set (DRF many-to-many PATCH
// semantics).
type documentUpdateWire struct {
	Title         *string            `json:"title"`
	Created       *string            `json:"created"`
	Correspondent *int               `json:"correspondent"`
	Tags          []int              `json:"tags"`
	DocumentType  *int               `json:"document_type"`
	CustomFields  []CustomFieldValue `json:"custom_fields"`
}

// routeDocumentDetail serves the per-document routes (PATCH/GET/DELETE on
// /api/documents/{id}/ and the download route). Unknown documents answer a
// legitimate 404 (no test failure — clients handle 404s).
func (s *Server) routeDocumentDetail(w http.ResponseWriter, r *http.Request) bool {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/documents/"), "/")
	segments := strings.Split(rest, "/")

	id, err := strconv.Atoi(segments[0])
	if err != nil {
		return false
	}

	switch {
	case len(segments) == 1:
		return s.routeDocumentByID(w, r, id)
	case len(segments) == 2 && segments[1] == "notes":
		return s.routeDocumentNotes(w, r, id)
	case len(segments) == 2 && segments[1] == "download":
		if r.Method != http.MethodGet {
			s.methodNotAllowed(w, r, http.MethodGet)

			return true
		}

		s.serveDocumentDownload(w, id)

		return true
	default:
		return false
	}
}

// routeDocumentByID serves the PATCH/GET/DELETE surface of one document.
func (s *Server) routeDocumentByID(w http.ResponseWriter, r *http.Request, id int) bool {
	switch r.Method {
	case http.MethodPatch:
		s.handleDocumentPatch(w, r, id)

		return true
	case http.MethodDelete:
		s.handleDocumentDelete(w, id)

		return true
	case http.MethodGet:
		s.handleDocumentGet(w, id)

		return true
	default:
		s.methodNotAllowed(w, r, http.MethodGet, http.MethodPatch, http.MethodDelete)

		return true
	}
}

// routeDocumentNotes serves one document's notes routes under the fake's
// lock (the notes handlers read the document store directly).
func (s *Server) routeDocumentNotes(w http.ResponseWriter, r *http.Request, id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.notes == nil {
		s.notes = map[int][]noteEntity{}
	}

	return s.handleDocumentNotesLocked(w, r, id)
}

// handleDocumentPatch applies and records one metadata patch. The caller
// must not hold s.mu.
func (s *Server) handleDocumentPatch(w http.ResponseWriter, r *http.Request, id int) {
	raw := readBody(r)

	var patch documentUpdateWire
	if err := json.Unmarshal(raw, &patch); err != nil {
		s.t.Errorf("paperlesstest: decode document patch: %v", err)
		s.writeJSON(w, http.StatusBadRequest, map[string]string{detailKey: "invalid patch payload"})

		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	doc := s.findDocumentLocked(id)
	if doc == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

		return
	}

	if patch.Title != nil {
		doc.Title = *patch.Title
	}

	if patch.Created != nil {
		doc.Created = parseServedCreated(*patch.Created)
	}

	if patch.Correspondent != nil {
		doc.Correspondent = *patch.Correspondent
	}

	if patch.Tags != nil {
		doc.TagIDs = patch.Tags
	}

	if patch.DocumentType != nil {
		doc.DocumentTypeID = *patch.DocumentType
	}

	if patch.CustomFields != nil {
		doc.CustomFields = patch.CustomFields
	}

	s.patches = append(s.patches, DocumentPatch{DocumentID: id, Body: slices.Clone(raw)})

	s.writeJSON(w, http.StatusOK, s.documentWireFor(doc))
}

// handleDocumentDelete removes one document and records the deletion.
func (s *Server) handleDocumentDelete(w http.ResponseWriter, id int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.findDocumentLocked(id) == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

		return
	}

	s.documents = slices.DeleteFunc(s.documents, func(doc *Document) bool {
		return doc.ID == id
	})

	s.deletedDocuments = append(s.deletedDocuments, id)

	w.WriteHeader(http.StatusNoContent)
}

// handleDocumentGet serves one document's serialized form.
func (s *Server) handleDocumentGet(w http.ResponseWriter, id int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc := s.findDocumentLocked(id)
	if doc == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

		return
	}

	s.writeJSON(w, http.StatusOK, s.documentWireFor(doc))
}

// serveDocumentDownload serves one document's stored original bytes.
func (s *Server) serveDocumentDownload(w http.ResponseWriter, id int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc := s.findDocumentLocked(id)
	if doc == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(doc.Content); err != nil {
		s.t.Errorf("paperlesstest: write download body: %v", err)
	}
}

// findDocumentLocked locates one stored document by ID, or nil. The caller
// must hold s.mu.
func (s *Server) findDocumentLocked(id int) *Document {
	for _, doc := range s.documents {
		if doc.ID == id {
			return doc
		}
	}

	return nil
}

// parseServedCreated parses the created values the fake serves/accepts
// (RFC 3339 and date-only, mirroring the SDK's parser tolerance).
func parseServedCreated(raw string) time.Time {
	for _, layout := range []string{time.RFC3339, time.DateOnly} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed
		}
	}

	return time.Time{}
}

// methodNotAllowed rejects a wrong method on a known path: the test code
// under development called the fake incorrectly, so the test fails.
func (s *Server) methodNotAllowed(w http.ResponseWriter, r *http.Request, allowed ...string) {
	s.t.Errorf(
		"paperlesstest: unexpected method %s on %s (allowed: %v)",
		r.Method,
		r.URL.Path,
		allowed,
	)
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	s.writeJSON(
		w,
		http.StatusMethodNotAllowed,
		map[string]string{detailKey: fmt.Sprintf("Method %q not allowed.", r.Method)},
	)
}
