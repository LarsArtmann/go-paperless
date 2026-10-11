# Status Report — Full-Code-Review Session (2026-10-11 06:36 CEST)

Scope: **this session only** (branching-flow verification → full-code-review skill run →
7 fixes → gates → HTML report → TODO_LIST/INDEX harvest). Prior backlog is listed
where it intersects, marked as pre-existing. Per instruction: `.md` format at an
explicit path (override of the HTML-canonical skill default, honored as demanded),
no research beyond this session's run.

---

## a) FULLY DONE

| #  | Work                                                                                                                                                                                                          | Evidence                                                                                       |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| 1  | branching-flow re-verified at **0 active findings** after all changes                                                                                                                                         | `branching-flow all . --format finding` → total 0, exit 0 (run twice: pre-review and post-fix) |
| 2  | Pareto plan produced and committed                                                                                                                                                                            | `docs/planning/2026-10-11_06-00_full-code-review-plan.html` (D2 graph, tiered task table)      |
| 3  | Every first-party file visited (37 files, 12,926 lines): client.go, ids.go, all 10 paperlesstest sources, all 15 test files, all configs + flake.nix                                                          | Session transcript; per-file reading, no sampling                                              |
| 4  | Fix: drift-guard hole — `findByName` added to the error-code drift test's tracked helpers (`docs_drift_test.go`); its dynamic `paperless.decode_<kind>s` codes are now enforced against `docs/ERROR_CODES.md` | Suite green; catalog rows already existed (human-added), now machine-checked                   |
| 5  | Fix: duplicated 14-line consumption-store block in the fake (`paperlesstest/tasks.go`) extracted into `existingDuplicateLocked` + `storeConsumedLocked`; both call sites reduced to 4-line bodies             | Round-trip suite green, behavior identical                                                     |
| 6  | Fix: `FindStoragePath` now wraps lookup failures with the entity name (parity with `FindCustomField`)                                                                                                         | `client.go:1104-1121`                                                                          |
| 7  | Fix: `findByName` parameter order normalized to `(client, ctx, …)` across all 3 call sites                                                                                                                    | `client.go`                                                                                    |
| 8  | Fix: `pageBounds` overflow clamp — absurd `page` values can no longer wrap negative and panic inside the fake                                                                                                 | `paperlesstest/documents.go:243`                                                               |
| 9  | Fix: `RetryPolicy` field docs now state "Zero (or negative) selects …" (was "Zero selects", implementation clamps `<= 0`)                                                                                     | `client.go:195-203`                                                                            |
| 10 | Fix: stale `.buildflow.yml` branching-flow comment ("~112 findings") updated to the 2026-10-11 triage state (0 active)                                                                                        | `.buildflow.yml`                                                                               |
| 11 | Review report written (episode 2, cross-referenced with the 2026-09-13 episode)                                                                                                                               | `docs/reviews/2026-10-11_06-10_full-code-review.html`                                          |
| 12 | Series index + TODO_LIST updated; 3 v2-scope observations folded into the existing v2 row (no tombstones)                                                                                                     | `docs/reviews/INDEX.md`, `TODO_LIST.md`                                                        |
| 13 | Full gate green at final state: `nix flake check` (exit 0), `.#test`, `.#test-race`, `.#lint` (0 issues), `nix fmt` clean, tree clean                                                                         | Session transcript                                                                             |

## b) PARTIALLY DONE

