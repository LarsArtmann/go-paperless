# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.1.0] - 2026-09-13

### Added

- Paperless-ngx REST client (API v10, token auth): `New` with
  `WithHTTPClient`/`WithTimeout` options
- Document upload (`Upload`, `UploadRequest`, `CustomFieldValue`) with async
  consumption-task polling (`GetTask`, `TaskOutcome`, `TaskStatus`)
- Idempotent lookups: `EnsureTag`, `EnsureCorrespondent`, `EnsureDocumentType`,
  `EnsureCustomField`, `FindCustomField`
- Document management: `ListDocumentChecksums`, `ListDocumentMetas`,
  `UpdateDocument`, `DeleteDocument`, `DownloadDocument`
- Version tolerance: `ProbeCapabilities`, `Capabilities.ChecksumShape`
- Respectful retries: `RetryAfterError` carrying `Retry-After` hints
- Typed errors via `github.com/larsartmann/go-error-family`
- httptest-based test suite, green under `-race`

[0.1.0]: https://github.com/LarsArtmann/go-paperless/releases/tag/v0.1.0
