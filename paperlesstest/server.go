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

// Server is a stateful in-memory fake of the Paperless-ngx REST API.
// Construct it with NewServer; the zero value is not usable.
type Server struct {
	t    testing.TB
	http *httptest.Server

	mu sync.Mutex

	apiVersion string

	requests []RequestRecord
}

// NewServer starts a fake Paperless-ngx server and registers its shutdown
// on t's cleanup list. Unexpected requests fail the test (t.Errorf) and
// answer 404.
func NewServer(t testing.TB, opts ...Option) *Server {
	server := &Server{
		t:          t,
		apiVersion: DefaultAPIVersion,
	}

	for _, opt := range opts {
		opt(server)
	}

	server.http = httptest.NewServer(server)
	t.Cleanup(server.Close)

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

	if !s.route(w, r) {
		s.t.Errorf("paperlesstest: unexpected request: %s %s", r.Method, r.URL.Path)
		s.writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Not found."})
	}
}

// route dispatches one request to the endpoint handlers. It reports whether
// the request matched a route (the response is then fully written).
func (s *Server) route(w http.ResponseWriter, r *http.Request) bool {
	// Endpoint families fill in as the fake grows; the scaffold routes
	// nothing yet, so every request lands on the 404 path above.
	_ = w
	_ = r

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
