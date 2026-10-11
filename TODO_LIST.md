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

| Task                                                                                                                                  | Status       | Impact | Effort       | Evidence                                                                                                                                                                                        |
| ------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Restore consumer-repo CI: GitHub Actions billing failure blocks every job start on InboxClean and bank-sync (go-paperless unaffected) | 🔵 `BLOCKED` | High   | owner action | Both post-push runs (2026-09-23) fail pre-job with "recent account payments have failed or your spending limit needs to be increased"; all code-level gates ran green locally before the pushes |
| Migrate bank-sync and InboxClean to go-paperless v0.5.0 (branded IDs, ADR 0004): bump the pin, let the compiler enumerate sites (`archiveMetadata` ints, `TaskOutcome.DocumentID` handling, paperlesstest fixtures), then drop the now-redundant int↔int64 conversions | 🔴 `TODO` | Medium | small | Both consumers pin v0.4.x and keep compiling; bank-sync already depends on go-branded-id for its own domain IDs (`internal/domain/ids.go`), so `PaperlessDocumentID` and `paperless.DocumentID` unify trivially |
