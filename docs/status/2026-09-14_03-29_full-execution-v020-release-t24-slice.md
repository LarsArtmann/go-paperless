# Status Report — Full execution: v0.2.0 release, T24 slice, fuzz-found bug fix

**Created:** 2026-09-14 03:29 CEST
**Repo:** `/home/lars/projects/go-paperless` (branch `main`, in sync with `origin/main`, HEAD `304d3e1`)
**Session:** Continuation of the 15:39 run. Scope: close out the P3 execution plan (tests, fuzz, T21), act on the user's three §g answers (Q1 leave v0.1.1 + note, Q2 cut v0.2.0, Q3 build T24), and ship the T24 endpoint slice.
**Baselines:** `docs/status/2026-09-13_15-39_pareto-execution-v011-release-p3-batch.md` (previous report), plan doc `docs/planning/2026-09-13_13-51_pareto-plan-sdk-hardening-and-release.md` (now fully annotated).
**Parallel session:** a second session ran concurrently in this repo (visible from ~18:40). Its work is flagged wherever it intersects; nothing of theirs was reverted.

> Format note: the status-report skill's canonical format is a styled HTML
> dashboard; the user explicitly requested `.md`. Honor-the-user override,
> same as the 15:39 report. Section sizes: user asked for up to 50 next
> items (f) and up to 3 questions (g); the skill defaults (25/1) are
> floors, not ceilings — the extra items are HARVEST fuel, not commitments.

---

## a) FULLY DONE

