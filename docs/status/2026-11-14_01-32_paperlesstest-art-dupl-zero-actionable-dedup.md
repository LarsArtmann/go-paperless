# Session Status: paperlesstest Dedup-to-Zero (art-dupl `-t 1` sweep)

**Session:** 2026-11-13 ~22:00 – 2026-11-14 01:32 CET · single Crush session, no task prompt
beyond "deduplicate to zero" over a pasted `art-dupl --sort total-tokens -t 1 --timing --html`
report (9 actionable groups / 19 clones, all in `paperlesstest/` production files).

**Session summary:** Triaged and dispositioned all 9 actionable clone groups from the
`-t 1` deep sweep — 8 eliminated by extraction, 1 accepted with `//art-dupl:accept`
markers — then re-verified the full gate cascade and annotated the open items from the
2026-10-24 triage session. art-dupl is now at **0 actionable, 0 total clones at `-t 1`**
(also 0 at `--type-aware -t 1` and `-t 5`).

| # | Disposition                                                                                                                                                                     | Evidence                                                                 |
| - | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| 1 | Groups 1/4/6 (custom-fields ↔ storage-paths list/create hand-copies): folded into the named-family machinery — new generic `routeNamedCollection[B, E namedListView]` + `finalize` hook; `routeCustomFields`/`routeStoragePaths` are ~20-line shells; `handleNamedList` generalized over `namedListView` (value-receiver `entityName` on all three entity types) | `paperlesstest/entities.go` (routeNamedCollection :338, shells :479/:504) |
| 2 | Groups 2/8 (ID-bump line pairs `Add*` ↔ route POST handlers): eliminated — the bump lives once inside the shared `handleNamedCreate`; `Add*` methods keep distinct field names, so no token pair remains | no bump lines in route shells; art-dupl re-scan clean                    |
| 3 | Group 7 (+ identical decode blocks in savedviews/sharelinks/documents): extracted `decodeBody` (`server.go:282`); unified messages to `decode <family> create/patch body`; new `invalidPatchDetail` const (wire details byte-identical: "invalid create payload" / "invalid patch payload") | `paperlesstest/server.go`, 5 call sites rewritten                       |
| 4 | Group 5 (savedviews ↔ sharelinks delete tails): extracted generic `deleteOne` (`entities.go:608`, DeleteFunc + 404/204 outcome); both delete handlers reduced to lock + one call | `paperlesstest/savedviews.go:128`, `sharelinks.go:133`                  |
| 5 | Group 3 (Documents/Patches/Uploads copy-loop tails): extracted `snapshotCopies[S, T]` (`documents.go:67`); three distinct per-type clone closures remain, tails gone. Bonus: `Requests()` simplified to `slices.Clone` | `paperlesstest/documents.go`, `tasks.go`, `records.go`                  |
| 6 | Group 9 (`default: return false` router fallthroughs, routeEntities ↔ routeDocumentDetail): ACCEPTED — idiomatic switch scaffolding, not shared logic; `//art-dupl:accept unmatched-route fallthrough is the router contract, not shared logic` markers on both sites | `entities.go:290`, `documents.go:382`                                   |
| 7 | Gate cascade: build · full tests · `nix fmt` (2 files reflowed, re-tested) · art-dupl ×3 scans (user flags / `--type-aware -t 1` / `-t 5`) all 0 actionable · golangci `nix run .#lint` 0 issues · branching-flow total 0 · `nix flake check` ALL PASSED (incl. race derivation) · erraudit `lint ./... --max --disable generic_return` exit 0 · LSP restarted, diagnostics clean | all exit codes observed in-session                                      |
| 8 | Docs/memory: 2026-10-24 report rows 1 + 8 annotated done (in-cell strikes, no dprint on that file); AGENTS.md routing bullet rewritten (routeNamedFamily/routeNamedCollection/finalize + decodeBody/deleteOne/snapshotCopies inventory) and art-dupl bullet extended (accepted-site inventory, `-t 1` = 0 actionable @ 2026-11-13, golines re-scan warning) | `AGENTS.md:131-141`, `AGENTS.md:183-191`, committed by daemon (a9f9ebb) |

## a) FULLY DONE

