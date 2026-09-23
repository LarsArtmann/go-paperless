package paperlesstest

import (
	"net/http"
	"net/url"
	"testing"
)

// RequestRecord is the capture of one request the fake received.
type RequestRecord struct {
	Method string
	Path   string
	Query  url.Values
	// Header holds the request headers as received. Authorization headers
	// carry the test token only — these are in-memory test fakes, but
	// consumers that log records should still redact.
	Header http.Header
	// Body is the raw request body; nil for bodyless requests. Upload
	// bodies are the full multipart payload (see Upload for the parsed
	// form).
	Body []byte
}

// Requests returns a copy of every request the fake has received, oldest
// first.
func (s *Server) Requests() []RequestRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := make([]RequestRecord, 0, len(s.requests))
	records = append(records, s.requests...)

	return records
}

// RequireAuthorized fails the test when any request the fake received was
// NOT authorized with the server's configured token. Requires WithToken to
// be configured (otherwise nothing is enforced and the assertion reports
// that).
func RequireAuthorized(tb testing.TB, server *Server) {
	tb.Helper()

	server.mu.Lock()
	defer server.mu.Unlock()

	if server.token == "" {
		tb.Fatal("paperlesstest: RequireAuthorized needs WithToken configured on the server")

		return
	}

	want := "Token " + server.token

	for _, record := range server.requests {
		if got := record.Header.Get("Authorization"); got != want {
			tb.Errorf(
				"request %s %s authorized with %q, want %q",
				record.Method,
				record.Path,
				got,
				want,
			)
		}
	}
}

// RequireUploadCount fails the test unless the fake captured exactly want
// uploads.
func RequireUploadCount(tb testing.TB, server *Server, want int) {
	tb.Helper()

	if got := len(server.Uploads()); got != want {
		tb.Errorf("captured uploads = %d, want %d", got, want)
	}
}

// recordRequest captures one incoming request. Bodyless requests record a
// nil body; bodyful ones are restored onto the request so handlers can
// parse them afterwards.
func (s *Server) recordRequest(r *http.Request) {
	record := RequestRecord{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.Query(),
		Header: r.Header.Clone(),
	}

	if r.Body != nil {
		record.Body = readBody(r)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests = append(s.requests, record)
}
