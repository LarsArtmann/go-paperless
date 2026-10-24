# Session Status: paperlesstest art-dupl Triage (2026-10-24 08:49–11:38)

**Scope:** this session only — the deduplication triage triggered by the
untracked `art-dupl-report.html` (generated 08:49) — plus things noticed in
passing while working. No fresh project-wide research was done, per
instruction.

**Session summary:** art-dupl reported 7 actionable clone groups (15 clones,
43 tokens) in the new `paperlesstest/` package; 6 were eliminated by one
extraction, 1 was accepted with markers. All gates re-verified green.

---

## a) FULLY DONE

| # | What                                                                                                                                                                                                                                                                                                                                          | Evidence                                                                 |
| - | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| 1 | Decoded and triaged all 7 actionable art-dupl clone groups from the HTML report (208 detected, 201 auto-suppressed, 7 actionable — all in `paperlesstest/` production files)                                                                                                                                                                  | extraction script in session log; group inventory matches report summary |
| 2 | Extracted `rejectWrongMethod` helper (`paperlesstest/documents.go:568`) and collapsed the single-method 405 guard at 5 sites: `documents.go:271` (document list), `documents.go:358` (download), `tasks.go:367`, `savedviews.go:70`, `sharelinks.go:66` — 5 lines to 1 at each site; eliminated clone groups 1–7 except the route-tail mirror | `rg rejectWrongMethod` → helper + 5 call sites; committed in `0f8f8de`   |
| 3 | Accepted the savedviews ↔ sharelinks route-tail family mirror (only 2 families share the collection+delete route shape; create semantics too distinct for `routeNamedFamily`): `//art-dupl:accept` markers on both anchor lines + design rationale in both route doc comments                                                                 | `savedviews.go:58`, `sharelinks.go:54` markers; final scan 0 actionable  |
| 4 | Gate re-verification: `nix run .#test` ok (both packages) · `nix run .#lint` 0 issues · `branching-flow all . --format finding` total: 0 · art-dupl re-scan after `nix fmt` → **0 actionable** (fmt was a no-op; markers survived) · `nix flake check` all passed (includes the race check derivation)                                        | session log, all exit codes observed                                     |
| 5 | Stale report artifact `art-dupl-report.html` removed via `trash` (recoverable), superseded by the post-refactor scan                                                                                                                                                                                                                          | git status shows `D`                                                     |

## b) PARTIALLY DONE

| # | What works                                                                                 | What remains open                                                                                                                                                                                                                                                                                         | Blocker                            | Effort |
| - | ------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- | ------ |
| 1 | Acceptance disposition for the mirror group is anchored and verified by one manual re-scan | Acceptance is prose, not a gate: no committed `.art-dupl-baseline.json`, nothing in CI enforces "0 actionable" (contrast: go-error-family pins accepted groups in a baseline and runs `art-dupl check -t 1 .` in CI; the deduplicate-code skill explicitly prefers gates over prose for recurring audits) | policy decision (see question g-2) | S–M    |
| 2 | Core gates ran: test, lint, branching-flow, art-dupl, flake check                          | **erraudit was not run** (AGENTS documents it as a working gate; my change adds no error returns, so risk ≈ 0 — but the discipline was missed). No full `buildflow` run either (govulncheck, go-licenses, lychee, dprint, vulnix) — mostly N/A for a pure-Go refactor, but unverified                     | none, just do it                   | S      |
| 3 | Marker/doc-comment edits are in the working tree and verified                              | Uncommitted at report time (harness forbids manual commits; auto-commit daemon will pick them up)                                                                                                                                                                                                         | daemon latency                     | —      |

## c) NOT STARTED

| # | What                                                                                                                                                                                                                   | Why not                                                                    | Priority |
| - | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------- |
| 1 | Deep sweep `art-dupl -t 1` over `paperlesstest/` (the 201 suppressed groups below `-t 5` were never eyeballed)                                                                                                         | routine gate is `-t 5`; deep sweeps are a separate deliberate activity     | Low      |
| 2 | Dedicated negative test for `rejectWrongMethod` (405 + `Allow` header + handled=true). Happy path is implicitly covered — inverted logic would fail every list test — but the rejection branch has no direct assertion | deliberate-405 tests are a style question for this fake (see question g-3) | Medium   |
| 3 | CHANGELOG decision for the internal refactor (behavior-neutral; fold into next release notes or skip)                                                                                                                  | no release in flight                                                       | Low      |
| 4 | AGENTS.md note that art-dupl markers share branching-flow's golines-reflow fragility ("re-run the tool after `nix fmt`" is documented for branching-flow only)                                                         | noticed at the end of the session                                          | Low      |
| 5 | Prose de-dup of the mirrored rationale now duplicated in both route doc comments                                                                                                                                       | see e-1                                                                    | Low      |

