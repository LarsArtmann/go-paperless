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

No open items.

## Low Impact

No open items.
