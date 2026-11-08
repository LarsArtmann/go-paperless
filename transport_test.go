package paperless

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-retry"
)

func TestStatusErrorExplainsFailedErrorBodyRead(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Promise 1024 bytes of error body but deliver only a fragment:
		// the server cannot fulfill the declared length and closes the
		// connection, so the client's best-effort diagnostic read fails
		// mid-body.
		w.Header().Set("Content-Length", "1024")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"detail": "boom`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	pingErr := client.Ping(t.Context())
	if pingErr == nil {
		t.Fatal("expected error for 500 response")
	}

	coded, ok := errors.AsType[*errorfamily.Error](pingErr)
	if !ok {
		t.Fatalf("expected *errorfamily.Error, got %T (%v)", pingErr, pingErr)
	}

	readErr, explained := coded.ErrorContext()["body_read_error"]
	if !explained || readErr == "" {
		t.Fatalf(
			"expected body_read_error context explaining the truncated snippet, got %v",
			coded.ErrorContext(),
		)
	}
}

func TestPingRejectsBadToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pathDocuments {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		if r.Header.Get("Authorization") != "Token bad" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client, err := New(server.URL, "bad")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	pingErr := client.Ping(t.Context())
	if pingErr == nil {
		t.Fatal("expected ping error")
	}

	if family := errorfamily.Classify(pingErr); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v (%v)", family, pingErr)
	}
}

func TestPingSucceeds(t *testing.T) {
	t.Parallel()

	var gotPath, gotQuery, gotAccept string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAccept = r.Header.Get("Accept")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[],"count":0}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := client.Ping(t.Context()); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	if gotPath != pathDocuments {
		t.Fatalf("ping path = %q, want %q", gotPath, pathDocuments)
	}

	if !strings.Contains(gotQuery, "page=1") || !strings.Contains(gotQuery, "page_size=1") {
		t.Fatalf("ping query = %q, want page=1 and page_size=1", gotQuery)
	}

	if gotAccept != "application/json; version="+apiVersion {
		t.Fatalf("ping Accept = %q, want versioned JSON", gotAccept)
	}
}

// TestPingAgainstHtmlOnlyApiRoot pins the live Paperless-ngx contract that
// broke production (paperless 3.0.5): the API root serves browsable HTML
// only and answers any JSON Accept header with 406 regardless of the token,
// while the document list view is the authenticated JSON endpoint. Ping
// must survive exactly this server shape.

