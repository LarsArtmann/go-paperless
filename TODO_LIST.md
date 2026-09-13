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

| Task                                            | Status    | Impact | Effort | Evidence                                                                                                                                                                                                                                                                                            |
| ----------------------------------------------- | --------- | ------ | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Add GitHub Actions CI running `nix flake check` | 🔴 `TODO` | High   | ~1h    | `.github/` holds only `dependabot.yml`; `flake.nix:95` defines `checks.build/lint/format` that nothing schedules; `README.md:78` calls `nix flake check` the "CI equivalent"                                                                                                                        |
| Make the sandboxed flake checks hermetic        | 🔴 `TODO` | High   | 1-2h   | `nix flake check` fails: `checks.build/build-standalone/lint` (`flake.nix:96`, `flake.nix:105`, `flake.nix:117`) are sandboxed builds whose `go build` must download `go-error-family` and the sandbox DNS is refused — vendor the dep or fetch modules via Nix; prerequisite for the CI task above |

## Medium Impact

| Task                                              | Status    | Impact | Effort | Evidence                                                                                                                                                                                  |
| ------------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Investigate + fix gopls "requires go1.27" warnings | 🔴 `TODO` | Med    | 30min  | 20 gopls warnings: `json.Unmarshal requires go1.27 or later (file is go1.26)`; `go.mod` pins `go 1.26.7` while tests pass under nix go_1_26 — decide go.mod bump vs toolchain pin (consumer constraint applies) |
| Add a `DownloadDocument` test                     | 🔴 `TODO` | Med    | 15min  | `client.go:1195`; no test references `DownloadDocument` (rg verified) — only public method without coverage                                                                               |
| Cover the 503 `Retry-After` branch                | 🔴 `TODO` | Med    | 15min  | `client.go:1443` wraps 503 in `RetryAfterError`; only the 429 path is tested (`client_test.go:583`)                                                                                       |
| Add `.golangci.yml` so the nolint directives bite | 🔴 `TODO` | Med    | 30min  | No `.golangci.yml` in the repo; 11 sites use `//nolint:tagliatelle` (e.g. `client.go:319`), which golangci-lint does not enable by default — JSON-tag regressions are currently unguarded |

## Low Impact

| Task                                          | Status    | Impact | Effort | Evidence                                                                                                                                                                        |
| --------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Resolve the `go.work` gitignore contradiction | 🔴 `TODO` | Low    | 10min  | `.gitignore:11` un-ignores `go.work` ("required for multi-module workspace") but the buildflow block re-ignores it at `.gitignore:64`; last match wins, so the override is dead |
