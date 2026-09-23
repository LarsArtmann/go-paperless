package paperlesstest

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

// tbStub is a minimal testing.TB standing in for a nested test: the fake
// reports failures through t.Errorf, and this stub captures them so tests
// can assert on the reporting itself.
type tbStub struct {
	testing.TB

	mu       sync.Mutex
	messages []string
	cleanups []func()
}

func (s *tbStub) Errorf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, fmt.Sprintf(format, args...))
}

func (s *tbStub) Cleanup(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanups = append(s.cleanups, fn)
}

func (s *tbStub) errorCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.messages)
}

func (s *tbStub) runCleanups() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := len(s.cleanups) - 1; i >= 0; i-- {
		s.cleanups[i]()
	}
}

func TestUnknownRouteAnswers404AndFailsTest(t *testing.T) {
	t.Parallel()

	stub := &tbStub{TB: t}
	server := NewServer(stub)
	defer server.Close()

	response, err := http.Get(server.URL() + "/api/unknown/")
	if err != nil {
		t.Fatalf("GET /api/unknown/: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for unknown route", response.StatusCode)
	}

	if got := stub.errorCount(); got != 1 {
		t.Errorf("t.Errorf call count = %d, want 1 for unknown route", got)
	}
}

func TestRequestsAreRecorded(t *testing.T) {
	t.Parallel()

	stub := &tbStub{TB: t}
	server := NewServer(stub)
	defer server.Close()

	request, err := http.NewRequest(
		http.MethodGet,
		server.URL()+"/api/missing/?page=1&page_size=1",
		http.NoBody,
	)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET /api/missing/: %v", err)
	}
	defer response.Body.Close()

	records := server.Requests()
	if len(records) != 1 {
		t.Fatalf("recorded requests = %d, want 1", len(records))
	}

	record := records[0]
	if record.Method != http.MethodGet || record.Path != "/api/missing/" {
		t.Errorf("recorded %s %s, want GET /api/missing/", record.Method, record.Path)
	}

	if record.Query.Get("page") != "1" || record.Query.Get("page_size") != "1" {
		t.Errorf("recorded query = %v, want page=1 page_size=1", record.Query)
	}
}