| # | Work | Evidence |
|---|------|----------|
| 1 | **P3 test batch (25 tests).** WithRetry: default-off single attempt, 5xx retry-then-success (3 requests), 404 never retried, Retry-After honored (elapsed ≥ ~1s), upload body replay byte-for-byte, negative `MaxAttempts` → `ErrInvalidConfig`, `DelayFunc` bridge unit test, zero-value defaults test. WaitForTask: pending→success, not-found→success, terminal failure → `paperless.task_failed` Rejection, duplicate refusal not an error, ctx deadline → Infrastructure + "last poll error", empty ID rejected with 0 HTTP, retry-interplay (the 15:39 report's f-26 semantics question, now pinned by a test). Plus `TaskOutcome.Duplicate` table, storage paths (create/find-keeps-template/empty-args/find-flag/list-pagination), hooks (mutation-no-leak + 512-byte capped error body) | `client_test.go:1706`–`2203` (retry/poll), `:2246`–`2684` (storage/hooks); suite green |
| 2 | **T17 fuzz targets ×4, all clean.** `FuzzParseRetryAfter`, `FuzzParseDocumentCreated`, `FuzzChecksumFrom`, `FuzzClassifyTask` with seeds + 5s runs each. **Found a real product bug:** `parseRetryAfter("12000000000000")` — `time.Duration(seconds)*time.Second` overflows into a NEGATIVE delay with `ok=true`. Fixed: delay-seconds too large for a Duration are now unparseable (`client.go:1819`); crash corpus committed as regression seeds (`testdata/fuzz/`) | `fuzz_test.go`; fix at daemon commit `acec996` |
| 3 | **T21 concurrency + page-cap tests.** Server always serves FULL 100-result pages → the scan stops at exactly 100 pages / 10,000 checksums (no hang); 8 concurrent `ListDocumentChecksums` callers, race-clean | `client_test.go:2586`, `:2627`; `nix run .#test-race` EXIT=0 |
| 4 | **Slug contract fix.** `EnsureStoragePath`'s POST was sending `"slug":""` (no `omitempty` on `storagePathPayload.Slug`) although the server generates the slug — the 15:39 report's f-46 contract test now exists AND the implementation was fixed to match it | `client.go` (payload struct), `TestEnsureStoragePathCreatesWhenMissing` asserts no slug key |
| 5 | **Stale vendorHash diagnosed + re-pinned.** `nix flake check` failed with "go-retry explicitly required … not marked as explicit in vendor/modules.txt": the pinned hash predated go-retry becoming a direct dep (P3 batch) and flake check hadn't run since. Re-pinned via the canonical fake-hash dance: `sha256-oLknr8l0AxmOxgbcCMRQqow//tSApuw+lrdZeFgXmgM=` | `flake.nix:52`; bare `nix flake check` EXIT=0 after |
| 6 | **staticcheck S1016 ×2 fixed** (identical payload/public structs): `StoragePath(entry)` in `ListStoragePaths`, `SavedViewFilterRule(rule)` in the saved-view mapping | `client.go`; lint 0 issues |
| 7 | **Docs batch v1** (pre-release): FEATURES re-anchored (~25 drifted citations) + 5 new API rows + new "Observability hooks" section; README features + Options table (+3 options); CHANGELOG `[Unreleased]`; ROADMAP themes 1–3 marked shipped with strikethroughs; TODO_LIST reseeded from the 15:39 report's f-list — **and the reseed exposed that plan tasks T06–T08 had never been executed** (no WithTimeout/negotiatedAPIVersion/ctx-cancel tests existed); 13:51 plan annotated inline: 18 of 24 task headings carry done-at hashes, T06–T08 + T18/T22/T23 left unmarked (open), T24 parked | `FEATURES.md`, `README.md`, `CHANGELOG.md`, `ROADMAP.md`, `TODO_LIST.md`, plan doc §T01–T24 |
| 8 | **Global AGENTS.md lessons** (15:39 report f-38): "commit per task when explicit commits are authorized; a security scanner must prove it scanned" added to Cross-Cutting Lessons | `/home/lars/.config/crush/AGENTS.md` |
| 9 | **Push + CI.** All work pushed; CI run 34761959684 fully green (gosec / nix flake check / govulncheck) — gosec now actually scans the new API code (the 15:39 "Files: 0" failure class is closed in practice) | `gh run list`; `origin/main` in sync |
| 10 | **v0.2.0 released** (user's Q2 answer). CHANGELOG `[0.2.0] - 2026-09-13` folded with empty `[Unreleased]` placeholders + footer link ref; flake `version = "0.2.0"`; **CI verified green on the EXACT tagged commit (`e2d6e7d`, run 34765795105) BEFORE tagging** — the v0.1.1 mistake, not repeated; annotated tag pushed; proxy verified (`go list -m -versions` lists v0.2.0); scratch-module `go get github.com/larsartmann/go-paperless@v0.2.0` exit 0; GitHub Release created from curated notes ([v0.2.0 release page](https://github.com/LarsArtmann/go-paperless/releases/tag/v0.2.0)) | tag `v0.2.0`, release URL, `/tmp/release-verify` go get |
| 11 | **v0.1.1 posture executed** (user's Q1 answer: leave). Tag untouched; CI-fix note appended to the release body via `gh release edit` (red gosec explained, fix commit 4fd2427 linked, supersession pointer to v0.2.0) | [v0.1.1 release page](https://github.com/LarsArtmann/go-paperless/releases/tag/v0.1.1) |
| 12 | **T24 wire shapes verified from SOURCE before coding** (Sourcegraph over paperless-ngx `serialisers.py`/`views.py`/`models.py`). This caught TWO wrong assumptions before they shipped: (a) share links carry a string `file_version` ("archive"/"original"), not a MIME `file_type`; (b) the notes endpoint is NOT paginated (bare JSON array; `pagination_class=None`), POST answers with the updated array, delete uses `?id=`. ShareLinkViewSet/SavedViewViewSet confirmed `StandardPagination` | Sourcegraph results (serialisers.py:1001, :1405, :2825; views.py:1818, :2802, :4622; models.py:971) |
| 13 | **T24 slice shipped, test-first (red-green).** 12 tests written BEFORE the implementation (compile-fail red, then green — the 15:39 lesson applied): notes `List/Add/DeleteDocumentNotes`, share links `List/Create/DeleteShareLink` (slug never sent; zero `FileVersion` omitted → server default archive; optional expiration), saved views `List/Create/DeleteSavedView` (stable-core fields incl. `filter_rules` rule_type+value; version-specific UI fields ignored). 9 methods + 5 exported types in `client.go:1875`–`2395`. Coverage 88.4% → **89.1%** | `client_test.go:2794`–`3312`; suite + race + lint + flake check green; commit `fee9cc0` (+ daemon `d878c67`, `53c5373`) |
| 14 | **T24 docs.** FEATURES gained a "Document notes, share links, and saved views" section (3 rows with citations); README features gained the three groups; ROADMAP theme 2 idea struck as shipped; the 13:51 plan's T24 heading annotation flipped from "⛔ Won't implement" to "✅ done at" with the Q3-override rationale; TODO_LIST rows closed (v0.1.1 note appended; pkg.go.dev gap tracked) | `FEATURES.md`, `README.md`, `ROADMAP.md`, TODO_LIST, plan doc §T24 |
| 15 | **CI green on the final state** (run on `6c25597`: success). Working tree clean, `origin/main` in sync at `304d3e1` | `gh run list`; `git status` |

## b) PARTIALLY DONE

| # | Work | Done | Missing |
|---|------|------|---------|
| 1 | **T24 in a release** | Code + tests + docs landed on main, CHANGELOG `[Unreleased]` carries the three groups (entries written by the parallel session — richer than my draft, left as-is to avoid duplication) | Not yet tagged; next release is v0.3.0 (minor — new APIs). A second session's plan doc (`docs/planning/2026-09-13_18-40_pareto-v030-landing-and-hardening.md`) targets exactly this |
| 2 | **pkg.go.dev indexing** | Module itself indexed (v0.1.0 shows, docs render); proxy serves v0.1.1 + v0.2.0; `go get` verified for both | v0.1.1 (12h+) and v0.2.0 still return 404 on pkg.go.dev. **Suspicious pattern:** v0.1.0 (go.mod `go 1.26`) indexed; everything pinning `go 1.27.1` did not — hypothesis: pkg.go.dev's doc builder vs the 1.27 toolchain. Not investigated (out of scope per user instruction); tracked in TODO_LIST |
| 3 | **TODO_LIST hygiene** | Both sessions update it; done items marked with evidence | Two conventions coexist: the parallel session keeps 🟢 `DONE` rows in place, docs-health's rule is delete-on-done. Needs one pass to reconcile (pick delete, archive the DONE rows into CHANGELOG pointers) |
| 4 | **CI coverage** | gosec now scans (Files > 0 verified via job conclusions); `nix flake check` + govulncheck green on every push | No `test-race` job (race only runs locally/hermetically); `pull_request` trigger still only push-path verified; gosec not mirrored into local `.golangci.yml` |
| 5 | **LSP experience** | `.crushrc` pins `GOTOOLCHAIN=auto` (committed last session) | This whole session again ran with hard-failing gopls/golangci-lint-ls diagnostics (`go.mod requires go >= 1.27.1`) — project `.crushrc` loads only at Crush startup and this session predates a restart. CLI gates were the source of truth throughout; next Crush start should clear it (unverified) |
| 6 | **Consumer propagation** (15:39 f-30/f-31) | Nothing this session — SDK-side requirements are documented (Go 1.27+ floor, README Requirements) | bank-sync/InboxClean flakes still on go_1_26 → they cannot build against v0.1.1+; trimmed-fork retirement decision untouched |

## c) NOT STARTED

| # | Work | Why it matters |
|---|------|----------------|
| 1 | **T18 — integration tier scaffold** (`//go:build integration`, env-driven `PAPERLESS_URL`/`PAPERLESS_TOKEN`, upload→poll→reconcile e2e) | The suite only ever speaks to httptest; T24's notes/share-links/saved-views shapes were verified from source, not against a live server |
| 2 | **T22 — ADR 0001: API-version policy beyond v10** | The ROADMAP's confidence theme references it; version-specific serializer drift (seen live in the saved-views source) is the argument for writing it |
| 3 | **T23 — streaming-upload spike note** | io.Reader body + Content-Length constraints; future-proofing evidence before promising the feature |
| 4 | **CI: `test-race` job + `pull_request` trigger verification** | Race gate currently local-only; PR path unexercised |
| 5 | **Examples: `ExampleClient_WaitForTask` + retry example** (15:39 f-27) | Example tests keep README↔code drift honest; the poll/retry semantics deserve a copy-paste-able example |
| 6 | **Consumer bumps** (bank-sync/InboxClean → go_1_27 + `go get` v0.2.0) | Blocks both consumers from building against the current SDK line |
| 7 | **Retire bank-sync's trimmed fork** (15:39 f-31) | The whole point of this SDK existing (AGENTS.md scope note); decision is the user's |
| 8 | **pkg.go.dev root cause / indexing request** (see b-2) | Release page + godoc discoverability |
| 9 | **ProbeCapabilities hook coverage** (15:39 f-34) | Probes route through `doRequestDetail`, so hooks fire there too — unasserted |
| 10 | **gopls verification after a Crush restart** (15:39 f-29) | Confirms the `.crushrc` fix actually clears the stdversion noise |

## d) TOTALLY FUCKED UP

1. **I baked a verdict into a historical doc before the decision owner spoke.** In the pre-release docs batch I annotated the 13:51 plan's T24 heading as "⛔ Won't implement — parked per the D3 default" — and hours later you answered Q3 with "build it next", forcing me to rewrite a historical annotation (the exact thing ANNOTATE mode says not to churn). Root cause: I recorded my *recommendation* as a *resolution*. The fix is mechanical: when a question is still open with the user, the annotation says "open — pending the user's §g answer", never a verdict.
2. **I let "indexing lag" stand in for investigation.** The 15:39 report called pkg.go.dev's missing v0.1.1 "lag". It has now been ~14h with v0.1.1 AND v0.2.0 missing while v0.1.0 (the only release whose go.mod says `go 1.26`) indexed fine. The go-1.27-builder hypothesis was visible in that pattern and I did not chase it — partly because your instruction scoped this session away from unrelated research, partly because I repeated the earlier report's framing instead of re-testing it. Same failure class as the 13:42 lesson ("status reports are point-in-time, re-verify before treating as current truth"), applied to my own earlier claim.
3. **Tool-selection failures burned round trips on large edits.** The T24 test append (~480 lines) failed twice via the edit tool — once on the mod-time guard, once on invalid-JSON arguments — before I switched to a bash heredoc append, which worked first try. Same class as the 15:39 session-close JSON failure: very large edit payloads are the wrong tool; heredoc/python is the right one, and I should have gone there immediately.
4. **The daemon raced my explicit commits twice more** (release-prep CHANGELOG+flake; docs closeout). Both times I checked `git status` immediately before `git add` per the recorded lesson and it STILL won the race (checked → daemon committed → add found nothing → "nothing to commit"). No damage (its commits were correct), but the "commit per task" history goal lost to the daemon again: the v0.2.0 CHANGELOG fold and the T24 bulk live in heuristic "auto-commit N file(s)" commits; only `fee9cc0` (lint fix) and `4fd2427`-style fixes carry real messages.
5. **My first T24 pagination test encoded the wrong server contract** (1-entry pages instead of full 100-entry pages, so the bounded-pagination stop condition fired after page 1). The suite caught it immediately and I fixed the TEST to mirror the real StandardPagination behavior — but the miss shows I wrote the test from imagination before re-checking the sibling tests' full-page pattern.

## e) WHAT WE SHOULD IMPROVE

1. **Test-first worked and must stay the default.** T24's red-green run (12 tests first, compile-fail red, then green) is the anti-pattern antidote for the 15:39 "implementation before verification" failure. Both sessions now have proof it costs nothing.
2. **Verify external wire shapes from source before coding — this session saved two bugs** (`file_version` vs `file_type`; bare-array notes). Make it a standing step for ANY new paperless-ngx endpoint: Sourcegraph/grep the serializer + viewset before writing types.
3. **Release discipline held: CI green on the exact tagged commit BEFORE tagging.** This is the single most valuable process fix from the v0.1.1 incident — keep it as a hard checklist item.
4. **Never pre-write verdicts into historical docs for open questions** (see d-1). Open = unmarked + a pointer to the pending decision.
5. **pkg.go.dev indexing belongs in the release checklist with a root-cause path**, not a "wait and see" (see d-2). Check within the release itself; escalate the go-1.27-builder hypothesis if the pattern repeats.
6. **Two-session coordination:** before editing shared living docs (CHANGELOG, TODO_LIST, ROADMAP), diff against HEAD first — I nearly duplicated the parallel session's CHANGELOG entries, and TODO_LIST now carries two DONE-row conventions. A one-line `git diff --stat` before touching shared docs would have caught both.
7. **Large edits: go straight to bash heredoc / python.** The edit tool's JSON envelope is the bottleneck past a few hundred lines (d-3). Rule of thumb: >~200 lines or >~8KB → heredoc.
8. **The daemon always wins the commit race** — accept heuristic messages for batch work and spend explicit, well-message commits only on single-purpose fixes. The 15:39 "commit per task" ideal is only achievable when committing immediately after each task, not in batches at phase end.
9. **Scanner gates: keep asserting what was scanned, not just exit codes.** gosec green on the new code was confirmed via job conclusions, and the lesson is now in global AGENTS.md — done, keep honoring it.
10. **Re-verify one's own prior claims as part of any status report** (d-2): "lag" aged into a multi-release anomaly. Every "temporary" state named in a previous report gets one fresh check in the next report.

## f) Top 50 things to get done next (brainstorm — HARVEST routes TODO_LIST vs ROADMAP)

| # | Thing | Impact | Effort |
|---|-------|--------|--------|
| 1 | Reconcile the parallel session's 18:40 v0.3.0 plan doc with TODO_LIST; pick the execution source | High | S |
| 2 | Cut v0.3.0 once both sessions' work is folded (T24 + T06–T08 tests + whatever the plan adds) | High | M |
| 3 | Investigate pkg.go.dev non-indexing of go-1.27 modules (builder hypothesis) + request indexing | Med | M |
| 4 | Bump bank-sync flake to go_1_27, then `go get` v0.2.0/v0.3.0 | High (consumer) | M |
| 5 | Bump InboxClean flake to go_1_27, same | High (consumer) | M |
| 6 | Decide + execute bank-sync trimmed-fork retirement (AGENTS.md scope note) | High (consumer) | M |
| 7 | T18: integration scaffold (`//go:build integration`, env-driven, upload→poll→reconcile) | Med | L |
| 8 | Run the integration tier against a live paperless-ngx once to validate T24's source-derived shapes | Med | M |
| 9 | CI: add `test-race` job | Med | S |
| 10 | CI: verify `pull_request` trigger via a scratch PR | Med | S |
| 11 | T22: ADR 0001 — API-version policy beyond v10 (saved-views field drift is the motivating evidence) | Med | S |
| 12 | T23: streaming-upload spike note + decision record | Low | S |
| 13 | `ExampleClient_WaitForTask` + retry example (README copy-paste) | Low | S |
| 14 | ProbeCapabilities hook-coverage assertion (probes go through `doRequestDetail`) | Low | S |
| 15 | Coverage 89.1% → ≥90% | Med | S |
| 16 | gopls/golangci-lint-ls verification after next Crush restart (`.crushrc` GOTOOLCHAIN pin) | Med | S |
| 17 | Dependabot: confirm go-retry updates flow in go_modules | Low | S |
| 18 | Fuzz targets wired into a scheduled CI job | Low | S |
| 19 | gosec rules mirrored into `.golangci.yml` (local lint ≈ CI security gate) | Low | S |
| 20 | CI scanner steps assert Files > 0 (generalize the 15:39 lesson into the workflow) | Med | S |
| 21 | `maxDocumentListPages` → generic `maxListPages` naming (now shared by 5 list methods) | Low | S |
| 22 | Saved-view `rule_type` typed constants (the server's numeric rule catalog) | Med | M |
| 23 | Share-link full-URL helper (`<base>/share/<slug>`) — consumers always need it | Med | S |
| 24 | `ListShareLinks(documentID)` filter variant (`?document=` filterset exists server-side) | Med | S |
| 25 | Saved-view Update/Retrieve methods (currently Create/List/Delete only) — if a consumer needs editing | Low | S |
| 26 | Document note-count/size guard documentation (notes endpoint is unpaginated) | Low | S |
| 27 | SECURITY.md: share-link threat-model paragraph (public unauthenticated URL exposure) | Med | S |
| 28 | Rate-limit interplay doc/test: WaitForTask polls + 429 Retry-After under WithRetry | Med | S |
| 29 | API-version compatibility matrix (which paperless-ngx versions serve which SDK methods) | Med | M |
| 30 | Annotate the 15:39 status report with this session's outcomes (docs-health ANNOTATE pass) | Low | S |
| 31 | TODO_LIST convention fix: delete DONE rows (docs-health rule) vs keep-marked (parallel session's style) — pick one | Low | S |
| 32 | AGENTS.md: add the paperless-ngx wire-fact pointers (notes bare array, share-link slug/file_version, saved-view stable fields) | Med | S |
| 33 | Verify go.sum minimality (no go-retry test deps leaked) | Low | S |
| 34 | `WithContextAny` consistency sweep (error contexts string-only today) | Low | S |
| 35 | Retry `IsRetryable` override hook — park unless a consumer asks | Low | S |
| 36 | Delete stale `/tmp/*.log` gate artifacts from this session | Low | S |
| 37 | README: quick-start recompile-check after this session's README edits (bullets only, but the 13:51 discipline says verify) | Low | S |
| 38 | `ProbeCapabilities`: assert `AcceptAPIVersion` echo value in the existing test | Low | S |
| 39 | InboxClean: replace its local wait-loop with `WaitForTask` | Med (consumer) | M |
| 40 | InboxClean/bank-sync: replace local retry logic with `WithRetry` | Med (consumer) | M |
| 41 | bank-sync: adopt `WaitForTask`+`WithRetry` interplay semantics (poll requests DO flow through retry — now pinned by `TestWaitForTaskWithRetryRecoversFromTransientPollFailure`) | Med (consumer) | S |
| 42 | notes/share-links/saved-views: `WithContext("document", …)` consistency check on error wrapping | Low | S |
| 43 | CHANGELOG at v0.3.0: fold `[Unreleased]` + footer link refs | Low | S |
| 44 | ROADMAP: retire the "Confidence" theme's integration bullet when T18 lands | Low | S |
| 45 | Consider `ListDocumentNotes` ordering guarantee in godoc (`-created`, verified from source) | Low | S |
| 46 | AddDocumentNote: research/record server-side note length limit | Low | S |
| 47 | Race-test the observability hooks (parallel callers appending snapshots) | Low | S |
| 48 | ShareLinkBundle endpoints — main-branch-only today; park until a stable release serves them | Low | S |
| 49 | docs-health VERIFY pass across the whole two-session batch (claims vs code) | Med | M |
| 50 | Next-session kickoff standing rule: bare `nix flake check` FIRST (gate-first audit, the 13:42 lesson) | Med | S |

## g) Three questions I cannot answer myself

1. **pkg.go.dev:** both v0.1.1 and v0.2.0 remain unindexed ~14h/~5h after push while v0.1.0 (go.mod `go 1.26`) is indexed and the module proxy serves everything. Is this acceptable to ride out (proxy + `go get` both verified working), or do you want me to actively chase it — file a pkg.go.dev support issue / test the "go.mod requires 1.27 breaks the doc builder" hypothesis by shipping a `go 1.26`-floor patch release? (A `go 1.26` floor would need json/v2 behind GOEXPERIMENT again — that tradeoff is yours to make, not mine.)
2. **T24 surface scope:** I shipped List/Create/Delete for all three groups — deliberately NOT saved-view Update/Retrieve, and NOT the document-scoped share-links listing (`GET /api/documents/{id}/share_links/`). Do bank-sync or InboxClean actually need those, or is the slice right for v0.3.0?
3. **Two sessions, two plans:** the parallel session dropped `docs/planning/2026-09-13_18-40_pareto-v030-landing-and-hardening.md` (v0.3.0 landing + hardening) while my lane finished the 13:51 plan end-to-end. For the next cycle — execute from their plan doc, extend this session's thread, or do you want a merged/reconciled plan first (and which session owns it)?
