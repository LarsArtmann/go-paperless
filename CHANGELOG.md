# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- Nothing yet.

### Fixed

- Nothing yet.

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

[0.2.0]: https://github.com/LarsArtmann/go-paperless/releases/tag/v0.2.0
[0.1.1]: https://github.com/LarsArtmann/go-paperless/releases/tag/v0.1.1
[0.1.0]: https://github.com/LarsArtmann/go-paperless/releases/tag/v0.1.0