| # | Work                                                                                 | Done                                             | Missing                                                                                                                                                     |
| - | ------------------------------------------------------------------------------------ | ------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Verify-suppression discipline for the new `//nolint:erraudit` (FindStoragePath wrap) | Annotation written, lint green                   | **erraudit itself was never run this session** (`nix develop -c erraudit ./… --type-aware --enforce-go-error-family --disable-extensions`)                  |
| 2 | "Test your fix fails when it should"                                                 | Drift-test change verified green                 | No **negative test** proving the guard now fires for an un-cataloged `findByName` kind                                                                      |
| 3 | Regression coverage for my own fixes                                                 | Existing suites green                            | `pageBounds` clamp has **no dedicated regression test**; nothing exercises page≈1e17                                                                        |
| 4 | Local tool breadth                                                                   | check/test/race/lint/fmt/branching-flow          | `.#fuzz` smoke and `.#coverage` not run (fuzz untouched parse paths, coverage baseline unmeasured)                                                          |
| 5 | CHANGELOG discipline                                                                 | Fixes documented in review report + TODO_LIST    | **No `CHANGELOG.md` entry** for the 7 fixes (repo convention: completed work lives there; next release commit must pick them up)                            |
| 6 | Commit hygiene                                                                       | All work committed (tree clean)                  | Fix-batch commit message lost to the auto-commit daemon (see d-1); history says "auto-commit 3 changed file(s)" instead of the prepared descriptive message |
| 7 | docs-health HARVEST loop                                                             | The 3 concrete v2 items harvested into TODO_LIST | This report's section (f) brainstorm is **not** yet harvested (by design: brainstorm ≠ commitment list)                                                     |

## c) NOT STARTED (pre-existing, untouched this session)

| # | Item                                                                                                                                                                        | Where tracked                                                       |
| - | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| 1 | Migrate bank-sync + InboxClean to go-paperless v0.5.0 (branded IDs)                                                                                                         | TODO_LIST High Impact                                               |
| 2 | v2 typed-API migration (brand `New(baseURL, token)`, unify public structs with payloads, drop `disabled_rules` + nolints) — extended this session with 3 concrete sub-items | TODO_LIST                                                           |
| 3 | Consumer-repo CI billing blockage                                                                                                                                           | TODO_LIST (BLOCKED, owner action)                                   |
| 4 | erraudit CI job                                                                                                                                                             | Blocked on module-proxy publication (AGENTS.md decision 2026-09-16) |
| 5 | govalid-generate unskip                                                                                                                                                     | Blocked on machine nixpkgs govalid rebuild (AGENTS.md)              |
| 6 | Next release cut (flake version + CHANGELOG heading + tag three-place agreement)                                                                                            | Not started; now carries the 7 fixes from this session              |
| 7 | Episode-3 review after the v2 migration                                                                                                                                     | ROADMAP/review series                                               |

## d) TOTALLY FUCKED UP

**Nothing functional.** No red gates, no broken tests, no data loss, no reverted
work, public API byte-identical. Two hygiene wounds, honestly:

1. **Lost commit message (process failure).** I ran the gate, then composed the
   commit message — the auto-commit daemon committed the tree seconds earlier, so
   the prepared "refactor: dedupe fake consumption path…" message was never used
   and the changes are buried under a heuristic auto-commit. The work is safe and
   documented in the review report, but `git log` no longer explains _why_ at that
   commit. Lesson: commit immediately after the gate, before composing prose.
2. **Careless edit construction (caught, zero damage).** My first multiedit on
   client.go invented a nonexistent `ctxReq` call; gopls flagged it instantly and
   I fixed it to the original `c.doRequest(...)` one step later. No harm done, but
   it should not have been constructible.

## e) WHAT WE SHOULD IMPROVE (self-review: forgotten / better / improve)

1. **Verify suppressions with the tool they target.** I wrote a new
   `//nolint:erraudit` and a new drift-test registry entry but ran neither
   erraudit nor a negative test. AGENTS.md explicitly warns that suppression
   anchoring is fragile under `nix fmt`. This is the single biggest gap.
2. **Mutation-verify guards.** A green suite after editing a _guard_ proves
   nothing; I should have temporarily added an un-cataloged kind and watched the
   guard go red, then revert.
3. **Regression-test my own fixes.** The pageBounds clamp and the dedup deserve
   pinning tests; I relied on "existing suites stayed green" which is weaker.
4. **Commit-then-prose.** Daemon beats slow commit messages; run `git add && git
   commit` immediately post-gate.
5. **Run the cheap extra gates when touching the fake or test infra:** fuzz smoke
   (`nix run .#fuzz -- 30s`) and coverage comparison (~88.6% baseline) cost
   minutes and would have caught regressions the unit suites can't.
6. **CHANGELOG as the work journal.** Fixes belong under an Unreleased heading at
   fix time, not reconstructed at release time.