## d) TOTALLY FUCKED UP

**Nothing introduced this session.** All gates green, no reverts, no data
loss, no behavioral change (guard semantics byte-identical: same 405, same
`Allow` header, same `return true`).

Pre-existing accepted-broken things noticed in passing (NOT session damage,
documented parks in AGENTS.md, listed so nobody thinks this session broke
them): gosec parked in CI (v2.29.0 cannot type-check Go 1.27 export data),
govalid step skipped in `.buildflow.yml`, erraudit kept out of CI until it
publishes on the module proxy.

One near-miss worth recording: the auto-commit daemon committed the
generated HTML report at 08:50 as `ba7c4a3 "docs: update report generation
logic for duplicate detection"` — a **misleading message** (the commit is a
generated report blob, not a logic change). I then deleted that committed
file via trash; the daemon will commit the deletion. No harm, but generated
artifacts landing in the repo root with heuristic messages is a real
hygiene problem (e-6).

## e) WHAT WE SHOULD IMPROVE

1. **Prose split-brain in the acceptance rationale** — the same 3-sentence
   mirror rationale now lives in both `routeSavedViews` and `routeShareLinks`
   doc comments. Irony: a dedup session added a (small) duplication. If the
   rationale ever changes, two places drift. Fix: one canonical sentence +
   cross-reference, or accept as standalone-file readability.
2. **Marker fragility is unguarded** — `nix fmt`/golines can reflow long
   anchor lines and silently break `//art-dupl:accept` suppression (same
   failure mode AGENTS documents for branching-flow). Today's marker lines
   are ≤ ~95 cols and survived, but nothing prevents a future reflow. Fix:
   keep markers short + re-scan after every fmt + document the protocol.
3. **Gate-stack discipline for small changes** — the session ran 5 of ~7
   relevant gates; erraudit (and full buildflow) were skipped by judgment
   call. Either always run them or record an explicit "small-change gate
   tier" in AGENTS.md so the judgment isn't ad hoc.
4. **Task-inference transparency** — the session opened with a generic
   "break this down" prompt and no task; I inferred dedup triage from the
   untracked report artifact (one-line statement, then proceeded). Worked
   here; the assumption should always be surfaced that explicitly.
5. **art-dupl machine-readable output** — triage began with 3 failed regex
   passes over the HTML before clean extraction. A `--format json` upstream
   would make agent triage deterministic (go-error-family already has a
   related flags-provenance TODO for art-dupl).
6. **Generated-report hygiene** — `art-dupl --html` artifacts land in the
   repo root, get daemon-committed under misleading heuristic messages, and
   then need manual trash-ing. Fix: write reports to `/tmp` (the
   deduplicate-code skill already says "view directly, do not save") or
   `.gitignore` root-level `*-report.html`.
7. **Acceptance-as-gate** (b-1): markers relitigate nothing; a baseline
   cannot be relitigated. If dedup audits recur here, pin the accepted
   group.

## f) Next tasks (session-derived; honest count is 20 — padding to 50 would

require the fresh research this report was told not to do)

