# go-paperless — AI Agent Configuration

Paperless-ngx REST client SDK. Single-module repo, root package `paperless`.

## Critical

- **Go 1.27+** (`go.mod` floors `go 1.27` — major.minor only; `encoding/json/v2` is the
  default there). The flake sets `GOEXPERIMENT=jsonv2`; on a Go ≤ 1.26 toolchain json/v2
  needs that var too, and bare older invocations fail with "build constraints exclude all
  Go files".
- Editor LSPs run the _machine's_ `go` (older; `go env -w GOTOOLCHAIN=local` is set
  host-wide) — the committed `.crushrc` pins `GOTOOLCHAIN=auto` for gopls/golangci-lint so
  they switch to the module's toolchain. Diagnostics still citing go 1.26.7 = the LSP
  process predates the pin (needs a Crush restart, startup-only lifecycle).
- Use `flake.nix` apps for everything: `nix run .#check|build|test|test-race|vet|lint|coverage|clean`, `nix fmt`. Bare `nix build` also works (`packages.default`).
- Machine `go env` (GOCACHE/GOMODCACHE/GOLANGCI_LINT_CACHE) targets `/mnt/buildcache`,
  which is sometimes a dead or 100%-full mount — full disk makes test/tidy/dprint fail
  with "no space left" or exit 20. Diagnose `df /mnt/buildcache`; prune with
  `go clean -cache` (retry once — concurrent gopls writers can abort it), then rerun. Bare
  golangci-lint additionally needs fresh temp dirs for all three caches.
- Host `erraudit`/`go`: the machine profile ships Go 1.27.0, but the flake dev shell stays
  canonical (pinned toolchain, `GOEXPERIMENT=jsonv2`). Caveat: `nix develop -c erraudit …`
  resolves the binary via the host PATH (`~/.local/bin/erraudit`), not a flake input — the
  hermetic pin is an open TODO_LIST item. Verify: `nix develop -c sh -c 'which erraudit'`.
- **BuildFlow**: run it as `nix develop -c buildflow --fix …` (dev shell supplies Go 1.27.1
  and every BuildFlow-probed tool). From a bare shell, `env -u GOTOOLCHAIN buildflow …`
  also works: `.buildflow.yml` sets `env: GOTOOLCHAIN: auto`, but caller-wins env lets an
  exported `GOTOOLCHAIN=local` preempt it (feedback filed in the BuildFlow repo, 2026-09-16).
- **go-auto-upgrade's samber/lo suggestions are accepted noise**: the two "manual Map" sites
  in client.go are pure type conversions (`StoragePath(payload)`, `CustomFieldValue(field)`);
  `lo.Map` would add a dependency for nothing. Don't add samber/lo for them.
- **Race coverage is deliberate and redundant by design**: CI runs `nix flake check` (which
  builds `checks.test-race`) AND an explicit `go test -race ./…` job, matching the
  standalone govulncheck/gosec job pattern — the explicit job exists so tooling that only
  scans CI YAML sees the race detector.
- **govalid-generate is SKIPPED in `.buildflow.yml`**: the machine's govalid
  (`/run/current-system/sw/bin/govalid`, nixpkgs snapshot, built on go1.26
  source-processing) exits 1 against this module's Go 1.27 floor while build/test pass;
  no upstream releases exist (`buildflow upgrade` 404s), and this repo has no govalid
  usage, so the step guards nothing while keeping every full run red. Remove the skip
  when govalid is adopted or the machine's rebuilds on Go ≥ 1.27.
- **branching-flow runs at 0 active findings** (verify: `branching-flow all . --format
  finding`; full triage: `docs/status/2026-10-11_03-41_branching-flow-findings-triage.md`).
  Of the original 60 findings: 13 eliminated in code (faults as `[]*Fault` +
  `slices.DeleteFunc`; the five fake entity/wire struct pairs merged into single tagged
  structs), 16 line-suppressed, 31 dismissed per rule. Mechanics that bite:
  - `//nolint:branching-flow // reason` works ONLY for the panic/phantom analyzers (tool
    v0.6.4); a bare directive (no `:rule`) suppresses ALL branching-flow findings on that
    line for those two (rule suffixes are accepted but unprobed). The directive must sit ON
    the finding line or directly above; keep anchor lines short or `nix fmt` (golines)
    reflows them and silently breaks suppression — always re-run the tool after formatting.
  - The 31 dismissals (DUPLICATE_TYPE, COMPOSITION_mixin, ANTI_PATTERN_large-struct in
    `.branching-flow.yml` `disabled_rules`) cannot be line-suppressed in v0.6.4; per-site
    rationale lives as doc-comment paragraphs on the flagged declarations (public structs
    stay tag-free = compatibility contract; SDK↔fake mirrors = the fake's round-trip
    contract; input-vs-response shapes diverge by design; `Server` = fixture state).
  - Revisit everything with the v2 typed-API migration (ROADMAP "Growing with the API") and
    when branching-flow ships line-suppression for dupe/composition. **strong-id** stays
    at zero (ADR 0004).
