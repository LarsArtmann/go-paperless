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

| Task                                            | Status    | Impact | Effort | Evidence                                                                                                                                                                                                                                     |
| ----------------------------------------------- | --------- | ------ | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Add GitHub Actions CI running `nix flake check` | 🔴 `TODO` | High   | ~1h    | `.github/` holds only `dependabot.yml`; `flake.nix` defines `checks.build/test/lint/format` that nothing schedules; `README.md:78` calls `nix flake check` the "CI equivalent"; hermetic since 2026-09-13 (`nix run .#check` exits 0) |

## Medium Impact

| Task                                              | Status    | Impact | Effort | Evidence                                                                                                                                                                                  |
| ------------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Add `.golangci.yml` so the nolint directives bite | 🔴 `TODO` | Med    | 30min  | No `.golangci.yml` in the repo; 11 sites use `//nolint:tagliatelle` (e.g. `client.go:319`), which golangci-lint does not enable by default — JSON-tag regressions are currently unguarded |

## Low Impact

| Task                                          | Status    | Impact | Effort | Evidence                                                                                                                                                                        |
| --------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Resolve the `go.work` gitignore contradiction | 🔴 `TODO` | Low    | 10min  | `.gitignore:11` un-ignores `go.work` ("required for multi-module workspace") but the buildflow block re-ignores it at `.gitignore:64`; last match wins, so the override is dead |
