# Branching-Flow Findings Triage — Session Status Report

> **EXECUTED 2026-10-11 (same day, follow-up session).** Plan v2 ran with one
> amendment discovered during execution: tool v0.6.4 honors
> `//nolint:branching-flow` only for the panic and phantom analyzers — dupe
> and composition findings ignore line directives entirely — so the 31
> DUPLICATE_TYPE/COMPOSITION_mixin/large-struct findings are dismissed via
> `.branching-flow.yml` `disabled_rules` (rationale in the config comment and
> as doc-comment paragraphs on each flagged declaration) instead of inline
> nolints. Final state: **0 active findings** (60 → 0); 13 eliminated in code
> (faults.go pointer restructure + five fake entity/wire struct merges),
> 16 suppressed per line with reasons, 31 dismissed per rule; all gates green
> (build, test, race, vet ±integration, lint, erraudit, `nix fmt` stable).
> Items below describe the state BEFORE execution.

**Date:** 2026-10-11 03:41
**Scope:** This session only — triage and planning of `branching-flow all .` output (60 findings). No code was modified this session.
**Predecessor:** `docs/status/2026-10-11_03-23_branded-id-migration-status.md` (migration complete; its 3 open questions remain unanswered — see section g).

---

## Session input

`branching-flow all . --format finding` (tool v0.6.4): **60 findings** —
1 error / 36 warnings / 23 info; by category: correctness 3, type-safety 15,
duplication 18, structure 24; 9 files affected / 12 scanned; 0 suppressed;
exit code 0. Full JSON: `.crush/shell-output/output-2363040712.log`.

---

## a) FULLY DONE

