# go-paperless — AI Agent Configuration

Paperless-ngx REST client SDK. Single-module repo, root package `paperless`.

## Critical

- **Go 1.27+** (`go.mod` floors `go 1.27` — major.minor only; `encoding/json/v2` is the default
  there). The flake sets `GOEXPERIMENT=jsonv2,simd` for the SIMD work; on a
  Go ≤ 1.26 toolchain json/v2 additionally needs `GOEXPERIMENT=jsonv2`, and
  bare older invocations fail with "build constraints exclude all Go files".
- Editor LSPs run the _machine's_ `go` (older, and `go env -w GOTOOLCHAIN=local`
  is set host-wide) — the committed `.crushrc` pins `GOTOOLCHAIN=auto` for
  gopls/golangci-lint so they switch to the module's toolchain. If LSP
  diagnostics still cite go 1.26.7, the LSP process predates the pin (needs
  a Crush restart, startup-only lifecycle).
- Use `flake.nix` apps for everything: `nix run .#check|build|test|test-race|vet|lint|coverage|clean`, `nix fmt`. Bare `nix build` also works (`packages.default`).
- Machine `go env` (GOCACHE/GOMODCACHE/GOLANGCI_LINT_CACHE) is read-only and may
  point at a dead mount — bare golangci-lint needs fresh temp dirs for all three.
- Host `erraudit`/`go` invocations run the machine's Go 1.26.7 and hard-fail on
  this module — run them inside `nix develop`.

## Scope

**Client only.** No pipeline abstractions, no ledger/projection code — those are
domain-coupled and belong to the consumers (InboxClean, bank-sync). Public API
names are a compatibility contract: bank-sync consumes this as a drop-in
replacement for its trimmed fork.

## Conventions

- Go 1.27+, functional patterns, early returns, descriptive names.
- Tests are httptest-based (no live server needed).
- `nolint` single-line with a reason: `//nolint:<linter> // reason`.
- `// art-dupl:<scope>` comments mark accepted duplication for the clone
  scanner (e.g. the Ping/`ProbeCapabilities` first-page query pair) — don't
  "fix" them away without checking the paired site named in the comment.
- Errors: `github.com/larsartmann/go-error-family` (`New*`/`Wrap*` with dot-notation codes).

## Docs

Feature inventory: `FEATURES.md` · Open work: `TODO_LIST.md` · Long-term
ideas: `ROADMAP.md` · Release history: `CHANGELOG.md` · Contributor setup:
`CONTRIBUTING.md` · Error-code catalog: `docs/ERROR_CODES.md` · Decision
records: `docs/adr/` · Point-in-time reports: `docs/status/` (annotated,
some archived under `docs/status/archived/`).
