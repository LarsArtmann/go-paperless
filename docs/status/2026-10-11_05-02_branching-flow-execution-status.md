# Branching-Flow Execution — Session Status Report

**Date:** 2026-10-11 05:02 CEST
**Scope:** This session only — execution of the Plan v2 triage from
[`docs/status/2026-10-11_03-41_branching-flow-findings-triage.md`](2026-10-11_03-41_branching-flow-findings-triage.md)
(annotated EXECUTED). Standing instruction: READ → UNDERSTAND → RESEARCH →
REFLECT, steps ≤12 min, execute and verify one at a time. Mandate:
"actually improve things!" — prefer genuine fixes over suppressions.
**Predecessor chain:** branded-ID migration report (2026-10-11 03-23) →
triage report (03-41) → this report.
**Bottom line: `branching-flow all .` went 60 → 0 active findings, 13
eliminated in code, 16 suppressed per line with reasons, 31 dismissed per
rule with documented rationale. Every gate green. Two significant execution
mistakes made, understood, and corrected along the way (section d).**

---

## a) FULLY DONE

1. **Pre-flight verification before any mass edit.** Diffed all 5
   entity/wire pairs field-by-field (all identical; wire side only adds JSON
   tags; the savedView pair confirmed to need the rule-pair merge first).
   Probed `//nolint:branching-flow // reason` on two real targets
   (paperlesstest/documents.go:71 nil-FP, client.go:1362 collision) — 60 →
   58, both registered. Evidence: branching-flow runs captured in
   `/tmp/bf-baseline.txt`, `/tmp/bf-probe.json`.
2. **faults.go restructure (2 INDEX_OUT_OF_RANGE eliminated).** Storage
   `[]Fault` → `[]*Fault`; `armedFaultLocked` returns `*Fault` (nil = none)
   instead of a `-1` sentinel index; disarm via `slices.DeleteFunc` pointer
   identity. Public API (`InjectFault(Fault)`, `WithFaults`) unchanged.
   Evidence: fault/retry tests pass (`TestFaultIsConsumedByTimes`,
   `TestFaultThenSuccessRecoversWithRetry`, …), finding count 58 → 56.
3. **Five fake entity/wire struct pairs merged (11 finding-slots net).**
   `savedViewRule` + `savedView` (rule pair first — element-type dependency
   honored), `shareLink`, `customField`, `storagePath`, `namedEntity[B]`.
   The fake now stores and serves the same shape per family; ~10 internal
   structs and every `X(y)` conversion deleted. Wire bytes preserved
   (list/create/delete handlers marshal the merged struct with identical
   tags and field order). One deliberate wire improvement: a saved view
   created without rules now stores `filter_rules: []` (real DRF parity)
   instead of round-tripping `null` into list responses. Evidence:
   `TestSavedViewsRoundTrip`, `TestFakeCoversEveryClientRoute`, full suite
   green; savedviews findings 56 → 53 → 50 → 45 across the merges.
4. **16 per-line suppressions with reasons, format-stable.** 1 nil-deref FP
   (invariant: only `&stored` ever appended), 4 PHANTOM-TRANSPOSE
   (`New(baseURL, token)`, `EnsureStoragePath(name, path)`,
   `AddCustomField(name, dataType)`, `AddStoragePath(name, path)` — all
   ROADMAP-v2 branded-params scope), 11 PHANTOM-COLLISION (parse-boundary
   wire strings). Directives sit standalone directly above short anchor
   lines; verified stable across two `nix fmt` passes. Evidence:
   `--include-suppressed` audit shows exactly 16 hidden, set-diff verified.
5. **31 rule-class dismissals via `.branching-flow.yml`** (new file,
   tracked). DUPLICATE_TYPE / COMPOSITION_mixin / ANTI_PATTERN_large-struct
   dropped with a rationale comment block in the config AND per-site
   rationale doc comments on all 27 flagged declarations (public-structs-
   stay-tag-free = compatibility contract; SDK↔fake mirrors = round-trip
   contract; input-vs-response shapes diverge by design; `Server` = fixture
   state under one mutex). Spot-verified the claims against the actual
   structs (`CustomFieldValue`, `DocumentNoteUser`, `StoragePath`,
   `UploadRequest` confirmed tag-free).
