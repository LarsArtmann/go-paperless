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
- Uploads are multipart bodies read fully into memory; size them
  accordingly for very large documents.

## Scanning

CI runs `govulncheck` (Go vulnerability database, code-reachability aware)
and `gosec` (static analysis) on every push and pull request.