// TestPingAgainstHtmlOnlyApiRoot pins the live Paperless-ngx contract that
// broke production (paperless 3.0.5): the API root serves browsable HTML
// only and answers any JSON Accept header with 406 regardless of the token,
// while the document list view is the authenticated JSON endpoint. Ping
// must survive exactly this server shape.
func TestPingAgainstHtmlOnlyApiRoot(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/":
			w.WriteHeader(http.StatusNotAcceptable)
		case pathDocuments:
			if r.Header.Get("Authorization") != "Token good" {
				w.WriteHeader(http.StatusUnauthorized)

				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"results":[{"checksum":"abc"}],"count":1}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "good")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := client.Ping(t.Context()); err != nil {
		t.Fatalf("Ping against HTML-only API root: %v", err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{"seconds", "120", 2 * time.Minute, true},
		{"zero seconds", "0", 0, true},
		{"negative seconds invalid", "-5", 0, false},
		{"empty invalid", "", 0, false},
		{"garbage invalid", "soon", 0, false},
		{"future date", now.Add(time.Minute).Format(http.TimeFormat), time.Minute, true},
		{"past date retries now", now.Add(-time.Hour).Format(http.TimeFormat), 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseRetryAfter(tc.value, now)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v (got %s)", ok, tc.ok, got)
			}

			if ok && got != tc.want {
				t.Fatalf("delay = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestWithRetryDefaultOffMakesSingleAttempt(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++

		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.ListStoragePaths(t.Context())
	if err == nil {
		t.Fatal("expected the 503 to surface")
	}

	if requests != 1 {
		t.Fatalf("requests = %d, want exactly 1 without WithRetry", requests)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Transient {
		t.Fatalf("expected Transient family, got %v (%v)", family, err)
	}
}

func TestWithRetryRetriesTransient5xxUntilSuccess(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)

			return
		}

		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token", WithRetry(RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		MaxDelay:     2 * time.Millisecond,
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	paths, err := client.ListStoragePaths(t.Context())
	if err != nil {
		t.Fatalf("ListStoragePaths: %v", err)
	}

	if len(paths) != 0 {
		t.Fatalf("paths = %v, want empty", paths)
	}

	if requests != 3 {
		t.Fatalf("requests = %d, want 3 (two 503s then success)", requests)
	}
}

func TestWithRetryNeverRetriesRejections(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := New(server.URL, "token", WithRetry(RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		MaxDelay:     2 * time.Millisecond,
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.DownloadDocument(t.Context(), NewDocumentID(42))
	if err == nil {
		t.Fatal("expected the 404 to surface")
	}

	if requests != 1 {
		t.Fatalf("requests = %d, want exactly 1 (rejections are never retried)", requests)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v (%v)", family, err)
	}
}

// TestWithRetryExhaustionSurfacesRetryOutcome pins the exhaustion surface
// (ERROR_CODES.md "Retry outcomes"): when every attempt fails retryably, the
// terminal error is go-retry's retry.exhausted (Infrastructure) with the SDK's
// uncoded "request failed after retries" message wrap on top; the last
// attempt's paperless classification stays reachable only through the chain.

// TestWithRetryExhaustionSurfacesRetryOutcome pins the exhaustion surface
// (ERROR_CODES.md "Retry outcomes"): when every attempt fails retryably, the
// terminal error is go-retry's retry.exhausted (Infrastructure) with the SDK's
// uncoded "request failed after retries" message wrap on top; the last
// attempt's paperless classification stays reachable only through the chain.
func TestWithRetryExhaustionSurfacesRetryOutcome(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		// Retry-After: 0 keeps the hint parseable (so the attempt error is a
		// *RetryAfterError) without extending the tiny test backoff.
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, err := New(server.URL, "token", WithRetry(RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		MaxDelay:     2 * time.Millisecond,
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Upload(t.Context(), UploadRequest{
		Filename: "invoice.pdf",
		Content:  []byte("%PDF-1.4 rate-limited"),
	})
	if err == nil {
		t.Fatal("expected the persistent 429 to surface")
	}

	if requests != 3 {
		t.Fatalf("requests = %d, want 3 (every attempt rate-limited)", requests)
	}

	if code := errorfamily.Code(err); code != "paperless.rate_limited" {
		t.Fatalf(
			"code = %q, want paperless.rate_limited (go-retry v0.9.0 surfaces the last attempt's code outermost)",
			code,
		)
	}

	if !errors.Is(err, retry.ErrExhausted) {
		t.Fatalf("exhaustion bookkeeping must stay reachable via errors.Is, got %v", err)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Infrastructure {
		t.Fatalf(
			"expected Infrastructure family for the client's exhaustion wrap, got %v (%v)",
			family,
			err,
		)
	}

	if !strings.Contains(err.Error(), "request failed after retries") {
		t.Fatalf("error %q should carry the documented exhaustion message wrap", err)
	}

	rateLimited, ok := errors.AsType[*RetryAfterError](err)
	if !ok {
		t.Fatalf("exhaustion should reach the last attempt's *RetryAfterError, got %v", err)
	}

	if code := errorfamily.Code(rateLimited); code != "paperless.rate_limited" {
		t.Fatalf("inner code = %q, want paperless.rate_limited", code)
	}
}

// TestWithRetryCancelDuringBackoffSurfacesRetryCanceled pins the WithRetry
// half of the cancellation surface: a context canceled while the first
// attempt is in flight keeps the canceled retry bookkeeping reachable via
// errors.Is (go-retry v0.9.0) while the last attempt's coded error surfaces
// outermost, and context.Canceled stays reachable through the chain. The
// handler gate makes the ordering deterministic: the cancel fires only
// after the server has entered the handler, so attempt 1 always dies with a
// transport-level cancel before any response exists, and the engine
// observes the cancel inside its backoff — the outcome no longer depends on
// whether the cancel goroutine outruns response processing, the race this
// test used to lose under parallel-build load.

// TestWithRetryCancelDuringBackoffSurfacesRetryCanceled pins the WithRetry
// half of the cancellation surface: a context canceled while the first
// attempt is in flight keeps the canceled retry bookkeeping reachable via
// errors.Is (go-retry v0.9.0) while the last attempt's coded error surfaces
// outermost, and context.Canceled stays reachable through the chain. The
// handler gate makes the ordering deterministic: the cancel fires only
// after the server has entered the handler, so attempt 1 always dies with a
// transport-level cancel before any response exists, and the engine
// observes the cancel inside its backoff — the outcome no longer depends on
// whether the cancel goroutine outruns response processing, the race this
// test used to lose under parallel-build load.
func TestWithRetryCancelDuringBackoffSurfacesRetryCanceled(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{})

	var once sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(entered) })
		// Hold attempt 1 in flight until the test's cancel wins the
		// ordering by construction. The ceiling only exists so
		// server.Close can never wedge if the connection were to linger.
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "token", WithRetry(RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		MaxDelay:     2 * time.Millisecond,
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		<-entered
		cancel()
	}()

	_, err = client.ListStoragePaths(ctx)
	if err == nil {
		t.Fatal("expected the canceled retry loop to surface")
	}

	if code := errorfamily.Code(err); code != "paperless.request_failed" {
		t.Fatalf(
			"code = %q, want paperless.request_failed (go-retry v0.9.0 surfaces the last attempt's code outermost)",
			code,
		)
	}

	if !errors.Is(err, retry.ErrCanceled) {
		t.Fatalf("canceled-backoff bookkeeping must stay reachable via errors.Is, got %v", err)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Infrastructure {
		t.Fatalf(
			"expected Infrastructure family for the client's canceled-backoff wrap, got %v (%v)",
			family,
			err,
		)
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want it to reach context.Canceled through the chain", err)
	}
}

// TestWithRetryCancelDuringBackoffPromotesRateLimitedCode pins the sibling
// ordering: attempt 1 has already completed as a rate limit (a
// *RetryAfterError is the last attempt on the books) and the context is
// canceled during the ensuing Retry-After-driven backoff. go-retry v0.9.0
// then surfaces the LAST attempt's code outermost — paperless.rate_limited,
// NOT paperless.request_failed — while retry.canceled and context.Canceled
// stay reachable via errors.Is (ERROR_CODES.md "Retry outcomes"). The
// response hook makes the ordering deterministic: the cancel fires only
// after the hook observed attempt 1's response, so the 30s hint window
// always outlives the scheduling gap before cancel runs. (This outcome was
// previously mislabeled a flake: the predecessor test raced the cancel
// against attempt 1's response processing and asserted the
// cancel-preempts-response winner's code for both orderings.)

// TestWithRetryCancelDuringBackoffPromotesRateLimitedCode pins the sibling
// ordering: attempt 1 has already completed as a rate limit (a
// *RetryAfterError is the last attempt on the books) and the context is
// canceled during the ensuing Retry-After-driven backoff. go-retry v0.9.0
// then surfaces the LAST attempt's code outermost — paperless.rate_limited,
// NOT paperless.request_failed — while retry.canceled and context.Canceled
// stay reachable via errors.Is (ERROR_CODES.md "Retry outcomes"). The
// response hook makes the ordering deterministic: the cancel fires only
// after the hook observed attempt 1's response, so the 30s hint window
// always outlives the scheduling gap before cancel runs. (This outcome was
// previously mislabeled a flake: the predecessor test raced the cancel
// against attempt 1's response processing and asserted the
// cancel-preempts-response winner's code for both orderings.)
func TestWithRetryCancelDuringBackoffPromotesRateLimitedCode(t *testing.T) {
	t.Parallel()

	hookSeen := make(chan struct{})

	var requests int

	var once sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		// Only attempt 1 carries the 30s hint the cancel must interrupt;
		// later attempts (reachable only under pathological scheduling)
		// stay hint-free so the loop fails fast instead of hanging.
		if requests == 1 {
			w.Header().Set("Retry-After", "30")
		}

		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, err := New(
		server.URL,
		"token",
		WithRetry(RetryPolicy{
			MaxAttempts:  3,
			InitialDelay: time.Millisecond,
			// Above the 30s hint so the hint IS the backoff window the
			// cancel must land inside.
			MaxDelay: 45 * time.Second,
		}),
		WithResponseHook(func(_ ResponseInfo) {
			once.Do(func() { close(hookSeen) })
		}),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		<-hookSeen
		cancel()
	}()

	_, err = client.ListStoragePaths(ctx)
	if err == nil {
		t.Fatal("expected the canceled retry loop to surface")
	}

	if code := errorfamily.Code(err); code != "paperless.rate_limited" {
		t.Fatalf(
			"code = %q, want paperless.rate_limited (go-retry v0.9.0 surfaces the last attempt's code outermost)",
			code,
		)
	}

	if !errors.Is(err, retry.ErrCanceled) {
		t.Fatalf("canceled-backoff bookkeeping must stay reachable via errors.Is, got %v", err)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Infrastructure {
		t.Fatalf(
			"expected Infrastructure family for the canceled-backoff wrap, got %v (%v)",
			family,
			err,
		)
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want it to reach context.Canceled through the chain", err)
	}
}

// TestUnfollowableRedirectClassifiedAsRedirect pins the 3xx surface: a
// redirect the http.Client cannot follow (a 301 without a Location header is
// returned as-is) classifies as paperless.redirect/Rejection, so it is never
// retried into the same redirect.

// TestUnfollowableRedirectClassifiedAsRedirect pins the 3xx surface: a
// redirect the http.Client cannot follow (a 301 without a Location header is
// returned as-is) classifies as paperless.redirect/Rejection, so it is never
// retried into the same redirect.
func TestUnfollowableRedirectClassifiedAsRedirect(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.ListStoragePaths(t.Context())
	if err == nil {
		t.Fatal("expected the 301 to surface as an error")
	}

	if code := errorfamily.Code(err); code != "paperless.redirect" {
		t.Fatalf("code = %q, want paperless.redirect", code)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family for an unfollowable redirect, got %v (%v)", family, err)
	}
}

// TestClassifyStatusMatrix pins the status-to-code/family mapping at the
// classification source. net/http swallows followable redirects, so only a
// synthetic response can exercise every status; the httptest-level tests
// above cover the reachable end-to-end surfaces.

// TestClassifyStatusMatrix pins the status-to-code/family mapping at the
// classification source. net/http swallows followable redirects, so only a
// synthetic response can exercise every status; the httptest-level tests
// above cover the reachable end-to-end surfaces.
func TestClassifyStatusMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status     int
		wantCode   string
		wantFamily errorfamily.Family
	}{
		{http.StatusUnauthorized, "paperless.auth_failed", errorfamily.Rejection},
		{http.StatusForbidden, "paperless.auth_failed", errorfamily.Rejection},
		{http.StatusTooManyRequests, "paperless.rate_limited", errorfamily.Transient},
		{http.StatusBadRequest, "paperless.client_error", errorfamily.Rejection},
		{http.StatusNotFound, "paperless.client_error", errorfamily.Rejection},
		{http.StatusInternalServerError, "paperless.server_error", errorfamily.Transient},
		{http.StatusServiceUnavailable, "paperless.server_error", errorfamily.Transient},
		{http.StatusMovedPermanently, "paperless.redirect", errorfamily.Rejection},
		{http.StatusNotModified, "paperless.redirect", errorfamily.Rejection},
		{http.StatusTemporaryRedirect, "paperless.redirect", errorfamily.Rejection},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.status), func(t *testing.T) {
			t.Parallel()

			resp := &http.Response{StatusCode: tt.status, Header: http.Header{}}
			err := classifyStatus(resp, []byte("snippet"), "/api/documents/", nil)

			if code := errorfamily.Code(err); code != tt.wantCode {
				t.Fatalf("code = %q, want %q (err: %v)", code, tt.wantCode, err)
			}

			if family := errorfamily.Classify(err); family != tt.wantFamily {
				t.Fatalf("family = %v, want %v (err: %v)", family, tt.wantFamily, err)
			}
		})
	}

	t.Run("redirect carries location context", func(t *testing.T) {
		t.Parallel()

		resp := &http.Response{
			StatusCode: http.StatusMovedPermanently,
			Header:     http.Header{"Location": []string{"https://elsewhere.example.com/docs/"}},
		}
		err := classifyStatus(resp, nil, "/api/documents/", nil)

		coded, ok := errors.AsType[*errorfamily.Error](err)
		if !ok {
			t.Fatalf("err = %v, want an *errorfamily.Error", err)
		}

		if got := coded.ErrorContext()["location"]; got != "https://elsewhere.example.com/docs/" {
			t.Fatalf(
				"location context = %q, want the Location header value (context: %v)",
				got,
				coded.ErrorContext(),
			)
		}
	})

	t.Run("nil classifies Rejection", func(t *testing.T) {
		t.Parallel()

		if errorfamily.Classify(nil) != errorfamily.Rejection {
			t.Error("errorfamily contract drifted: nil must classify Rejection")
		}
	})
}

// TestWithRetryClampsWireHintAtMaxDelay pins the wire-to-delay seam
// without a real second-long sleep: parseRetryAfter only accepts whole
// seconds, so a verbatim wire hint costs a full second of test time.
// Instead a 2s hint with a 50ms ceiling proves the hint was parsed from
// the response and reaches the retry delay (elapsed >= ~45ms, above the
// ~1ms computed backoff) AND that the ceiling clamps it (elapsed well
// under the verbatim 2s). Verbatim hint honoring below the ceiling is
// pinned pure by TestRetryPolicyDelayFuncBridgesRetryAfterHint.
func TestWithRetryClampsWireHintAtMaxDelay(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusServiceUnavailable)

			return
		}

		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token", WithRetry(RetryPolicy{
		MaxAttempts:  2,
		InitialDelay: time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	started := time.Now()
	_, err = client.ListStoragePaths(t.Context())
	elapsed := time.Since(started)

	if err != nil {
		t.Fatalf("ListStoragePaths: %v", err)
	}

	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}

	if elapsed < 45*time.Millisecond {
		t.Fatalf(
			"elapsed = %s, want >= ~45ms so the server's hint reaches the retry delay",
			elapsed,
		)
	}

	if elapsed >= 2*time.Second {
		t.Fatalf(
			"elapsed = %s, want well under the verbatim 2s hint — MaxDelay must clamp wire hints",
			elapsed,
		)
	}
}

