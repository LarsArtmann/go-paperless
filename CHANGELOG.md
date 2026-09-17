# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- Nothing yet.

### Changed

- Nothing yet.

### Fixed

- Nothing yet.

## [0.3.2] - 2026-09-17

### Fixed

- `updateMatchingAlgorithm` (the legacy auto-tag self-heal behind
  `EnsureTag`) sent `{"id":0,"name":"","matching_algorithm":0}` —
  Paperless-ngx's DRF serializer rejects a PATCH that re-sends a blank
  required name with 400 Bad Request, so the demote never landed and every
  papersync tick retried it forever (live symptom: `Bad Request:
  /api/tags/1/` once per sync). The PATCH now carries only
  `matching_algorithm`, and the self-heal test pins that `name` stays out
  of the request body.

## [0.3.1] - 2026-09-14

A hardening patch: no public API changes, no dependency changes — the module
content consumers build against is identical to v0.3.0. What ships is
confidence: race-detector coverage in CI, a real-server test scaffold, the
error-code catalog, and supply-chain/tooling cleanup.

### Added

- Real-server integration tests behind the `integration` build tag
  (`integration_test.go`), env-driven via `PAPERLESS_INTEGRATION_URL` /
  `PAPERLESS_INTEGRATION_TOKEN`
- Error-code catalog: [docs/ERROR_CODES.md](docs/ERROR_CODES.md) documents
  every `paperless.*` code by family and retryability
- `BenchmarkUpload` (multipart cost visibility); a request-hook and
  response-hook assertion test for `ProbeCapabilities`; a plain-request
  context-cancellation test
- CI runs the race detector hermetically (`checks.test-race` via
  `nix flake check`); PR template and maintainer release checklist added

### Changed

- Matching-algorithm constants are now a distinct unexported type
  (`matchingAlgorithm`), so the tag self-heal path cannot PATCH an
  arbitrary int
- Test suite harmonized on `t.Context()`; ADR 0001 records the
  API-version policy (pin one, tolerate decoding, never emulate)
- The flake provides `packages.default` / `packages.go-paperless`, so bare
  `nix build` works, and its version attribute tracks the release tag
  (previously stuck at 0.2.0 after the v0.3.0 tag)

### Fixed

- `.golangci.yml` now analyzes the Go 1.27 language version (`run.go` was
  pinned to the host's older 1.26.7 — a drift trap against the module's
  `go 1.27` floor)

## [0.3.0] - 2026-09-14

### Added

- Document notes: `ListDocumentNotes`, `AddDocumentNote`, `DeleteDocumentNote`
  (`DocumentNote`, `DocumentNoteUser`) — the notes endpoint is a bare JSON
  array; every mutation answers with the full updated list, newest first
- Share links: `ListShareLinks`, `CreateShareLink`, `DeleteShareLink`
  (`ShareLink`, `ShareLinkFileVersion`) — the slug is server-generated, the
  create POST sends only document + optional file version + optional
  expiration (default rendition: archive); expiration zero = never expires
- Saved views: `ListSavedViews`, `CreateSavedView`, `DeleteSavedView`
  (`SavedView`, `SavedViewFilterRule`) — models the stable serializer core
  (name, visibility flags, sort, filter rules); unknown UI fields are ignored
- Tests: `TestWithTimeoutBoundsSlowResponses`,
  `TestWithHTTPClientRoutesRequestsThroughSuppliedClient`, and
  `TestNegotiatedAPIVersionReadsContentType` close the Options/transport
  test gaps; `ListDocumentMetas` now asserts the custom-fields round-trip.
  Coverage 88.4% → 90.1%

### Changed

- `go.mod` floors `go 1.27` (major.minor) instead of the `1.27.1` patch
  floor, so environments trailing the newest 1.27 patch release build again
- The three paginated listings (`ListDocumentChecksums`,
  `ListDocumentMetas`, `ListStoragePaths`) share one generic
  `fetchAllPages` helper; pagination policy lives in a single place
- `FindCustomField` sends `page_size=1` like every other exact-name lookup
- CI pins `govulncheck@v1.8.0` (was `@latest`); dev shells set
  `GOTOOLCHAIN=local` so the flake's Go toolchain is authoritative

### Fixed

- Removed a stale duplicated doc comment on `doRequest`; `WithHTTPClient`'s
  doc no longer claims `WithTimeout` has no effect (the last option wins)
- Error classification no longer reads the response body twice (the capped
  snippet is passed to `classifyStatus` directly)
- `ListShareLinks` and `ListSavedViews` reuse the same shared pagination
  helper as the document listings (one bounded-walk policy everywhere);
  each listing gained a cap-parity test proving the 100-page stop
- CI hardening: workflow actions pinned to commit SHAs, a grouped weekly
  github-actions Dependabot source, and a tightened golangci-lint set
  (37 findings cleared, including a wrong-error classification in a
  storage-path rejection test)

## [0.2.0] - 2026-09-13

### Added

- `WaitForTask` (plus `DefaultTaskPollInterval`): polls one consumption task
  to a terminal state — immediate first poll, not-found and transient-error
  tolerance, duplicate refusals are honest outcomes (not errors), a context
  deadline bounds the wait and surfaces the last poll error
- `TaskOutcome.Duplicate()`: accessor for duplicate-refusal details
  (document ID, in-trash flag, refused flag)
