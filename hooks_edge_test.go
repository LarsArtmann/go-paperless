package paperless

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestResponseHookFiresOnEveryAttemptIncludingFailures pins the hook
// branches the happy-path tests miss: the response hook observes error
// responses too (the capped error body, not a skipped call), and with
// WithRetry it fires once PER ATTEMPT — so an observer counting calls
// sees the retries, not one summarized event.
func TestResponseHookFiresOnEveryAttemptIncludingFailures(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex

	var bodies []string

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++

		if requests == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)

			_, _ = w.Write([]byte(`{"detail": "slow down"}`))

			return
		}

		w.Header().Set("Content-Type", "application/json; version=10")

		_, _ = w.Write([]byte(`{"results": [], "count": 0, "next": null, "previous": null}`))
	}))
	defer server.Close()

	client, err := New(
		server.URL,
		"token",
		WithRetry(RetryPolicy{MaxAttempts: 2, InitialDelay: time.Millisecond}),
		WithResponseHook(func(info ResponseInfo) {
			mu.Lock()
			defer mu.Unlock()

			bodies = append(bodies, string(info.Body))
		}),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err = client.ListStoragePaths(t.Context()); err != nil {
		t.Fatalf("ListStoragePaths: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(bodies) != 2 {
		t.Fatalf("hook calls = %d, want 2 (one per attempt)", len(bodies))
	}

	if !strings.Contains(bodies[0], "slow down") {
		t.Errorf("first observation = %q, want the capped error body", bodies[0])
	}

	if !strings.Contains(bodies[1], "results") {
		t.Errorf("second observation = %q, want the success body", bodies[1])
	}
}