1. **Dedup goal reached**: `art-dupl --sort total-tokens -t 1` (the user's exact command) →
   244 detected groups, **0 shown / 0 total clones**. Same under `--type-aware -t 1` and `-t 5`.
2. All 9 groups dispositioned with zero behavioral change (wire responses byte-identical;
   only t.Errorf decode diagnostics reworded from URL-path-based to family-label-based —
   verified no test asserts the old strings before changing).
3. Repo conventions honored: free generic functions (golines-safe), markers on anchor
   lines re-verified post-`nix fmt`, no dprint on strike-bearing status files, no new
   third-party imports in `paperlesstest/` (stdlib + existing deps only).
4. Full verification cascade green (see table row 7); `nix flake check` as CI ground truth.
5. Memory maintenance: AGENTS.md routing + art-dupl bullets updated; prior-session open
   items (deep sweep rows) annotated done in the 2026-10-24 status report.
6. Suppression-marker inventory recorded in AGENTS.md (Ping pair, savedviews/sharelinks
   route shells, named.go findByName tails, example_test boilerplate, two router
   fallthroughs) so the next sweep knows the accepted floor.

## b) PARTIALLY DONE

1. **erraudit**: only the sanctioned maximum-strictness form was run
   (`erraudit lint ./... --max --disable generic_return`, exit 0). The routine form from
   AGENTS (`erraudit ./... --type-aware --enforce-go-error-family --disable-extensions`)
   was NOT run this session — the --max form is the stricter audit, but both documented
   forms exist and only one was exercised.
2. **Gate tier**: focused Go gates all green, but a full `nix develop -c buildflow --fix`
   loop (govulncheck, gosec, go-licenses, lychee, dprint, vulnix, …) was not run after
   the refactor. Same discipline miss the 2026-10-24 session self-reported (its item 2
   of "could have done better") — repeated, not newly invented.
3. **Doc-tail commits**: AGENTS.md + status annotations landed only via the auto-commit
   daemon (a9f9ebb); no deliberate, message-bearing commit was made (user did not ask
   for one — noted, not judged a defect).
4. **Coverage evidence**: tests green, but `nix run .#coverage` was not run to prove no
   coverage drop in the refactored files (behavior-preserving refactor, existing
   round-trip + coverage tests pass — indirect evidence only).

## c) NOT STARTED (open items noticed this session, incl. prior-session leftovers)

1. Old row 5: prose de-dup of the mirrored route-shell rationale still duplicated in
   both `routeSavedViews`/`routeShareLinks` doc comments.
2. Old row 2: dedicated negative test for `rejectWrongMethod` (405 + Allow + handled).
3. Old row 3: explicit CHANGELOG policy decision for behavior-neutral internal refactors
   (I skipped the changelog this session; decision not yet recorded anywhere binding).
4. Old rows 7/9/10/11/14/17: .gitignore for `art-dupl-report.html`; marker survival
   re-verify at next treefmt/dprint bump; "small-change gate tier" note in AGENTS.md;
   upstream ask for `art-dupl --format json`; re-anchor example_test.go markers;
   skim suppressed groups for ≥4-occurrence patterns.
5. `writeNotFound` consolidation (~7 identical find→nil→404 sites across
   entities/documents handlers) — deliberately left out: scanner-clean today, and the
   surgical floor "only what art-dupl flags" was kept.
6. `data_type` default literal `"string"` exists twice (WithCustomFields + finalize
   hook); no shared const.
7. `handleDocumentDelete` keeps its hand-rolled delete+record shape (differs from
   `deleteOne`: pre-check + deletedDocuments append) — no doc comment explains why it
   must not use `deleteOne`.
8. gosec probe not re-run post-refactor (new code has no security surface; the G125
   suppression site was untouched).

## d) TOTALLY FUCKED UP

**Nothing destructive.** No data loss, no reverts of others' work, no broken gates at
session end. Honest process defects, all caught and corrected in-session:

1. **Design dead-end on the generics**: first attempt constrained `handleNamedCreate`
   over `namedEntityView[B]` with pointer-receiver methods — value-typed stores don't
   satisfy pointer-receiver interfaces, so it cannot compile (method-set rule I should
   have worked out before editing, per the read→understand→think discipline). Cost:
   ~4 extra edit rounds and a switch to the simpler `finalize`-hook design. No damage;
   the final design is simpler than the first idea (a good outcome arrived at
   inefficiently).
2. **Stale gopls diagnostics**: after the pointer-receiver failure the LSP kept serving
   a frozen error list (citing removed lines) for several edits; I burned attention
   reconciling it against real builds instead of restarting gopls immediately
   (AGENTS.md even documents startup-only LSP lifecycle). Restarted late; clean after.
3. **One malformed annotation edit**: first attempt at the status-report strikes had a
   missing opening `~~` and hit the read-before-edit guard; caught on the very next
   step and re-applied correctly (verified by sed).
4. Cosmetic: a verification `grep -c` returned exit 1 on zero matches (expected grep
   semantics, not a tool failure).

## e) WHAT WE SHOULD IMPROVE

1. **Think type-param design through before typing**: generics method sets (value vs
   pointer receivers vs `*E` constraint terms) deserve the same upfront reflection the
   data-models-first rule mandates for data shapes. The `finalize`-hook design (no
   interface, closures at the shell) should have been the first candidate — it is both
   simpler and the one that survived.
2. **Restart the LSP as soon as diagnostics stop tracking edits** — stale-diagnosis
   reconciliation is wasted motion; the restart is one call.
3. **Run the complete documented gate tier for a ~10-file refactor**: both erraudit
   forms + one full buildflow loop, not just the focused gates. Every gate the repo
   documents as "working" should either run or have a recorded N/A rationale in the
   session report.
4. **Prefer the boring design first**: the session's winning pattern (one hook
   parameter, zero new interfaces) is also the most obvious one — reach for it before
   constraint-type machinery, even when the constraint version looks more "typed".
