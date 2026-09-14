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
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
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
// configured Paperless should never construct a client.
var ErrInvalidConfig = errors.New("paperless: base URL and token are required")

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
	// InitialDelay is the pause before the second attempt. Zero selects
	// DefaultRetryInitialDelay.
	InitialDelay time.Duration
	// MaxDelay caps the backoff between attempts. Zero selects
	// DefaultRetryMaxDelay.
	MaxDelay time.Duration
	// Multiplier is the exponential backoff factor. Zero selects
	// DefaultRetryMultiplier.
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
				return hint.After
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
// and API token. Returns ErrInvalidConfig when either is empty, the URL is
// not parseable, or a WithRetry policy is malformed. Options customize the
// HTTP transport.
func New(baseURL, token string, opts ...Option) (*Client, error) {
	if baseURL == "" || token == "" {
		return nil, ErrInvalidConfig
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
		return nil, fmt.Errorf("%w: retry MaxAttempts must be >= 0", ErrInvalidConfig)
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

// UploadRequest describes one document to hand to Paperless-ngx's consumer.
type UploadRequest struct {
	// Filename is the original file name (e.g. "invoice.pdf"). Paperless-ngx
	// derives the stored document name and archive serial numbering from it.
	Filename string
	// Content holds the raw file bytes. Gmail attachments are capped at 25 MB,
	// so buffering in memory is safe.
	Content []byte
	// Title overrides the consumer-assigned title when set.
	Title string
	// Created sets the document's creation date (typically the email date).
	Created time.Time
	// CorrespondentID is the Paperless-ngx correspondent (e.g. the email
	// sender) assigned on consumption. Zero leaves the field unset.
	CorrespondentID int
	// TagIDs are Paperless-ngx tag IDs applied on consumption.
	TagIDs []int
	// DocumentTypeID is the Paperless-ngx document type (e.g. "Email")
	// applied on consumption. Zero leaves the type unset.
	DocumentTypeID int
	// CustomFields are custom-field values applied on consumption, e.g. the
	// provenance field carrying the Gmail message ID. Nil leaves them unset.
	CustomFields []CustomFieldValue
}

// CustomFieldValue assigns a value to one custom field by its definition ID.
type CustomFieldValue struct {
	Field int
	Value string
}

// Upload posts a document to the consumption queue. It returns the task ID
// (UUID) Paperless-ngx assigned; consumption itself is asynchronous — the
// caller can poll `/api/tasks/?task_id=<id>` for the resulting document.
func (c *Client) Upload(ctx context.Context, req UploadRequest) (string, error) {
	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	filePart, err := writer.CreateFormFile("document", req.Filename)
	if err != nil {
		return "", errorfamily.WrapInfrastructure(err, "paperless.build_multipart", "could not build upload form").
			WithContext("filename", req.Filename)
	}

	if _, writeErr := filePart.Write(req.Content); writeErr != nil {
		return "", errorfamily.WrapInfrastructure(writeErr, "paperless.write_document", "could not write document bytes").
			WithContext("filename", req.Filename)
	}

	if metaErr := writeUploadMetadata(writer, req); metaErr != nil {
		return "", metaErr
	}

	if closeErr := writer.Close(); closeErr != nil {
		return "", errorfamily.WrapInfrastructure(
			closeErr,
			"paperless.close_multipart",
			"could not finalize upload form",
		)
	}

	taskIDBytes, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathUploadDocument,
		"",
		&body,
		writer.FormDataContentType(),
	)
	if err != nil {
		return "", fmt.Errorf(
			"upload %q: %w",
			req.Filename,
			err,
		) //nolint:erraudit // uncoded context wrap keeps the inner coded error's code+family
	}

	taskID := strings.Trim(strings.TrimSpace(string(taskIDBytes)), `"'`)
	if taskID == "" {
		return "", errorfamily.NewCorruption("paperless.empty_task_id",
			"Paperless-ngx accepted the upload but returned no task ID").
			WithContext("filename", req.Filename)
	}

	return taskID, nil
}

// TaskStatus is a Paperless-ngx consumption task's lifecycle state. The v10
// API serves lowercase values; terminal states are success and failure
// (verified against paperless-ngx 3.x PaperlessTask.Status).
type TaskStatus string

// Task consumption statuses served by the v10 API (lowercase values).
const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusStarted TaskStatus = "started"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusFailure TaskStatus = "failure"
)

// Terminal reports whether the status is a final task state. Unknown status
// strings (future server versions) poll forever — the safe default for a
// caller deciding whether to retry.
func (s TaskStatus) Terminal() bool {
	return s == TaskStatusSuccess || s == TaskStatusFailure
}

// TaskOutcome is the classified result of one consumption-task poll.
type TaskOutcome struct {
	Status TaskStatus
	// DocumentID is the server document this task points at: the newly
	// created document after a successful consumption, or the pre-existing
	// duplicate when the server refused the upload. Zero when unknown.
	DocumentID int64
	// DuplicateRefused is true when the consumer rejected the upload because
	// an identical document already exists (checksum dedup). The ledger uses
	// this to separate honest refusals from real failures.
	DuplicateRefused bool
	// DuplicateInTrash refines DuplicateRefused: the duplicate document is
	// in the trash (server treats trash duplicates as re-consumable).
	DuplicateInTrash bool
	// ErrorMessage is the server's failure reason for non-duplicate failures.
	ErrorMessage string
}

// Duplicate reports the duplicate-refusal details of an outcome. refused is
// true when the consumer rejected the upload because an identical document
// already exists (checksum dedup); documentID is that pre-existing duplicate
// and inTrash says whether it sits in the trash (the server treats trash
// duplicates as re-consumable). For non-refusals all returns are zero.
func (o TaskOutcome) Duplicate() (int64, bool, bool) {
	if !o.DuplicateRefused {
		return 0, false, false
	}

	return o.DocumentID, o.DuplicateInTrash, true
}

// taskResultData mirrors the free-form result_data dict Paperless-ngx stores
// on a finished task: {"document_id": N} after a successful consumption,
// {"duplicate_of": N, "duplicate_in_trash": bool} for refused duplicates
// (the postrun handler demotes such tasks to failure), and
// {"error_type", "error_message"} for real failures. Pointers distinguish
// absent keys from zero values.
type taskResultData struct {
	DocumentID       *int64  `json:"document_id"`
	DuplicateOf      *int64  `json:"duplicate_of"`
	DuplicateInTrash bool    `json:"duplicate_in_trash"`
	ErrorMessage     *string `json:"error_message"`
}

// taskPayload mirrors the Paperless-ngx v10 task object (TaskSerializerV10).
type taskPayload struct {
	TaskID             string         `json:"task_id"`
	Status             string         `json:"status"`
	ResultData         taskResultData `json:"result_data"`
	RelatedDocumentIDs []int64        `json:"related_document_ids"`
}

// classifyTask turns the server's task payload into the classified outcome.
// DocumentID prefers result_data.document_id, falls back to duplicate_of,
// then to the first related_document_ids entry (the server's projection of
// exactly those two fields).
func classifyTask(payload taskPayload) TaskOutcome {
	status := TaskStatus(strings.ToLower(strings.TrimSpace(payload.Status)))
	outcome := TaskOutcome{Status: status}

	switch {
	case payload.ResultData.DocumentID != nil:
		outcome.DocumentID = *payload.ResultData.DocumentID
	case payload.ResultData.DuplicateOf != nil:
		outcome.DocumentID = *payload.ResultData.DuplicateOf
		outcome.DuplicateRefused = true
		outcome.DuplicateInTrash = payload.ResultData.DuplicateInTrash
	case len(payload.RelatedDocumentIDs) > 0:
		outcome.DocumentID = payload.RelatedDocumentIDs[0]
	}

	if payload.ResultData.ErrorMessage != nil {
		outcome.ErrorMessage = *payload.ResultData.ErrorMessage
	}

	return outcome
}

