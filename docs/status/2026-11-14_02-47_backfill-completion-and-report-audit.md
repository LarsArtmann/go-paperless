# Status Report: v0.5.3 Release-Page Backfill Completion and Report Audit

**2026-11-14 02:47 CET** · go-paperless · continuation of the v0.5.3 release
session. Covers the owner-directed backfill ("Backfill all missing GitHub
Releases!") executed after
`2026-11-14_02-09_v053-release-cut-and-release-flow-gaps.md` (the 02:09
report), plus an audit of that report's own quality — where it caught
**this session's author fabricating a number**, corrected in place.

## Scope line

**Verified this session (fresh command output):**

- All 5 missing Releases created: v0.1.0, v0.3.2, v0.4.0, v0.4.1, v0.4.2 —
  `gh release create` returned all five tag URLs on first attempt.
- Set equality **by count command, not memory**: `gh release list --limit 20 | wc -l` → **13**;
  per-tag `gh release view` loop over `git tag` → 13/13 resolve, zero
  "STILL MISSING" lines.
- **No `latest` steal**: `gh release list --limit 1` → v0.5.3 still marked
  `Latest`. Every backfill passed explicit `--latest=false`.
- Release body spot-check: `gh release view v0.4.0 --json body` returns the
  notes file content intact (the long one — v0.4.0 — chosen as the sample).
- CI on the docs commit `cf83641`: run `43539778715`, completed **success**
  (16m0s) — closes the 02:09 report's (b)(3).
- Daemon committed the report annotations + AGENTS.md lesson
  (`6b03439`, `7817425`); a CI run for `7817425` was `in_progress` (43s in)
  at write time — docs-only diff again.
- The 02:09 report's annotations applied via multiedit, tool-confirmed
  (4 edits): (b)(1) resolved with evidence, (g) Q2 answered,
  (f)#1 struck done, arithmetic corrected, execution appendix appended.

**NOT verified here (and why):**

- Bodies of the other four backfilled releases not spot-checked (one of
  five sampled; v0.4.0 chosen as the largest/most complex). Low risk —
  `gh release create --notes-file` either posts the file or fails.
- CI on `7817425` still running at write time (docs-only).
- GitHub-side rendering (blockquote, inline code, em-dashes) of the
  backfilled bodies not inspected in a browser — only the JSON body
  content was verified.
- Everything else carried from the 02:09 report's scope line unchanged.

---

## a) FULLY DONE

1. **The GitHub Releases page is complete: 13/13 version tags have
   Release entries.** Owner directive executed: five backfills created from
   their CHANGELOG sections (v0.4.0's full Added/Changed/Fixed/Verified,
   v0.3.2 including the recovered-verbatim blockquote, all with the
   italic backfill-provenance footer naming the original ship date).
   Evidence: count command 13, per-tag loop 13/13, listed in the appendix
   of the annotated 02:09 report.
2. **The `latest`-flag lesson was applied, not just recorded.** Every
   backfill used explicit `--latest=false`; post-backfill check confirms
   v0.5.3 remains `Latest`. The v0.5.1 steal from earlier in the session
   did not recur.
3. **The 02:09 report was annotated to resolution** (docs-health ANNOTATE
   convention: inline strikes + end-of-file appendix, no rewriting):
   (b)(1) closed with the 13/13 evidence, (f)#1 struck done, (g) Q2 marked
   answered by directive, and the report's own count error corrected in
   place (see (d)(1)).
4. **AGENTS.md now carries the complete release-page lesson**: run
   `gh release create` as part of every cut; diff `git tag` against
   `gh release list` (never sample the list head); pass
   `--latest`/`--latest=false` explicitly on EVERY create (GitHub
   auto-pick is NOT highest-semver — demonstrated, not assumed).
   Committed by the daemon (`7817425`).
5. **CI on the release-session docs commit is green** (run `43539778715`,
   success) — the last open verification from the release cut is closed.

## b) PARTIALLY DONE

1. **Release-page consistency is now cosmetic-only.** All 13 entries
   exist and v0.5.3 is Latest; what remains mixed: two old titles lack the
   `vX.Y.Z — headline` style (v0.3.1, v0.2.0), and five backfills show
   `publishedAt` = today while their bodies carry the original ship dates
   (GitHub cannot backdate a publish — this is platform behavior, not
   drift; the body footer is the only honest record). Remaining work is
   title normalization; the date thing is accepted-as-is. Effort: S.
2. **Release-flow mechanical gates.** The lessons live in AGENTS.md prose;
   `release-verify` still cannot see the Releases page, no set-equality
   sweep is scripted, no `release-create` app exists. The failure this
   guards against already happened once (v0.5.1/v0.5.2) and my manual
   process missed 5 of 7 gaps on first pass. Remaining: three small flake
   work items (02:09 report f#4/#6/#9). Effort: M total. Gated on (g) Q3.
3. **Consumer bumps to v0.5.3** — unchanged from the 02:09 report: TODO
   row exists, not executed, blocked on timing decision and billing-dead
   consumer CI ((g) Q1).

## c) NOT STARTED

1. **`release-verify` Releases-page check** (02:09 f#4) — the structural
   blind spot that let v0.5.1/v0.5.2 ship page-less for 7 days.
2. **Scripted tag↔release set-equality sweep** (f#6) — now run manually
   three times this session; rule-of-three says it becomes a command.
3. **`release-create` flake app** (f#9): draft notes from the CHANGELOG
   section, create with explicit flags, run the sweep.
4. **Proxy `.info` step in `release-verify`** (f#8).
5. **Title normalization for v0.3.1/v0.2.0** (f#29).
6. **Wire-correctness SLA decision** (f#11) — the quantified 6-day FK
   exposure window argument sits in the 02:09 report (d)(4).
7. **Everything carried from the 02:09 report not listed above** — art-dupl
   enforcement pair (gated Q3), savedviews/sharelinks prose de-dup,
   `rejectWrongMethod` negative test, full buildflow loop, erraudit
   hermetic pin (proxy watch), govalid-skip watch, branching-flow/v2,
   consumer billing fix (owner action). Not re-researched; state per the
   02:09 report stands.

## d) TOTALLY FUCKED UP

1. **The 02:09 report fabricated a count: "9 of 14 version tags covered"
   was never produced by any command.** True numbers: 8 of 13 had
   Releases. I composed the figure from memory fragments (a release-list
   glance plus the five-missing set) instead of running
   `gh release list | wc -l` — exactly the "impression sold as evidence"
   failure the scope-line rule exists to prevent, and it shipped in a
   report whose entire premise was evidence discipline. Caught during the
   backfill annotation when I re-derived totals; corrected inline in the
   annotated file. Root cause: no counting command in the verification
   batch, so the number was invented to fill the sentence. Severity:
   moderate — every other number in that report IS command-backed, but one
   fabricated count taxes trust in all of them. Mitigation: corrected in
   place; the count-from-command rule is this report's (e)(1).
2. **The original in-session backfill verified the visible, not the
   complete** (2 of 7 gaps found from the list head; 5 more found only by
   the set-equality loop during report writing). Already admitted at 02:09
   (d)(3); restated one line because it is the direct cause of (d)(1) —
   same failure mode (sampling instead of measuring), two artifacts
   (process, then report).
3. **The backfill execution itself had no failures**: five creates, zero
   retries, zero steals, no 404s. Nothing else is fucked up *in this
   continuation* — (d) is short because the work was small and the
   verification loop closed everything it touched.

## e) WHAT WE SHOULD IMPROVE

1. **Count-from-command rule for every report**: any N in a status report
   must cite the command that produced it (`wc -l`, a loop, a jq count).
   Composed arithmetic ("9 had + 5 missing = 14") is how (d)(1) happened.
   Concretely: the verification batch template gains a count command
   alongside every existence check.
2. **Script the set-equality sweep.** Third manual run this session. One
   loop over `git tag` with `gh release view` already exists as a
   session command; promote it to `nix run .#release-check` (or fold into
   `release-verify`) so the next cut cannot skip it.
3. **Post-create body verification**: `gh release view --json body` after
   each create (I sampled 1 of 5 this time; the loop costs seconds and
   removes the only unverified claim in this session's scope line).
4. **Title style at creation time**: backfills copied the body convention
   but the em-dash title style was applied by hand each time — normalize
   the two bare titles (v0.3.1, v0.2.0) and make the style part of
   `release-create` (f#9).
5. **Prose decays; gates don't.** AGENTS.md now documents three
   release-page rules; the 2026-11-07 cuts had the skill available and
   still skipped the page; my manual pass missed 5/7 gaps. The (b)(2)
   gates are the only durable fix — priority signal is in (g) Q3.

## f) Up to 50 things to get done next

Refreshed from the 02:09 report's table (30 items, #1 now DONE — removed).
Same harvest routing: TODO_LIST-eligible items marked; watch items route
to ROADMAP/AGENTS.md watch lists. Items gated on (g) questions say so.

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Bump bank-sync to v0.5.3: go.mod, `go mod vendor` + `buildflow -s nix-hash-fix --fix`, re-prove suite (gated Q1) | High | M | Feature |
| 2 | Bump journal/InboxClean to v0.5.3, same two-step tail + suite re-proof (gated Q1) | High | M | Feature |
| 3 | Resolve consumer Actions billing — owner action; unblocks CI re-proof of #1/#2 | High | S | Bug |
| 4 | Extend `release-verify` with a Releases-page existence check (`gh release view vX.Y.Z` must resolve) (gated Q3) | High | S | Quality |
| 5 | Script the tag↔Release set-equality sweep as a flake app (third manual run = rule of three) | High | S | Quality |
| 6 | Drop papersync's `retryafter.Clamp` for `MaxDelay: 60s` during the bank-sync bump | Medium | S | Cleanup |
| 7 | Add a `release-create` flake app: CHANGELOG-section notes draft, explicit-flag create, run the sweep (gated Q3) | Medium | M | Feature |
| 8 | Add proxy `.info` step to `release-verify` before the pkg.go.dev step | Medium | S | Quality |
| 9 | Normalize the two bare release titles (v0.3.1, v0.2.0) to the `vX.Y.Z — headline` style | Low | S | Cleanup |
| 10 | Decide + record the wire-correctness SLA (immediate patch cut for consumer-visible wire bugs) in AGENTS.md | Medium | S | Documentation |
| 11 | Carry-over: art-dupl baseline gate policy — baseline file + CI job at `-t 1` vs prose sweeps (gated Q3 / 01-32 report) | Medium | M | Quality |
| 12 | Carry-over: record the dedup scope-floor decision in AGENTS.md (gated Q3) | Medium | S | Documentation |
| 13 | Verify CI on daemon commit `7817425` (in progress at write time) and archive evidence | Low | S | Documentation |
| 14 | Carry-over: savedviews/sharelinks prose de-dup | Low | S | Cleanup |
| 15 | Carry-over: `rejectWrongMethod` negative test | Medium | S | Quality |
| 16 | Carry-over: full buildflow loop (last full run predates today's docs commits) | Medium | M | Quality |
| 17 | Carry-over: erraudit public-proxy re-check (unblocks the CI job) | Medium | S | Quality |
| 18 | Carry-over: govalid-skip removal watch (machine Go ≥ 1.27 rebuild) | Low | S | Cleanup |
| 19 | Carry-over: branching-flow v2 revisit (line-suppression for dupe/composition) | Low | L | Quality |
| 20 | Carry-over: v2 typed-API migration design (ROADMAP) | Low | L | Feature |
| 21 | Local `nix flake check` for a fresh post-release all-gates artifact | Low | M | Quality |
| 22 | CONTRIBUTING: document the release command sequence proven this session (gorelease → bump → CI watch → tag → proxy .info → gh release → release-verify → set-equality) | Medium | S | Documentation |
| 23 | Confirm MIGRATION.md renders in the pkg.go.dev doc set for v0.5.3 | Low | S | Documentation |
| 24 | Tag consumers' changelogs when the v0.5.3 bumps land (part of #1/#2) | Low | S | Documentation |
| 25 | Decide the CHANGELOG `[Unreleased]` convention (bare vs placeholders) in CONTRIBUTING | Low | S | Documentation |
| 26 | Backfill-provenance footer: promote the italic format from report convention to AGENTS.md line (superseded if #7 lands) | Low | S | Documentation |
| 27 | Triage the 2026-11-08 Dependabot update runs (what did they bump?) | Low | S | Cleanup |
| 28 | Fuzz watch: confirm the next scheduled run exercises the 2026-11-08 corpus additions | Low | S | Quality |
| 29 | `release-verify` SSH-signature assertion (config regression would go unnoticed today) | Low | S | Quality |
| 30 | Update FEATURES.md if the Releases-page completeness counts as an ops feature worth listing (likely not — decide and close) | Low | S | Documentation |

Honest stop at 30, same rationale as 02:09: the remaining known work is
represented above; padding to 50 would invent items no session observed.

## g) Questions I cannot answer myself

1. **Consumer bump timing vs billing** (carried from 02:09, still
   gating the row's whole purpose): run the bank-sync and
   journal/InboxClean bumps to v0.5.3 now — and if yes, is local suite
   re-proof acceptable while consumer Actions is billing-dead, or does
   billing land first? I cannot decide consumer priorities or pay the
   GitHub bill.
2. **Prose vs gates for the release flow**: are the three mechanical
   release gates (Releases-page check in `release-verify`, set-equality
   sweep app, `release-create` app) worth building now (≈M total), or do
   you accept prose-only protection given the failure already occurred
   once and my manual process missed 5 of 7 gaps? I can build them either
   way; the risk acceptance is yours.
3. **Carry-over, asked twice, still unanswered**: enforce paperlesstest
   0-actionable mechanically (`.art-dupl-baseline.json` + CI at `-t 1`) vs
   prose+manual sweeps, and is the dedup scope floor scanner-only or does
   it include proactive near-miss consolidations? Owner preference; gates
   f#11/#12.

---

**HARVEST note:** (f) refreshes the 02:09 table (its #1 is done and
struck there). New TODO_LIST-eligible: #5, #9, #13, #22, #26; the rest
route as marked. Report ends in WAIT per standing instruction — harvest
and the gated items move when you answer (g).
