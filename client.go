// Package paperless provides a Go client for the Paperless-ngx REST API:
// document upload, tag/correspondent/type/custom-field management, task
// polling, and document management.
//
// It targets API version 10 and authenticates via the static token scheme
// (`Authorization: Token <token>`), which users generate in the Paperless-ngx
// web UI under "My Profile".
//
// # Requirements
//
// Go 1.27 or newer. The module decodes with `encoding/json/v2`, which is
// part of the standard library there. On a Go 1.26 or older toolchain it
// only exists behind `GOEXPERIMENT=jsonv2` and builds fail without it.
//
// # Options
//
//	WithHTTPClient(*http.Client) — use a custom HTTP client (transport, proxies, timeouts)
//	WithTimeout(time.Duration)   — per-request timeout on the default client
package paperless

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-retry"
)

// Default transport tuning shared by every consumer of this SDK. The default
// transport keeps only 2 idle connections per host, so parallel requests to
// the single Paperless-ngx host would close and re-handshake TLS connections
// constantly; a cloned transport with a per-host idle pool turns that into
// connection reuse.
const (
	// DefaultMaxIdleConns is the total idle connection pool size.
	DefaultMaxIdleConns = 100
	// DefaultMaxIdleConnsPerHost is the per-host idle pool — the value that
	// actually matters when all traffic targets one API host.
	DefaultMaxIdleConnsPerHost = 8
	// DefaultIdleConnTimeout matches the default transport's 90s so pooled
	// connections do not outlive typical server keep-alive windows.
	DefaultIdleConnTimeout = 90 * time.Second
)

const (
	// apiVersion is the Accept-header API version this client targets.
	apiVersion = "10"

	// maxSuccessStatusCode is the inclusive upper bound of the 2xx range.
	maxSuccessStatusCode = 299

	// requestTimeout bounds every API call.
	requestTimeout = 60 * time.Second

	// maxErrorBodyBytes limits how much of an error response is read for context.
	maxErrorBodyBytes = 512

	// pathUploadDocument posts a file into the consumption queue.
	pathUploadDocument = "/api/documents/post_document/"

	// pathTasks serves the consumption-task queue; task outcome polling
	// filters it by the exact task ID (task_id=<uuid>).
	pathTasks = "/api/tasks/"

	// pathTags lists and creates tags.
	pathTags = "/api/tags/"

	// pathCorrespondents lists and creates correspondents.
	pathCorrespondents = "/api/correspondents/"

	// pathDocumentTypes lists and creates document types.
	pathDocumentTypes = "/api/document_types/"

	// pathCustomFields lists and creates custom field definitions.
	pathCustomFields = "/api/custom_fields/"

	// pathStoragePaths lists and creates storage paths.
	pathStoragePaths = "/api/storage_paths/"

	// pathDocumentDetail is the per-document DRF route with the ID spliced
	// in (view/update/delete single documents).
	pathDocumentDetail = "/api/documents/%d/"

	// pathDocumentDownload serves one document's stored ORIGINAL file
	// bytes (the decrypt-repair scan reads them to detect encryption).
	pathDocumentDownload = "/api/documents/%d/download/"

	// pathDocumentNotes reads, adds, and deletes a document's notes.
	// Not paginated: the response is a bare JSON array.
	pathDocumentNotes = "/api/documents/%d/notes/"

	// pathShareLinks lists, creates, and deletes share links. The slug is
	// server-generated; create sends only document, file_version, and the
	// optional expiration.
	pathShareLinks = "/api/share_links/"

	// pathSavedViews lists, creates, and deletes saved views.
	pathSavedViews = "/api/saved_views/"

	// pathDocuments lists documents; used for ledger reconciliation.
	pathDocuments = "/api/documents/"

	// documentListPageSize is the page size for document listing scans.
	documentListPageSize = 100

	// maxDocumentListPages bounds reconciliation scans (100 pages x 100
	// documents = 10,000 documents).
	maxDocumentListPages = 100
)

// matchingAlgorithm is Paperless-ngx's matching_algorithm enum. A distinct
// type keeps the self-heal path from PATCHing an arbitrary int into the tag
// payload.
type matchingAlgorithm int