6. **Root-cause discovery, documented:** tool v0.6.4 honors line directives
   ONLY for the panic and phantom analyzers —
   `cmd/branching-flow/container.go` wires `isSuppressed` solely for those
   two runners, and the composition output path has no suppression filter at
   all. The README's "All analyzers support two ignore comment styles" is
   inaccurate for dupe/composition in 0.6.4. Recorded in AGENTS.md and the
   config comment.
7. **All quality gates green at session end.** `go build ./...`, `go test
   ./...` (both packages), `go test -race ./...`, `go vet ./...`, `go vet
   -tags integration ./...`, `nix run .#lint` → **0 issues**, erraudit →
   **No violations**, `nix fmt` → 0 changed (stable), branching-flow →
   **0 active**.
8. **Docs updated.** AGENTS.md: stale "branching-flow is skipped by policy"
   note replaced with the three-tier disposition + suppression conventions
   (line directives work only for panic/phantom; directive placement rules;
   re-run the tool after formatting). CHANGELOG.md: internal bullet under
   0.5.0 (public surface unchanged). TODO_LIST.md: new v2-follow-up row.
   The 03-41 triage report annotated EXECUTED with the plan amendment.
9. **Lint regressions introduced and cleared within the session:** 31 lll
   (long rationale lines), 11 godoclint (misordered doc comments), 1
   nlreturn (blank line I dropped) — all fixed; final lint run clean.

## b) PARTIALLY DONE

1. **Plan v2's "15 fixes / 45 suppressions" landed as 13 / 16 / 31.** The
   fix half is 13, not 15: merging the fake pairs re-classified some
   same-package duplicates as cross-package mirrors (e.g.
   `savedViewRule` ↔ `savedViewFilterRulePayload`,
   `namedEntity` ↔ `namedPayload`) — genuinely unfixable without v2 API
   changes, so they moved from "fix" to "dismiss" rather than being
   eliminated. Nothing was lost; the categories shifted. Open: none
   technically, but the report/CHANGELOG numbers say 13 where the plan said
   15 — make sure that reads as re-classification, not shortfall.
~~2. **Rule-scoped nolint mechanism (`:panic`, `:phantom`) unprobed.** Source~~ → TODO_LIST Low (2026-10-11 harvest): rule-scoped nolint probe
   read confirms the suffix is lowercased and validated against the
   IgnoreCommentType enum, but I shipped bare `//nolint:branching-flow`
   (suppress-all) everywhere. Open: probe `:panic`/`:phantom` scoping so
   future directives suppress only the intended kind. Effort: S.
~~3. **v0.5.0 still untagged.** CHANGELOG/flake/README all say 0.5.0; the~~ → TODO_LIST release row (2026-10-11 harvest): tag decision is the owner's
   predecessor session's 3 questions (tag timing, consumer-migration
   ownership, ADR 0004 sign-off) are still unanswered and still block the
   release flow. Effort: S once answered.
~~4. **TODO_LIST harvest of the next-things list.** I added the one~~ done — harvested this pass (2026-10-11 docs-health)
   branching-flow-v2 row; the fuller (f) list below (50 items) is not yet
   harvested into TODO_LIST/ROADMAP. Per the status-report skill, section
   (f) feeds docs-health HARVEST — that pass should run after you release
   me. Effort: M.

## c) NOT STARTED

1. **Upstream branching-flow feedback** — the README inaccuracy and the
   missing `isSuppressed` wiring for dupe/composition. Not started; wants
   the verify-before-filing gate and a probe against the tool's own main
   branch (0.6.4 binary may already differ from repo main).
