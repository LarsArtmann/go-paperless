# go-paperless — AI Agent Configuration

Paperless-ngx REST client SDK. Single-module repo, root package `paperless`.

## Critical

- **Go 1.27+** (`go.mod` floors `go 1.27` — major.minor only; `encoding/json/v2` is the default
  there). The flake sets `GOEXPERIMENT=jsonv2` (the `simd` experiment was
  dropped 2026-09-16: it only enables the `simd` stdlib package, which this
  module never imports); on a
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
- Host `erraudit`/`go`: the machine profile ships Go 1.27.0 since the
  2026-10-11 update (was 1.26.7, which hard-failed on this module), but the
  flake dev shell stays the canonical environment (pinned toolchain,
  `GOEXPERIMENT=jsonv2`). Caveat: `nix develop -c erraudit …` resolves the
  binary via the host PATH (`~/.local/bin/erraudit`), not a flake input —
  the hermetic pin is an open TODO_LIST item. Verify with
  `nix develop -c sh -c 'which erraudit'`.
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
- **branching-flow runs at 0 active findings** (verified 2026-10-11,
  `branching-flow all . --format finding`). Disposition of the original 60
  findings (full triage: `docs/status/2026-10-11_03-41_branching-flow-findings-triage.md`):
  - **13 eliminated in code**: `faults.go` stores `[]*Fault` and disarms via
    `slices.DeleteFunc` (no more sentinel-index arithmetic); the five fake
    entity/wire struct pairs merged into single tagged structs
    (`namedEntity`, `customField`, `storagePath`, `savedView`/`savedViewRule`,
    `shareLink`) — the fake stores and serves the same shape, deleting ~10
    structs and all conversion ceremony.
  - **16 suppressed per line**: `//nolint:branching-flow // reason` works ONLY
    for the panic and phantom analyzers in tool v0.6.4 (dupe/composition
    findings ignore line directives — `cmd/branching-flow/container.go` wires
    `isSuppressed` solely for panic/phantom). A bare directive (no `:rule`)
    suppresses ALL branching-flow findings on that line for those two
    analyzers (suppress-all semantics, `parseNolintComment`); rule-scoped
    suffixes (`:panic`, `:phantom`) are accepted but unprobed. Directives must sit ON the
    finding line or the line DIRECTLY above; keep anchor lines short or
    `nix fmt` (golines) reflows them and silently breaks suppression — always
    re-run the tool after formatting.
  - **31 dismissed per rule** in `.branching-flow.yml` `disabled_rules`
    (DUPLICATE_TYPE, COMPOSITION_mixin, ANTI_PATTERN_large-struct): these
    classes cannot be line-suppressed in v0.6.4. Per-site rationale lives as
    doc-comment paragraphs on the flagged declarations (public structs stay
    tag-free = compatibility contract; SDK↔fake mirrors = the fake's
    round-trip contract; input-vs-response shapes diverge by design;
    `Server` = fixture state under one mutex).
  - Revisit everything with the v2 typed-API migration (ROADMAP "Growing
    with the API") and when branching-flow ships line-suppression for
    dupe/composition. The **strong-id** analysis stays at zero (ADR 0004).
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
replacement for its trimmed fork. Breaking type changes ship as 0.x minor
releases (v0.5.0 did for branded IDs, ADR 0004) — consumers pin versions, and
the compiler walks migrations when they upgrade.

## Conventions

- Go 1.27+, functional patterns, early returns, descriptive names.
- Tests are httptest-based (no live server needed).
- **Branded entity IDs = ADR 0004** (`docs/adr/0004-branded-entity-ids.md`):
  every entity reference is `TagID`/`CorrespondentID`/`DocumentID`
  (int64-backed!)/`TaskID` (string-backed)/… from
  `github.com/larsartmann/go-branded-id`, defined in `ids.go` with exported
  brands (`Name()` → `tag:7` display) and `NewXxxID` constructors. Rules:
  error messages/codes format `.Get()` (byte-identical to raw-int output,
  never the branded `String()`); guards use `.IsZero()`; named-entity
  plumbing is generic over the brand (`findNamed[B]`, `createNamed[B]`,
  `namedPayload[B]`) as FREE functions taking `*Client` — the Go 1.27
  compiler accepts type-parameterized methods but golines (in `nix fmt`)
  does not parse them, so generic methods are banned here. Zero-value IDs
  marshal as JSON `null` (create bodies send `"id":null`; paperless-ngx
  serializers ignore read-only ids).
- **`paperlesstest/` is the consumer-testing fake** (plan:
  `docs/planning/2026-09-23_16-19_paperlesstest-consumer-testing-sdk.md`).
  Its non-test files may import ONLY the root `paperless` package's ID
  types (ADR 0004 contract amendment) — no SDK logic, no other third-party
  imports (round-trip tests live in its `_test.go` files); fixtures,
  counters, and introspection speak the SDK's branded ID vocabulary so
  fake state compares equal to SDK results. The shared named-family
  routing is free generic functions (`routeNamedFamily[B]`) because
  methods cannot carry type parameters onto that reuse. Every route must
  be registered in `servedRoutes` AND
  round-tripped through the real `Client`;
  `TestFakeCoversEveryClientRoute` enforces set equality with client.go's
  `path*` constants in both directions, so a new SDK endpoint fails the
  suite until the fake speaks it. Consumers' hand-rolled fakes should be
  replaced by this package, not extended.
- `nolint` single-line with a reason: `//nolint:<linter> // reason`.
- **erraudit gate WORKS again (verified 2026-09-23)**: the 2026-09-16
  breakage ("packages contain errors" from go1.26-era x/tools) is healed
  upstream — `erraudit ./... --type-aware --enforce-go-error-family
  --disable-extensions` exits 0 inside `nix develop` on the clean state
  and correctly flags real violations (it caught an error-discard in
  paperlesstest). Run it inside `nix develop` (the machine's bare `go`
  is too old). Deliberate stdlib-constructor sites carry
  `//nolint:erraudit // reason` ON the anchor line (`nix fmt` may move it
  to the closing paren — suppression still anchors). golangci's "Found
  unknown linters: erraudit" warning is unavoidable noise. `--no-suppress`
  is audit mode: suppressed findings reappearing there is by design.
  **CI decision (2026-09-16)**: erraudit stays out of CI until it is (a)
  published on the public module proxy (proxy.golang.org 404s it as of
  v0.4.0 — re-verify before revisiting) AND (b) rebuilt on go1.27
  source-processing (done). Private-runner auth and a version-pinned
  golangci plugin were rejected as worse tradeoffs. When the proxy
  unblocks: one CI job, `go run
  github.com/larsartmann/erraudit/cmd/erraudit@latest` with the flags above.
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
- **Test cleanup: `defer server.Close()` stays.** Decision (2026-09-16):
  httptest servers and body readers close via `defer` at the top of the test
  that created them (~63 sites, per-test, no subtest survives its parent).
  `t.Cleanup` is reserved for cleanups a helper registers on the test's
  behalf (see `TestIntegrationUploadReconcile`) and for things that must run
  after subtests. Don't migrate the defer sites.
- **Release version lives in three places and must agree**: flake.nix
  `version`, the newest `## [X.Y.Z]` CHANGELOG heading, and the git tag.
  `TestFlakeVersionMatchesChangelog` (`docs_drift_test.go`) enforces the
  first two mechanically; `nix run .#release-verify -- vX.Y.Z` checks all
  three plus push status, CI, pkg.go.dev, and a race run. Bump both files
  inside the release commit (the v0.3.1 flow; the v0.3.0 drift is closed).
- **Standalone gosec honors `#nosec`, not `nolint`**: the CI gosec job runs
  the bare scanner (`go run .../gosec@v2.29.0`), which only reads gosec's own
  `// #nosec Gxxx` annotations — a golangci-style `//nolint:gosec` comment
  suppresses nothing there (the v0.4.2 red CI was exactly this). Suppressions
  aimed at the CI gate go in gosec syntax, with the rationale as a preceding
  comment.
- **Doc-drift tests are the catalog's CI** (`docs_drift_test.go`):
  `TestErrorCodesAreDocumented` extracts every `paperless.*` code — static
  literals AND the kind-parameterized helper sites (`findNamed`,
  `createNamed`, `findByName`, ...) resolve to — from client.go and requires a
  docs/ERROR_CODES.md row for each; a non-literal kind argument fails the
  test so new sites can't hide codes. `TestREADMEListsLookupVerbs` requires
  every exported `Find*`/`Get*`/`Ensure*` method in README.md.

## Docs

Feature inventory: `FEATURES.md` · Open work: `TODO_LIST.md` · Long-term
ideas: `ROADMAP.md` · Release history: `CHANGELOG.md` · Contributor setup:
`CONTRIBUTING.md` · Error-code catalog: `docs/ERROR_CODES.md` · Decision
records: `docs/adr/` · Point-in-time reports: `docs/status/` (annotated,
some archived under `docs/status/archived/`).