const (
	// matchingAlgorithmNone is Paperless-ngx's "none" matching algorithm:
	// the object never participates in automatic (machine-learning) matching.
	matchingAlgorithmNone matchingAlgorithm = 0

	// matchingAlgorithmAuto is Paperless-ngx's "auto" matching algorithm.
	matchingAlgorithmAuto matchingAlgorithm = 6
)

// ErrInvalidConfig is returned when the client is constructed with a missing
// base URL or token. The integration is optional; callers that have not
// configured Paperless should never construct a client. New wraps it with the
// paperless.invalid_config code (Rejection family), so errors.Is keeps
// matching while Code/Classify speak.
var ErrInvalidConfig = errors.New(
	"paperless: base URL and token are required",
) //nolint:erraudit // documented errors.Is sentinel (ERROR_CODES.md); coded via paperless.invalid_config at the New call site

// Option configures a Client at construction time.
type Option func(*Client)

// WithHTTPClient uses the given HTTP client instead of the default one.
// Note that a later WithTimeout option still sets Timeout on the supplied
// client — options apply in order and the last writer wins.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithTimeout sets the per-request timeout of the client's HTTP client.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// Default retry tuning for WithRetry, mirroring go-retry's defaults.
const (
	// DefaultRetryMaxAttempts is the total attempt count including the first
	// call when a RetryPolicy leaves MaxAttempts unset.
	DefaultRetryMaxAttempts = 3
	// DefaultRetryInitialDelay is the pause before the second attempt when a
	// RetryPolicy leaves InitialDelay unset.
	DefaultRetryInitialDelay = 100 * time.Millisecond
	// DefaultRetryMaxDelay caps the exponential backoff when a RetryPolicy
	// leaves MaxDelay unset.
	DefaultRetryMaxDelay = 5 * time.Second
	// DefaultRetryMultiplier is the backoff growth factor when a RetryPolicy
	// leaves Multiplier unset.
	DefaultRetryMultiplier = 2.0
)

// RetryPolicy configures opt-in automatic retries. The zero value means
// "use the defaults" (DefaultRetryMaxAttempts attempts, 100ms initial
// delay, 5s cap, 2x growth).
//
// Retries apply to every API round-trip of the client: transient failures
// (network errors, 429/503 with Retry-After, 5xx) are retried with
// exponential backoff + jitter, server-provided Retry-After hints take
// precedence, and rejections (4xx) fail immediately. Without WithRetry the
// client makes exactly one attempt per call.
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts including the first call.
	// Zero selects DefaultRetryMaxAttempts; negative values are rejected by
	// New with ErrInvalidConfig.
	MaxAttempts int
	// InitialDelay is the pause before the second attempt. Zero (or
	// negative) selects DefaultRetryInitialDelay.
	InitialDelay time.Duration
	// MaxDelay caps the backoff between attempts — computed exponential
	// backoff AND a server's Retry-After hint alike. Zero (or negative)
	// selects DefaultRetryMaxDelay. Capping hints keeps a broken or hostile
	// server from stalling the caller indefinitely while still honoring
	// every hint that stays under the ceiling.
	MaxDelay time.Duration
	// Multiplier is the exponential backoff factor. Zero (or negative)
	// selects DefaultRetryMultiplier.
	Multiplier float64
}

// WithRetry turns on automatic retries for transient failures according to
// the given policy. The fail-fast single-attempt behavior is unchanged
// unless this option is passed.
func WithRetry(policy RetryPolicy) Option {
	return func(c *Client) {
		c.retry = &policy
	}
}

// WithRequestHook installs an observer called with a snapshot of every
// outgoing API request before it is sent. The header INCLUDES the
// Authorization token — hooks that log headers must redact it. The hook
// observes only; mutating the snapshot has no effect on the request.
func WithRequestHook(hook func(RequestInfo)) Option {
	return func(c *Client) {
		if hook != nil {
			c.requestHook = hook
		}
	}
}

// ResponseInfo is the observation snapshot of one API response. Body is the
// full body for 2xx responses and the error snippet (capped, see
// maxErrorBodyBytes) otherwise.
type ResponseInfo struct {
	Status int
	Header http.Header
	Body   []byte
}