5. **Record skipped-changelog decisions** in the session report at decision time (done
   here, section c.3) so the next release-cut session doesn't have to re-litigate.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

**Gate & policy (highest leverage)**

| #  | What                                                                                                                                                          | Priority | Size | Category    |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ---- | ----------- |
| 1  | Decide the art-dupl gate policy (old g-2): committed `.art-dupl-baseline.json` + CI `art-dupl check` job pinning "0 actionable" (go-error-family precedent) vs prose+manual sweeps — and pin the threshold (`-t 5` routine, `-t 1` sweep?)        | Medium   | M    | Quality     |
| 2  | Implement whichever baseline policy is chosen in #1 (CI job + baseline file + AGENTS.md note)                                                                 | Medium   | M    | Quality     |
| 3  | Record the "small-change gate tier" in AGENTS.md (which gates are mandatory for <50-line vs ~10-file internal refactors) — this session is the worked example | Medium   | S    | Documentation |
| 4  | Run the full `nix develop -c buildflow --fix` loop once over the post-refactor tree and record the findings baseline                                          | Medium   | S    | Quality     |
| 5  | Re-run the routine erraudit form (`--type-aware --enforce-go-error-family`) as evidence; consider documenting that `--max` form subsumes it or keep both listed | Low      | S    | Quality     |
| 6  | Re-run the gosec probe post-refactor (extend the 2026-11-08 unparking evidence)                                                                              | Low      | S    | Security    |
| 7  | `nix run .#coverage` pass over the refactored files to pin no-coverage-drop evidence                                                                          | Low      | S    | Quality     |
| 8  | Decide CHANGELOG policy for behavior-neutral internal refactors (old row 3) and record the decision in AGENTS.md or CONTRIBUTING                              | Low      | S    | Documentation |
| 9  | Old row 9: verify acceptance markers survive the next treefmt/dprint plugin bump (calendar-triggered check)                                                   | Low      | S    | Quality     |
| 10 | Old row 11: verify art-dupl flags and file the upstream ask for machine-readable output (`--format json`) — verify-external-claims first                      | Low      | S    | Tooling     |

**Code cleanliness (scanner-clean floor already reached; these are judgment calls)**

