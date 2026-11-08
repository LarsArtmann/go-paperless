package paperless

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestNewRejectsMissingConfig(t *testing.T) {
	t.Parallel()

	if _, err := New("", "token"); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("empty URL: expected ErrInvalidConfig, got %v", err)
	}

	if _, err := New("https://paperless.example.com", ""); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("empty token: expected ErrInvalidConfig, got %v", err)
	}

	_, err := New("", "token")
	if err == nil {
		t.Fatal("expected an error for an empty base URL")
	}

	if code := errorfamily.Code(err); code != "paperless.invalid_config" {
		t.Fatalf("code = %q, want paperless.invalid_config", code)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v (%v)", family, err)
	}

	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("err = %v, want the sentinel matchable through the coded wrap", err)
	}
}

func TestNewRejectsNonHTTPScheme(t *testing.T) {
	t.Parallel()

	_, err := New("ftp://paperless.example.com", "token")
	if err == nil {
		t.Fatal("expected error for ftp scheme")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v", errorfamily.Classify(err))
	}
}

// The capability probe must classify all three checksum delivery shapes:
// flat (pre-3.x), versions[] (3.x), and neither (the degraded case the
// doctor warns about).
func TestProbeCapabilitiesDetectsChecksumShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantShape  string
		wantFlat   bool
		wantVers   bool
		wantSample int
	}{
		{
			name:       "flat pre-3x",
			body:       `{"results":[{"id":1,"checksum":"sha-flat"}]}`,
			wantShape:  "flat (pre-3.x)",
			wantFlat:   true,
			wantSample: 1,
		},
		{
			name:       "versions 3x",
			body:       `{"results":[{"id":2,"versions":[{"id":2,"checksum":"sha-vers","is_root":true}]}]}`,
			wantShape:  "versions[] (3.x)",
			wantVers:   true,
			wantSample: 1,
		},
		{
			name: "both shapes",
			body: `{"results":[{"id":3,"checksum":"a","versions":[` +
				`{"id":3,"checksum":"b","is_root":true}]},` +
				`{"id":4,"versions":[{"id":4,"checksum":"c","is_root":true}]}]}`,
			wantShape:  "flat+versions[]",
			wantFlat:   true,
			wantVers:   true,
			wantSample: 2,
		},
		{
			name:      "neither shape",
			body:      `{"results":[{"id":5,"title":"no checksums here"}]}`,
			wantShape: "none",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != pathDocuments {
						w.WriteHeader(http.StatusNotFound)

						return
					}

					w.Header().Set("Content-Type", "application/json; version=10")
					_, _ = w.Write([]byte(tc.body))
				}),
			)
			defer server.Close()

			client, err := New(server.URL, "token")
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			caps, err := client.ProbeCapabilities(t.Context())
			if err != nil {
				t.Fatalf("ProbeCapabilities: %v", err)
			}

			if caps.ChecksumShape() != tc.wantShape {
				t.Errorf("shape = %q, want %q", caps.ChecksumShape(), tc.wantShape)
			}

			if caps.FlatChecksum != tc.wantFlat || caps.VersionedChecksum != tc.wantVers {
				t.Errorf("flat=%v versions=%v, want flat=%v versions=%v",
					caps.FlatChecksum, caps.VersionedChecksum, tc.wantFlat, tc.wantVers)
			}

			if caps.DocumentsSampled != tc.wantSample {
				t.Errorf("sampled = %d, want %d", caps.DocumentsSampled, tc.wantSample)
			}

			if caps.AcceptAPIVersion != "10" {
				t.Errorf("AcceptAPIVersion = %q, want 10", caps.AcceptAPIVersion)
			}
		})
	}
}

func TestRetryPolicyZeroValueFillsDefaults(t *testing.T) {
	t.Parallel()

	config := (RetryPolicy{}).retryConfig()

	if config.MaxAttempts != DefaultRetryMaxAttempts {
		t.Fatalf("MaxAttempts = %d, want %d", config.MaxAttempts, DefaultRetryMaxAttempts)
	}

	if config.InitialDelay != DefaultRetryInitialDelay {
		t.Fatalf("InitialDelay = %s, want %s", config.InitialDelay, DefaultRetryInitialDelay)
	}

	if config.MaxDelay != DefaultRetryMaxDelay {
		t.Fatalf("MaxDelay = %s, want %s", config.MaxDelay, DefaultRetryMaxDelay)
	}

	if config.Multiplier != DefaultRetryMultiplier {
		t.Fatalf("Multiplier = %f, want %f", config.Multiplier, DefaultRetryMultiplier)
	}
}