// RequestInfo is the observation snapshot of one outgoing API request. Body
// is not included (multipart uploads are single-read streams).
type RequestInfo struct {
	Method string
	URL    string
	Header http.Header
}

// WithResponseHook installs an observer called with a snapshot of every API
// response after its body has been read. The header INCLUDES any
// Set-Cookie values — hooks that log headers must redact secrets. The hook
// observes only; mutating the snapshot has no effect on the caller's data.
func WithResponseHook(hook func(ResponseInfo)) Option {
	return func(c *Client) {
		if hook != nil {
			c.responseHook = hook
		}
	}
}

// retryConfig maps the policy onto go-retry's engine, bridging server
// Retry-After hints into the delay computation.
func (p RetryPolicy) retryConfig() retry.Config {
	maxAttempts := p.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = DefaultRetryMaxAttempts
	}

	initialDelay := p.InitialDelay
	if initialDelay <= 0 {
		initialDelay = DefaultRetryInitialDelay
	}

	maxDelay := p.MaxDelay
	if maxDelay <= 0 {
		maxDelay = DefaultRetryMaxDelay
	}

	multiplier := p.Multiplier
	if multiplier <= 0 {
		multiplier = DefaultRetryMultiplier
	}

	return retry.Config{
		MaxAttempts:  maxAttempts,
		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
		Multiplier:   multiplier,
		DelayFunc: func(_ int, err error) time.Duration {
			if hint, ok := errors.AsType[*RetryAfterError](err); ok {
				// The hint overrides the computed backoff but honors the
				// same MaxDelay ceiling; 0 (or below) keeps the computed
				// backoff per go-retry's DelayFunc contract.
				return min(hint.After, maxDelay)
			}

			return 0
		},
	}
}

// Client talks to a single Paperless-ngx instance. The zero value is not
// usable — construct via New.
type Client struct {
	baseURL      *url.URL
	token        string
	httpClient   *http.Client
	retry        *RetryPolicy
	requestHook  func(RequestInfo)
	responseHook func(ResponseInfo)
}

// New creates a client for the given base URL (e.g. "https://paperless.example.com")
// and API token. Returns ErrInvalidConfig (coded paperless.invalid_config,
// Rejection) when either is empty, the URL is not parseable, or a WithRetry
// policy is malformed. Options customize the HTTP transport.

//nolint:branching-flow // public API; a swapped token fails scheme validation; brands are ROADMAP v2 scope
func New(baseURL, token string, opts ...Option) (*Client, error) {
	if baseURL == "" || token == "" {
		missing := "baseURL and token"

		switch {
		case baseURL == "":
			missing = "baseURL"
		case token == "":
			missing = "token"
		}

		return nil, errorfamily.WrapRejection(
			ErrInvalidConfig,
			"paperless.invalid_config",
			missing+" must not be empty",
		).WithContext("url", baseURL)
	}

	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "paperless.invalid_url", "base URL is not parseable").
			WithContext("url", baseURL)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errorfamily.NewRejection("paperless.invalid_url", "base URL must use http or https").
			WithContext("url", baseURL)
	}

	client := &Client{
		baseURL: parsed,
		token:   token,
		httpClient: &http.Client{
			Timeout:   requestTimeout,
			Transport: defaultTransport(),
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.retry != nil && client.retry.MaxAttempts < 0 {
		return nil, errorfamily.WrapRejectionf(
			ErrInvalidConfig,
			"paperless.invalid_retry", //nolint:erraudit // MaxAttempts value is in the message; baseURL is irrelevant
			"retry MaxAttempts must be >= 0, got %d",
			client.retry.MaxAttempts,
		)
	}

	return client, nil
}

// defaultTransport clones the default transport with a per-host idle-connection
// pool sized for concurrent uploads. A nil result makes http.Client fall back
// to the untouched default transport (the defensive path taken when the
// default is not a *http.Transport).
func defaultTransport() *http.Transport {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil
	}

	tuned := transport.Clone()
	tuned.MaxIdleConns = DefaultMaxIdleConns
	tuned.MaxIdleConnsPerHost = DefaultMaxIdleConnsPerHost
	tuned.IdleConnTimeout = DefaultIdleConnTimeout

	return tuned
}