7. **File-size guideline drift is real:** client.go (2,671) and client_test.go
   (4,098) are ~8–12× the 350-line guideline. I noted it and deferred; a split by
   domain (upload/tasks/documents/named/notes/sharelinks/savedviews/transport) is
   mechanical and package-internal.
8. **Minor split brain, newly named:** three "created-string" parsers exist with
   subtly different layout sets (`parseDocumentCreated`, `parseServedCreated`,
   `parseUploadDate`). Fake-vs-SDK duplication is contractually deliberate, but
   the _fake-side_ duplication between parseServedCreated and parseUploadDate
   could be one function.
9. **Did I lie?** No. Every "fixed" claim is backed by a green gate run in this
   session; every "kept" item has its reason written down.
10. **Ghost systems?** None found this session: every route the fake serves is
    round-tripped (`TestFakeCoversEveryClientRoute`), every error code is
    catalog-checked, every doc example compiles as a test.
11. **Scope creep trap?** Held. 7 fixes, 0 speculative rewrites, public API
    untouched; the three tempting API changes (Duplicate() accessor, nil-vs-empty
    FilterRules, FileVersion default) were routed to the existing v2 row instead
    of "fixed" against the compatibility contract.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

Sorted by impact; items 1–12 are concrete follow-ups from this session, the rest
are backlog/brainstorm (ROADMAP fuel, not commitments).