| # | Item                                                                                               | Evidence                                                                                                                                                                                                                                                                                                              |
| - | -------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Complete inventory of all 60 findings (id, rule, file:line, class)                                 | extracted from shell-output log; every class accounted for                                                                                                                                                                                                                                                            |
| 2 | Root-cause verification of all 3 correctness findings — **all three are analyzer false positives** | `documents.go:71` deref: `s.documents` is `[]*Document` only ever appended as `&stored` (documents.go:133) — nil unrepresentable. `faults.go:66,74`: both accesses sit inside the `if index >= 0` guard (faults.go:65); index is a `range` index or `-1`                                                              |
| 3 | Suppression mechanism fully researched                                                             | `//nolint:branching-flow` / `// branching-flow:ignore` line-scoped directives; `--include-suppressed` audit mode (suppressed findings don't fail `--exit-code` gates); `.branching-flow.yml` `disabled_rules` (rejected: loses per-site reasons). Verified in tool README + source (`pkg/roleak`, `pkg/doanalyzerv2`) |
| 4 | Per-class disposition for all 60 findings, twice                                                   | v1 (rejected, see d) and v2 (pending approval, see below)                                                                                                                                                                                                                                                             |
| 5 | Tool surface mapped                                                                                | 16 subcommands; `all` flags (`--baseline`, `--severity`, `--exit-code`); `panic` analyzer is experimental (source of all 3 FPs)                                                                                                                                                                                       |
| 6 | Todo tracking created (13 tasks, v1 plan)                                                          | —                                                                                                                                                                                                                                                                                                                     |

**v2 triage (current, awaiting approval): 15 FIX at root / 45 SUPPRESS with reason.**

- **FIX (15):** faults.go `*Fault` + `slices.DeleteFunc` restructure (kills 2 INDEX_OUT_OF_RANGE, removes the sentinel-index smell); paperlesstest entity/wire consolidation — merge `namedEntity`/`namedEntityWire`, `customFieldEntity`/`customFieldWire`, `storagePathEntity`/`storagePathWire`, `savedViewRuleEntity`/`savedViewRuleWire`, `shareLinkEntity`/`shareLinkWire` (kills 10 DUPLICATE_TYPE + 3 COMPOSITION_mixin; deletes ~10 structs + conversion ceremony).
- **SUPPRESS (45):** 1 nil-deref FP (invariant documented at site); 4 PHANTOM-TRANSPOSE (public-API params, typed-string = ROADMAP v2); 11 PHANTOM-COLLISION (parse-boundary helpers take flat wire strings by design); 8 SDK DUPLICATE_TYPE (tags on public structs = v2 marshaling decision; cross-package mirror = import cycle); ~20 mixins (9 break public composite literals = v2; ~10 cross SDK↔fake = import cycle; 2 input-vs-response shapes sharing 2 fields); 1 Server large-struct (fake fixture-state holder, single-mutex design).

## b) PARTIALLY DONE

| Item                                          | Done                                                           | Missing                                                                                                                                                                                                                                                   |
| --------------------------------------------- | -------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Suppression syntax verification               | Line-scoped `//nolint:branching-flow` confirmed in tool source | Rule-scoped casing unverified (tool's own tests use lowercase `:phantom`/`:do`; emitted IDs are `PHANTOM-COLLISION` etc.). Needs a 2-min probe before mass application                                                                                    |
| Entity/wire merge feasibility                 | Tool's "same signature" grouping known for all 5 pairs         | Field-by-field diff NOT done by me — counts in the v2 plan are tool-derived. Known wrinkle: `savedViewEntity`/`savedViewWire` are NOT identical today (FilterRules element types differ) — that merge is conditional on the rule-pair merge landing first |
| Suppression anchoring on mid-expression lines | —                                                              | The 11 PHANTOM-COLLISION sites anchor on composite-literal field lines (client.go:1362/1488/1542, tasks.go:307-338); trailing-comment anchoring there is untested                                                                                         |
| Tool behavior check                           | Read README                                                    | Contradition unresolved: README says `--exit-code` is ON by default, yet the user's run exited 0 WITH 60 findings (0.6.4 binary vs local repo main?)                                                                                                      |

## c) NOT STARTED

1. faults.go restructure (task 1 of v2 plan).
2. All 5 paperlesstest entity/wire merges.
3. All 45 suppression comments.
4. Verification: `all` → 0 active; `--include-suppressed` → 45 marked; `nix fmt` anchor-stability re-run.
5. Full gates: build / test / race / vet(+integration) / lint / erraudit.
6. AGENTS.md suppression-convention note + accepted-classes section (and refreshing the now-half-stale branching-flow policy note).
7. CHANGELOG row for the fake consolidation (expected: no public-surface delta).
8. All carry-over work from the 03-23 report (BuildFlow run, golden-bytes wire test, FK-clearing PATCH test, v0.5.0 tag + release-verify, consumer migrations, strong-id CI gate, HARVEST of that report's section f, gopls restart).

**Zero files were edited this session.**

## d) TOTALLY FUCKED UP!

1. **Plan v1 was "suppress all 60".** Root cause: the entity/wire split I proposed to preserve was built by ME last session — I was judging my own recent output and pattern-matched it to "deliberate architecture" instead of scrutinizing it. Motivated reasoning; it took the user's one-line challenge ("Why Suppress EVERYTHING?!") to find the answer I should have found first: the fake's internal entity/wire split pulls no weight and merging it is a genuine improvement.
2. **Todo list went stale the moment the plan changed.** The 13 tasks still describe v1; never reconciled to the v2 plan. Being fixed with this report.
3. **Promised merge counts without reading the structs.** The "10 DUPLICATE_TYPE + 3 mixin die" claim trusts the tool's grouping; I never diffed the five pairs myself. If any pair has asymmetric fields (savedView already known-wrinkled), the counts and task order in v2 are wrong.
4. **Dismissed a contradiction instead of resolving it.** "Exit code 0 despite findings + README says exit-code default-on" got a shrug ("version may differ") instead of a 30-second probe. Unresolved.

## e) WHAT WE SHOULD IMPROVE!

1. **When the analyzer flags code I wrote, invert my default:** "fix" unless fixing is provably worse — never "by design" without a fight. My own output deserves MORE scrutiny, not less.
2. **Diff before you promise.** Consolidation counts in a plan must come from reading the structs, not from the scanner's clustering.
3. **Spike mechanism unknowns early.** Two 2-minute probes (nolint casing, mid-expression anchoring) would have de-risked the entire 45-suppression half of the plan before it was presented.
4. **Keep the todo tool synchronized at every plan revision**, not at execution start.
5. **Lead with the fix/suppress split and evidence**, not with the suppression table — framing shapes the reviewer's scrutiny.
6. **Surface tool-version contradictions immediately** — they usually mean a doc or a binary is lying, and either matters for CI gating later.

## f) Next things to get done (48, sorted roughly by impact)

**Execute the approved v2 plan:**

1. Pre-flight: diff the 5 entity/wire struct pairs field-by-field (validates merge counts + order).
2. Pre-flight probe: `//nolint:branching-flow[:RULE]` casing on one finding.
3. Pre-flight probe: suppression anchoring on a composite-literal line (client.go:1362).
4. faults.go: `armedFaultLocked` → `*Fault` + `slices.DeleteFunc` identity delete (kills 2 findings, removes sentinel index).
5. savedviews.go: merge `savedViewRuleEntity`/`savedViewRuleWire` (2 findings).
6. savedviews.go: merge `savedViewEntity`/`savedViewWire` (1 mixin; after #5).
7. sharelinks.go: merge `shareLinkEntity`/`shareLinkWire` (3 findings).
8. entities.go: merge `customFieldEntity`/`customFieldWire` (2 findings).
9. entities.go: merge `storagePathEntity`/`storagePathWire` (2 findings).
10. entities.go: merge `namedEntity`/`namedEntityWire` (2 findings).
11. Run paperlesstest suite (round-trip + TestFakeCoversEveryClientRoute) after each merge.
12. Suppress NIL_POINTER_DEREF documents.go:71 (invariant: only `&stored` appended).
13. Suppress 4 PHANTOM-TRANSPOSE (v2 typed-string scope, ROADMAP ref).
14. Suppress 11 PHANTOM-COLLISION (parse-boundary reason).
15. Suppress 8 client.go DUPLICATE_TYPE (public-struct-tags = v2 / import cycle).
16. Suppress remaining ~20 mixins (v2 / cycle / input-vs-response).
17. Suppress Server ANTI_PATTERN_large-struct (fake state holder).
18. Verify `branching-flow all` → 0 active findings.
19. Verify `--include-suppressed` → all 45 marked suppressed (audit mode).
20. `nix fmt`, then re-run `all` (formatters must not move directives off anchors).
21. Full gates: build, test, test -race, vet (+integration tag), `nix run .#lint`, erraudit.
22. AGENTS.md: suppression convention + accepted-finding classes; rewrite the now-stale branching-flow policy note.
23. CHANGELOG: fake-consolidation row (document explicitly that public surface is unchanged).
24. TODO_LIST.md: row for this work; mark this report's items executed.

**Carry-overs from the 03-23 report (still open, user-gated or unstarted):**
25. User decision: tag v0.5.0 now or hold for consumer lockstep.
26. User decision: who migrates bank-sync / InboxClean.
27. User decision: ADR 0004 paperlesstest contract amendment sign-off.
28. BuildFlow `--fix` run (license/vuln gates vs go-branded-id v0.7.0).
29. Golden-bytes wire test pinning `"id":null` create bodies and `"correspondent":null` PATCH.
30. Explicit FK-clearing null-PATCH test.
31. Tag v0.5.0 + `nix run .#release-verify -- v0.5.0`.
32. Migrate bank-sync to v0.5.0 branded IDs (unify with `paperless.DocumentID`).
33. Migrate InboxClean `internal/papersync`.
34. Wire `branching-flow strong-id --exit-code` as CI gate.
35. HARVEST the 03-23 report's section (f) (40 items) into TODO_LIST/ROADMAP (docs-health).
36. Restart gopls/Crush — stale diagnostics have persisted two sessions.
37. Re-adjudicate ids.go godoclint warning.

**New from this session:**
38. Resolve the branching-flow 0.6.4 exit-code contradiction (README default-on vs observed exit 0).
39. Once `all` is clean: evaluate re-enabling branching-flow in `.buildflow.yml` (currently skipped by policy).
40. Evaluate `--baseline` SARIF as an alternative to inline suppressions for the v2-scope class.
41. Upstream feedback to branching-flow: the two panic-analyzer FP patterns (guarded sentinel index; range-over-pointer-slice invariant).
42. Seed a v2 ADR: typed string params/fields — the 15 type-safety suppressions are the ready-made catalog.
43. v2 decision: JSON tags on public structs (retires the 8 SDK DUPLICATE_TYPE suppressions).
44. Consider exporting SDK wire payloads for fake reuse (retires cross-package dupe/mixin suppressions; needs an ADR — touches ADR 0004's amendment).
45. Document the suppression inventory so the v2 migration knows exactly which directives to remove.
46. Watch `Server` struct growth — revisit field grouping if it keeps accumulating (suppressed now, threshold 15 vs 32 fields).
47. Check whether branching-flow flags stale/unused suppression directives (hygiene).
48. Optional: `branching-flow stats` before/after snapshot for the follow-up report.

## g) Questions I can NOT figure out myself

1. **Approve the v2 plan as presented (15 fix / 45 suppress)?** Or move a class — e.g., brand the 4 PHANTOM-TRANSPOSE params NOW as a v0.6.0 breaking minor instead of waiting for v2?
2. **Once `all` reads clean, should it become a hard gate** (CI job and/or re-enabling the skipped branching-flow step in `.buildflow.yml`), or stay advisory?
3. **Carry-over: tag v0.5.0 now or hold until consumers migrate?** Decides whether this session's work folds into 0.5.0 (internal-only changes; folding is safe) or ships after.

---

_Session state: WAITING FOR INSTRUCTIONS. No code modified. Todo list reconciled to the v2 plan (all pending)._