// GetTask fetches one consumption task by ID (the UUID Upload returns).
// found is false when the server has no such task — the record may not be
// persisted yet (polling callers retry) or was pruned.
//
// The response is expected in the v10 paginated envelope
// (`{"results": [...]}`); a bare JSON array is tolerated defensively for
// deployments that predate pagination.
func (c *Client) GetTask(ctx context.Context, taskID string) (TaskOutcome, bool, error) {
	query := url.Values{}
	query.Set("task_id", taskID)

	raw, reqErr := c.doRequest(ctx, http.MethodGet, pathTasks, query.Encode(), nil, "")
	if reqErr != nil {
		return TaskOutcome{}, false, fmt.Errorf("get task %s: %w", taskID, reqErr)
	}

	var envelope struct {
		Results []taskPayload `json:"results"`
	}

	if unmarshalErr := json.Unmarshal(raw, &envelope); unmarshalErr != nil {
		// Tolerate a bare JSON array too (defensive, mirrors checksumFrom):
		// the v10 endpoint paginates, but older deployments may not. The
		// reported error wraps the ENVELOPE failure, not the array attempt,
		// because the envelope shape is what v10 promises — the array shape
		// is a fallback, and its failure adds no diagnostic value. The
		// second Unmarshal reallocates envelope.Results; negligible at a
		// one-element payload.
		envelope.Results = nil

		if arrayErr := json.Unmarshal(raw, &envelope.Results); arrayErr != nil {
			return TaskOutcome{}, false, errorfamily.WrapCorruption(
				unmarshalErr,
				"paperless.decode_task",
				"could not decode task poll result",
			).WithContext("task_id", taskID)
		}
	}

	if len(envelope.Results) == 0 {
		return TaskOutcome{}, false, nil
	}

	return classifyTask(envelope.Results[0]), true, nil
}

// DefaultTaskPollInterval is the pause between polls used by WaitForTask when
// the caller passes a non-positive interval.
const DefaultTaskPollInterval = 2 * time.Second

// WaitForTask polls one consumption task until it reaches a terminal state
// (success or failure), the context is done, or the server stops answering.
//
// Polling starts immediately; interval is the pause between polls (a
// non-positive interval means DefaultTaskPollInterval). Bounds come from the
// context: pass ctx with a deadline/timeout to cap total wait time.
//
// The task may not be visible yet right after Upload returns (the server
// persists it asynchronously) — that case keeps polling. Transient poll
// failures (network blips, decode hiccups) also keep polling; the most
// recent error surfaces only when the context expires.
//
// On a terminal outcome the outcome is fully populated. err is non-nil only
// for a real terminal failure (status failure without a duplicate refusal —
// duplicate refusals are honest outcomes, not errors).
func (c *Client) WaitForTask(
	ctx context.Context,
	taskID string,
	interval time.Duration,
) (TaskOutcome, error) {
	if taskID == "" {
		return TaskOutcome{}, errorfamily.NewRejection("paperless.empty_task_id",
			"task ID is required to poll a consumption task")
	}

	if interval <= 0 {
		interval = DefaultTaskPollInterval
	}

	var lastErr error

	for {
		outcome, found, err := c.GetTask(ctx, taskID)
		if err != nil {
			lastErr = err
		} else if found && outcome.Status.Terminal() {
			if outcome.Status == TaskStatusFailure && !outcome.DuplicateRefused {
				return outcome, errorfamily.NewRejection("paperless.task_failed",
					"Paperless-ngx consumption task failed").
					WithContext("task_id", taskID).
					WithContextAny("error_message", outcome.ErrorMessage)
			}

			return outcome, nil
		}

		select {
		case <-ctx.Done():
			reason := errorfamily.WrapInfrastructure(ctx.Err(),
				"paperless.task_poll_abandoned",
				"gave up waiting for the consumption task",
			).WithContext("task_id", taskID)

			if lastErr != nil {
				return TaskOutcome{}, fmt.Errorf("%w (last poll error: %w)", reason, lastErr)
			}

			return TaskOutcome{}, reason
		case <-time.After(interval):
		}
	}
}

// writeUploadMetadata adds the optional metadata form fields shared by all uploads.
func writeUploadMetadata(writer *multipart.Writer, req UploadRequest) error {
	if req.Title != "" {
		if err := writer.WriteField("title", req.Title); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"paperless.write_title",
				"could not write title field",
			)
		}
	}

	if !req.Created.IsZero() {
		if err := writer.WriteField("created", req.Created.Format(time.DateOnly)); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"paperless.write_created",
				"could not write created field",
			)
		}
	}

	if req.CorrespondentID > 0 {
		if err := writer.WriteField(
			"correspondent",
			strconv.Itoa(req.CorrespondentID),
		); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"paperless.write_correspondent",
				"could not write correspondent field",
			)
		}
	}

	for _, tagID := range req.TagIDs {
		if err := writer.WriteField("tags", strconv.Itoa(tagID)); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"paperless.write_tags",
				"could not write tags field",
			)
		}
	}

	if req.DocumentTypeID > 0 {
		if err := writer.WriteField(
			"document_type",
			strconv.Itoa(req.DocumentTypeID),
		); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"paperless.write_document_type",
				"could not write document_type field",
			)
		}
	}

	return writeCustomFieldsField(writer, req.CustomFields)
}

// writeCustomFieldsField serializes the custom-field values into the JSON
// form field Paperless-ngx's consumption endpoint expects.
func writeCustomFieldsField(writer *multipart.Writer, fields []CustomFieldValue) error {
	if len(fields) == 0 {
		return nil
	}

	params := make([]customFieldValuePayload, 0, len(fields))
	for _, field := range fields {
		params = append(params, customFieldValuePayload(field))
	}

	encoded, err := json.Marshal(params)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"paperless.write_custom_fields",
			"could not encode custom_fields field",
		)
	}

	if err := writer.WriteField("custom_fields", string(encoded)); err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"paperless.write_custom_fields",
			"could not write custom_fields field",
		)
	}

	return nil
}

// EnsureTag returns the ID of the tag with the given name, creating it when
// it does not exist yet.
//
// Tags are provenance metadata applied deterministically at upload time, so
// they are created with the "none" matching algorithm: a "gmail" tag that
// Paperless-ngx's classifier learns to predict would silently leak onto
// manually scanned documents that merely look similar. An existing tag still
// carrying "auto" (created by older InboxClean versions) is self-healed to
// "none" — the classifier then no longer trains against it. Self-healing is
// tag-specific on purpose: correspondents deliberately keep "auto".
func (c *Client) EnsureTag(ctx context.Context, name string) (int, error) {
	existing, found, findErr := c.findNamed(ctx, pathTags, "tag", name)
	if findErr != nil {
		return 0, fmt.Errorf("find tag %q: %w", name, findErr)
	}

	if found {
		if existing.MatchingAlgorithm == int(matchingAlgorithmAuto) {
			if err := c.updateMatchingAlgorithm(
				ctx,
				pathTags,
				"tag",
				name,
				existing.ID,
				matchingAlgorithmNone,
			); err != nil {
				return 0, fmt.Errorf("demote auto tag %q: %w", name, err)
			}
		}

		return existing.ID, nil
	}

	return c.createNamed(ctx, pathTags, "tag", name, matchingAlgorithmNone)
}

// updateMatchingAlgorithm PATCHes an existing object's matching algorithm
// (used to self-heal legacy auto tags down to none).
func (c *Client) updateMatchingAlgorithm(
	ctx context.Context,
	endpoint,
	kind,
	name string,
	id int,
	algorithm matchingAlgorithm,
) error {
	payload, err := json.Marshal(namedPayload{MatchingAlgorithm: int(algorithm)})
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"paperless.marshal_"+kind+"_update",
			"could not encode "+kind+" update payload",
		).WithContext(kind, name)
	}

	if _, reqErr := c.doRequest(
		ctx,
		http.MethodPatch,
		endpoint+strconv.Itoa(id)+"/",
		"",
		bytes.NewReader(payload),
		"application/json",
	); reqErr != nil {
		return fmt.Errorf("update %s %q: %w", kind, name, reqErr)
	}

	return nil
}

