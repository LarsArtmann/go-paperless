# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.

## Themes

### 1. Consumption peace of mind

Uploading is only half the story: Paperless-ngx consumes asynchronously and
refuses duplicates. The SDK hands callers the raw pieces (`GetTask`,
`TaskOutcome`, `RetryAfterError`) and every consumer re-implements the wait
loop and backoff on top.

Raw ideas:

- A polling helper that blocks until `TaskStatus.Terminal()` with a deadline
  and context cancellation
- A client-side retry policy that honors `RetryAfterError.After` instead of
  blind exponential backoff in each consumer
- First-class "was this a duplicate refusal?" ergonomics layered on
  `TaskOutcome`

### 2. Growing with the API

The client targets API v10 and the endpoints its two consumers need
(InboxClean, bank-sync). Coverage expands only when a real consumer needs it.

Raw ideas:

- Notes, share links, and saved-view endpoints
- Storage-path management alongside tags/correspondents/document types
- Hook points for request/response logging so operators can trace sync runs
- Streaming multipart upload for sources larger than the memory-safe
  envelope the current in-memory buffering assumes (`client.go:195`)

### 3. Confidence

Version tolerance is probed (`ProbeCapabilities`), but the suite only ever
speaks to synthetic httptest servers, and the tolerant parsers
(`checksumFrom`, `parseRetryAfter`, `parseDocumentCreated`) are exactly the
kind of hand-rolled fallback logic that benefits from property-based
scrutiny.

Raw ideas:

- An integration test tier against a real paperless-ngx (containers)
  exercising upload → poll → reconcile end to end
- Fuzz or property tests for the checksum/status/date parsers

## Non-goals

- **Pipeline abstractions** (ledgers, projections, sync orchestration):
  domain-coupled; they belong to the consumers. This repo is a client,
  deliberately.
- **Multi-API-version emulation**: version differences are handled by
  capability probing and tolerant decoding, not by emulating every
  historical API version.
- **Auto-generating the client from an OpenAPI spec**: paperless-ngx serves
  no complete spec, and the handwritten client carries domain decisions
  (matching-algorithm policy, duplicate-refusal classification) a generator
  would lose.
