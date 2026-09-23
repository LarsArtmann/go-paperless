package paperlesstest

import (
	"bytes"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const (
	// DefaultToken is the API token consumers should configure on their
	// paperless.Client unless a test enforces a different one via WithToken.
	DefaultToken = "test-token"

	// DefaultAPIVersion is the API version the fake echoes in every JSON
	// response's Content-Type ("application/json; version=N"), matching the
	// SDK's Accept-header target.
	DefaultAPIVersion = "10"

	// detailKey is the DRF error body's message key.
	detailKey = "detail"
)

// ChecksumShape selects the document checksum wire shape the fake serves
// for stored documents.
type ChecksumShape int

const (
	// ChecksumFlat serves the legacy pre-3.x shape: the checksum in the
	// document's flat "checksum" field.
	ChecksumFlat ChecksumShape = iota
	// ChecksumVersions serves the paperless-ngx 3.x shape: the checksum in
	// the document's versions[] array (is_root marks the current original)
	// with no flat field at all.
	ChecksumVersions
)

// Option configures a Server at construction time.
type Option func(*Server)

// WithToken enforces token authentication on every request: requests
// without the exact "Authorization: Token <token>" header are answered 401
// ("Invalid token.") without reaching any endpoint. With the default empty
// token the fake accepts every request.
func WithToken(token string) Option {
	return func(s *Server) {
		s.token = token
	}
}

// WithChecksumShape selects the checksum wire shape the fake serves for
// stored documents: ChecksumFlat (pre-3.x flat field, the default) or
// ChecksumVersions (paperless-ngx 3.x versions[] array).
func WithChecksumShape(shape ChecksumShape) Option {
	return func(s *Server) {
		s.shape = shape
	}
}

// WithDocuments seeds the fake with stored documents. IDs and checksums
// follow the Document fixture rules (auto-assigned when unset).
func WithDocuments(documents ...Document) Option {
	return func(s *Server) {
		for _, doc := range documents {
			s.addDocument(doc)
		}
	}
}

// Server is a stateful in-memory fake of the Paperless-ngx REST API.
// Construct it with NewServer; the zero value is not usable.
type Server struct {
	t    testing.TB
	http *httptest.Server

	mu sync.Mutex

	apiVersion string
	shape      ChecksumShape

	documents      []*Document
	nextDocumentID int

	patches          []DocumentPatch
	deletedDocuments []int

	uploads        []Upload
	script         []TaskPlan
	tasks          []*taskRecord
	nextTaskNumber int

	tags              []namedEntity
	correspondents    []namedEntity
	documentTypes     []namedEntity
	nextTagID         int
	nextCorrespondentID int
	nextDocumentTypeID  int

	customFields      []customFieldEntity
	storagePaths      []storagePathEntity
	nextCustomFieldID int
	nextStoragePathID int

	requests []RequestRecord
	token    string
	faults   []Fault
}

// NewServer starts a fake Paperless-ngx server and registers its shutdown
// on tb's cleanup list. Unexpected requests fail the test (tb.Errorf) and
// answer 404.
func NewServer(tb testing.TB, opts ...Option) *Server {
	tb.Helper()

	server := &Server{
		t:          tb,
		apiVersion: DefaultAPIVersion,
	}

	for _, opt := range opts {
		opt(server)
	}

	server.http = httptest.NewServer(server)
	tb.Cleanup(server.Close)

	return server
}

// Close shuts the fake server down. NewServer already schedules this via
// t.Cleanup; Close exists for tests that must release the port early.
func (s *Server) Close() {
	s.http.Close()
}

// URL returns the base URL the fake listens on; hand it to paperless.New.
func (s *Server) URL() string {
	return s.http.URL
}

// ServeHTTP routes one request against the fake's endpoints. Unknown
// method/path combinations fail the test and answer 404.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.recordRequest(r)

	if s.serveFault(w, r) {
		return
	}

	if !s.requestAuthorized(r) {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{detailKey: "Invalid token."})

		return
	}

	if !s.route(w, r) {
		s.t.Errorf("paperlesstest: unexpected request: %s %s", r.Method, r.URL.Path)
		s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: "Not found."})
	}
}

// requestAuthorized reports whether the request may reach an endpoint.
// With no token configured the fake accepts every request; otherwise the
// static token scheme must match exactly.
func (s *Server) requestAuthorized(r *http.Request) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token == "" {
		return true
	}

	return r.Header.Get("Authorization") == "Token "+s.token
}

// route dispatches one request to the endpoint handlers. It reports whether
// the request matched a route (the response is then fully written).
func (s *Server) route(w http.ResponseWriter, r *http.Request) bool {
	switch {
	case s.routeDocuments(w, r):
		return true
	case s.routeUpload(w, r):
		return true
	case s.routeTasks(w, r):
		return true
	case s.routeEntities(w, r):
		return true
	}

	return false
}

// writeJSON writes one JSON response with the fake's versioned Content-Type.
func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		s.t.Errorf("paperlesstest: encode response payload: %v", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json; version="+s.apiVersion)
	w.WriteHeader(status)

	if _, writeErr := w.Write(encoded); writeErr != nil {
		s.t.Errorf("paperlesstest: write response body: %v", writeErr)
	}
}

// readBody snapshots the request body so it can be recorded and parsed
// twice (recording consumes the stream; handlers re-read from the copy).
func readBody(r *http.Request) []byte {
	if r.Body == nil {
		return nil
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil
	}

	r.Body = io.NopCloser(bytes.NewReader(raw))

	return raw
}
