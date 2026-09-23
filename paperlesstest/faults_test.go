package paperlesstest

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	paperless "github.com/larsartmann/go-paperless"
)

func TestFaultThenSuccessRecoversWithRetry(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithFaults(Fault{
		Method:     http.MethodGet,
		PathPrefix: "/api/documents/",
		Times:      2,
		Status:     http.StatusServiceUnavailable,
	}))
	server.addDocument(Document{Title: "one", Checksum: "c1"})

	retrying, err := paperless.New(
		server.URL(),
		DefaultToken,
		paperless.WithRetry(paperless.RetryPolicy{MaxAttempts: 3, InitialDelay: time.Millisecond}),
	)
	if err != nil {
		t.Fatalf("paperless.New: %v", err)
	}

	checksums, err := retrying.ListDocumentChecksums(context.Background())
	if err != nil {
		t.Fatalf("ListDocumentChecksums after two 503s: %v", err)
	}

	if _, ok := checksums["c1"]; !ok {
		t.Error("checksum c1 missing after fault-then-success recovery")
	}
}

func TestFaultIsConsumedByTimes(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	server.InjectFault(Fault{
		Method:     http.MethodGet,
		PathPrefix: "/api/documents/",
		Times:      1,
		Status:     http.StatusInternalServerError,
	})

	ctx := context.Background()

	if err := newTestClient(t, server).Ping(ctx); err == nil {
		t.Error("first Ping = nil error, want the armed 500")
	}

	if err := newTestClient(t, server).Ping(ctx); err != nil {
		t.Errorf("second Ping after the fault was consumed: %v", err)
	}
}

func TestRateLimitWithRetryAfterSeconds(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithFaults(Fault{
		Method:     http.MethodGet,
		PathPrefix: "/api/documents/",
		Times:      1,
		Status:     http.StatusTooManyRequests,
		RetryAfter: "0",
	}))

	fastRetry, err := paperless.New(
		server.URL(),
		DefaultToken,
		paperless.WithRetry(paperless.RetryPolicy{MaxAttempts: 2, InitialDelay: time.Millisecond}),
	)
	if err != nil {
		t.Fatalf("paperless.New: %v", err)
	}

	if pingErr := fastRetry.Ping(context.Background()); pingErr != nil {
		t.Fatalf("Ping after a 429+Retry-After: %v", pingErr)
	}
}

func TestRateLimitYieldsRetryAfterError(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	server.InjectFault(Fault{
		Method:     http.MethodGet,
		PathPrefix: "/api/documents/",
		Times:      1,
		Status:     http.StatusTooManyRequests,
		RetryAfter: "3",
	})

	pingErr := newTestClient(t, server).Ping(context.Background())
	if pingErr == nil {
		t.Fatal("Ping = nil error, want the rate-limit rejection chain")
	}

	retryAfter, ok := errors.AsType[*paperless.RetryAfterError](pingErr)
	if !ok {
		t.Fatalf("errors.AsType[*RetryAfterError] = false (%v)", pingErr)
	}

	if retryAfter.After != 3*time.Second {
		t.Errorf("Retry-After = %v, want 3s", retryAfter.After)
	}

	if errorfamily.Classify(retryAfter.Err) != errorfamily.Transient {
		t.Errorf("wrapped family = %v, want Transient", errorfamily.Classify(retryAfter.Err))
	}
}

func TestRetryAfterHTTPDateInThePastMeansImmediateRetry(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithFaults(Fault{
		Method:     http.MethodGet,
		PathPrefix: "/api/documents/",
		Times:      1,
		Status:     http.StatusServiceUnavailable,
		RetryAfter: time.Now().Add(-time.Minute).UTC().Format(http.TimeFormat),
	}))

	fastRetry, err := paperless.New(
		server.URL(),
		DefaultToken,
		paperless.WithRetry(paperless.RetryPolicy{MaxAttempts: 2, InitialDelay: time.Millisecond}),
	)
	if err != nil {
		t.Fatalf("paperless.New: %v", err)
	}

	if pingErr := fastRetry.Ping(context.Background()); pingErr != nil {
		t.Fatalf("Ping with a past HTTP-date Retry-After: %v", pingErr)
	}
}

func TestMalformedJSONFaultSurfacesCorruption(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithFaults(Fault{
		Method:     http.MethodGet,
		PathPrefix: "/api/tasks/",
		Times:      1,
		Status:     http.StatusOK,
		Body:       `{"results":[{"task_id"`,
	}))

	_, _, err := newTestClient(t, server).GetTask(context.Background(), "task-1")
	if err == nil {
		t.Fatal("GetTask against a truncated body = nil error, want a decode corruption")
	}

	if errorfamily.Classify(err) != errorfamily.Corruption {
		t.Errorf("family = %v, want Corruption", errorfamily.Classify(err))
	}
}

func TestFaultOnWrongMethodDoesNotFire(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	server.InjectFault(Fault{
		Method:     http.MethodPost,
		PathPrefix: "/api/documents/",
		Times:      1,
		Status:     http.StatusInternalServerError,
	})
	server.addDocument(Document{Title: "one", Checksum: "c1"})

	if err := newTestClient(t, server).Ping(context.Background()); err != nil {
		t.Errorf("GET Ping hit a POST-only fault: %v", err)
	}
}