| #  | What                                                                                                                                                          | Priority | Size | Category    |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ---- | ----------- |
| 11 | De-dup the mirrored route-shell rationale prose in `routeSavedViews`/`routeShareLinks` doc comments (old row 5) — one shared paragraph + cross-reference      | Low      | S    | Documentation |
| 12 | Extract `customFieldDefaultDataType` const; use it in `WithCustomFields` + the finalize hook (two `"string"` literals today)                                   | Low      | S    | Cleanup     |
| 13 | Document (or remove) why `handleDocumentDelete` doesn't use `deleteOne` (pre-check + deletedDocuments recording is deliberate — say so at the site)            | Low      | S    | Documentation |
| 14 | Judgment: `writeNotFound` helper for the ~7 find→nil→404 sites — adopt only if the scanner or review ever flags them; otherwise leave the surgical floor       | Low      | S    | Cleanup     |
| 15 | Judgment: generic seed helper for the five `Add*` methods' lock/bump/append skeleton — likely ACCEPT (bodies differ by construction); if accepted, no marker needed (not token-clones); document the pattern in AGENTS.md instead | Low      | S    | Cleanup     |
| 16 | Verify the finalize-hook semantics are directly pinned by tests (custom-field create defaults `data_type`; storage-path create derives `path-N` slug); add missing assertions if only implicitly covered | Medium   | S    | Testing     |
| 17 | Old row 2: dedicated negative test for `rejectWrongMethod` (405 + Allow header + handled=true)                                                                  | Medium   | S    | Testing     |
| 18 | Old row 17: skim the suppressed art-dupl groups for ≥4-occurrence patterns (cross-family guards)                                                                | Low      | M    | Quality     |
| 19 | Old row 14: re-anchor `example_test.go` art-dupl markers that sit above their clone ranges (re-scan to see if they re-surface first)                            | Low      | S    | Cleanup     |
| 20 | Old row 7: confirm `.gitignore` rule for root-level `art-dupl-report.html` exists (or add it); keep writing reports to /tmp per the skill                       | Low      | S    | Cleanup     |

**Docs & memory**

| #  | What                                                                                                                                                          | Priority | Size | Category    |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ---- | ----------- |
| 21 | Fold this refactor into the next release notes (one line, paperlesstest-internal, behavior-neutral) when the release is cut                                    | Low      | S    | Documentation |
| 22 | Add a one-paragraph architecture note to `paperlesstest/README.md` (routeNamedFamily vs routeNamedCollection split) — only if the README grows an internals section; today it is wire-level and correct | Low      | S    | Documentation |
| 23 | AGENTS.md: extend the paperlesstest bullet with "new named family = implement entityName + a route shell over routeNamedCollection" checklist                  | Low      | S    | Documentation |
| 24 | Consider a `docs/adr/` entry only if the finalize-hook pattern gets reused outside paperlesstest — today it is package-internal, ADR likely overkill            | Low      | S    | Documentation |

**Consumer & ecosystem (noticed, adjacent)**

| #  | What                                                                                                                                                          | Priority | Size | Category    |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ---- | ----------- |
| 25 | After the next release, let bank-sync/InboxClean fake-swap E2Es re-confirm the refactor changed nothing observable (their existing TODO rows already track this)| Medium   | M    | Consumers   |
| 26 | TODO_LIST row 49 (split-brain sweep pinning the fake's wire-mirror struct tags) — untouched, still valid                                                       | Medium   | M    | Testing     |
| 27 | Keep the example_test.go marker style consistent with the new accepted-site inventory in AGENTS.md (single source of truth: AGENTS list vs scattered comments) | Low      | S    | Documentation |
| 28 | Sweep for any remaining `paperless.New*ID(...Get() + 1)` hand-bumps outside the shared create path that could silently drift from it                            | Low      | S    | Quality     |
| 29 | Verify `decodeBody`'s double `readBody` in `handleDocumentPatch` (raw captured, then re-read) is documented at the site — it is safe (readBody restores) but non-obvious | Low      | S    | Documentation |
| 30 | Re-run `branching-flow all .` after the next `nix fmt` that touches the new closure-heavy shells (the known reflow class; cheap standing check)                 | Low      | S    | Quality     |

(30 curated items; items 1–3 are the ones I'd do first — they convert this session's
manual sweep into an enforced floor so it cannot silently regress.)

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Gate policy**: Should "0 actionable" become an *enforced* floor — committed
   `.art-dupl-baseline.json` + CI `art-dupl check` job (the go-error-family repo
   precedent, validated by the deduplicate-code skill) — or stay prose + periodic
   manual sweeps like today? And if enforced: at which threshold (`-t 5` routine with
   `-t 1` as deliberate sweeps, or `-t 1` outright)?
2. **Dedup scope floor**: Is "eliminate everything art-dupl flags at `-t 1`, touch
   nothing else" the standing rule, or do you also want the near-miss consolidations
   that the scanner does not flag (e.g. `writeNotFound` across ~7 sites, generic seed
   helper for the five `Add*` methods) done proactively?
3. **CHANGELOG policy**: For behavior-neutral `paperlesstest/`-internal refactors like
   this one — fold a one-liner into the next release's CHANGELOG, or keep skipping
   them entirely until a consumer-visible change ships?

---
_All gates green at session end: tests, lint, flake check, branching-flow 0, erraudit
exit 0, art-dupl 0 actionable at every threshold scanned._
