# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, use ROADMAP.md.
> Items are ranked by impact. Status is verified, not assumed. Completed rows
> are deleted (they live in `CHANGELOG.md`), never retained.

## Status legend

| Status           | Meaning                                                     |
| ---------------- | ----------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                   |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                   |
| 🔵 `BLOCKED`     | Cannot proceed, external dependency or decision needed.     |
| 🟢 `DONE`        | Completed. Remove from this list and log in `CHANGELOG.md`. |

## High Impact

| Task                                                                                                                             | Status       | Impact | Effort | Evidence                                                                                                                                                                                                             |
| -------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Build `paperlesstest/` consumer testing SDK (stateful fake server, fixtures, fault injection, round-trip-verified); then migrate InboxClean's two hand-rolled fakes + bank-sync | 🔴 `TODO`    | High   | ~14h   | No exported test surface today; InboxClean re-implements the wire protocol twice in `paperless_command_test.go` (:245, ~:1100-1259). Full plan: `docs/planning/2026-09-23_16-19_paperlesstest-consumer-testing-sdk.md` |
| Push bank-sync + InboxClean consumer bumps (go 1.27 + v0.3.0)                                                                    | 🔵 `BLOCKED` | High   | ~10min | Both bumped and green locally, not pushed — the v0.3.0 rollout exists on one machine until then (`docs/status/…05-24….md` b-4, g-1)                                                                                  |
| Decide: retire bank-sync's trimmed fork in favor of this module                                                                  | 🔵 `BLOCKED` | High   | ~2h    | `AGENTS.md` scope note calls this module a drop-in replacement for the fork; decision + migration belongs to bank-sync (`docs/status/…05-24….md` c-2)                                                                |
| Rebuild erraudit on go1.27 source-processing (x/tools)                                                                           | 🔵 `BLOCKED` | Med    | ~30min | erraudit v0.4.0 bundles go1.26-era x/tools and rejects this module with "packages contain errors" — reproduced on pristine v0.3.1; fix lives in the erraudit repo (`AGENTS.md` erraudit bullet)                      |
| Cut v0.3.2 release carrying dep floors (error-family v0.10.1, go-retry v0.6.0) + CHANGELOG [Unreleased]; then re-point consumers | 🔵 `BLOCKED` | High   | ~20min | Deps bumped + fully verified 2026-09-16, CHANGELOG entry written; awaiting user call on coordinating with the integration-test workstream (`docs/status/2026-09-16_17-35_dependency-upgrade-status.md` §g-2, §f-4/5) |
| Pin erraudit hermetically in flake once upstream rebuilds (flake input + preparedSource + `packages.erraudit`)                   | 🔵 `BLOCKED` | Med    | ~45min | Same upstream unblock as the rebuild row; recipe: bank-sync `flake.nix:83-89,295-347,563-570` (`docs/status/2026-09-16_17-35_dependency-upgrade-status.md` §f-7)                                                     |