func TestNewRejectsNegativeRetryMaxAttempts(t *testing.T) {
	t.Parallel()

	_, err := New("https://paperless.example.com", "token", WithRetry(RetryPolicy{MaxAttempts: -1}))
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig for negative MaxAttempts, got %v", err)
	}

	if code := errorfamily.Code(err); code != "paperless.invalid_retry" {
		t.Fatalf("code = %q, want paperless.invalid_retry (%v)", code, err)
	}

	if !strings.Contains(err.Error(), "got -1") {
		t.Fatalf("error must name the offending value, got %v", err)
	}
}

func TestRequestHookObservesWithoutMutating(t *testing.T) {
	t.Parallel()

	var (
		gotInfo     RequestInfo
		gotInjected string
		gotAuth     string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotInjected = r.Header.Get("X-Injected")
		gotAuth = r.Header.Get("Authorization")

		_, _ = w.Write([]byte(`{"results":[{"task_id":"task-uuid-9","status":"pending"}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token", WithRequestHook(func(info RequestInfo) {
		gotInfo = info
		info.Header.Set("X-Injected", "yes")
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, _, err := client.GetTask(t.Context(), NewTaskID("task-uuid-9")); err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if gotInfo.Method != http.MethodGet {
		t.Fatalf("Method = %q, want GET", gotInfo.Method)
	}

	if !strings.Contains(gotInfo.URL, "/api/tasks/?task_id=task-uuid-9") {
		t.Fatalf("URL = %q, want the tasks endpoint with the task_id query", gotInfo.URL)
	}

	if gotInfo.Header.Get("Authorization") != "Token token" {
		t.Fatalf(
			"hooked Authorization = %q, want the token (hooks must see it to redact it)",
			gotInfo.Header.Get("Authorization"),
		)
	}

	if gotInjected != "" {
		t.Fatal("hook mutation leaked into the real request")
	}

	if gotAuth != "Token token" {
		t.Fatalf("server Authorization = %q", gotAuth)
	}
}

func TestResponseHookSees2xxBodyAndCappedErrorBody(t *testing.T) {
	t.Parallel()

	var responses []ResponseInfo

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/tasks/") {
			_, _ = w.Write([]byte(`{"results":[{"task_id":"task-uuid-10","status":"success",` +
				`"result_data":{"document_id":42}}]}`))

			return
		}

		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(bytes.Repeat([]byte("x"), 2000))
	}))
	defer server.Close()

	client, err := New(server.URL, "token", WithResponseHook(func(info ResponseInfo) {
		responses = append(responses, info)
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, _, err := client.GetTask(t.Context(), NewTaskID("task-uuid-10")); err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if len(responses) != 1 || responses[0].Status != http.StatusOK {
		t.Fatalf("responses = %+v, want one 200 snapshot", responses)
	}

	if !strings.Contains(string(responses[0].Body), "document_id") {
		t.Fatalf("2xx hook body = %q, want the full response body", responses[0].Body)
	}

	if responses[0].Header == nil {
		t.Fatal("hooked response header must be present")
	}

	_, err = client.DownloadDocument(t.Context(), NewDocumentID(42))
	if err == nil {
		t.Fatal("expected the 404 to surface")
	}

	if len(responses) != 2 || responses[1].Status != http.StatusNotFound {
		t.Fatalf("responses = %+v, want a second 404 snapshot", responses)
	}

	if len(responses[1].Body) != maxErrorBodyBytes {
		t.Fatalf(
			"404 hook body = %d bytes, want the %d-byte cap",
			len(responses[1].Body),
			maxErrorBodyBytes,
		)
	}
}

func TestWithTimeoutBoundsSlowResponses(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client, err := New(server.URL, "token", WithTimeout(20*time.Millisecond))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	started := time.Now()

	pingErr := client.Ping(t.Context())
	if pingErr == nil {
		t.Fatal("expected the ping to fail against a server that never answers")
	}

	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("ping returned after %s, want the ~20ms deadline to cut it off", elapsed)
	}

	if family := errorfamily.Classify(pingErr); family != errorfamily.Transient {
		t.Fatalf("expected Transient family for a timeout, got %v (%v)", family, pingErr)
	}
}

func TestProbeCapabilitiesFiresHooks(t *testing.T) {
	t.Parallel()

	var (
		requests  int
		responses int
		gotPath   string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

		_, _ = w.Write([]byte(`{"results":[{"id":1,"checksum":"sha-probe"}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token",
		WithRequestHook(func(RequestInfo) { requests++ }),
		WithResponseHook(func(ResponseInfo) { responses++ }),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	caps, err := client.ProbeCapabilities(t.Context())
	if err != nil {
		t.Fatalf("ProbeCapabilities: %v", err)
	}

	if gotPath != "/api/documents/" {
		t.Errorf("probe path = %q, want /api/documents/", gotPath)
	}

	if !caps.FlatChecksum || caps.VersionedChecksum {
		t.Errorf("checksum shapes = flat:%v versioned:%v, want flat-only",
			caps.FlatChecksum, caps.VersionedChecksum)
	}

	if requests != 1 || responses != 1 {
		t.Fatalf("hooks fired request=%d response=%d, want 1/1", requests, responses)
	}
}

type recordingTransport struct {
	gotAuth   string
	gotCalled int
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.gotCalled++
	r.gotAuth = req.Header.Get("Authorization")

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"results":[]}`)),
		Request:    req,
	}, nil
}

func TestWithHTTPClientRoutesRequestsThroughSuppliedClient(t *testing.T) {
	t.Parallel()

	transport := &recordingTransport{}

	client, err := New("https://paperless.example.com", "secret-token",
		WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := client.Ping(t.Context()); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	if transport.gotCalled != 1 {
		t.Fatalf(
			"round trips = %d, want exactly 1 through the supplied client",
			transport.gotCalled,
		)
	}

	if transport.gotAuth != "Token secret-token" {
		t.Fatalf("Authorization seen by the supplied transport = %q", transport.gotAuth)
	}
}

// WithHTTPClient(nil) must be a no-op: the option's defensive branch keeps
// the default transport instead of planting a nil client.

// WithHTTPClient(nil) must be a no-op: the option's defensive branch keeps
// the default transport instead of planting a nil client.
func TestWithHTTPClientNilKeepsDefaultTransport(t *testing.T) {
	t.Parallel()

	client, err := New("https://paperless.example.com", "token", WithHTTPClient(nil))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tuned, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want the default *http.Transport", client.httpClient.Transport)
	}

	if tuned.MaxIdleConnsPerHost != DefaultMaxIdleConnsPerHost {
		t.Fatalf("MaxIdleConnsPerHost = %d, want the default %d",
			tuned.MaxIdleConnsPerHost, DefaultMaxIdleConnsPerHost)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[],"count":0}`))
	}))
	defer server.Close()

	client, err = New(server.URL, "token", WithHTTPClient(nil))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := client.Ping(t.Context()); err != nil {
		t.Fatalf("Ping with nil WithHTTPClient: %v", err)
	}
}

func TestDefaultTransportHonorsExportedConstants(t *testing.T) {
	t.Parallel()

	client, err := New("https://paperless.example.com", "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tuned, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.httpClient.Transport)
	}

	if tuned.MaxIdleConns != DefaultMaxIdleConns {
		t.Errorf("MaxIdleConns = %d, want %d", tuned.MaxIdleConns, DefaultMaxIdleConns)
	}

	if tuned.MaxIdleConnsPerHost != DefaultMaxIdleConnsPerHost {
		t.Errorf("MaxIdleConnsPerHost = %d, want %d",
			tuned.MaxIdleConnsPerHost, DefaultMaxIdleConnsPerHost)
	}

	if tuned.IdleConnTimeout != DefaultIdleConnTimeout {
		t.Errorf("IdleConnTimeout = %s, want %s",
			tuned.IdleConnTimeout, DefaultIdleConnTimeout)
	}
}

// failingPartWriter makes every multipart part write fail, forcing the
// error branches in the upload-metadata writers without a live server.
type failingPartWriter struct{}

func (failingPartWriter) Write([]byte) (int, error) {
	return 0, errPartWriteFailed
}

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
