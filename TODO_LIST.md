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

| Task                                                                     | Status      | Impact | Effort | Evidence                                                                                                                                                                            |
| ------------------------------------------------------------------------ | ----------- | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Cut v0.2.0 (the P3 batch is tested + documented)                          | 🔵 `BLOCKED` | High   | ~30min | Needs answers to the three open questions in `docs/status/2026-09-13_15-39_pareto-execution-v011-release-p3-batch.md` §g; CHANGELOG `[Unreleased]` + FEATURES rows are ready        |
| Bump consumer flakes to Go 1.27 before `go get`-ing v0.1.1+               | 🔴 `TODO`   | High   | ~1h    | `go.mod` pins `go 1.27.1`; bank-sync/InboxClean run `go_1_26` in their flakes and fail to build against this module until bumped (`docs/status/…15-39….md` f-30)                     |
| Decide: retire bank-sync's trimmed fork in favor of this module           | 🔴 `TODO`   | High   | ~2h    | `AGENTS.md` scope note calls this module a drop-in replacement for the fork; decision + migration belongs to bank-sync (`docs/status/…15-39….md` f-31)                              |

## Medium Impact

| Task                                                        | Status    | Impact | Effort | Evidence                                                                                                                                                    |
| ----------------------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| T18: integration test scaffold (`//go:build integration`)   | 🔴 `TODO` | Med    | ~3h    | Env-driven (`PAPERLESS_URL`/`PAPERLESS_TOKEN`), skip-if-unset, upload → poll → reconcile e2e (`docs/status/…15-39….md` f-18); suite is httptest-only otherwise |
| CI: add a `test-race` job + verify the `pull_request` trigger | 🔴 `TODO` | Med    | ~30min | `ci.yml` runs `nix flake check`/govulncheck/gosec on push+PR but no race job; only the push path was exercised (`docs/status/…15-39….md` f-22, f-23)          |
| T22: ADR 0001 — API-version policy beyond v10                | 🔴 `TODO` | Med    | ~1h    | Probe-vs-negotiate-vs-break decision record the ROADMAP's confidence theme references (`docs/status/…15-39….md` f-20)                                        |
| Coverage: 88.4% → ≥90%                                       | 🔴 `TODO` | Med    | ~1h    | `nix run .#coverage` after the P3 batch (`docs/status/…15-39….md` f-28)                                                                                       |
| Plan tasks T06–T08: UpdateDocument shape, Options/transport, error-surface tests | 🔴 `TODO` | Med | ~2h | Micro tasks in `docs/planning/2026-09-13_13-51_pareto-plan-sdk-hardening-and-release.md` §T06–T08 were never executed (no `WithTimeout`/`negotiatedAPIVersion`/ctx-cancel tests exist in `client_test.go`) |
| Append a note to the v0.1.1 GitHub release body about the red gosec job | 🔵 `BLOCKED` | Med | ~5min | Pending the tag-posture answer (supersede vs leave) in `docs/status/…15-39….md` §g-1; the tag itself stays untouched either way                              |

## Low Impact

| Task                                                     | Status    | Impact | Effort | Evidence                                                                                                |
| -------------------------------------------------------- | --------- | ------ | ------ | -------------------------------------------------------------------------------------------------------- |
| T23: streaming-upload spike note                          | 🔴 `TODO` | Low    | ~1h    | io.Reader body + Content-Length constraints, decision record before promising the feature (ROADMAP theme 2) |
| `ExampleClient_WaitForTask` + retry example               | 🔴 `TODO` | Low    | ~30min | Example tests keep README↔code drift honest (pattern: `example_test.go:104`); `docs/status/…15-39….md` f-27 |
| Confirm `.crushrc` cleared the gopls/golangci-lint-ls failures | 🔴 `TODO` | Low | ~5min | Project `.crushrc` pins `GOTOOLCHAIN=auto`; only effective after a Crush restart (startup-only lifecycle)   |
| Hook-coverage for `ProbeCapabilities` requests            | 🔴 `TODO` | Low    | ~15min | Probes go through `doRequestDetail`, so hooks fire there too — assert it (`docs/status/…15-39….md` f-34)    |