- **`nix fmt` (treefmt) lags bare `dprint` after plugin bumps**: when `dprint check` flags
  files after `dprint config update`, fix with bare `dprint fmt <file>`; `nix fmt` is a
  no-op there, so no ping-pong.
- **Never run bare `dprint fmt` over annotated status reports** (`docs/status/**` carrying
  `~~line~~` verdict strikes): the markdown table formatter cannot represent a
  strikethrough spanning cells and destroys the annotations (2026-10-11 incident, 11 files
  restored from daemon commits). Annotate AFTER formatting; `dprint check` flagging those
  historical files is the accepted state (the canonical `nix flake check` gate runs
  treefmt, which does not format markdown, so it stays green). Strike-free status files
  (grep `~~` = 0) with padding-only diffs CAN be formatted losslessly — verify by diffing
  space-stripped before/after (done 2026-10-30 for two files).

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
- **Name interface-method params** (`Cleanup(fn func())`, not `Cleanup(func())`): local
  golangci relies on inamedparam's `skip-single-param` default, but the nix build gate runs
  the older nixpkgs golangci-lint, which flags single unnamed params (build-red
  2026-10-30 until named).
- **Branded entity IDs = ADR 0004** (`docs/adr/0004-branded-entity-ids.md`): every entity
  reference is `TagID`/`CorrespondentID`/`DocumentID` (int64-backed!)/`TaskID`
  (string-backed)/… from `github.com/larsartmann/go-branded-id`, defined in `ids.go` with
  exported brands (`Name()` → `tag:7` display) and `NewXxxID` constructors. Rules: error
  messages/codes format `.Get()` (byte-identical to raw-int output, never the branded
  `String()`); guards use `.IsZero()`; named-entity plumbing is generic over the brand
  (`findNamed[B]`, `createNamed[B]`, `namedPayload[B]`) as FREE functions taking `*Client`
  — the compiler accepts type-parameterized methods but golines (in `nix fmt`) does not
  parse them, so generic methods are banned here. Zero-value IDs marshal as JSON `null`
  (create bodies send `"id":null`; paperless-ngx serializers ignore read-only ids).
- **`paperlesstest/` is the consumer-testing fake** (plan:
  `docs/planning/2026-09-23_16-19_paperlesstest-consumer-testing-sdk.md`). Its non-test
  files may import ONLY the root `paperless` package's ID types (ADR 0004 contract
  amendment) — no SDK logic, no other third-party imports (round-trip tests live in its
  `_test.go` files); fixtures, counters, and introspection speak the SDK's branded ID
  vocabulary so fake state compares equal to SDK results. The shared named-family routing
  is free generic functions (`routeNamedFamily[B]`) for the same golines reason. Every
  route must be registered in `servedRoutes` AND round-tripped through the real `Client`;
  `TestFakeCoversEveryClientRoute` enforces set equality with client.go's `path*`
  constants in both directions, so a new SDK endpoint fails the suite until the fake
  speaks it. Consumers' hand-rolled fakes should be replaced by this package, not extended.
