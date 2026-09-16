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
  (`client.go`)
- ~~A client-side retry policy that honors `RetryAfterError.After`~~ →
  opt-in `WithRetry(RetryPolicy)` backed by
  [go-retry](https://github.com/larsartmann/go-retry), fail-fast default
  preserved (`WithRetry`)
- ~~First-class "was this a duplicate refusal?" ergonomics~~ →
  `TaskOutcome.Duplicate()`

### 2. Growing with the API

The client targets API v10 and the endpoints its two consumers need
(InboxClean, bank-sync). Coverage expands only when a real consumer needs it.

Raw ideas:

- ~~Notes, share links, and saved-view endpoints~~ → shipped (built on
  the user's go: `ListDocumentNotes`/`AddDocumentNote`/`DeleteDocumentNote`,
  `CreateShareLink`/`ListShareLinks`/`DeleteShareLink`,
  `ListSavedViews`/`CreateSavedView`/`DeleteSavedView`)
- ~~Storage-path management alongside tags/correspondents/document types~~ →
  shipped: `FindStoragePath` / `EnsureStoragePath` / `ListStoragePaths`
- ~~Hook points for request/response logging so operators can trace sync
  runs~~ → shipped: `WithRequestHook` / `WithResponseHook`
- Streaming multipart upload for sources larger than the memory-safe
  envelope the current in-memory buffering assumes (`Upload`)
- Share-link conveniences, demand-gated: a full-URL helper
  (`<base>/share/<slug>`) and a document-scoped listing
  (`GET /api/documents/{id}/share_links/`)
- Saved-view Update/Retrieve methods (Create/List/Delete ship today) if a
  consumer needs editing
- Capability-probe caching — `ProbeCapabilities` performs a full documents
  request per call today; probe once per client lifetime if consumers feel it
- Typed public fields (phantom/branded types for `Slug`, `Checksum`,
  `SortField`, ... — the branching-flow analyzer's PHANTOM_TYPE suggestions).
  Breaking API change: bank-sync consumes this as a drop-in replacement, so a
  typed-field migration is v2 scope at the earliest. branching-flow is
  skipped in `.buildflow.yml` until then (rationale there).

### 3. Confidence

Version tolerance is probed (`ProbeCapabilities`), and the tolerant parsers
now have property-based scrutiny. A real-server tier exists in scaffold form
(`integration_test.go`); what httptest still cannot prove is server-side
timing and reconciliation against live data.

Raw ideas:

- ~~An integration test tier against a real paperless-ngx (containers)
  exercising upload → poll → reconcile end to end~~ → scaffold shipped:
  `integration_test.go` behind the `integration` build tag (ping + the five
  listings); the upload → poll → checksum-reconcile loop itself is written
  (`TestIntegrationUploadReconcile`, compiled by `checks.integration-vet`)
  and needs a live server to run (`nix run .#integration`)
- ~~Fuzz or property tests for the checksum/status/date parsers~~ → shipped:
  `FuzzParseRetryAfter`, `FuzzParseDocumentCreated`, `FuzzChecksumFrom`,
  `FuzzClassifyTask` (`fuzz_test.go`; the retry-after fuzzer found and fixed
  a real overflow)

### 4. Tooling & workflow (raw)

Fleet-level workflow and automation ideas surfaced by the 2026-09-16
dependency-upgrade session
(`docs/status/2026-09-16_17-35_dependency-upgrade-status.md`):

- Parallel-session protocol: two agents + the auto-commit daemon on one
  worktree tangled commit attribution and raced builds; consider
  per-session worktrees or an explicit repo-busy handoff convention
- flake.lock ownership: who runs `nix flake update`, on what cadence, and
  what guards drift (the daemon updated it unprompted on 2026-09-16)
- Dependabot vs manual dependency sweeps: keep the weekly gomod PRs AND
  coordinated manual bumps, or restrict dependabot (open user decision)
- Weekly dep-freshness guard (`go list -m -u all` report) complementing
  dependabot, if the policy question above lands on "keep both"
- CHANGELOG convention: a dedicated "Dependencies" subsection instead of
  burying dep bumps under Changed

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

## Decision records

Architecture decisions that constrain the ideas above live in
[docs/adr/](docs/adr/README.md) — notably ADR 0003, which records that
streaming upload is explicitly NOT promised.
