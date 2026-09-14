# Status Report — Docs-Health Audit: Verify + Harvest + Annotate + Archive over the 2026-0* backlog

- **Timestamp:** 2026-09-14 07:23 CEST
- **Repo:** `go-paperless` @ `d143ecf` (main; the auto-commit daemon swept the session's changes into heuristic commits, as designed)
- **Session scope:** full docs-health run requested as "view ALL `**/2026-0*` files" — AUDIT mode (VERIFY + HARVEST + BUILD-fix + ANNOTATE + ARCHIVE) over the six living docs and the entire timestamped `docs/` backlog.
- **Format note:** the status-report skill's canonical format is a styled HTML dashboard; you explicitly requested `.md`, so this is Markdown — same honor-the-user override as every prior report in this series.

---

## Headline numbers

| Metric | Value |
| --- | --- |
| `**/2026-0*` files viewed | 19 of 19 (6 status `.md`, 2 planning `.md`, 6 HTML reviews, 2 D2, 2 SVG) |
| Historical items resolved inline | ~450 rows/items across 8 files (strikethrough + `done at`/`Won't implement`/routed markers) |
| Living docs updated | 6 of 6 (TODO_LIST, FEATURES, ROADMAP, CHANGELOG, AGENTS, README) |
| TODO_LIST rebuilt | 9 DONE rows deleted, 1 ghost row killed, 1 overclaim corrected, 14 verified rows added → 3 High / 11 Med / 11 Low, 100% open |
| Code/config fixes | 1 (`.golangci.yml` `run.go: 1.26.7` → `1.27`) |
| Archived | 1 file → `docs/status/archived/` (the fully-resolved 13:42 audit) |
| Claim verifications against code | 20+ (grep/fetch/live), 3 overclaims found |
| Gates | `nix run .#check` "all checks passed!" · `.#vet` exit 0 · `nix fmt` 0 changed · coverage 88.6% re-verified first-hand |
| Final health scores | Accuracy 10/10 · Fitness 10/10 (post-fix; prior baseline 9.5/10 from the 13:42 audit) |

---

## a) FULLY DONE

1. **All 19 `**/2026-0*` files viewed** — the 6 status reports and 2 plans in full; the 6 HTML review reports via title/header extraction; the 2 D2 sources and 2 SVGs inventoried. Nothing in the glob was skipped.
2. **Skill + reference load before acting:** docs-health SKILL.md plus doc-ownership, harvest-guide, verify-checklist, health-report-format, resolving-items, annotation-placement — and both annotator scripts read before use.
3. **VERIFY pass with 20+ code checks.** Symbol line drift (client.go is now 2,371 lines — nearly every `client.go:NNN` citation in FEATURES was stale); integration env names (`PAPERLESS_INTEGRATION_URL/_TOKEN` confirmed); example inventory (no `ExampleClient_WaitForTask`, 5 `testableexamples` suppressions); `.crushrc` pin syntax; `.golangci.yml` `run.go: 1.26.7`; `ci.yml` gosec strict but no Files>0 assert; flake `version = "0.3.0"` + `packages.default`; dependabot gomod + github-actions groups; 9 test symbols grepped; **cap-parity tests exist for only 3 of 5 listings** (Metas + StoragePaths missing — the 05:24 report's "all five listings" is an overclaim); **no `defaultTransport`/`WithHTTPClient(nil)`/custom_fields-PATCH tests** (the T06–T08 "DONE" claim overcovered); **pkg.go.dev verified LIVE — all four versions (v0.1.0…v0.3.0) indexed**, killing the oldest open TODO row.
4. **TODO_LIST.md rebuilt.** All 9 🟢 rows deleted (done work lives in CHANGELOG — the delete-on-done convention finally enforced after two reports flagged the split convention); the ghost pkg.go.dev row removed; the T06–T08 row replaced with a precise tests-tail row; 14 open rows harvested from the three most recent reports, every one verified against code first. Result: 3 High / 11 Med / 11 Low, zero historical content.
5. **FEATURES.md de-roted.** Every `client.go:NNN` / `client_test.go:NNN` citation replaced with symbol/test-name anchors — the durable fix the 18:37 and 05:24 sessions both recommended. All ~60 cited symbols/tests verified to exist before writing.
6. **ROADMAP.md corrections.** Stale "the suite still only speaks to synthetic httptest servers" fixed; the integration-tier raw idea struck as scaffold-shipped (e2e remainder explicit); four raw ideas routed in (share-link conveniences, saved-view Update/Retrieve, probe caching).
7. **CHANGELOG.md completion.** `[Unreleased]` gained the post-v0.3.0 flake work (`packages.default`/`packages.go-paperless`, version-attribute alignment) and the `.golangci.yml` language-version fix; the missing `[0.3.0]` footer link ref added.
8. **AGENTS.md refreshed.** Flake line now mentions `packages.default`; Docs section points at `docs/ERROR_CODES.md`, `docs/adr/`, and the annotated/archived status series; new gotcha: host `erraudit`/`go` run 1.26.7 and hard-fail — run inside `nix develop`.
9. **README.md** development block gained `nix build` (packages.default has worked since `39cb43b` but was undocumented).
10. **`.golangci.yml` fixed:** `run.go` aligned `1.26.7` → `"1.27"` (closes the 04:13 report's D4 drift trap); `nix run .#lint` = 0 issues after.
11. **ANNOTATE over all 8 historical `.md` files** — every numbered item got a verdict: `done at <hash>` (hashes only from annotated plans/commit messages I read), `Won't implement — reason`, `done — evidence`, or routed-to-TODO_LIST/ROADMAP markers. Sections a/d/e (work logs, confessions, lessons) left alone per the placement rules; open items left unmarked so absence-of-marker stays the open signal. One in-place factual correction: the 05:24 report's "cap-parity tests for all five listings" now reads corrected-with-pointer-to-TODO_LIST.
12. **Archive executed:** `docs/status/2026-09-13_13-42_docs-health-audit.md` → `docs/status/archived/` via `git mv`, with a Resolution appendix. It is the only file whose every thread terminated (its b/c/d shipped; its 50-row brainstorm fully dispatched through the 13:51 plan, which is itself annotated).
13. **Gates green at session end:** `nix run .#check` → "all checks passed!" (build/test/lint/fmt + the race check); `nix run .#vet` exit 0; `nix fmt` 0 changed; coverage re-run first-hand: **88.6%** (the TODO row now cites a number I measured, not transcribed).
14. **Health report printed inline** with both scores and visible math; no report file written for the audit itself (per docs-health: audits are living diagnoses, not snapshots).

## b) PARTIALLY DONE

1. **~35 f-table leftovers in old reports are neither routed nor Won't-marked** (15:39 rows 40/41/47; 03:29 rows 26/29/34/39–42/45/46; 05:24 ~23 brainstorm rows: rate-limit interplay doc, ADR 0002 pagination, decode-code normalization test, multipart property test, benchmarks, PR-template consumer checkbox…). Deliberate — they are idea-grade and I chose not to bloat TODO_LIST — but "unmarked in an old report" is a resting place, not a decision. See g-1.
2. **HTML review reports viewed, not annotated.** INDEX.md carries the point-in-time warning, so per-finding verdicts felt like noise; the alternative (per-finding inline resolution) would be genuinely useful but is a session of its own. See g-3.
3. **Two annotation styles now coexist:** strikethrough rows in the status reports vs the 13:51 plan's pre-existing `✅ done at` heading style (which I extended, not converted). Consistent per file, inconsistent across the series. See g-2.
4. **The lint-fix mechanism audit (04:13 B2/C5)** was routed to TODO_LIST, not performed — out of docs scope, but it means "37 findings cleared" still rests on green-gate evidence, not read hunks.
5. **DONE-evidence homes are uneven:** code-side completions live in CHANGELOG as the convention demands, but the consumer-bump row's evidence (bank-sync/InboxClean bumped, unpushed) lives only in the 05:24 report — CHANGELOG is silent on consumer-repo work by design.

## c) NOT STARTED

1. `art-dupl` re-run over the enlarged test file (last clean scan predates the hardening tail's new tests) — verified only that the marker convention is intact.
2. erraudit re-verification **this session**: my bare `erraudit lint ./...` hard-failed on the host's Go 1.26.7 (now documented in AGENTS.md); the "zero findings" claim in TODO_LIST still cites the 05:24 session's dev-shell run, not a fresh one.
3. A status-series index (`docs/status/` has no INDEX.md; `docs/reviews/` does) — the archive move makes a pointer to `archived/` more valuable now.
4. Any consumer-repo work: pushes of bank-sync/InboxClean remain blocked on your go (unchanged from 05:24).
5. `client.go` split watch: 2,371 lines, under the ~3k trigger — correctly a watch item, untouched.
6. gitleaks/trivy adoption — closed as Won't in this pass's annotations; nothing started.

## d) TOTALLY FUCKED UP

1. **I mangled a backtick in ROADMAP.md inside a 7-edit multiedit** — the first edit's new_string dropped the closing backtick of `` `DefaultTaskPollInterval` `` while trying to restructure the citation on the next line. The final citation sweep (`rg 'client.go:[0-9]+'`) plus a view caught it within minutes and it is fixed, but a formatting-sensitive edit buried in a batch is exactly the failure mode the exact-match rules warn about.
2. **Wrong annotator for the 18:37 f-list:** I ran the table-row script against a prose numbered list (found 0 matches). I had read that file's f-section earlier and it was visibly prose — the dry-run discipline saved the file from corruption, but the round trip was avoidable by checking the list shape before picking the tool.
3. **The edit tool's mtime guard rejected my Fine-Plan note insertion** because my own annotation script had touched the file after my last `view` — self-inflicted staleness. I should have ordered the work: all scripted passes first, hand-edits last (the re-run worked first try).
4. **I almost inherited the T06–T08 overclaim wholesale.** The old TODO row said DONE and my first rebuild plan deleted it without inspection; only the custom_fields grep (run for unrelated citation verification) exposed that `TestUpdateDocumentSendsPatchBody` never asserts custom_fields or the empty-`TagIDs` omitempty. The catch was luck-adjacent — theVERIFY grep happened to cover it. A deliberate "diff every DONE row's evidence against its claimed tests" pass would have found all four gaps (transport constants, nil client, custom_fields PATCH, error-code snapshot) systematically instead of two-by-two.
5. **Coverage claim provenance:** the TODO row originally cited "88.6%" transcribed from the 05:24 report. I re-ran coverage and it matched — but shipping the row before re-measuring would have been exactly the "status reports are point-in-time" violation this repo keeps re-learning. The fix landed this session; the near-miss is on the record.

## e) WHAT WE SHOULD IMPROVE

1. **Verify every cited symbol/test exists before writing the citation** — the grep-first pass is why FEATURES' anchors are all real. Make it mechanical (a checklist column, not a habit).
2. **Never batch formatting-sensitive markdown edits.** One logical edit per call when backticks/pipes are in play; re-view after any multiedit that restructures line content.
3. **Check list shape (table vs prose) before selecting an annotator**, and dry-run row 1 of the actual target section — not a nearby section.
4. **Scripted passes first, hand-edits last** on the same file — the mtime guard is doing its job; don't fight it, sequence around it.
5. **Diff DONE-row evidence against the claimed tests mechanically** (`rg "^func Test<name>"` per row) — the T06–T08 overclaim class survived three sessions because verification was ad hoc.
6. **pkg.go.dev indexing belongs in the release checklist as a check, not an assumption** — this session it had self-healed, but only because someone fetched the page. The post-release-ritual row in TODO_LIST already covers it; keep it.
7. **Give DONE evidence a stated home per category** (this repo → CHANGELOG; consumer repos → the status report that executed it) in the TODO_LIST legend, so deletions never orphan evidence.
8. **Decide the resting place for idea-grade leftovers at annotation time** — either route to ROADMAP or mark `Won't implement`. ~35 items across four files currently sit in neither state (g-1).

## f) Up to 50 things to get done next

Impact-ordered. Items 1–24 are the verified TODO_LIST (do not duplicate here in full — it is the source of truth); 25–50 are fresh from this session or routed-adjacent.

| # | Thing | Impact | Effort |
| --- | --- | --- | --- |
| 1 | Cut v0.4.0 with the hardening tail (race-in-check, integration scaffold, error catalog, ADR 0001, typed enum, packages.default, lint alignment) — TODO_LIST High | High | 30m |
| 2 | Push bank-sync + InboxClean consumer bumps (blocked on your go) — TODO_LIST High | High | 10m |
| 3 | bank-sync trimmed-fork retirement decision — TODO_LIST High, BLOCKED | High | 2h |
| 4 | Integration scaffold e2e: upload → `WaitForTask` → checksums reconcile — TODO_LIST Med | Med | 2h |
| 5 | Verify v0.2.0/v0.3.0 tags build clean in-sandbox (settles the red-gate-at-tag question) — TODO_LIST Med | Med | 30m |
| 6 | Audit the 8 non-mechanical lint fixes — TODO_LIST Med | Med | 1h |
| 7 | gosec CI step: assert Files > 0 — TODO_LIST Med | Med | 15m |
| 8 | erraudit CI gate decision — TODO_LIST Med | Med | 30m |
| 9 | Coverage recovery (88.6% → top-3 uncovered blocks named + closed) — TODO_LIST Med | Med | 1h |
| 10 | Cap-parity tests for `ListDocumentMetas` + `ListStoragePaths` — TODO_LIST Med (found this session) | Med | 30m |
| 11 | Wire ERROR_CODES.md into CI (catalog-drift grep test) — TODO_LIST Med | Med | 30m |
| 12 | Fuzz `savedViewPayload`/`shareLinkPayload` — TODO_LIST Med | Med | 30m |
| 13 | SECURITY.md share-link threat model — TODO_LIST Med | Med | 15m |
| 14 | Typed enums for `FileVersion`/expiry + `rule_type` — TODO_LIST Med | Med | 1h |
| 15 | Flake version git-derived or release-commit-bumped — TODO_LIST Med | Med | 45m |
| 16 | T23 streaming-upload spike note — TODO_LIST Low | Low | 1h |
| 17 | Examples batch (WaitForTask + retry; `// Output:` for the two runnable examples) — TODO_LIST Low | Low | 45m |
| 18 | `.crushrc` LSP verification after a real Crush restart — TODO_LIST Low | Low | 10m |
| 19 | `docs/adr/` index + template — TODO_LIST Low | Low | 20m |
| 20 | `t.Cleanup`-vs-`defer` call, recorded once in AGENTS.md — TODO_LIST Low | Low | 15m |
| 21 | Integration wiring: `go vet -tags integration` gate + opt-in `checks.integration` — TODO_LIST Low | Low | 30m |
| 22 | Verb-table drift test — TODO_LIST Low | Low | 20m |
| 23 | README quick-start snippets for the three v0.3.0 families — TODO_LIST Low | Low | 30m |
| 24 | Tests tail: transport constants, `WithHTTPClient(nil)`, custom_fields PATCH — TODO_LIST Low | Low | 30m |
| 25 | Route-or-close the ~35 unmarked f-table leftovers (see g-1): rate-limit interplay doc, ADR 0002 pagination policy, `decode_*` normalization test, multipart property test, pagination benchmarks, PR-template consumer checkbox, probe AcceptAPIVersion integration assert | Low-Med | 30m |
| 26 | Re-run `erraudit` inside the dev shell so the TODO row's "zero findings" is fresh, not inherited | Low | 5m |
| 27 | Re-run `art-dupl -t 5` over the enlarged test file (last clean scan predates the tail's tests) | Low | 10m |
| 28 | Add a `docs/status/INDEX.md` (or fold into reviews INDEX) covering the six status reports + `archived/` | Low | 15m |
| 29 | Add "DONE evidence home" note to the TODO_LIST legend (CHANGELOG vs status report) | Low | 5m |
| 30 | Annotate the 6 HTML review reports per-finding, or formalize LEAVE-ALONE in INDEX.md (g-3) | Low | 1h+ |
| 31 | Standardize annotation style across plans vs reports (g-2) | Low | 30m |
| 32 | Post-release ritual as one command — TODO_LIST Low | Low | 1h |
| 33 | INDEX.md link-check in CI — declined this session (low churn); revisit if the report series grows | Low | 20m |
| 34 | Release-notes-verbatim check: diff the v0.3.0 GitHub Release body against CHANGELOG once | Low | 10m |
| 35 | `decode_*` code normalization: dedicated test for `fetchAllPages` space→underscore codes | Low | 15m |
| 36 | `BenchmarkListDocumentChecksums` next to `BenchmarkUpload` | Low | 20m |
| 37 | ROADMAP: capability-probe caching already routed; consider probe-cost measurement first so the idea has data | Low | 20m |
| 38 | go-error-family upgrade check (v0.10.0 pinned; verify at next ecosystem sweep) | Low | 10m |
| 39 | PR template: "consumer impact" checkbox (bank-sync/InboxClean pin this module) | Low | 10m |
| 40 | Note-count/size guard documentation for the unpaginated notes endpoint | Low | 15m |
| 41 | `WithRetry` + `WaitForTask` deadline math: one godoc paragraph (test already pins behavior) | Low | 10m |
| 42 | Dependabot actions-group triage when the first PR lands (event-driven; on notice) | Low | 10m |
| 43 | Consumer repos: nolintlint unknown-directive warnings (deferloop/erraudit/legacyerrors) — bank-sync-owned | Low | 15m |
| 44 | Consumer repos: dependabot gomod pin for go-paperless bumps — bank-sync/InboxClean-owned | Low | 15m |
| 45 | `checks.integration` demo-server workflow_dispatch (after item 21) | Low | 30m |
| 46 | Multi-system flake decision (restrict `systems` vs portability work) — needs your call first | Med (decision) | — |
| 47 | 13:51 + 18:40 plans: once C08/C14/C25 resolve, re-run docs-health and archive them | Low | 20m |
| 48 | Quarterly ROADMAP prune (standing hygiene; last full pass was this session) | Low | 15m |
| 49 | `client.go` split at ~3k lines — watch item, currently 2,371 | Low | — |
| 50 | Session-hygiene standing rule held: gate first, cite only verified hashes, dry-run scripts — keep all three | — | — |

**HARVEST note:** 1–24 already live in TODO_LIST (this session's harvest); 25–35 are the fresh residue of this session. The rest is ROADMAP-grade or event-driven.

## g) Three questions I cannot answer myself

1. **Where should idea-grade leftovers rest?** I left ~35 brainstorm rows unmarked across the annotated f-tables (rate-limit interplay doc, ADR 0002 pagination policy, decode-code test, multipart property test, pagination benchmarks, …). Options: (a) route them all into ROADMAP now so TODO_LIST stays lean but nothing is entombed, (b) mark them `Won't implement` in the reports (harsh for genuine ideas), or (c) accept "unmarked in an old report" as the idea archive. Your call decides whether docs/status remains a quasi-backlog.
2. **Standardize the annotation style?** Status reports now use strikethrough rows; the 13:51 plan uses `✅ done at` heading markers (its pre-existing convention, which I extended). One house style would make the series scannable, but retrofitting the plans means churning historical files a second time. Which do you value more: consistency or minimal churn?
3. **The six HTML review reports:** per-finding inline annotation (real value: each finding's disposition — fixed / nolint'd / Won't — becomes visible next to the claim), or formalize LEAVE-ALONE in INDEX.md and spend the effort on code? An hour-plus either way.

---

*Point-in-time snapshot — goes stale by design. Section (f) is the HARVEST input; items 1–24 were already routed into `TODO_LIST.md` this session. Annotate non-destructively via docs-health ANNOTATE mode; do not rewrite.*
