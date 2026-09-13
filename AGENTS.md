# go-paperless — AI Agent Configuration

Paperless-ngx REST client SDK. Single-module repo, root package `paperless`.

## Critical

- **Every** `go` invocation needs `GOEXPERIMENT=jsonv2` (module uses
  `encoding/json/v2`). The flake devShell + apps set the full
  `jsonv2,goroutineleakprofile,simd`. Bare invocations fail with "build
  constraints exclude all Go files".
- Use `flake.nix` apps for everything: `nix run .#build|test|test-race|vet|lint|coverage|clean`, `nix fmt`.
- Machine `go env` (GOCACHE/GOMODCACHE/GOLANGCI_LINT_CACHE) is read-only and may
  point at a dead mount — bare golangci-lint needs fresh temp dirs for all three.

## Scope

**Client only.** No pipeline abstractions, no ledger/projection code — those are
domain-coupled and belong to the consumers (InboxClean, bank-sync). Public API
names are a compatibility contract: bank-sync consumes this as a drop-in
replacement for its trimmed fork.

## Conventions

- Go 1.26+, functional patterns, early returns, descriptive names.
- Tests are httptest-based (no live server needed).
- `nolint` in block form with a reason.
- Errors: `github.com/larsartmann/go-error-family` (`New*`/`Wrap*` with dot-notation codes).