- Opt-in automatic retries: `RetryPolicy` + `WithRetry` (backed by
  go-retry v0.5.0). Transient failures only (network, 429/503, 5xx);
  rejections fail fast; request bodies replay byte-for-byte per attempt;
  server `Retry-After` hints override exponential backoff; a negative
  `MaxAttempts` is rejected via `ErrInvalidConfig`. Default behavior is
  unchanged (single attempt, fail fast)
- Storage paths: `FindStoragePath`, `EnsureStoragePath` (an existing path
  keeps its configured directory template; the create POST sends only
  `name` and `path` — the server generates the slug), `ListStoragePaths`
  (bounded pagination like the document listings)
- Observability hooks: `WithRequestHook` / `WithResponseHook` deliver
  `RequestInfo` / `ResponseInfo` value snapshots with cloned headers (they
  include the Authorization token / Set-Cookie — redact before logging);
  non-2xx bodies are capped at 512 bytes
- Tests: the five new APIs are exercised end-to-end over httptest (25 new
  tests), four fuzz targets (`FuzzParseRetryAfter`,
  `FuzzParseDocumentCreated`, `FuzzChecksumFrom`, `FuzzClassifyTask`) with
  crash corpus seeds, a `maxDocumentListPages` page-cap test and a
  concurrent-caller race test; coverage 86.1% → 88.4%

### Fixed

- `parseRetryAfter` now treats delay-seconds values too large for
  `time.Duration` as unparseable — they previously overflowed into a
  negative delay (found by fuzzing)
- `EnsureStoragePath` no longer sends an empty `slug` field on create

## [0.1.1] - 2026-09-13

### Changed

- Go floor raised to 1.27.1: `encoding/json/v2` is the default toolchain
  there, so consumers no longer need `GOEXPERIMENT=jsonv2`; consumer
  toolchains must be Go 1.27+ to build against this version
- Added `.golangci.yml`: `tagliatelle` now actively enforces snake_case
  JSON tags (the Paperless-ngx wire convention) and `nolintlint` requires
  explained, used directives; the 11 now-redundant `//nolint:tagliatelle`
  suppressions were removed
- Hygiene: deleted ghost `dprint.json` (treefmt owns formatting, nothing
  invoked dprint) and the dead `!go.work` gitignore override that the
  buildflow block re-ignored anyway
- `nix flake check` now passes fully hermetic: `checks.build`, a new
  `checks.test` (full suite in the sandbox) and `checks.lint` use
  `buildGoModule` (modules fetched via Nix into a fixed-output derivation,
  materialised as `vendor/`) instead of downloading `go-error-family` at
  build time; the sandbox no longer needs DNS
- Removed `checks.build-standalone` (single-module repo, no `go.work`;
  it duplicated `checks.build`)
- All flake apps export `GOEXPERIMENT` themselves and carry
  `meta.description` (visible in `nix flake show`); new `nix run .#check`
  app runs `nix flake check` as the one-command CI equivalent
- GitHub Actions CI: `nix flake check` plus `govulncheck` and `gosec` on
  every push and pull request
- Package godoc now states the Go requirement, the `GetTask` bare-array
  tolerance, and the UTC reading of timezone-less dates; new
  `ExampleNew_invalidConfig` shows `errors.Is` against `ErrInvalidConfig`
- Governance: `SECURITY.md` (private reporting path, threat-model notes);
  CONTRIBUTING points at `TODO_LIST.md` for open work

### Fixed

- Test coverage gaps: `DownloadDocument` (happy path + 404 error wrap) and
  the 503 `Retry-After` branch now have dedicated tests — every public
  method is exercised by the suite
- `nix run .#fmt` (and `nix flake check` evaluation): the fmt app passed an
  attrset of treefmt programs where a list of packages was required

## [0.1.0] - 2026-09-13

### Added

- Paperless-ngx REST client (API v10, token auth): `New` with
  `WithHTTPClient`/`WithTimeout` options
- Document upload (`Upload`, `UploadRequest`, `CustomFieldValue`) with async
  consumption-task polling (`GetTask`, `TaskOutcome`, `TaskStatus`)
- Idempotent lookups: `EnsureTag`, `EnsureCorrespondent`, `EnsureDocumentType`,
  `EnsureCustomField`, `FindCustomField`
- Name resolution: `GetCorrespondentName`, `GetDocumentTypeName`
- Document management: `ListDocumentChecksums`, `ListDocumentMetas`,
  `UpdateDocument`, `DeleteDocument`, `DownloadDocument`
- Version tolerance: `ProbeCapabilities`, `Capabilities.ChecksumShape`
- Respectful retries: `RetryAfterError` carrying `Retry-After` hints
- Tunable default transport constants: `DefaultMaxIdleConns`,
  `DefaultMaxIdleConnsPerHost`, `DefaultIdleConnTimeout` (shipped in 0.1.0,
  unlisted until now)
- `ErrInvalidConfig` sentinel for `New` validation failures (shipped in
  0.1.0, unlisted until now)
- Typed errors via `github.com/larsartmann/go-error-family`
- httptest-based test suite, green under `-race`

[0.3.1]: https://github.com/LarsArtmann/go-paperless/releases/tag/v0.3.1
[0.3.0]: https://github.com/LarsArtmann/go-paperless/releases/tag/v0.3.0
[0.2.0]: https://github.com/LarsArtmann/go-paperless/releases/tag/v0.2.0
[0.1.1]: https://github.com/LarsArtmann/go-paperless/releases/tag/v0.1.1
[0.1.0]: https://github.com/LarsArtmann/go-paperless/releases/tag/v0.1.0
