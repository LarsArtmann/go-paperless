# Security Policy

## Supported Versions

| Version          | Supported |
| ---------------- | --------- |
| latest on `main` | yes       |

The SDK is consumed as a Go module (v0.x); security fixes land on `main`
and ship in the next tagged release. Pin versions via `go get` and update
promptly.

## Reporting a Vulnerability

**Do not open a public issue for security vulnerabilities.**

Use GitHub's private vulnerability reporting:
[Report a vulnerability](https://github.com/LarsArtmann/go-paperless/security/advisories/new).

You will get an acknowledgment within 7 days and a fix-or-assessment plan
for confirmed issues.

## Threat Model Notes

This is a client SDK for a self-hosted Paperless-ngx instance. What it
holds, and what matters:

- **API tokens are caller-supplied.** The client never logs or serializes
  them; it sends them as `Authorization: Token <token>` headers. Construct
  the client with the token from your secret manager, not from source code.
- **The base URL and all request/response bodies flow over the
  caller-configured `http.Client`.** Point it at TLS endpoints
  (`https://…`) for anything beyond localhost; `New` accepts plain HTTP
  for self-hosted LAN deployments by design.
- **No credentials are persisted** by the SDK — no config files, no caches.
- **Response hooks see raw `Set-Cookie` headers.** `WithResponseHook`
  receives a value snapshot with cloned headers, including any cookies the
  server sets. If your hook logs, redact the `Set-Cookie` (and
  `Authorization`) headers first — see the redaction example in the hook
  test suite.
- **Share links are public, unauthenticated URLs.** `CreateShareLink` asks
  the server for a slug; anyone who learns the resulting
  `<base>/share/<slug>` URL can download the document (or its original
  file, depending on the file version) with no login and no token. The SDK
  cannot revoke knowledge of a URL — it can only delete the link
  (`DeleteShareLink`), which kills every copy of the URL at once. Set an
  `Expiration` for time-boxed access, treat slugs as bearer secrets (don't
  log them, don't embed them in shared documents), and delete links as
  soon as their audience no longer needs them. Slug length and entropy are
  server-controlled — the SDK cannot guarantee unpredictability, so for
  highly sensitive documents prefer time-boxed links plus channel hygiene
  (or skip share links entirely and provision accounts).
- Uploads are multipart bodies read fully into memory; size them
  accordingly for very large documents.

## Scanning

CI runs `govulncheck` (Go vulnerability database, code-reachability aware)
and `gosec` (static analysis) on every push and pull request. A scheduled
`Fuzz` workflow exercises the parsing fuzz targets weekly (and on demand).
