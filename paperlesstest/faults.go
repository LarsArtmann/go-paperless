package paperlesstest

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// Fault scripts server-side failures for matching requests: the fake
// answers with the fault's status (and optional body / Retry-After hint)
// instead of running the endpoint handler. Consume the same shape as
// retry and rate-limit testing against the real server: 5xx-then-success
// sequences, 429 with Retry-After, and malformed payloads.
type Fault struct {
	// Method is the HTTP method the fault matches ("" = any method).
	Method string
	// PathPrefix is the path prefix the fault matches, e.g.
	// "/api/documents/" (required).
	PathPrefix string
	// Times is how many matching requests the fault serves before the
	// endpoint behaves normally again. Zero or negative means every
	// matching request fails forever.
	Times int
	// Status is the HTTP status the fault answers with (required).
	Status int
	// Body overrides the response body verbatim (e.g. malformed JSON to
	// exercise decode-error paths). Empty answers a DRF-style detail body.
	Body string
	// RetryAfter sets a Retry-After response header (delay-seconds or
	// HTTP-date form), exercising the client's RetryAfterError handling
	// on 429/503 responses.
	RetryAfter string
}

// WithFaults arms faults at construction time (the runtime equivalent is
// InjectFault).
func WithFaults(faults ...Fault) Option {
	return func(s *Server) {
		for _, fault := range faults {
			s.InjectFault(fault)
		}
	}
}

// InjectFault arms one fault. The first matching armed fault wins; faults
// with a positive Times are consumed by each match and disarmed once spent.
func (s *Server) InjectFault(fault Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.faults = append(s.faults, fault)
}

// serveFault answers one request from an armed fault when one matches. It
// reports whether the request was handled by a fault. The caller must not
// hold s.mu.
func (s *Server) serveFault(w http.ResponseWriter, r *http.Request) bool {
	s.mu.Lock()

	index := s.armedFaultLocked(r)

	var fault Fault

	if index >= 0 {
		fault = s.faults[index]

		if fault.Times > 0 {
			fault.Times--

			if fault.Times == 0 {
				s.faults = slices.Delete(s.faults, index, index+1)
			} else {
				s.faults[index] = fault
			}
		}
	}

	s.mu.Unlock()

	if index < 0 {
		return false
	}

	if fault.RetryAfter != "" {
		w.Header().Set("Retry-After", fault.RetryAfter)
	}

	if fault.Body != "" {
		s.writeRaw(w, fault.Status, []byte(fault.Body))

		return true
	}

	s.writeJSON(
		w,
		fault.Status,
		map[string]string{detailKey: "Faulted by " + strconv.Quote(fault.PathPrefix) + "."},
	)

	return true
}

// armedFaultLocked finds the first fault matching the request. Returns -1
// when no fault is armed for it. The caller must hold s.mu.
func (s *Server) armedFaultLocked(r *http.Request) int {
	for index := range s.faults {
		fault := &s.faults[index]

		if fault.matches(r) {
			return index
		}
	}

	return -1
}

// matches reports whether the fault applies to the request.
func (f *Fault) matches(r *http.Request) bool {
	if f.Method != "" && f.Method != r.Method {
		return false
	}

	if f.PathPrefix == "" {
		return false
	}

	return strings.HasPrefix(r.URL.Path, f.PathPrefix)
}

// writeRaw writes one response body verbatim (fault bodies may be
// intentionally malformed JSON).
func (s *Server) writeRaw(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json; version="+s.apiVersion)
	w.WriteHeader(status)

	if _, err := w.Write(body); err != nil {
		s.t.Errorf("paperlesstest: write fault body: %v", err)
	}
}