var errUnrelatedFailure = errors.New("boom")

func TestRetryPolicyDelayFuncBridgesRetryAfterHint(t *testing.T) {
	t.Parallel()

	delayFunc := (RetryPolicy{MaxAttempts: 2}).retryConfig().DelayFunc

	// Below the default MaxDelay ceiling the hint is honored verbatim.
	if got := delayFunc(1, &RetryAfterError{After: 3 * time.Second}); got != 3*time.Second {
		t.Fatalf("DelayFunc = %s, want the server hint 3s", got)
	}

	// Beyond the ceiling the hint is clamped, so a broken or hostile
	// server cannot stall the caller indefinitely.
	if got := delayFunc(1, &RetryAfterError{After: 7 * time.Second}); got != DefaultRetryMaxDelay {
		t.Fatalf("DelayFunc = %s, want the hint clamped to MaxDelay %s", got, DefaultRetryMaxDelay)
	}

	if got := delayFunc(1, errUnrelatedFailure); got != 0 {
		t.Fatalf("DelayFunc = %s, want 0 (fall back to exponential backoff)", got)
	}
}

func TestWithRetryReplaysUploadBodyByteForByte(t *testing.T) {
	t.Parallel()

	var (
		bodies   [][]byte
		requests int
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		raw, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("read request body: %v", readErr)
		}

		bodies = append(bodies, raw)

		if requests == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)

			return
		}

		_, _ = w.Write([]byte(`"task-uuid-retry"`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token", WithRetry(RetryPolicy{
		MaxAttempts:  2,
		InitialDelay: time.Millisecond,
		MaxDelay:     2 * time.Millisecond,
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	taskID, err := client.Upload(t.Context(), UploadRequest{
		Filename: "invoice.pdf",
		Content:  []byte("%PDF-1.4 replay-me"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if taskID.Get() != "task-uuid-retry" {
		t.Fatalf("task ID = %q", taskID)
	}

	if requests != 2 || len(bodies) != 2 {
		t.Fatalf("requests = %d, bodies = %d, want 2 each", requests, len(bodies))
	}

	if !bytes.Equal(bodies[0], bodies[1]) {
		t.Fatal("retried upload body differs from the first attempt")
	}

	if !strings.Contains(string(bodies[1]), "%PDF-1.4 replay-me") {
		t.Fatalf("replayed body lost the file content: %q", bodies[1])
	}
}

func TestPlainRequestHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	pingErr := client.Ping(ctx)
	if pingErr == nil {
		t.Fatal("expected the ping to fail against an already-cancelled context")
	}

	if !errors.Is(pingErr, context.Canceled) {
		t.Fatalf("pingErr = %v, want it to wrap context.Canceled", pingErr)
	}

	if code := errorfamily.Code(pingErr); code != "paperless.request_failed" {
		t.Fatalf("code = %q, want paperless.request_failed (cancel without retry)", code)
	}

	if family := errorfamily.Classify(pingErr); family != errorfamily.Transient {
		t.Fatalf(
			"expected Transient family for a canceled single attempt, got %v (%v)",
			family,
			pingErr,
		)
	}
}

func TestNegotiatedAPIVersionReadsContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		{"echoed version", "application/json; version=10", "10"},
		{"version with charset", "application/json; version=9; charset=utf-8", "9"},
		{"no version parameter", "application/json", ""},
		{"empty header", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			header := http.Header{}
			if tc.contentType != "" {
				header.Set("Content-Type", tc.contentType)
			}

			if got := negotiatedAPIVersion(header); got != tc.want {
				t.Fatalf("negotiatedAPIVersion(%q) = %q, want %q", tc.contentType, got, tc.want)
			}
		})
	}
}
