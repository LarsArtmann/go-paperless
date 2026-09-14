# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.

## Themes

### 1. Consumption peace of mind — SHIPPED 2026-09-13

Uploading is only half the story: Paperless-ngx consumes asynchronously and
refuses duplicates. All three raw ideas below shipped in the v0.2.0 batch
(see FEATURES.md "Async consumption tracking" and "Error handling and
retries", CHANGELOG [Unreleased]):

- ~~A polling helper that blocks until `TaskStatus.Terminal()` with a
  deadline and context cancellation~~ → `WaitForTask` + `DefaultTaskPollInterval`
  (`client.go:580`)
- ~~A client-side retry policy that honors `RetryAfterError.After`~~ →
  opt-in `WithRetry(RetryPolicy)` backed by
  [go-retry](https://github.com/larsartmann/go-retry), fail-fast default
  preserved (`client.go:184`)
- ~~First-class "was this a duplicate refusal?" ergonomics~~ →
  `TaskOutcome.Duplicate()` (`client.go:465`)

### 2. Growing with the API

The client targets API v10 and the endpoints its two consumers need
(InboxClean, bank-sync). Coverage expands only when a real consumer needs it.

Raw ideas:

- ~~Notes, share links, and saved-view endpoints~~ → shipped (built on
  the user's go: `ListDocumentNotes`/`AddDocumentNote`/`DeleteDocumentNote`,
  `CreateShareLink`/`ListShareLinks`/`DeleteShareLink`,
  `ListSavedViews`/`CreateSavedView`/`DeleteSavedView`; `client.go:1944`)
- ~~Storage-path management alongside tags/correspondents/document types~~ →
  shipped: `FindStoragePath` / `EnsureStoragePath` / `ListStoragePaths`
  (`client.go:971`)
- ~~Hook points for request/response logging so operators can trace sync
  runs~~ → shipped: `WithRequestHook` / `WithResponseHook` (`client.go:194`)
- Streaming multipart upload for sources larger than the memory-safe
  envelope the current in-memory buffering assumes (`client.go:372`)

### 3. Confidence

Version tolerance is probed (`ProbeCapabilities`), and the tolerant parsers
now have property-based scrutiny — but the suite still only speaks to
synthetic httptest servers.

Raw ideas:

- An integration test tier against a real paperless-ngx (containers)
  exercising upload → poll → reconcile end to end
- ~~Fuzz or property tests for the checksum/status/date parsers~~ → shipped:
  `FuzzParseRetryAfter`, `FuzzParseDocumentCreated`, `FuzzChecksumFrom`,
  `FuzzClassifyTask` (`fuzz_test.go`; the retry-after fuzzer found and fixed
  a real overflow)

## Testing infrastructure

- Real-server integration tests exist behind the `integration` build tag
  (`integration_test.go`, env-driven). Growing that suite is the cheapest
  path to confidence in behaviors httptest cannot fake (server-side
  pagination quirks, negotiated versions, task timing).

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
