package paperless

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-retry"
)

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
		return fmt.Errorf(
			"ping %s: %w",
			c.baseURL,
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	return nil
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

		return data, err //nolint:erraudit // passthrough: inner error already carries method+path
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
		return data, fmt.Errorf(
			"request failed after retries: %w",
			err,
		) //nolint:erraudit // keeps inner code+family
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

	defer func() { _ = resp.Body.Close() }() //nolint:erraudit // cleanup only: body fully read or error already returned

	var data []byte

	var bodyReadErr error

	success := resp.StatusCode >= http.StatusOK && resp.StatusCode < maxSuccessStatusCode
	if success {
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.Header, errorfamily.WrapInfrastructure(err,
				"paperless.read_response", "could not read response body",
			).WithContext("path", path)
		}
	} else {
		// Best-effort diagnostics: keep whatever body arrived even when the
		// read fails midway (a partial snippet still narrows the failure);
		// classifyStatus attaches the read failure so a missing or truncated
		// snippet is explained rather than silent.
		data, bodyReadErr = io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	}

	if c.responseHook != nil {
		c.responseHook(ResponseInfo{
			Status: resp.StatusCode,
			Header: resp.Header.Clone(),
			Body:   data,
		})
	}

	if !success {
		return nil, resp.Header, classifyStatus(resp, data, path, bodyReadErr)
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
		return Capabilities{}, fmt.Errorf(
			"probe capabilities: %w",
			err,
		) //nolint:erraudit // keeps inner code+family
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
// 3xx responses (which only arrive when the redirect could not be followed)
// classify as paperless.redirect/Rejection instead of retrying into the same
// redirect again. snippet is the (already capped) error body the caller read;
// bodyReadErr, when non-nil, says that read failed and the snippet may be
// truncated or empty, and is attached as context so the gap is explained, not
// silent.
func classifyStatus(resp *http.Response, snippet []byte, path string, bodyReadErr error) error {
	statusCode := resp.StatusCode

	wrapped := errorfamily.NewTransient("paperless.server_error", "Paperless-ngx returned a retryable error").
		WithContext("status", strconv.Itoa(statusCode)).
		WithContext("path", path).
		WithContext("body", string(snippet))

	switch {
	case statusCode >= 300 && statusCode < 400:
		// Reached only for redirects the http.Client could not follow (e.g.
		// a 3xx without a Location header is returned as-is). A followable
		// redirect is transparently followed and never reaches this switch.
		wrapped = errorfamily.NewRejection("paperless.redirect",
			"Paperless-ngx returned a redirect the client could not follow").
			WithContext("status", strconv.Itoa(statusCode)).
			WithContext("path", path).
			WithContext("location", resp.Header.Get("Location"))
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

	if bodyReadErr != nil {
		wrapped = wrapped.WithContext("body_read_error", bodyReadErr.Error())
	}

	if statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable {
		if after, ok := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); ok {
			return &RetryAfterError{Err: wrapped, After: after}
		}
	}

	return wrapped
}