// namedPayload mirrors the create/list subset shared by Paperless-ngx's tag and
// correspondent objects (id + name + matching algorithm).
type namedPayload struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	MatchingAlgorithm int    `json:"matching_algorithm"`
}

// findNamed looks up a Paperless-ngx named object (tag, correspondent) by
// exact (case-insensitive) name. found is false when no match exists.
func (c *Client) findNamed(
	ctx context.Context,
	endpoint, kind, name string,
) (namedPayload, bool, error) {
	query := url.Values{}
	query.Set("name__iexact", name)
	query.Set("page_size", "1")

	raw, reqErr := c.doRequest(ctx, http.MethodGet, endpoint, query.Encode(), nil, "")
	if reqErr != nil {
		return namedPayload{}, false, reqErr
	}

	list := struct {
		Results []namedPayload `json:"results"`
	}{}

	if unmarshalErr := json.Unmarshal(raw, &list); unmarshalErr != nil {
		return namedPayload{}, false, errorfamily.WrapCorruption(
			unmarshalErr,
			"paperless.decode_"+kind+"s",
			"could not decode "+kind+" search result",
		).WithContext(kind, name)
	}

	if len(list.Results) == 0 {
		return namedPayload{}, false, nil
	}

	return list.Results[0], true, nil
}

// EnsureCorrespondent returns the ID of the correspondent with the given
// name, creating it when it does not exist yet.
//
// Correspondents keep the "auto" matching algorithm (deliberately, unlike
// tags): learning "documents from sender X look like this" is exactly the
// value auto matching adds for documents NOT uploaded by this pipeline
// (manual scans), and correspondents carry no provenance semantics that
// auto prediction could corrupt.
func (c *Client) EnsureCorrespondent(ctx context.Context, name string) (int, error) {
	return c.ensureNamed(ctx, pathCorrespondents, "correspondent", name)
}

// EnsureDocumentType returns the ID of the document type with the given
// name, creating it with the "none" matching algorithm when missing. A
// document type applied deterministically at upload time is provenance
// metadata: classifier-learned matching would leak the type onto manual
// scans, mirroring the tags rationale. An EXISTING type keeps its
// configured algorithm — unlike tags there is no legacy-release shape to
// heal, and overriding a deliberate user config would be wrong.
func (c *Client) EnsureDocumentType(ctx context.Context, name string) (int, error) {
	existing, found, findErr := c.findNamed(ctx, pathDocumentTypes, "document type", name)
	if findErr != nil {
		return 0, fmt.Errorf("find document type %q: %w", name, findErr)
	}

	if found {
		return existing.ID, nil
	}

	return c.createNamed(ctx, pathDocumentTypes, "document type", name, matchingAlgorithmNone)
}

// customFieldPayload mirrors one custom field definition.
type customFieldPayload struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	DataType string `json:"data_type"`
}

// FindCustomField looks up a custom field definition by name WITHOUT
// creating it — the read-only half of EnsureCustomField, used by dry-run
// backfills to report what they would record without mutating the server.
func (c *Client) FindCustomField(ctx context.Context, name string) (int, bool, error) {
	query := url.Values{}
	query.Set("name__iexact", name)
	query.Set("page_size", "1")

	raw, err := c.doRequest(ctx, http.MethodGet, pathCustomFields, query.Encode(), nil, "")
	if err != nil {
		return 0, false, fmt.Errorf("find custom field %q: %w", name, err)
	}

	list := struct {
		Results []customFieldPayload `json:"results"`
	}{}

	if unmarshalErr := json.Unmarshal(raw, &list); unmarshalErr != nil {
		return 0, false, errorfamily.WrapCorruption(
			unmarshalErr,
			"paperless.decode_custom_fields",
			"could not decode custom field search result",
		).WithContext("name", name)
	}

	if len(list.Results) == 0 {
		return 0, false, nil
	}

	return list.Results[0].ID, true, nil
}

// EnsureCustomField returns the ID of the custom field definition with the
// given name, creating it as a string field when missing (paperless-ngx 2.x+
// custom fields). Used for the provenance field carrying the Gmail message
// ID. An existing definition of any data type is kept — the caller decides
// whether a non-string field is acceptable by the value it assigns.
func (c *Client) EnsureCustomField(ctx context.Context, name string) (int, error) {
	fieldID, found, findErr := c.FindCustomField(ctx, name)
	if findErr != nil {
		return 0, findErr
	}

	if found {
		return fieldID, nil
	}

	payload, err := json.Marshal(customFieldPayload{Name: name, DataType: "string"})
	if err != nil {
		return 0, errorfamily.WrapInfrastructure(
			err,
			"paperless.marshal_custom_field",
			"could not encode custom field payload",
		).WithContext("name", name)
	}

	raw, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathCustomFields,
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return 0, fmt.Errorf("create custom field %q: %w", name, err)
	}

	created := customFieldPayload{}
	if unmarshalErr := json.Unmarshal(raw, &created); unmarshalErr != nil {
		return 0, errorfamily.WrapCorruption(
			unmarshalErr,
			"paperless.decode_custom_field",
			"could not decode created custom field",
		).WithContext("name", name)
	}

	return created.ID, nil
}

// StoragePath is one Paperless-ngx storage path definition: the server-side
// directory template (e.g. "{created_year}/{correspondent}") documents are
// filed into, addressed by name.
type StoragePath struct {
	ID   int
	Slug string
	Name string
	Path string
}