2. **CI gate for branching-flow.** The 0-active state is enforced by
   nothing automated; a CI job (or `nix flake check` app) would pin it.
   Blocked in practice by the unresolved `--exit-code` semantics (README
   says on-by-default; observed exit 0 with 60 findings) — must probe
   first.
3. **Pinning test for the deliberate wire delta.** Create-response for a
   rule-less saved view changed `null` → `[]`. No test asserts the exact
   JSON; only parsed round-trips pass. I judge it correct (DRF parity) but
   unpinned.
4. **crush-config `references/lessons.md` entries** (cross-project
   lessons, need commits in that repo): (a) probe suppression mechanisms
   per rule CLASS, not per mechanism; (b) standalone-directive-above-anchor
   is the only golines-stable nolint placement.
5. **ADR for the new fake convention** — "entity families store and serve
   the same tagged struct" contradicts the old entity/wire split convention
   that previous sessions established; nothing records the decision except
   CHANGELOG prose.
6. **v2 typed-API items** (brand `New`/`EnsureStoragePath` params, JSON-tag
   decision for public structs, drop the scaffolding) — ROADMAP scope,
   intentionally untouched this session.

## d) TOTALLY FUCKED UP!

1. **I generalized suppression evidence across rule classes.** The two
   probes (pre-flight) were panic-class and phantom-class findings. From
   them I concluded the mechanism worked and applied 27 trailing directives
   to dupe/composition targets — every one inert. The tool's own runner
   wiring (`container.go:40`: `newStandardRunner(LinterKeyDupe, …)` without
   suppression) proved it in a 2-minute read I did only AFTER the mass edit
   failed. Caught empirically (31 findings still active) before anything
   shipped wrong, but the batch was waste and cost three rework cycles.
   Severity: none at HEAD (corrected), high wasted effort. Root cause:
   treating a handoff's "suppression mechanism verified" as class-universal
   instead of re-verifying per class.
2. **`nix fmt` silently broke 15 of 16 working suppressions.** I anchored
   long trailing comments on lines near golines' width cap; golines
   exploded those lines multi-line and the directives stranded on the
   closing-paren lines, out of reach of the tool's line/line-1 matching.
   Worse: my first post-fmt verification piped `--include-suppressed` output
   into an "active" check and I misread it — briefly concluded the one
   working suppression (documents.go:71) was broken and the broken ones
   fine. The set-diff re-check exposed the truth. Severity: was silently
   broken between two commits; now corrected and stable across repeated
   fmt passes. Root cause: no post-format verification step in the plan
   (the 03-41 report even predicted "formatter risk" — I verified only once
   and misread the run).
