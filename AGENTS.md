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
- **BuildFlow**: run it as `nix develop -c buildflow --fix …` (dev shell supplies
  Go 1.27.1 + every BuildFlow-probed tool: go-licenses, govulncheck, lychee,
  dprint, vulnix). From a bare shell, `env -u GOTOOLCHAIN buildflow …` also
  works: `.buildflow.yml` sets `env: GOTOOLCHAIN: auto`, but BuildFlow's
  caller-wins env contract lets an exported shell `GOTOOLCHAIN=local` preempt
  it (feedback filed in the BuildFlow repo, docs/feedback/new/ 2026-09-16).
- **go-auto-upgrade's samber/lo suggestions are accepted noise**: the two
  "manual Map" sites in client.go are pure type conversions
  (`StoragePath(payload)`, `CustomFieldValue(field)`); `lo.Map` would add a
  dependency for nothing. Don't add samber/lo for them.
- **Race coverage is deliberate and redundant by design**: CI runs
  `nix flake check` (which builds `checks.test-race`) AND an explicit
  `go test -race ./…` job, matching the standalone govulncheck/gosec job
  pattern. The explicit job exists so tooling that only scans CI YAML sees
  the race detector.
- **govalid-generate fails on the machine's stale govalid, not on this repo**:
  `/run/current-system/sw/bin/govalid` is a nixpkgs snapshot
  (`govalid-0-unstable-2026-05-16`) built against go1.26 source-processing;
  against this module's Go 1.27 floor (encoding/json/v2) its markers analysis
  exits 1, so BuildFlow's `govalid-generate` step fails while build/test pass.
  No upstream releases exist (`buildflow upgrade` 404s); it heals when the
  machine's nixpkgs bumps govalid. SKIPPED in `.buildflow.yml` for now: this
  project has no govalid usage (no `go:generate`, no govalid config), so the
  step guards nothing here while keeping every full run red. Remove the skip
  when govalid is adopted or the machine's govalid rebuilds on Go ≥ 1.27.
- **branching-flow is skipped by policy**: its PHANTOM_TYPE findings land at
  error severity with no repair path, gating the pipeline on public string
  fields (`Title`, `Slug`, `Checksum`, ...). Typed public fields are a
  breaking API change under the compatibility contract — v2 scope, tracked in
  ROADMAP ("Growing with the API"). Re-enable branching-flow together with a
  typed-API migration.
- **`nix fmt` (treefmt) lags bare `dprint` after plugin bumps**: dprint
  plugins update via `dprint config update` (bumped 2026-09-16 to
  json 0.24.0 / markdown 0.24.0 / dockerfile 0.6.0 / npm-yaml URL), but
  treefmt doesn't apply the new plugin's formatting (e.g. markdown table
  padding). When `dprint check` flags files after a plugin bump, fix with
  bare `dprint fmt <file>`; `nix fmt` is a no-op there, so no ping-pong.

## Scope

**Client only.** No pipeline abstractions, no ledger/projection code — those are
domain-coupled and belong to the consumers (InboxClean, bank-sync). Public API
names are a compatibility contract: bank-sync consumes this as a drop-in
replacement for its trimmed fork.

## Conventions

- Go 1.27+, functional patterns, early returns, descriptive names.
- Tests are httptest-based (no live server needed).
- `nolint` single-line with a reason: `//nolint:<linter> // reason`.
- **erraudit gate** (inside `nix develop`): `erraudit ./... --type-aware
  --enforce-go-error-family --disable-extensions` exits 0. Deliberate
  stdlib-constructor sites carry `//nolint:erraudit // reason` ON the anchor
  line (`nix fmt` may move it to the closing paren — suppression still
  anchors). golangci's "Found unknown linters: erraudit" warning is
  unavoidable noise. `--no-suppress` is audit mode: suppressed findings
  reappearing there is by design.
- **Error model = ADR 0002** (`docs/adr/0002-error-model.md`): codes+family
  live at the failure source; call-site wraps are uncoded `fmt.Errorf`
  context wraps (an outer code would shadow the inner HTTP classification —
  a tested contract). One code, one meaning, one family.
  `--enforce-generic-return` stays off: bare `error` returns are the SDK
  contract.
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