| #  | Task                                                                                                                                                                                                                                    | Impact | Effort | Category      |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Run `erraudit ./... --type-aware --enforce-go-error-family --disable-extensions` inside `nix develop` over the session's changed files                                                                                                  | Medium | S      | Quality       |
| 2  | Decide baseline policy (question g-2); if yes: `art-dupl baseline`, commit `.art-dupl-baseline.json`, add CI job `art-dupl check -t 5 .`                                                                                                | Medium | S–M    | Quality       |
| 3  | Add negative test pinning `rejectWrongMethod`: wrong method → 405 + `Allow` header + `t.Errorf` fired; right method → false, no write                                                                                                   | Medium | S      | Test          |
| 4  | Decide CHANGELOG disposition for the refactor (fold into next release vs. skip)                                                                                                                                                         | Low    | S      | Documentation |
| 5  | De-dup the mirrored route rationale prose (e-1) — canonical sentence + cross-reference                                                                                                                                                  | Low    | S      | Cleanup       |
| 6  | Add AGENTS.md note: art-dupl marker anchors break under golines reflow; re-scan after `nix fmt` (extend the existing branching-flow warning)                                                                                            | Medium | S      | Documentation |
| 7  | Add `.gitignore` rule for root-level `art-dupl-report.html` / write future reports to `/tmp` per the skill                                                                                                                              | Low    | S      | Cleanup       |
| 8  | Deep sweep `art-dupl --type-aware -t 1` over `paperlesstest/`, triage what surfaces                                                                                                                                                     | Low    | M      | Quality       |
| 9  | Re-verify the acceptance markers survive the next `dprint`/treefmt plugin bump (same class as the known treefmt-lag gotcha)                                                                                                             | Low    | S      | Quality       |
| 10 | Record a "small-change gate tier" in AGENTS.md (which gates are mandatory for <50-line internal refactors)                                                                                                                              | Medium | S      | Documentation |
| 11 | File/track the upstream ask: art-dupl `--format json` for agent triage (verify current flags first per verify-external-claims)                                                                                                          | Low    | S      | Tooling       |
| 12 | Consider teaching the auto-commit daemon to skip `*-report.html` blobs at repo root or route them under `docs/reviews/` with honest names                                                                                               | Low    | M      | Tooling       |
| 13 | (Read, not researched) TODO_LIST already tracks the v2 typed-API migration items that supersede today's acceptance markers — revisit the savedviews/sharelinks mirror when `routeNamedFamily`-style generics grow a delete-only variant | Medium | L      | Feature       |
| 14 | (Noticed) `example_test.go` art-dupl markers from a prior session sit well above their clone ranges in spots — harmless today, but a future re-scan may re-flag those groups; re-anchor if that happens                                 | Low    | S      | Cleanup       |
| 15 | (Noticed) branching-flow suppressions share the same reflow fragility documented in AGENTS — today's zero held; nothing to do unless a fmt bump moves anchor lines                                                                      | Low    | S      | Watch-item    |
| 16 | Verify the flake's `checks.test-race` covers `./paperlesstest` too (it built a race derivation this session; confirm package list once)                                                                                                 | Low    | S      | Test          |
| 17 | Skim the 201 suppressed art-dupl groups for any ≥4-occurrence pattern even at low tokens (cross-file guards like today's)                                                                                                               | Low    | M      | Quality       |
| 18 | If baseline adopted (f-2), wire it into `nix flake check` so the canonical gate covers it                                                                                                                                               | Medium | M      | Quality       |
| 19 | Add the session's one-liner (markers accepted, helper extracted) to the next release CHANGELOG if 4 decides "fold in"                                                                                                                   | Low    | S      | Documentation |
| 20 | Re-read `docs/planning/2026-09-23_16-19_paperlesstest-consumer-testing-sdk.md` for any dedup-related guidance the triage should have consulted (it was not opened this session)                                                         | Low    | S      | Documentation |

> (f) is **pending docs-health HARVEST** — deliberately not harvested now
> because the instruction was report-then-wait.

## g) Questions I cannot answer myself

1. **Intent:** the session opened with a generic "break this down and
   execute" prompt and no task. I inferred "process the art-dupl report"
   from the untracked artifact. Was that the intended task, or was a
   different task meant to accompany the prompt?
2. **Baseline policy:** should this repo adopt the go-error-family pattern
   (committed `.art-dupl-baseline.json` + CI `art-dupl check` gate), or stay
   marker-only? The repo's own convention so far is markers; the skill
   prefers gates for recurring audits. This is a policy call, not a
   grep-able fact.
3. **Fake-testing philosophy:** should `paperlesstest` carry deliberate
   negative tests (assert 405 + `Allow` on wrong-method requests), given the
   fake's design makes unexpected calls fail tests via `t.Errorf` — i.e. is
   asserting the failure channel in-scope for the fake's contract, or
   consumer territory?

---

_Written 2026-10-24 11:38. Point-in-time snapshot; annotate, never rewrite.
Manual commit skipped (harness contract: never commit without explicit
request — auto-commit daemon handles it)._
