# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, use ROADMAP.md.
> Items are ranked by impact. Status is verified, not assumed.

## Status legend

| Status           | Meaning                                                     |
| ---------------- | ----------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                   |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                   |
| 🔵 `BLOCKED`     | Cannot proceed, external dependency or decision needed.     |
| 🟢 `DONE`        | Completed. Remove from this list and log in `CHANGELOG.md`. |

## High Impact

| Task                                                            | Status    | Impact | Effort | Evidence                                                                                                                                                                                                    |
| --------------------------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Cut v0.2.0 (the P3 batch is tested + documented)                | 🟢 `DONE` | High   | ~30min | Tag `v0.2.0` exists at HEAD and `CHANGELOG.md` has the `[0.2.0] - 2026-09-13` section; the three §g questions were resolved by cutting the release                                                          |
| Bump consumer flakes to Go 1.27 before `go get`-ing v0.1.1+     | 🔴 `TODO` | High   | ~1h    | `go.mod` floors `go 1.27` (normalized from the `1.27.1` patch floor); bank-sync/InboxClean run `go_1_26` in their flakes and fail to build against this module until bumped (`docs/status/…15-39….md` f-30) |
| Decide: retire bank-sync's trimmed fork in favor of this module | 🔴 `TODO` | High   | ~2h    | `AGENTS.md` scope note calls this module a drop-in replacement for the fork; decision + migration belongs to bank-sync (`docs/status/…15-39….md` f-31)                                                      |

## Medium Impact

| Task                                                                             | Status    | Impact | Effort | Evidence                                                                                                                                                                                                    |
| -------------------------------------------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| T18: integration test scaffold (`//go:build integration`)                        | 🔴 `TODO` | Med    | ~3h    | Env-driven (`PAPERLESS_URL`/`PAPERLESS_TOKEN`), skip-if-unset, upload → poll → reconcile e2e (`docs/status/…15-39….md` f-18); suite is httptest-only otherwise                                              |
| CI: add a `test-race` job + verify the `pull_request` trigger                    | 🔴 `TODO` | Med    | ~30min | `ci.yml` runs `nix flake check`/govulncheck/gosec on push+PR but no race job; only the push path was exercised (`docs/status/…15-39….md` f-22, f-23)                                                        |
| CI: add an `erraudit lint ./... --type-aware` gate                               | 🔴 `TODO` | Med    | ~15min | Error-modernization regression guard: `erraudit` currently reports zero findings (verified 2026-09-13); pin it in CI before the first `errors.As`/`errors.Is` regression lands                              |
| T22: ADR 0001 — API-version policy beyond v10                                    | 🔴 `TODO` | Med    | ~1h    | Probe-vs-negotiate-vs-break decision record the ROADMAP's confidence theme references (`docs/status/…15-39….md` f-20)                                                                                       |
| Coverage: 88.4% → ≥90%                                                           | 🟢 `DONE` | Med    | ~1h    | 90.1% after the custom-fields round-trip + options/transport tests (2026-09-13); remaining gap is dominated by unreachable `bytes.Buffer` write-error branches                                              |
| pkg.go.dev has not indexed v0.1.1/v0.2.0 (v0.1.0 is the latest shown)            | 🔴 `TODO` | Med    | ~10min | Proxy serves both (`go list -m -versions` + scratch `go get` v0.2.0 exit 0); use the pkg.go.dev "Request indexing" form or wait out the indexer backlog                                                     |
| Plan tasks T06–T08: UpdateDocument shape, Options/transport, error-surface tests | 🟢 `DONE` | Med    | ~2h    | All three named gaps closed: `TestWithTimeoutBoundsSlowResponses`, `TestWithHTTPClientRoutesRequestsThroughSuppliedClient`, `TestNegotiatedAPIVersionReadsContentType`, plus the existing ctx-deadline test |
| Append a note to the v0.1.1 GitHub release body about the red gosec job          | 🟢 `DONE` | Med    | ~5min  | Body appended via `gh release edit v0.1.1` (tag untouched; supersede ruled out by the Q1 answer) — visible on the release page                                                                              |

## Low Impact

| Task                                                           | Status    | Impact | Effort | Evidence                                                                                                                                                      |
| -------------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| T23: streaming-upload spike note                               | 🔴 `TODO` | Low    | ~1h    | io.Reader body + Content-Length constraints, decision record before promising the feature (ROADMAP theme 2)                                                   |
| `ExampleClient_WaitForTask` + retry example                    | 🔴 `TODO` | Low    | ~30min | Example tests keep README↔code drift honest (pattern: `example_test.go:104`); `docs/status/…15-39….md` f-27                                                   |
| Confirm `.crushrc` cleared the gopls/golangci-lint-ls failures | 🔴 `TODO` | Low    | ~5min  | In-session `lsp_restart` did NOT pick up the pin (2026-09-13): diagnostics still cite go 1.26.7 / `GOTOOLCHAIN=local`; a full Crush restart is still required |
| Hook-coverage for `ProbeCapabilities` requests                 | 🔴 `TODO` | Low    | ~15min | Probes go through `doRequestDetail`, so hooks fire there too — assert it (`docs/status/…15-39….md` f-34)                                                      |