- `nolint` single-line with a reason: `//nolint:<linter> // reason`.
- **erraudit gate works (healed upstream, verified 2026-09-23)**: run `erraudit ./...
  --type-aware --enforce-go-error-family --disable-extensions` inside `nix develop` (the
  machine's bare `go` is too old); exits 0 on the clean state and correctly flags real
  violations. Deliberate stdlib-constructor sites carry `//nolint:erraudit // reason` ON
  the anchor line (`nix fmt` may move it to the closing paren — suppression still
  anchors). golangci's "Found unknown linters: erraudit" warning is unavoidable noise.
  `//nolint` suppressions are honored even in audit modes (`--no-suppress`/`--max`).
  Bare `--max` exits 2 with one `generic_return` finding per client.go method returning
  bare `error` (27 as of 2026-10-27): the per-method-error-type alternative ADR 0002
  explicitly rejected, not a regression — do NOT refactor signatures or add nolints for
  them. The sanctioned maximum-strictness audit is `erraudit lint ./... --max --disable
  generic_return` (exit 0). **CI decision (2026-09-16)**: erraudit stays out of CI until
  it is on the public module proxy (proxy.golang.org 404s it as of v0.4.0 — re-verify
  before revisiting; the go1.27 rebuild is done; private-runner auth and a version-pinned
  golangci plugin were rejected as worse tradeoffs). When the proxy unblocks: one CI job,
  `go run github.com/larsartmann/erraudit/cmd/erraudit@latest` with the flags above.
- **Error model = ADR 0002** (`docs/adr/0002-error-model.md`): codes+family live at the
  failure source; call-site wraps are uncoded `fmt.Errorf` context wraps (an outer code
  would shadow the inner HTTP classification — a tested contract). One code, one meaning,
  one family. Amendments (2026-10-27): go-retry's terminal `retry.*` outcomes
  (`retry.exhausted`/`retry.canceled`/`retry.deadline`, Infrastructure) are the one
  sanctioned coded wrap ABOVE SDK errors — `Code`/`Classify` surface them outermost while
  the `paperless.*` cause stays reachable via `errors.Is`/`AsType` (documented in
  `docs/ERROR_CODES.md` "Retry outcomes"); the `ErrInvalidConfig` sentinel is coded
  (`paperless.invalid_config`, Rejection) at the `New` return site and never returned
  bare; unfollowable 3xx → `paperless.redirect` (Rejection).
  `TestEveryConstructedErrorIsCoded` (`docs_drift_test.go`) mechanically enforces that
  every constructed error is coded or a `%w` wrap. Consequently golangci's wrapcheck
  ignores `github.com/larsartmann/go-error-family` (`.golangci.yml`
  `wrapcheck.ignore-package-globs`): errorfamily constructors are the ADR 0002-sanctioned
  error model, not foreign errors escaping unwrapped — without the glob every constructor
  site is a false positive (exposed 2026-10-18 when `max-same-issues` was lifted 10 → 0;
  don't remove the glob and don't re-cap the issue limits). `--enforce-generic-return`
  stays off: bare `error` returns are the SDK contract (see the erraudit bullet for the
  sanctioned audit invocation).
- `// art-dupl:<scope>` comments mark accepted duplication for the clone scanner (e.g. the
  Ping/`ProbeCapabilities` first-page query pair) — don't "fix" them away without checking
  the paired site named in the comment.
- Errors: `github.com/larsartmann/go-error-family` (`New*`/`Wrap*` with dot-notation codes).
- **Test cleanup: `defer server.Close()` stays** (2026-09-16 decision): httptest servers
  and body readers close via `defer` at the top of the test that created them (~63 sites,
  per-test, no subtest survives its parent). `t.Cleanup` is reserved for cleanups a helper
  registers on the test's behalf (`TestIntegrationUploadReconcile`) and for things that
  must run after subtests. Don't migrate the defer sites.
- **Release version lives in three places and must agree**: flake.nix `version`, the newest
  `## [X.Y.Z]` CHANGELOG heading, and the git tag. `TestFlakeVersionMatchesChangelog`
  (`docs_drift_test.go`) enforces the first two mechanically; `nix run .#release-verify --
  vX.Y.Z` checks all three plus push status, CI, pkg.go.dev, and a race run. Bump both
  files inside the release commit (the v0.3.1 flow; the v0.3.0 drift is closed).
- **gosec in CI**: the standalone job runs the bare scanner, which honors only gosec's own
  `// #nosec Gxxx` annotations — a golangci-style `//nolint:gosec` suppresses nothing
  there (the v0.4.2 red CI); suppressions aimed at the CI gate go in gosec syntax with a
  preceding rationale comment. **The job is PARKED (since 2026-10-11)**: v2.29.0 (latest)
  cannot type-check Go 1.27 modules — its x/tools reads export data v4, Go 1.27 emits v5,
  so every stdlib import fails and the run exits 1 with Issues: 0 (`encoding/json/v2` was
  a red herring). Fail-closed: an instrument that cannot load the module measured nothing,
  so it must not gate. Restore the scan step in `.github/workflows/ci.yml` when gosec
  ships a go1.27-capable release, verifying locally first: `nix develop -c sh -c
  'GOEXPERIMENT=jsonv2 go run github.com/securego/gosec/v2/cmd/gosec@v2.29.0 ./...'`
  (expect Issues: 0, exit 0). Same pattern as the govalid skip in `.buildflow.yml`.
- **Doc-drift tests are the catalog's CI** (`docs_drift_test.go`):
  `TestErrorCodesAreDocumented` extracts every `paperless.*` code — static literals AND
  the kind-parameterized helper sites (`findNamed`, `createNamed`, `findByName`, ...)
  resolve to — from client.go and requires a docs/ERROR_CODES.md row for each; a
  non-literal kind argument fails the test so new sites can't hide codes.
  `TestREADMEListsLookupVerbs` requires every exported `Find*`/`Get*`/`Ensure*` method in
  README.md.

## Docs

Feature inventory: `FEATURES.md` · Open work: `TODO_LIST.md` · Long-term
ideas: `ROADMAP.md` · Release history: `CHANGELOG.md` · Contributor setup:
`CONTRIBUTING.md` · Error-code catalog: `docs/ERROR_CODES.md` · Decision
records: `docs/adr/` · Point-in-time reports: `docs/status/` (annotated,
some archived under `docs/status/archived/`).
