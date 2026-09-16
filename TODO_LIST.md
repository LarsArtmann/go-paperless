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

| Task                                                            | Status       | Impact | Effort | Evidence                                                                                                                                              |
| --------------------------------------------------------------- | ------------ | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Push bank-sync + InboxClean consumer bumps (go 1.27 + v0.3.0)   | 🔵 `BLOCKED` | High   | ~10min | Both bumped and green locally, not pushed — the v0.3.0 rollout exists on one machine until then (`docs/status/…05-24….md` b-4, g-1)                   |
| Decide: retire bank-sync's trimmed fork in favor of this module | 🔵 `BLOCKED` | High   | ~2h    | `AGENTS.md` scope note calls this module a drop-in replacement for the fork; decision + migration belongs to bank-sync (`docs/status/…05-24….md` c-2) |
| Rebuild erraudit on go1.27 source-processing (x/tools)          | 🔵 `BLOCKED` | Med    | ~30min | erraudit v0.4.0 bundles go1.26-era x/tools and rejects this module with "packages contain errors" — reproduced on pristine v0.3.1; fix lives in the erraudit repo (`AGENTS.md` erraudit bullet) |
