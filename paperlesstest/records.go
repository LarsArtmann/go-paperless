package paperlesstest

import (
	"bytes"
	"net/http"
	"net/url"
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

	records := make([]RequestRecord, len(s.requests))
	copy(records, s.requests)

	return records
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