| #  | Task                                                                                                         | Impact | Effort       |
| -- | ------------------------------------------------------------------------------------------------------------ | ------ | ------------ |
| 1  | Run erraudit in `nix develop` to verify the new nolint anchor survives                                       | High   | 5 min        |
| 2  | Negative-verify the drift guard: temp unknown `findByName` kind → expect red → revert                        | High   | 10 min       |
| 3  | Add regression test for the `pageBounds` overflow clamp                                                      | High   | 15 min       |
| 4  | Add CHANGELOG entry (Unreleased) for the 7 review fixes                                                      | High   | 10 min       |
| 5  | Fuzz smoke `nix run .#fuzz -- 30s` after this session's edits                                                | Medium | 5 min        |
| 6  | `nix run .#coverage` and compare vs ~88.6% baseline                                                          | Medium | 5 min        |
| 7  | Decide + pin the nil-vs-empty `FilterRules` contract with a test                                             | Medium | 20 min       |
| 8  | Consolidate `parseServedCreated`/`parseUploadDate` in the fake (one parser)                                  | Medium | 15 min       |
| 9  | Split client.go by domain (same package, mechanical)                                                         | Medium | 1–2 h        |
| 10 | Split client_test.go the same way                                                                            | Medium | 1–2 h        |
| 11 | `records.Requests()`: deep-clone Header (cheap consistency with Documents/Patches)                           | Low    | 10 min       |
| 12 | Fold `newTestClientWithToken` into `newTestClient` (variadic token)                                          | Low    | 10 min       |
| 13 | Release v0.5.1 with the review fixes (version/CHANGELOG/tag + release-verify)                                | High   | 30 min       |
| 14 | Migrate bank-sync + InboxClean to v0.5.0 (TODO_LIST row)                                                     | High   | small        |
| 15 | Give `tbStub` a `Fatalf` that panics so `NewServer(&tbStub{})` misuse fails loudly                           | Low    | 10 min       |
| 16 | Verify paperless server semantics for zero-byte/empty-filename uploads; then decide Upload pre-validation    | Medium | research     |
| 17 | Consider attaching response body context to `auth_failed` (needs consumer/test agreement)                    | Low    | 20 min       |
| 18 | Remove the dead `return` after `tb.Fatal` in `RequireAuthorized`                                             | Low    | 2 min        |
| 19 | Add a test asserting the fake's fall-through failure mode for non-numeric entity IDs                         | Low    | 15 min       |
| 20 | Consider duplicate-name 400 in `handleNamedCreate` (match DRF uniqueness) — decide lenient-vs-honest         | Low    | 20 min       |
| 21 | v2: brand `New(baseURL, token)` / `EnsureStoragePath(name, path)` params                                     | Medium | medium       |
| 22 | v2: unify public structs with wire payloads (JSON-tag decision)                                              | Medium | medium       |
| 23 | v2: public typed `MatchingAlgorithm` enum                                                                    | Low    | small        |
| 24 | v2: typed `TaskOutcome.Duplicate()` accessor (replace double-bool)                                           | Low    | small        |
| 25 | v2: typed ShareLink FileVersion default (replace absent-string)                                              | Low    | small        |
| 26 | v2: drop `.branching-flow.yml` disabled_rules + nolints when the tool ships line suppression                 | Low    | small        |
| 27 | Re-run branching-flow triage after v2 (AGENTS.md revisit note)                                               | Low    | small        |
| 28 | erraudit CI job once the module proxy publishes it (re-verify 404)                                           | Medium | small        |
| 29 | Unskip govalid-generate when machine govalid rebuilds on Go ≥ 1.27                                           | Low    | small        |
| 30 | Unify context constructors in paperlesstest tests (`t.Context()`)                                            | Low    | 30 min       |
| 31 | Create `docs/DOMAIN_LANGUAGE.md` (named entities, checksum shapes, consumption task, self-heal, dup refusal) | Medium | 1 h          |
| 32 | Spot-verify FEATURES.md claims against post-ADR-0004 code                                                    | Low    | 30 min       |
| 33 | AGENTS.md: name `findByName` explicitly in the doc-drift test note                                           | Low    | 2 min        |
| 34 | Verify govulncheck pin v1.8.0 is still current; bump if newer exists                                         | Low    | 5 min        |
| 35 | Consider flake.lock auto-update workflow (weekly nixpkgs bumps)                                              | Low    | 30 min       |
| 36 | Periodic benchmark run (Upload/ListChecksums) to catch perf regressions                                      | Low    | 30 min setup |
| 37 | Add runnable examples for UpdateDocument / ListDocumentMetas / EnsureStoragePath                             | Low    | 30 min       |
| 38 | Table-driven upload-metadata writer (optional; per-field codes make this debatable)                          | Low    | 40 min       |
| 39 | Saved-view fake: document lenient rule_type handling as contract (unknown IDs pass through)                  | Low    | 10 min       |
| 40 | Fuzz seeds: add `Retry-After` overflow edge (seconds > MaxInt64/time.Second)                                 | Low    | 5 min        |
| 41 | Harness: assert `RequireAuthorized` message content in its self-test                                         | Low    | 10 min       |
| 42 | `writeNotes`: share `defaultNoteUser()` per response instead of per note (micro-alloc)                       | Low    | 5 min        |
| 43 | Consider `ChecksumShape` as a typed enum with String() (public, v2-adjacent)                                 | Low    | small        |
| 44 | CI: job to run `branching-flow all .` on PRs (currently local + BuildFlow only)                              | Medium | 30 min       |
| 45 | CI: cache golangci-lint builds in the lint check (build time)                                                | Low    | 20 min       |
| 46 | Docs: link the review series from README (For Contributors section)                                          | Low    | 10 min       |
| 47 | Roadmap: capture items 15–46 overflow into ROADMAP.md via docs-health HARVEST                                | Low    | 20 min       |
| 48 | Schedule episode-3 full review after the v2 migration lands                                                  | Low    | —            |
| 49 | Re-verify TODO_LIST/ROADMAP freshness via docs-health VERIFY (post-session)                                  | Low    | 20 min       |
| 50 | Re-run lychee link check over the two new HTML reports                                                       | Low    | 5 min        |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Release cadence:** cut **v0.5.1 now** with the 7 review fixes, or batch them
   into the next consumer-migration release? (Three-place version agreement means
   the fixes sit unreleased until you decide.)
2. **Live-server fact I cannot verify:** does your real Paperless-ngx accept
   zero-byte or empty-filename uploads? The answer decides whether client-side
   `Upload` pre-validation (a Rejection before any HTTP) is safe or would block a
   valid use. No integration credentials were available this session.
3. **v2 API shape preference:** should the v2 migration actually brand
   `New(baseURL, token)` (typed BaseURL/Token), or only unify the public structs
   and keep the string signatures? bank-sync consumes this drop-in, so the
   consumer's preference should win and I cannot infer it from this repo.

---

**Then wait for instructions** — no further work executed after this report.