3. **My first doc-fold script was defective twice over.** It batched by
   line numbers that its own first insertion invalidated (folded 1, skipped
   21), inserted the rationale at the TOP of doc comments (violating
   godoclint's name-first rule → 11 new lint findings), and used Python
   `capitalize()`, which lowercased SDK → Sdk, ROADMAP → Roadmap, API → api
   in 27 comments. Fixed by a second, pattern-based script that rebuilt all
   27 blocks deterministically with hand-written casing. Severity: none at
   HEAD; the intermediate states are in git history via the auto-commit
   daemon. Root cause: I ran a multi-step mutation script without dry-run
   or diff review.
4. **Self-inflicted lint regression:** an early multiedit on entities.go
   dropped a blank line before a return (nlreturn finding). Trivial, caught
   by lint, fixed — but it was my own edit, not pre-existing debt.
5. **History noise from mid-rework auto-commits.** The daemon committed
   broken intermediates (inert directives, exploded lines, mangled docs) as
   "chore: auto-commit N file(s)". Expected daemon behavior, not a
   violation, but the history now contains several states I would rather it
   didn't; the final state is correct and verified.

## e) WHAT WE SHOULD IMPROVE!

1. **Probe per rule CLASS, not per mechanism.** One probe per emitted rule
   ID family (panic, phantom, dupe, composition, antipattern) before any
   batch suppression. Would have saved the entire 27-edit detour.
2. **Read the gatekeeper's wiring before writing comments.** For
   suppression tooling, find the code that consumes the directive (here:
   `container.go` runner construction) before mass-applying. 2 minutes
   versus 3 rework cycles.
3. **Standalone directive line directly above the anchor is the only
   golines-stable nolint placement** in this repo. Trailing comments push
   lines over the cap and get exploded. This belongs in
   crush-config `references/lessons.md` and possibly the how-to-golang
   skill.
4. **One canonical verification run per checkpoint.** My mixed
   default/include-suppressed piping produced the misread. Rule: save the
   default run to a file, assert on that file, and never feed
   `--include-suppressed` output into an "active" assertion.
5. **Mutation scripts need a dry-run gate.** Any script inserting/moving
   lines should print a unified diff (or run on a copy) for review before
   writing. The capitalize()/top-insert defects would both have been caught
   in a diff review.
6. **Template-assigned rationale claims need per-site verification.** The
   27 doc comments were assembled from 11 reason templates; I spot-checked
   the load-bearing claim (public structs tag-free) only after writing the
   report of my own work — it held, but the check should have preceded the
   write.
7. **Post-format verification is a mandatory step, not a follow-up.** The
   plan listed "nix fmt then re-run" — it must also re-run the _analyzer_
   and re-assert the suppression set, which is what finally caught issue
   (d)2.
8. **The tool's `--format finding` JSON has no obvious suppressed marker**
   (I diffed two runs by set to count suppressions). Worth one more look
   and, if truly absent, folding into the upstream feedback item.

## f) Next things to get done (50, ranked by impact)

| #  | Task                                                                                                                                                                                      | Impact   | Effort | Category           |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------------ |
~~| 1  | Answer the 3 predecessor questions and tag v0.5.0 (`nix run .#release-verify -- v0.5.0`; CHANGELOG/flake/README already agree)                                                            | Critical | S      | Release            |~~ → TODO_LIST release row (2026-10-11 harvest)
~~| 2  | Migrate bank-sync to go-paperless v0.5.0 (branded IDs; compiler-driven; drop redundant int↔int64 conversions)                                                                             | High     | M      | Feature            |~~ already harvested — TODO_LIST consumer-migration row
~~| 3  | Migrate InboxClean to go-paperless v0.5.0 (same shape as #2)                                                                                                                              | High     | M      | Feature            |~~ already harvested — TODO_LIST consumer-migration row
~~| 4  | Probe `branching-flow --exit-code` semantics (README claims on-by-default; observed 0 with 60 findings) and record the truth in AGENTS.md                                                 | High     | S      | Quality            |~~ → TODO_LIST (2026-10-11 harvest): exit-code probe
~~| 5  | Wire the 0-active state into CI or `nix flake check` (depends on #4; candidate: `--baseline` SARIF gate failing only on new findings)                                                     | High     | M      | Quality            |~~ → TODO_LIST Medium (2026-10-11 harvest): CI/baseline gate
~~| 6  | File upstream branching-flow issue: README overclaims line-suppression for dupe/composition (after verify-before-filing against tool main)                                                | Medium   | S      | Bug (upstream)     |~~ → TODO_LIST Low (2026-10-11 harvest): upstream issue
~~| 7  | File upstream feature request: `isSuppressed` wiring for dupe/composition/antipattern runners (mirror the panic/phantom path in container.go)                                             | Medium   | S      | Feature (upstream) |~~ → TODO_LIST Low (2026-10-11 harvest): upstream feature request
~~| 8  | Add pinning test: saved-view create without rules answers `"filter_rules":[]` (exact JSON), list keeps `[]`                                                                               | Medium   | S      | Quality            |~~ → TODO_LIST (2026-10-11 harvest): saved-view pinning test
~~| 9  | Add pinning test: `slices.DeleteFunc` disarm removes exactly the consumed fault (multiple armed faults, one Times=1) — direct unit test                                                   | Medium   | S      | Quality            |~~ → TODO_LIST Low (2026-10-11 harvest): fault-disarm unit test
~~| 10 | Probe rule-scoped nolint (`:panic`, `:phantom`); if working, narrow the 16 bare directives to their kinds                                                                                 | Medium   | S      | Quality            |~~ → TODO_LIST Low (2026-10-11 harvest): rule-scoped nolint probe
~~| 11 | Write ADR: paperlesstest entity families store-and-serve one tagged struct (replaces the entity/wire split convention; notes/uploads families stay split because shapes genuinely differ) | Medium   | S      | Documentation      |~~ → TODO_LIST Medium (2026-10-11 harvest): store==serve ADR
~~| 12 | Add crush-config lessons.md: probe suppression per rule class (needs commit in that repo)                                                                                                 | Medium   | S      | Process            |~~ routed — crush-config lessons.md (external repo, commit required)
~~| 13 | Add crush-config lessons.md: standalone-directive-above-anchor under golines (needs commit in that repo)                                                                                  | Medium   | S      | Process            |~~ routed — crush-config lessons.md (external repo, commit required)
~~| 14 | HARVEST this report's (f) into TODO_LIST.md / ROADMAP.md (docs-health HARVEST mode)                                                                                                       | Medium   | M      | Documentation      |~~ done — harvested this pass (2026-10-11 docs-health)
~~| 15 | Investigate why `--format finding` JSON lacks a suppressed marker; fold finding into #6/#7 feedback                                                                                       | Low      | S      | Quality            |~~ → TODO_LIST Low (2026-10-11 harvest): finding-JSON marker
~~| 16 | Restart Crush so gopls/golangci-lint reload with the pinned GOTOOLCHAIN (stale "go-branded-id not in go.mod" + example_test diagnostics are pre-session LS state, contradict green gates) | Low      | S      | Cleanup            |~~ routed — Crush restart is a session-start action, not repo work
~~| 17 | Verify AGENTS.md's paperlesstest conventions section reflects the merged store==serve structs (I updated the branching-flow note, not the convention text)                                | Low      | S      | Documentation      |~~ done — verified this pass: AGENTS.md conventions reflect the merged store==serve structs
~~| 18 | Consider CONTRIBUTING.md note: branching-flow gate, `.branching-flow.yml`, and the nolint conventions for contributors                                                                    | Low      | S      | Documentation      |~~ → TODO_LIST Low (2026-10-11 harvest): CONTRIBUTING branching-flow note
~~| 19 | Re-check FEATURES.md for stale "entity/wire split" wording about paperlesstest                                                                                                            | Low      | S      | Documentation      |~~ done — verified this pass: FEATURES carries no stale entity/wire wording
~~| 20 | Decide whether the CHANGELOG internal bullet belongs under 0.5.0 (ships with the tag) vs Unreleased — current placement ships it with v0.5.0                                              | Low      | S      | Documentation      |~~ routed — placement consistent: the internal bullet ships with untagged 0.5.0
~~| 21 | ROADMAP: link "Growing with the API" to the concrete v2 items the suppressions encode (#22–#25)                                                                                           | Low      | S      | Documentation      |~~ → TODO_LIST Low (2026-10-11 harvest): ROADMAP v2 cross-link
~~| 22 | v2: brand `New(baseURL, token)` params → drop 1 transpose nolint                                                                                                                          | Low      | M      | Feature            |~~ already harvested — TODO_LIST v2 row
~~| 23 | v2: brand `EnsureStoragePath(name, path)` params → drop 1 transpose nolint                                                                                                                | Low      | M      | Feature            |~~ already harvested — TODO_LIST v2 row
~~| 24 | v2: JSON-tag decision for public structs → drop 8 DUPLICATE_TYPE dismissals + wire-mirror rationales                                                                                      | Low      | L      | Feature            |~~ already harvested — TODO_LIST v2 row
~~| 25 | v2: unify public structs with payloads (embedding/embedded tags) → drop ~10 mixin rationales                                                                                              | Low      | L      | Feature            |~~ already harvested — TODO_LIST v2 row
~~| 26 | v2: decide cross-package wire-shape sharing (exported wire types vs permanent fake mirrors) → drop 2 cross-pkg dismissals                                                                 | Low      | M      | Feature            |~~ already harvested — TODO_LIST v2 row
~~| 27 | When branching-flow ships dupe/composition line-suppression: replace disabled_rules with per-site nolints (config comment already points here)                                            | Low      | M      | Cleanup            |~~ routed — TODO_LIST v2 row (scaffolding drop)
~~| 28 | Meta-test idea: flake check script that runs branching-flow and diffs against an expected-suppressed manifest in-repo (catches formatter-induced breaks mechanically)                     | Medium   | M      | Quality            |~~ → TODO_LIST Medium (2026-10-11 harvest): suppressed-manifest gate
~~| 29 | Add `nix fmt` + branching-flow re-verify to the session checklist in AGENTS.md (already partially there; make the re-assert explicit)                                                     | Low      | S      | Process            |~~ done — AGENTS.md documents the fmt-then-re-verify suppression rule
~~| 30 | Watch item: golangci "Found unknown linters: branching-flow" warning — confirm nolintlint stays disabled so it never hard-fails                                                           | Low      | S      | Quality            |~~ → TODO_LIST Low (2026-10-11 harvest): nolintlint watch
~~| 31 | Confirm `documents.go` Document/documentWire split is still the right call (public fixture input vs wire shape) and say so in the ADR from #11                                            | Low      | S      | Documentation      |~~ → TODO_LIST Low (2026-10-11 harvest): Document split re-check
~~| 32 | Sweep for other near-cap comment lines that a future plugin bump could explode (the dprint-plugin-bump lag is a known pattern in this repo)                                               | Low      | S      | Cleanup            |~~ → TODO_LIST Low (2026-10-11 harvest): near-cap comment sweep
~~| 33 | After the daemon commits the 4 doc files: one final `git status` + gate sweep to close the session                                                                                        | Low      | S      | Process            |~~ routed — standing session-close rule
~~| 34 | Consider renaming `savedViewCreateWire` alias or documenting why the input-shape alias survives the merge (self-documenting? verify readers agree)                                        | Low      | S      | Cleanup            |~~ → TODO_LIST Low (2026-10-11 harvest): alias naming
~~| 35 | Check whether `TaskPlan`/`Upload`/`taskRecord` families deserve the same store==serve treatment later (shapes differ today — verify, don't assume)                                        | Low      | S      | Cleanup            |~~ → TODO_LIST Low (2026-10-11 harvest): store==serve family re-check
~~| 36 | Report the "historical nolint-hides behavior" comment vs composition-path divergence to upstream as part of #7 context                                                                    | Low      | S      | Bug (upstream)     |~~ → TODO_LIST Low (2026-10-11 harvest): upstream context
~~| 37 | Evaluate `--baseline` SARIF as the CI mode if #4 shows exit-code semantics are unusable                                                                                                   | Low      | S      | Quality            |~~ → ROADMAP (2026-10-11 harvest): SARIF baseline mode
~~| 38 | Consider publishing the expected-suppressed manifest as a small `docs/BRANCHING_FLOW.md` for humans (config + AGENTS note may suffice — decide)                                           | Low      | S      | Documentation      |~~ → TODO_LIST Low (2026-10-11 harvest): BRANCHING_FLOW.md decision
~~| 39 | Re-run `nix run .#coverage` once post-tag to confirm no coverage regression from the merges (deleted structs shouldn't move it, but verify)                                               | Low      | S      | Quality            |~~ → TODO_LIST Low (2026-10-11 harvest): post-tag coverage run
~~| 40 | Ask upstream whether `Matching`'s line-1 rule is contractual (my placement strategy depends on it)                                                                                        | Low      | S      | Quality (upstream) |~~ → TODO_LIST Low (2026-10-11 harvest): Matching line-1 rule
~~| 41 | Add a regression note to the 03-41 triage report's "formatter risk" line pointing at what actually happened (teacherly cross-reference)                                                   | Low      | S      | Documentation      |~~ done — 03-41 formatter-risk line cross-references this report's d-2
~~| 42 | Sweep the repo for `//nolint:branching-flow` directives that no longer suppress anything after future refactors (tie to #28's manifest)                                                   | Low      | S      | Cleanup            |~~ → TODO_LIST Low (2026-10-11 harvest): stale-directive sweep
~~| 43 | Confirm `slices.DeleteFunc` order-preservation is intentional for fault priority ("first matching wins") and pin with a two-fault test if cheap                                           | Low      | S      | Quality            |~~ → TODO_LIST Low (2026-10-11 harvest): DeleteFunc order pin
~~| 44 | Document in AGENTS.md that bare `//nolint:branching-flow` suppresses ALL branching-flow findings on the line (suppress-all semantics, from parseNolintComment)                            | Low      | S      | Documentation      |~~ done — AGENTS.md documents bare-nolint suppress-all semantics (this pass)
~~| 45 | Once consumers migrate (#2/#3): collect their hand-rolled fakes and replace with paperlesstest per the AGENTS.md contract                                                                 | Low      | L      | Cleanup            |~~ → TODO_LIST Low (2026-10-11 harvest): consumer fake replacement
~~| 46 | Evaluate golines max-width pinning in flake.nix (explicit > default) so comment-cap behavior is stable across nixpkgs bumps                                                               | Low      | S      | Quality            |~~ → ROADMAP (2026-10-11 harvest): golines width pinning
~~| 47 | Re-check that `example_test.go` LS errors vanish after the Crush restart (#16); if they persist in a fresh session, investigate for real                                                  | Low      | S      | Bug                |~~ → TODO_LIST Low (2026-10-11 harvest): LS-error re-check
~~| 48 | Consider a tiny `make`-less flake app `.#triage` wrapping `branching-flow all . --format finding` for one-command re-verification                                                         | Low      | S      | Quality            |~~ → TODO_LIST Low (2026-10-11 harvest): .#triage app
~~| 49 | Clean stale `/tmp/bf-*.json` scratch artifacts (session hygiene; they are outside the repo, so optional)                                                                                  | Low      | S      | Cleanup            |~~ Won't implement — /tmp scratch is ephemeral and machine-local
~~| 50 | Close the loop on this report: annotate it EXECUTED-HARVESTED once #14 runs                                                                                                               | Low      | S      | Documentation      |~~ done — annotated EXECUTED-HARVESTED this pass

**HARVEST note:** items 1–5, 8–14 are TODO_LIST-grade; 22–27, 40 belong in
ROADMAP under "Growing with the API"; the rest are small cleanups. Run
docs-health HARVEST to route them.

## g) Questions I can NOT figure out myself

1. **When do we tag v0.5.0 — now, or after the consumer migrations?** The
   repo is internally consistent at 0.5.0 (CHANGELOG, flake version, README)
   and fully green, but the release-verify flow also checks push status and
   pkg.go.dev. I cannot decide whether you want the tag to precede the
   bank-sync/InboxClean bumps (so consumers can pin immediately) or follow
   them (so the tag certifies migrated consumers). This has blocked the
   release flow since the predecessor session.
2. **Who owns the consumer migrations (bank-sync, InboxClean to branded
   IDs)?** If you want me to do them in follow-up sessions I will start with
   the compiler-driven sweep; if the consumer repos have their own owners or
   you want to drive them, I should not touch those pins.
3. **Should the fake's wire delta (saved-view create with no rules now
   answers `"filter_rules":[]` instead of `null`) be kept and pinned, or do
   you want strict byte-for-byte parity with the pre-merge fake?** I chose
   DRF parity (the real server never emits `null` for nested serializer
   lists) and the SDK parses both identically — but if a consumer's
   hand-rolled tests assert on the raw `null`, this change would surface
   there, and only you know what those tests expect.

---

_Prepared 2026-10-11 05:02 CEST. Waiting for instructions._
