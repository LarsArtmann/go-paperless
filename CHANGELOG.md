# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Changed

- Go floor raised to 1.27.1: `encoding/json/v2` is the default toolchain
  there, so consumers no longer need `GOEXPERIMENT=jsonv2`; consumer
  toolchains must be Go 1.27+ to build against this version
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
- Typed errors via `github.com/larsartmann/go-error-family`
- httptest-based test suite, green under `-race`

[0.1.0]: https://github.com/LarsArtmann/go-paperless/releases/tag/v0.1.0