// storagePathPayload mirrors Paperless-ngx's storage path serializer fields
// the SDK relies on.
type storagePathPayload struct {
	ID   int    `json:"id"`
	Slug string `json:"slug,omitempty"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// FindStoragePath looks up a storage path by exact (case-insensitive) name
// WITHOUT creating it — the read-only lookup for dry-run reporting.
func (c *Client) FindStoragePath(ctx context.Context, name string) (int, bool, error) {
	query := url.Values{}
	query.Set("name__iexact", name)
	query.Set("page_size", "1")

	raw, reqErr := c.doRequest(ctx, http.MethodGet, pathStoragePaths, query.Encode(), nil, "")
	if reqErr != nil {
		return 0, false, reqErr
	}

	list := struct {
		Results []storagePathPayload `json:"results"`
	}{}

	if unmarshalErr := json.Unmarshal(raw, &list); unmarshalErr != nil {
		return 0, false, errorfamily.WrapCorruption(
			unmarshalErr,
			"paperless.decode_storage_paths",
			"could not decode storage path search result",
		).WithContext("name", name)
	}

	if len(list.Results) == 0 {
		return 0, false, nil
	}

	return list.Results[0].ID, true, nil
}

// EnsureStoragePath returns the ID of the storage path with the given name,
// creating it with the given directory template when missing. An EXISTING
// path keeps its configured template — mirroring the document type policy:
// the template is deliberate user config, and overriding it from a sync
// pipeline would be wrong.
func (c *Client) EnsureStoragePath(ctx context.Context, name, path string) (int, error) {
	if name == "" || path == "" {
		return 0, errorfamily.NewRejection("paperless.empty_storage_path",
			"storage path name and directory template are required")
	}

	existing, found, findErr := c.FindStoragePath(ctx, name)
	if findErr != nil {
		return 0, fmt.Errorf("find storage path %q: %w", name, findErr)
	}

	if found {
		return existing, nil
	}

	payload, err := json.Marshal(storagePathPayload{Name: name, Path: path})
	if err != nil {
		return 0, errorfamily.WrapInfrastructure(
			err,
			"paperless.marshal_storage_path",
			"could not encode storage path payload",
		).WithContext("name", name)
	}

	raw, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathStoragePaths,
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return 0, fmt.Errorf("create storage path %q: %w", name, err)
	}

	created := storagePathPayload{}
	if unmarshalErr := json.Unmarshal(raw, &created); unmarshalErr != nil {
		return 0, errorfamily.WrapCorruption(
			unmarshalErr,
			"paperless.decode_storage_path",
			"could not decode created storage path",
		).WithContext("name", name)
	}

	return created.ID, nil
}

// ListStoragePaths returns every storage path definition on the server,
// paginating the same bounded way as the document listings.
func (c *Client) ListStoragePaths(ctx context.Context) ([]StoragePath, error) {
	payloads, listErr := fetchAllPages[storagePathPayload](ctx, c, pathStoragePaths, "storage path")
	if listErr != nil {
		return nil, listErr
	}

	paths := make([]StoragePath, 0, len(payloads))
	for _, payload := range payloads {
		paths = append(paths, StoragePath(payload))
	}

	return paths, nil
}

// GetCorrespondentName resolves a correspondent ID to its display name.
// The backfill needs it to decide whether a document's existing non-null
// correspondent actually matches the derived sender name — an ID alone
// cannot answer that question.
func (c *Client) GetCorrespondentName(ctx context.Context, id int) (string, error) {
	return c.getNamedDetail(ctx, pathCorrespondents, "correspondent", id)
}

// GetDocumentTypeName resolves a document type ID to its display name, the
// document-type sibling of GetCorrespondentName: the backfill compares the
// stored name against the configured type to repair drifted assignments.
func (c *Client) GetDocumentTypeName(ctx context.Context, id int) (string, error) {
	return c.getNamedDetail(ctx, pathDocumentTypes, "document type", id)
}

// getNamedDetail fetches one named Paperless-ngx object (correspondent,
// document type) by ID and returns its display name.
func (c *Client) getNamedDetail(
	ctx context.Context,
	endpoint, kind string,
	id int,
) (string, error) {
	raw, err := c.doRequest(
		ctx,
		http.MethodGet,
		endpoint+strconv.Itoa(id)+"/",
		"",
		nil,
		"",
	)
	if err != nil {
		return "", fmt.Errorf("get %s %d: %w", kind, id, err)
	}

	detail := namedPayload{}
	if unmarshalErr := json.Unmarshal(raw, &detail); unmarshalErr != nil {
		return "", errorfamily.WrapCorruption(
			unmarshalErr,
			"paperless.decode_"+strings.ReplaceAll(kind, " ", "_"),
			"could not decode "+kind,
		).WithContext("id", strconv.Itoa(id))
	}

	return detail.Name, nil
}

// ensureNamed returns the ID of the named Paperless-ngx object (tag,
// correspondent), creating it with the "auto" matching algorithm when it
// does not exist yet. kind names the object family for error codes,
// messages, and log context.
func (c *Client) ensureNamed(ctx context.Context, endpoint, kind, name string) (int, error) {
	existing, found, findErr := c.findNamed(ctx, endpoint, kind, name)
	if findErr != nil {
		return 0, fmt.Errorf("find %s %q: %w", kind, name, findErr)
	}

	if found {
		return existing.ID, nil
	}

	return c.createNamed(ctx, endpoint, kind, name, matchingAlgorithmAuto)
}

// createNamed POSTs a new named object with the requested matching
// algorithm and returns its ID.
func (c *Client) createNamed(
	ctx context.Context,
	endpoint,
	kind,
	name string,
	algorithm matchingAlgorithm,
) (int, error) {
	payload, err := json.Marshal(namedPayload{Name: name, MatchingAlgorithm: int(algorithm)})
	if err != nil {
		return 0, errorfamily.WrapInfrastructure(
			err,
			"paperless.marshal_"+kind,
			"could not encode "+kind+" payload",
		).WithContext(kind, name)
	}

	raw, err := c.doRequest(
		ctx,
		http.MethodPost,
		endpoint,
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return 0, fmt.Errorf("create %s %q: %w", kind, name, err)
	}

	created := namedPayload{}
	if err := json.Unmarshal(raw, &created); err != nil {
		return 0, errorfamily.WrapCorruption(
			err,
			"paperless.decode_"+kind,
			"could not decode created "+kind,
		).WithContext(kind, name)
	}

	return created.ID, nil
}

// Ping verifies base URL and token by hitting the authenticated document
// list view (a real JSON endpoint, bounded to one entry). The API root is
// deliberately avoided: Paperless-ngx serves it as browsable HTML only, so
// a JSON Accept header is answered 406 regardless of token validity
// (observed on paperless 3.0.5 — 406 classified as a client rejection,
// silently skipping every sync). A 401/403 means the token is wrong; a
// transport error means the URL is unreachable.
func (c *Client) Ping(ctx context.Context) error {
	// art-dupl:accept first-page query boilerplate shared with the document list pagination below
	query := url.Values{}
	query.Set("page", "1")
	query.Set("page_size", "1")

	if _, err := c.doRequest(
		ctx,
		http.MethodGet,
		pathDocuments,
		query.Encode(),
		nil,
		"",
	); err != nil {
		return fmt.Errorf("ping %s: %w", c.baseURL, err)
	}

	return nil
}

// documentListEntry mirrors the checksum-bearing shape of Paperless-ngx's
// document objects; only the checksum is needed for ledger reconciliation.
// Since paperless-ngx 3.x the flat checksum is gone from the serializer and
// the versions array carries it instead.
type documentListEntry struct {
	Checksum string                   `json:"checksum"`
	Versions []documentVersionPayload `json:"versions"`
}

// effectiveChecksum returns the document's content checksum across API
// shapes: the flat field served by older paperless-ngx versions first, then
// the root version's checksum (3.x), then any non-empty version checksum.
func (e documentListEntry) effectiveChecksum() string {
	return checksumFrom(e.Checksum, e.Versions)
}

// checksumFrom resolves a document checksum across API shapes; shared by
// the verify (documentListEntry) and backfill (documentMetaPayload) paths.
func checksumFrom(flat string, versions []documentVersionPayload) string {
	if flat != "" {
		return flat
	}

	for _, version := range versions {
		if version.IsRoot && version.Checksum != "" {
			return version.Checksum
		}
	}

	for _, version := range versions {
		if version.Checksum != "" {
			return version.Checksum
		}
	}

	return ""
}

// fetchAllPages walks a paginated DRF list endpoint (page/page_size query
// parameters, `{"results": [...]}` envelope) up to maxDocumentListPages and
// returns the concatenated entries. A short page ends the scan; the page cap
// bounds it defensively so a server serving full pages forever cannot hang
// callers. resource is the singular object name, used to build error messages
// and error codes ("storage path" → "list storage paths (page 2)").
func fetchAllPages[T any](
	ctx context.Context,
	c *Client,
	path, resource string,
) ([]T, error) {
	items := []T{}

	for page := 1; page <= maxDocumentListPages; page++ {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("page_size", strconv.Itoa(documentListPageSize))

		raw, err := c.doRequest(ctx, http.MethodGet, path, query.Encode(), nil, "")
		if err != nil {
			return nil, fmt.Errorf("list %ss (page %d): %w", resource, page, err)
		}

		list := struct {
			Results []T `json:"results"`
		}{}

		if unmarshalErr := json.Unmarshal(raw, &list); unmarshalErr != nil {
			code := "paperless.decode_" + strings.ReplaceAll(resource, " ", "_") + "s"

			return nil, errorfamily.WrapCorruption(unmarshalErr, code,
				"could not decode "+resource+"s list").
				WithContext("page", strconv.Itoa(page))
		}

		items = append(items, list.Results...)

		if len(list.Results) < documentListPageSize {
			break
		}
	}

	return items, nil
}

// ListDocumentChecksums returns the SHA-256 checksum of every document
// currently stored in Paperless-ngx. Used to reconcile the local upload
// ledger against reality: ledger entries whose checksum disappeared from
// Paperless-ngx (documents deleted there) are candidates for re-upload.
func (c *Client) ListDocumentChecksums(ctx context.Context) (map[string]struct{}, error) {
	entries, err := fetchAllPages[documentListEntry](ctx, c, pathDocuments, "document")
	if err != nil {
		return nil, err
	}

	checksums := map[string]struct{}{}

	for _, entry := range entries {
		if checksum := entry.effectiveChecksum(); checksum != "" {
			checksums[checksum] = struct{}{}
		}
	}

	return checksums, nil
}

// documentMetaPayload mirrors the subset of Paperless-ngx's document object
// the metadata backfill needs. correspondent and tags decode to IDs; a null
// correspondent decodes as 0. created stays a string so both the datetime
// form Paperless-ngx serves and legacy date-only values parse.
type documentMetaPayload struct {
	ID            int                       `json:"id"`
	Title         string                    `json:"title"`
	Correspondent int                       `json:"correspondent"`
	Created       string                    `json:"created"`
	Tags          []int                     `json:"tags"`
	DocumentType  int                       `json:"document_type"`
	CustomFields  []customFieldValuePayload `json:"custom_fields"`
	Checksum      string                    `json:"checksum"`
	Versions      []documentVersionPayload  `json:"versions"`
}

// documentVersionPayload mirrors one entry of Paperless-ngx's document
// versions array. Since paperless-ngx 3.x the flat document checksum is no
// longer part of the serializer fields; the checksum moved into the
// versions array (is_root marks the current original).
type documentVersionPayload struct {
	ID       int    `json:"id"`
	Checksum string `json:"checksum"`
	IsRoot   bool   `json:"is_root"`
}

// effectiveChecksum returns the document's content checksum across API
// shapes: the flat field served by older paperless-ngx versions first, then
// the root version's checksum (3.x), then any non-empty version checksum.
func (p documentMetaPayload) effectiveChecksum() string {
	return checksumFrom(p.Checksum, p.Versions)
}

// parseDocumentCreated parses the created value a stored document serves
// back: full RFC 3339 datetimes, timezone-less datetimes, and bare dates
// (uploads send date-only created fields). Timezone-less values are read
// as UTC (time.Parse's zero-offset default); Paperless-ngx stores its
// dates in UTC. Unparseable values yield the zero time.
func parseDocumentCreated(raw string) time.Time {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		time.DateOnly,
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed
		}
	}

	return time.Time{}
}

// DocumentMeta is the metadata snapshot of one stored Paperless-ngx document.
type DocumentMeta struct {
	ID             int
	Title          string
	Correspondent  int
	Created        time.Time
	TagIDs         []int
	DocumentTypeID int
	CustomFields   []CustomFieldValue
	Checksum       string
}

// ListDocumentMetas returns id, title, correspondent, created date, tags,
// and checksum of every stored document. The backfill matches ledger
// records against server documents (by checksum) and detects duplicates
// (same checksum stored more than once).
func (c *Client) ListDocumentMetas(ctx context.Context) ([]DocumentMeta, error) {
	entries, err := fetchAllPages[documentMetaPayload](ctx, c, pathDocuments, "document meta")
	if err != nil {
		return nil, err
	}

	metas := make([]DocumentMeta, 0, len(entries))
	for i := range entries {
		entry := &entries[i]

		metas = append(metas, DocumentMeta{
			ID:             entry.ID,
			Title:          entry.Title,
			Correspondent:  entry.Correspondent,
			Created:        parseDocumentCreated(entry.Created),
			TagIDs:         entry.Tags,
			DocumentTypeID: entry.DocumentType,
			CustomFields:   customFieldsFromPayload(entry.CustomFields),
			Checksum:       entry.effectiveChecksum(),
		})
	}

	return metas, nil
}

// UpdateDocumentRequest carries the fields a metadata backfill may change.
// Nil pointers leave the field untouched; TagIDs replaces the full tag set
// (Paperless-ngx PATCH semantics for many-to-many fields).
type UpdateDocumentRequest struct {
	Title           *string
	Created         *time.Time
	CorrespondentID *int
	TagIDs          []int
	DocumentTypeID  *int
	CustomFields    []CustomFieldValue
}

// UpdateDocument PATCHes one document's metadata. Empty requests are
// rejected so callers never burn a round trip on a no-op.
func (c *Client) UpdateDocument(
	ctx context.Context,
	documentID int,
	req UpdateDocumentRequest,
) error {
	if req.Title == nil && req.Created == nil && req.CorrespondentID == nil && req.TagIDs == nil &&
		req.DocumentTypeID == nil && req.CustomFields == nil {
		return errorfamily.NewRejection("paperless.empty_update", "no metadata fields to update").
			WithContext("document_id", strconv.Itoa(documentID))
	}

	body, err := json.Marshal(buildDocumentUpdatePayload(req))
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "paperless.marshal_document_update", "could not encode document update").
			WithContext("document_id", strconv.Itoa(documentID))
	}

	if _, err := c.doRequest(
		ctx,
		http.MethodPatch,
		fmt.Sprintf(pathDocumentDetail, documentID),
		"",
		bytes.NewReader(body),
		"application/json",
	); err != nil {
		return fmt.Errorf("update document %d: %w", documentID, err)
	}

	return nil
}

// documentUpdatePayload mirrors the writable subset of Paperless-ngx's
// document serializer. Pointer fields are emitted only when set; an empty
// TagIDs slice is omitted so other-field updates never touch tags.
type documentUpdatePayload struct {
	Title         *string                   `json:"title,omitempty"`
	Created       *string                   `json:"created,omitempty"`
	Correspondent *int                      `json:"correspondent,omitempty"`
	TagIDs        []int                     `json:"tags,omitempty"`
	DocumentType  *int                      `json:"document_type,omitempty"`
	CustomFields  []customFieldValuePayload `json:"custom_fields,omitempty"`
}

// customFieldValuePayload is the wire shape of one document custom-field
// value (paperless-ngx 2.x+): the field definition ID plus the value.
type customFieldValuePayload struct {
	Field int    `json:"field"`
	Value string `json:"value"`
}

// customFieldsFromPayload converts the wire shape to the public type.
func customFieldsFromPayload(fields []customFieldValuePayload) []CustomFieldValue {
	if len(fields) == 0 {
		return nil
	}

	result := make([]CustomFieldValue, 0, len(fields))
	for _, field := range fields {
		result = append(result, CustomFieldValue(field))
	}

	return result
}

// buildDocumentUpdatePayload converts the caller-facing update request into
// its wire shape; only the fields the caller set are included.
func buildDocumentUpdatePayload(req UpdateDocumentRequest) documentUpdatePayload {
	payload := documentUpdatePayload{}

	if req.Title != nil {
		payload.Title = req.Title
	}

	if req.Created != nil {
		formatted := req.Created.Format(time.DateOnly)
		payload.Created = &formatted
	}

	if req.CorrespondentID != nil {
		id := *req.CorrespondentID
		payload.Correspondent = &id
	}

	if req.DocumentTypeID != nil {
		id := *req.DocumentTypeID
		payload.DocumentType = &id
	}

	if req.CustomFields != nil {
		payload.CustomFields = make([]customFieldValuePayload, 0, len(req.CustomFields))
		for _, field := range req.CustomFields {
			payload.CustomFields = append(payload.CustomFields, customFieldValuePayload(field))
		}
	}

	payload.TagIDs = req.TagIDs

	return payload
}

// DeleteDocument permanently removes one document from Paperless-ngx.
// Used by the backfill's --prune to drop duplicate server-side copies;
// callers own the destructive-action decision.
func (c *Client) DeleteDocument(ctx context.Context, documentID int) error {
	if _, err := c.doRequest(
		ctx,
		http.MethodDelete,
		fmt.Sprintf(pathDocumentDetail, documentID),
		"",
		nil,
		"",
	); err != nil {
		return fmt.Errorf("delete document %d: %w", documentID, err)
	}

	return nil
}

// DownloadDocument fetches one document's stored ORIGINAL file bytes
// (GET /api/documents/{id}/download/ — not the archive rendition). The
// decrypt-repair scans them for the /Encrypt trailer before replacing.
func (c *Client) DownloadDocument(ctx context.Context, documentID int) ([]byte, error) {
	raw, err := c.doRequest(
		ctx,
		http.MethodGet,
		fmt.Sprintf(pathDocumentDownload, documentID),
		"",
		nil,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("download document %d: %w", documentID, err)
	}

	return raw, nil
}

// doRequest performs one authenticated API round-trip and returns the
// response body. rawQuery is appended verbatim when non-empty; contentType
// is only required for requests with a body. See doRequestDetail for the
// header-exposing variant used by capability probing.
func (c *Client) doRequest(
	ctx context.Context,
	method string,
	path string,
	rawQuery string,
	body io.Reader,
	contentType string,
) ([]byte, error) {
	if c.retry == nil {
		data, _, err := c.doRequestDetail(ctx, method, path, rawQuery, body, contentType)

		return data, err
	}

	// Retries replay the body per attempt, so it must be re-readable:
	// snapshot it once up front (uploads already buffer fully in memory).
	var snapshot []byte

	if body != nil {
		var readErr error

		snapshot, readErr = io.ReadAll(body)
		if readErr != nil {
			return nil, errorfamily.WrapInfrastructure(readErr,
				"paperless.buffer_request_body",
				"could not buffer the request body for retries",
			).WithContext("path", path)
		}
	}

	data, err := retry.DoWithValue(
		ctx,
		c.retry.retryConfig(),
		func(_ context.Context, _ int) ([]byte, error) {
			var attemptBody io.Reader
			if body != nil {
				attemptBody = bytes.NewReader(snapshot)
			}

			attemptData, _, attemptErr := c.doRequestDetail(
				ctx,
				method,
				path,
				rawQuery,
				attemptBody,
				contentType,
			)

			return attemptData, attemptErr
		},
	)
	if err != nil {
		return data, fmt.Errorf("request failed after retries: %w", err)
	}

	return data, nil
}

// doRequestDetail is doRequest plus the response headers, so probes can
// read server-advertised metadata (API version negotiation echo, link
// headers, rate-limit hints).
func (c *Client) doRequestDetail(
	ctx context.Context,
	method string,
	path string,
	rawQuery string,
	body io.Reader,
	contentType string,
) ([]byte, http.Header, error) {
	endpoint := c.baseURL.JoinPath(path)
	endpoint.RawQuery = rawQuery

	var req *http.Request

	var err error

	req, err = http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(err, "paperless.build_request", "could not build request").
			WithContext("method", method).
			WithContext("path", path)
	}

	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("Accept", "application/json; version="+apiVersion)

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if c.requestHook != nil {
		c.requestHook(RequestInfo{
			Method: req.Method,
			URL:    req.URL.String(),
			Header: req.Header.Clone(),
		})
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, errorfamily.WrapTransient(err, "paperless.request_failed", "request to Paperless-ngx failed").
			WithContext("method", method).
			WithContext("path", path)
	}

	defer func() { _ = resp.Body.Close() }()

	var data []byte

	success := resp.StatusCode >= http.StatusOK && resp.StatusCode < maxSuccessStatusCode
	if success {
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.Header, errorfamily.WrapInfrastructure(err,
				"paperless.read_response", "could not read response body",
			).WithContext("path", path)
		}
	} else {
		data, _ = io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	}

	if c.responseHook != nil {
		c.responseHook(ResponseInfo{
			Status: resp.StatusCode,
			Header: resp.Header.Clone(),
			Body:   data,
		})
	}

	if !success {
		return nil, resp.Header, classifyStatus(resp, data, path)
	}

	return data, resp.Header, nil
}

// Capabilities reports what the connected Paperless-ngx server actually
// serves. The checksum shape is the load-bearing part: --verify, --prune,
// and --backfill all reconcile by checksum, and since paperless-ngx 3.x the
// flat checksum field is gone from the serializers (served via versions[]
// instead) — a server serving NEITHER shape makes every ledger row look
// drifted. Doctor surfaces this before it bites.
type Capabilities struct {
	// AcceptAPIVersion echoes the API version the server negotiated from
	// the Accept header (empty when the response omits it).
	AcceptAPIVersion string
	// FlatChecksum: sampled documents carry the legacy flat checksum field.
	FlatChecksum bool
	// VersionedChecksum: sampled documents carry checksums in versions[].
	VersionedChecksum bool
	// DocumentsSampled is how many documents the probe inspected.
	DocumentsSampled int
}

// ChecksumShape names the checksum delivery the server uses, for humans.
func (c Capabilities) ChecksumShape() string {
	switch {
	case c.FlatChecksum && c.VersionedChecksum:
		return "flat+versions[]"
	case c.FlatChecksum:
		return "flat (pre-3.x)"
	case c.VersionedChecksum:
		return "versions[] (3.x)"
	default:
		return "none"
	}
}

// ProbeCapabilities fetches one small documents page and inspects the
// checksum shapes the server serves. Read-only, one request, safe to run
// against any paperless-ngx version.
func (c *Client) ProbeCapabilities(ctx context.Context) (Capabilities, error) {
	// art-dupl:accept first-page query boilerplate shared with the Ping probe above
	query := url.Values{}
	query.Set("page", "1")
	query.Set("page_size", "5")

	raw, header, err := c.doRequestDetail(
		ctx, http.MethodGet, pathDocuments, query.Encode(), nil, "",
	)
	if err != nil {
		return Capabilities{}, fmt.Errorf("probe capabilities: %w", err)
	}

	caps := Capabilities{AcceptAPIVersion: negotiatedAPIVersion(header)}

	list := struct {
		Results []documentListEntry `json:"results"`
	}{}
	if err := json.Unmarshal(raw, &list); err != nil {
		return caps, errorfamily.WrapCorruption(err, "paperless.decode_documents",
			"could not decode document list while probing capabilities")
	}

	for _, entry := range list.Results {
		if entry.effectiveChecksum() == "" {
			continue
		}

		caps.DocumentsSampled++

		if entry.Checksum != "" {
			caps.FlatChecksum = true
		}

		if checksumFrom("", entry.Versions) != "" {
			caps.VersionedChecksum = true
		}
	}

	return caps, nil
}

// negotiatedAPIVersion extracts the API version the server echoed in its
// Content-Type (DRF Accept-header versioning echoes "version=N").
func negotiatedAPIVersion(header http.Header) string {
	for part := range strings.SplitSeq(header.Get("Content-Type"), ";") {
		if version, ok := strings.CutPrefix(strings.TrimSpace(part), "version="); ok {
			return version
		}
	}

	return ""
}

// RetryAfterError reports a rate-limit (or maintenance) response that
// carried a Retry-After header. It wraps the classified family error so
// retry policies can honor the server's hint instead of blind exponential
// backoff, while IsRetryable keeps working through Unwrap.
type RetryAfterError struct {
	Err error
	// After is the parsed hint. Seconds-only or HTTP-date forms are
	// accepted; a date in the past parses as 0 (retry immediately).
	After time.Duration
}

func (e *RetryAfterError) Error() string {
	return fmt.Sprintf("server requested retry after %s: %v", e.After, e.Err)
}

func (e *RetryAfterError) Unwrap() error { return e.Err }

// parseRetryAfter reads a Retry-After header value: either delay-seconds
// or an HTTP-date. Unparseable values return ok=false so callers fall
// back to their own backoff policy. Delay-seconds values too large to
// represent as a non-negative duration are treated as unparseable
// (multiplying them into a time.Duration would overflow and wrap
// negative).
func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 || int64(seconds) > math.MaxInt64/int64(time.Second) {
			return 0, false
		}

		return time.Duration(seconds) * time.Second, true
	}

	if deadline, err := http.ParseTime(value); err == nil {
		delay := deadline.Sub(now)
		if delay < 0 {
			return 0, true
		}

		return delay, true
	}

	return 0, false
}

// classifyStatus converts a non-2xx response into an error-family error,
// wrapping a Retry-After hint (429/503) in a RetryAfterError when present.
// snippet is the (already capped) error body the caller read.
func classifyStatus(resp *http.Response, snippet []byte, path string) error {
	statusCode := resp.StatusCode

	wrapped := errorfamily.NewTransient("paperless.server_error", "Paperless-ngx returned a retryable error").
		WithContext("status", strconv.Itoa(statusCode)).
		WithContext("path", path).
		WithContext("body", string(snippet))

	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		wrapped = errorfamily.NewRejection("paperless.auth_failed", "Paperless-ngx rejected the API token").
			WithContext("status", strconv.Itoa(statusCode)).
			WithContext("path", path)
	case statusCode == http.StatusTooManyRequests:
		wrapped = errorfamily.NewTransient("paperless.rate_limited", "Paperless-ngx rate limit hit, retry later").
			WithContext("path", path)
	case statusCode >= 400 && statusCode < 500:
		wrapped = errorfamily.NewRejection("paperless.client_error", "Paperless-ngx rejected the request").
			WithContext("status", strconv.Itoa(statusCode)).
			WithContext("path", path).
			WithContext("body", string(snippet))
	}

	if statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable {
		if after, ok := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); ok {
			return &RetryAfterError{Err: wrapped, After: after}
		}
	}

	return wrapped
}

// DocumentNoteUser is the (possibly anonymous) author of a document note.
type DocumentNoteUser struct {
	ID        int
	Username  string
	FirstName string
	LastName  string
}

// DocumentNote is one operator comment attached to a stored document.
type DocumentNote struct {
	ID      int
	Note    string
	Created time.Time
	// User is the note's author, or nil when the server cannot resolve it
	// (e.g. the author account was deleted).
	User *DocumentNoteUser
}

// documentNoteUserPayload mirrors the BasicUserSerializer the notes
// endpoint embeds.
type documentNoteUserPayload struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// documentNotePayload mirrors the Note serializer fields the SDK relies on.
type documentNotePayload struct {
	ID      int                      `json:"id"`
	Note    string                   `json:"note"`
	Created time.Time                `json:"created"`
	User    *documentNoteUserPayload `json:"user"`
}

// decodeDocumentNotes maps the server's note array (the notes endpoint is
// not paginated and always answers with a bare JSON array).
func decodeDocumentNotes(raw []byte) ([]DocumentNote, error) {
	var payloads []documentNotePayload
	if err := json.Unmarshal(raw, &payloads); err != nil {
		return nil, errorfamily.WrapCorruption(err, "paperless.decode_notes",
			"could not decode document notes")
	}

	notes := make([]DocumentNote, 0, len(payloads))
	for _, payload := range payloads {
		note := DocumentNote{
			ID:      payload.ID,
			Note:    payload.Note,
			Created: payload.Created,
		}

		if payload.User != nil {
			note.User = &DocumentNoteUser{
				ID:        payload.User.ID,
				Username:  payload.User.Username,
				FirstName: payload.User.FirstName,
				LastName:  payload.User.LastName,
			}
		}

		notes = append(notes, note)
	}

	return notes, nil
}

// ListDocumentNotes returns the notes attached to one document, newest
// first (the server's ordering). The notes endpoint is not paginated —
// the response is a bare JSON array.
func (c *Client) ListDocumentNotes(ctx context.Context, documentID int) ([]DocumentNote, error) {
	raw, err := c.doRequest(
		ctx,
		http.MethodGet,
		fmt.Sprintf(pathDocumentNotes, documentID),
		"",
		nil,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list document notes (document %d): %w",
			documentID,
			err,
		) //nolint:erraudit // uncoded context wrap keeps the inner coded error's code+family
	}

	return decodeDocumentNotes(raw)
}

// AddDocumentNote appends one note to a document and returns the server's
// updated note list (the notes endpoint answers every mutation with the
// full remaining list, newest first).
func (c *Client) AddDocumentNote(
	ctx context.Context,
	documentID int,
	note string,
) ([]DocumentNote, error) {
	if documentID <= 0 {
		return nil, errorfamily.NewRejection("paperless.empty_document_id",
			"document ID is required to add a note")
	}

	if note == "" {
		return nil, errorfamily.NewRejection("paperless.empty_note",
			"note text is required")
	}

	payload, err := json.Marshal(struct {
		Note string `json:"note"`
	}{Note: note})
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "paperless.marshal_note",
			"could not encode the note payload")
	}

	raw, err := c.doRequest(
		ctx,
		http.MethodPost,
		fmt.Sprintf(pathDocumentNotes, documentID),
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return nil, fmt.Errorf("add document note (document %d): %w", documentID, err)
	}

	return decodeDocumentNotes(raw)
}

// DeleteDocumentNote removes one note from a document and returns the
// server's updated note list. The note is addressed by the query
// parameter the endpoint expects (DELETE .../notes/?id=<noteID>).
func (c *Client) DeleteDocumentNote(
	ctx context.Context,
	documentID, noteID int,
) ([]DocumentNote, error) {
	if documentID <= 0 {
		return nil, errorfamily.NewRejection("paperless.empty_document_id",
			"document ID is required to delete a note")
	}

	if noteID <= 0 {
		return nil, errorfamily.NewRejection("paperless.empty_note_id",
			"note ID is required to delete a note")
	}

	query := url.Values{}
	query.Set("id", strconv.Itoa(noteID))

	raw, err := c.doRequest(
		ctx,
		http.MethodDelete,
		fmt.Sprintf(pathDocumentNotes, documentID),
		query.Encode(),
		nil,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("delete document note (document %d, note %d): %w",
			documentID, noteID, err)
	}

	return decodeDocumentNotes(raw)
}

// ShareLinkFileVersion names which rendition of a document a share link
// exposes. The server's default (and the zero-value fallback it applies
// when the field is omitted) is the archive version.
type ShareLinkFileVersion string

const (
	// ShareLinkFileVersionArchive exposes the OCR'd archive rendition.
	ShareLinkFileVersionArchive ShareLinkFileVersion = "archive"
	// ShareLinkFileVersionOriginal exposes the untouched original file.
	ShareLinkFileVersionOriginal ShareLinkFileVersion = "original"
)

// ShareLink is one public download link for a document. Slug is
// server-generated; the full share URL is <base>/share/<slug>.
type ShareLink struct {
	ID      int
	Created time.Time
	// Expiration is when the link stops working; zero means it never
	// expires.
	Expiration time.Time
	Slug       string
	DocumentID int
	// FileVersion says which rendition the link serves (archive or
	// original).
	FileVersion ShareLinkFileVersion
}

// shareLinkPayload mirrors the ShareLink serializer fields the SDK relies
// on (id, created, expiration, slug, document, file_version).
type shareLinkPayload struct {
	ID          int        `json:"id"`
	Created     time.Time  `json:"created"`
	Expiration  *time.Time `json:"expiration"`
	Slug        string     `json:"slug"`
	Document    int        `json:"document"`
	FileVersion string     `json:"file_version"`
}

func (p shareLinkPayload) shareLink() ShareLink {
	link := ShareLink{
		ID:          p.ID,
		Created:     p.Created,
		Slug:        p.Slug,
		DocumentID:  p.Document,
		FileVersion: ShareLinkFileVersion(p.FileVersion),
	}

	if p.Expiration != nil {
		link.Expiration = *p.Expiration
	}

	return link
}

// CreateShareLinkRequest asks for one share link. A zero FileVersion is
// omitted from the POST so the server applies its default (archive); a
// nil Expiration means the link never expires.
type CreateShareLinkRequest struct {
	DocumentID  int
	FileVersion ShareLinkFileVersion
	Expiration  *time.Time
}

// ListShareLinks returns every share link on the server, paginating the
// same bounded way as the document listings.
func (c *Client) ListShareLinks(ctx context.Context) ([]ShareLink, error) {
	payloads, listErr := fetchAllPages[shareLinkPayload](ctx, c, pathShareLinks, "share link")
	if listErr != nil {
		return nil, listErr
	}

	links := make([]ShareLink, 0, len(payloads))

	for _, payload := range payloads {
		links = append(links, payload.shareLink())
	}

	return links, nil
}

// CreateShareLink makes one public download link for a document. The
// server generates the slug — the POST deliberately sends only the
// document, the optional file version, and the optional expiration.
func (c *Client) CreateShareLink(
	ctx context.Context,
	req CreateShareLinkRequest,
) (ShareLink, error) {
	if req.DocumentID <= 0 {
		return ShareLink{}, errorfamily.NewRejection("paperless.empty_document_id",
			"document ID is required to create a share link")
	}

	payload, err := json.Marshal(struct {
		Document    int                  `json:"document"`
		FileVersion ShareLinkFileVersion `json:"file_version,omitempty"`
		Expiration  *time.Time           `json:"expiration,omitempty"`
	}{
		Document:    req.DocumentID,
		FileVersion: req.FileVersion,
		Expiration:  req.Expiration,
	})
	if err != nil {
		return ShareLink{}, errorfamily.WrapInfrastructure(err, "paperless.marshal_share_link",
			"could not encode the share link payload").
			WithContext("document", strconv.Itoa(req.DocumentID))
	}

	raw, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathShareLinks,
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return ShareLink{}, fmt.Errorf("create share link (document %d): %w",
			req.DocumentID, err)
	}

	var created shareLinkPayload
	if err := json.Unmarshal(raw, &created); err != nil {
		return ShareLink{}, errorfamily.WrapCorruption(err, "paperless.decode_share_link",
			"could not decode the created share link").
			WithContext("document", strconv.Itoa(req.DocumentID))
	}

	return created.shareLink(), nil
}

// DeleteShareLink revokes one public link; every consumer of the slug
// loses access immediately.
func (c *Client) DeleteShareLink(ctx context.Context, linkID int) error {
	if linkID <= 0 {
		return errorfamily.NewRejection("paperless.empty_share_link_id",
			"share link ID is required to delete a share link")
	}

	if _, err := c.doRequest(
		ctx,
		http.MethodDelete,
		pathShareLinks+strconv.Itoa(linkID)+"/",
		"",
		nil,
		"",
	); err != nil {
		return fmt.Errorf("delete share link %d: %w", linkID, err)
	}

	return nil
}

// SavedViewFilterRule is one filter criterion of a saved view: the
// server's numeric rule type (e.g. 6 = "has tag ...") and its string
// value.
type SavedViewFilterRule struct {
	RuleType int
	Value    string
}

// savedViewFilterRulePayload mirrors the SavedViewFilterRule serializer.
type savedViewFilterRulePayload struct {
	RuleType int    `json:"rule_type"`
	Value    string `json:"value"`
}

// SavedView is one stored filter view ("Inbox", ...) in the web UI. The
// SDK models the stable core fields; servers add or drop UI fields
// (icon, page_size, display mode, ...) across versions and unknown JSON
// fields are ignored.
type SavedView struct {
	ID   int
	Name string
	// ShowOnDashboard and ShowInSidebar mirror the view's visibility
	// flags; a server that no longer serves them decodes as false.
	ShowOnDashboard bool
	ShowInSidebar   bool
	SortField       string
	SortReverse     bool
	FilterRules     []SavedViewFilterRule
}

// savedViewPayload mirrors the SavedView serializer's stable fields.
type savedViewPayload struct {
	ID              int                          `json:"id"`
	Name            string                       `json:"name"`
	ShowOnDashboard bool                         `json:"show_on_dashboard"`
	ShowInSidebar   bool                         `json:"show_in_sidebar"`
	SortField       string                       `json:"sort_field"`
	SortReverse     bool                         `json:"sort_reverse"`
	FilterRules     []savedViewFilterRulePayload `json:"filter_rules"`
}

func (p savedViewPayload) savedView() SavedView {
	view := SavedView{
		ID:              p.ID,
		Name:            p.Name,
		ShowOnDashboard: p.ShowOnDashboard,
		ShowInSidebar:   p.ShowInSidebar,
		SortField:       p.SortField,
		SortReverse:     p.SortReverse,
		FilterRules:     make([]SavedViewFilterRule, 0, len(p.FilterRules)),
	}

	for _, rule := range p.FilterRules {
		view.FilterRules = append(view.FilterRules, SavedViewFilterRule(rule))
	}

	return view
}

// CreateSavedViewRequest asks for one saved view. Name is required; the
// filter rules carry the server's numeric rule types.
type CreateSavedViewRequest struct {
	Name            string
	ShowOnDashboard bool
	ShowInSidebar   bool
	SortField       string
	SortReverse     bool
	FilterRules     []SavedViewFilterRule
}

// ListSavedViews returns every saved view on the server, paginating the
// same bounded way as the document listings.
func (c *Client) ListSavedViews(ctx context.Context) ([]SavedView, error) {
	payloads, listErr := fetchAllPages[savedViewPayload](ctx, c, pathSavedViews, "saved view")
	if listErr != nil {
		return nil, listErr
	}

	views := make([]SavedView, 0, len(payloads))

	for _, payload := range payloads {
		views = append(views, payload.savedView())
	}

	return views, nil
}

// CreateSavedView stores one saved view and returns its ID.
func (c *Client) CreateSavedView(
	ctx context.Context,
	req CreateSavedViewRequest,
) (int, error) {
	if req.Name == "" {
		return 0, errorfamily.NewRejection("paperless.empty_saved_view",
			"saved view name is required")
	}

	rules := make([]savedViewFilterRulePayload, 0, len(req.FilterRules))
	for _, rule := range req.FilterRules {
		rules = append(rules, savedViewFilterRulePayload(rule))
	}

	payload, err := json.Marshal(struct {
		Name            string                       `json:"name"`
		ShowOnDashboard bool                         `json:"show_on_dashboard"`
		ShowInSidebar   bool                         `json:"show_in_sidebar"`
		SortField       string                       `json:"sort_field"`
		SortReverse     bool                         `json:"sort_reverse"`
		FilterRules     []savedViewFilterRulePayload `json:"filter_rules"`
	}{
		Name:            req.Name,
		ShowOnDashboard: req.ShowOnDashboard,
		ShowInSidebar:   req.ShowInSidebar,
		SortField:       req.SortField,
		SortReverse:     req.SortReverse,
		FilterRules:     rules,
	})
	if err != nil {
		return 0, errorfamily.WrapInfrastructure(err, "paperless.marshal_saved_view",
			"could not encode the saved view payload").
			WithContext("name", req.Name)
	}

	raw, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathSavedViews,
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return 0, fmt.Errorf("create saved view %q: %w", req.Name, err)
	}

	var created savedViewPayload
	if err := json.Unmarshal(raw, &created); err != nil {
		return 0, errorfamily.WrapCorruption(err, "paperless.decode_saved_view",
			"could not decode the created saved view").
			WithContext("name", req.Name)
	}

	return created.ID, nil
}

// DeleteSavedView removes one saved view from the server.
func (c *Client) DeleteSavedView(ctx context.Context, viewID int) error {
	if viewID <= 0 {
		return errorfamily.NewRejection("paperless.empty_saved_view_id",
			"saved view ID is required to delete a saved view")
	}

	if _, err := c.doRequest(
		ctx,
		http.MethodDelete,
		pathSavedViews+strconv.Itoa(viewID)+"/",
		"",
		nil,
		"",
	); err != nil {
		return fmt.Errorf("delete saved view %d: %w", viewID, err)
	}

	return nil
}
