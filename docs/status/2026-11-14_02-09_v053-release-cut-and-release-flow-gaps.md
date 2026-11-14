# Status Report: v0.5.3 Release Cut and Release-Flow Audit

**2026-11-14 02:09 CET** · go-paperless · session scope: the v0.5.3 release
(`f56b2e4`) plus the release-flow gaps it exposed. Companion to
`2026-11-14_01-32_paperlesstest-art-dupl-zero-actionable-dedup.md` (the
immediately preceding session; its open questions reappear in (g)).

## Scope line

**Verified this session (fresh command output, quoted where load-bearing):**

- gorelease vs `v0.5.2`: `Suggested version: v0.5.3`, zero additions, zero
  breaks (`nix develop -c env GOFLAGS= go run golang.org/x/exp/cmd/gorelease@latest -base=v0.5.2`).
- Full test suite green locally after the version bump (`nix run .#test`:
  both packages ok, `TestFlakeVersionMatchesChangelog` included).
- CI green on the exact release commit: run `43535672998`, 12m53s,
  completed/success, on `f56b2e4` (read back from `gh run list`).
- Tag `v0.5.3` is annotated, SSH-signed, and points at `f56b2e4`
  (`git rev-parse v0.5.3^{commit}` == `git rev-parse HEAD` at tag time;
  `%(contents)` shows the SSH signature block).
- Proxy indexed the version at the right commit:
  `proxy.golang.org/.../@v/v0.5.3.info` → `{"Version":"v0.5.3",…,"Hash":"f56b2e4…","Ref":"refs/tags/v0.5.3"}`.
- `nix run .#release-verify -- v0.5.3` → **ALL GREEN**: tag exists,
  CHANGELOG+flake agree, tree clean and pushed, CI green, pkg.go.dev
  indexed, race suite green over the tagged worktree.
- Clean-dir consumer test: fresh module in `/tmp/release-verify`,
  `go get github.com/larsartmann/go-paperless@v0.5.3` + a one-line
  `paperless.New` reference compiles ("CONSUMER TEST GREEN").
- GitHub Releases: v0.5.3 created and marked `Latest`; v0.5.1/v0.5.2
  backfilled (`gh release list` read back after the fix).
- Full tag↔release set equality (checked during report writing, not during
  the cut): 5 tags lack Releases — v0.1.0, v0.3.2, v0.4.0, v0.4.1, v0.4.2
  (per-tag `gh release view` loop, output quoted in (b)).

**NOT verified here (and why):**

- CI on the docs commit `cf83641` (run `43538242206`) was still
  `in_progress` at write time (6m57s in). Docs-only change; the release
  commit's CI is green, which is the gate that mattered for tagging.
- Full local `nix flake check` was NOT run this session — only
  `nix run .#test`. Rationale: the authoritative gate per the release skill
  is CI on the exact commit, which was green; the previous session ran
  `nix flake check` green on the same Go tree (only CHANGELOG/flake.nix
  version lines changed since). Named as memory, not evidence.
- `erraudit` not run this session: no Go files changed (only
  CHANGELOG.md, flake.nix, AGENTS.md, TODO_LIST.md).
- Consumer repos (bank-sync, journal/InboxClean) not touched and not
  inspected — out of scope; state taken from TODO_LIST rows.
- Whether v0.1.0's missing Release is intentional: ~~unknown (see (g)).~~ answered by owner directive 2026-11-14: backfill all — done.

---

## a) FULLY DONE

1. **v0.5.3 release cut, end to end.** Assessed 46+ commits since v0.5.2;
   bump class decided by evidence, not vibes (gorelease: no API changes →
   PATCH). CHANGELOG `[Unreleased]` folded into `## [0.5.3] - 2026-11-14`
   (repo's bare-`[Unreleased]` convention preserved), flake.nix
   `version = "0.5.3"`, release commit `f56b2e4`, pushed, CI green on that
   commit, annotated+SSH-signed tag, pushed. Scope: CHANGELOG.md,
   flake.nix, git tag v0.5.3.
2. **release-verify ALL GREEN** over the tagged tree, including the race
   suite in a clean worktree of `v0.5.3`. This is the project's own
   sanctioned release gate; output quoted in the scope line.
3. **Proxy and pkg.go.dev confirmed.** `.info` endpoint returns the exact
   commit hash `f56b2e4…`; `release-verify`'s pkg.go.dev step passed.
   (Initial `/fetch` 404 was propagation delay, retried via the later gate.)
4. **Clean-dir consumer test green.** Fresh module, `go get …@v0.5.3`
   (sumdb-verified), compiles against the package. The definitive
   consumer-facing check.
5. **GitHub Release v0.5.3 live and `Latest`.** Title
   "v0.5.3 — FK-clearing PATCH fix, api-diff gate, QA depth"; notes
   drafted to /tmp and passed via `--notes-file`; `--latest` explicit.
6. **v0.5.1 and v0.5.2 Release entries backfilled** from their CHANGELOG
   sections, with a backfill provenance note in each body.
7. **Release-flow lessons recorded in AGENTS.md** (commit `cf83641`):
   release-verify's Releases-page blind spot, the `gh release create`
   latest-flag steal, the proxy `.info` check (vendor mode breaks
   `go list -m -versions` in-repo).
8. **TODO_LIST row 37 retargeted** (same commit): the done release-cut
   half removed per the table's DONE rule ("remove + log in CHANGELOG",
   satisfied by `[0.5.3]`), the consumer-bump half kept as its own
   actionable row; dprint-formatted, diff verified content-safe.

## b) PARTIALLY DONE

1. **GitHub Releases page completeness — ~~9 of 14 version tags covered~~ 13 of 13 (RESOLVED 2026-11-14 ~02:20, owner directive "Backfill all missing GitHub Releases!").**
   ~~Works: v0.5.0–v0.5.3, v0.3.0, v0.3.1, v0.2.0, v0.1.1 have Releases.
   Missing (verified by per-tag `gh release view`): **v0.1.0, v0.3.2,
   v0.4.0, v0.4.1, v0.4.2**.~~ All five backfilled from their CHANGELOG
   sections with explicit `--latest=false`; post-backfill per-tag
   `gh release view` loop resolves 13/13 tags, no steals, v0.5.3 remains
   `Latest`. (Report's original count was wrong: 8 had Releases, not 9;
   13 tags total, not 14 — arithmetic corrected here, not rewritten.)
2. **Consumer propagation to v0.5.3.** Works: the TODO row exists, is
   High/Medium-impact, and now names the two-step vendor tail. Not done:
   the actual bumps (bank-sync, journal/InboxClean), because timing is an
   owner decision and consumer Actions CI is billing-dead (TODO row 36),
   which constrains "re-prove their suites". Effort: M + owner input.
3. **CI on docs commit `cf83641`.** In progress at write time
   (run `43538242206`). Nothing to do unless it goes red; docs-only diff.

## c) NOT STARTED

1. **Mechanical release-page enforcement** — extend `release-verify` with
   a Releases-page check and/or add a `release-create` flake app (draft
   notes from the CHANGELOG section, `gh release create --latest`,
   set-equality verify). Currently prevented only by prose in AGENTS.md.
2. **Proxy `.info` step inside release-verify** — pkg.go.dev lags the
   proxy; `.info` is the earlier, commit-exact signal.
3. **Full tag↔release set-equality in the release checklist** — the check
   that caught (b)(1) is not written down anywhere mechanical.
4. **Backfilling the 5 old Releases** (subset of (b)(1) execution).
5. **Carry-over, untouched this session** (state from TODO_LIST/prior
   report, not re-researched): art-dupl baseline gate + CI threshold;
   dedup scope-floor decision; savedviews/sharelinks prose de-dup;
   `rejectWrongMethod` negative test; full buildflow loop; erraudit
   hermetic pin (blocked on public proxy); govalid-skip removal watch;
   branching-flow/v2 typed-API revisit; consumer Actions billing fix.

## d) TOTALLY FUCKED UP

1. **v0.5.1 and v0.5.2 shipped with no GitHub Release entry — for 7
   days.** The 2026-11-07 cuts produced tags, proxy versions, and CHANGELOG
   sections but nothing on the Releases page, and nothing caught it:
   `release-verify` checks tag/CHANGELOG/flake/push/CI/pkg.go.dev/race and
   is structurally blind to the Releases page. Severity: moderate —
   release *discovery* was broken (the page humans and some tooling read),
   downloads were never broken. Root cause: the release flow had no
   mechanical gate for this artifact. Mitigation: backfilled; lesson in
   AGENTS.md. Residual: **it can recur tomorrow** — the guard is prose.
2. **My backfill stole the repo's `latest` release flag.** Creating
   v0.5.1's entry without an explicit latest flag made v0.5.1 — not
   v0.5.3 — the repo's Latest release for about a minute. Self-inflicted,
   caught on the very next `gh release list`, fixed with
   `gh release edit v0.5.3 --latest=true`. Root cause: I applied `--latest`
   only to v0.5.3 and trusted auto-pick for the rest; GitHub's auto-pick
   then did **not** choose the highest semver (picked v0.5.1 over v0.5.3 —
   the exact picker behavior is unknown to me). Lesson: never trust
   auto-pick; set `--latest`/`--latest=false` on every create.
3. **My backfill was itself incomplete, and I didn't know it.** I backfilled
   the two gaps visible at the top of `gh release list` and moved on;
   full set-equality against `git tag` — run only during this report —
   showed **five** missing Releases, not two. Process failure in one
   sentence: I verified the visible, not the complete. The same trap the
   scope-line rule exists for.
4. **The FK-fix exposure window was ~6 days by policy.** Consumers pin
   v0.5.2, whose FK-clearing PATCH silently no-ops; the fix has been on
   main since ~2026-11-08 but only reached a tag today. Any consumer
   reconcile pass in that window "successfully" failed to clear foreign
   keys. Not a code or process accident — the direct cost of the
   batched-cut cadence, now quantified. (Backfill caveat: consumers may
   also have been pinned even older; only go.mod v0.5.2 pins are verified,
   per the TODO row.)

Nothing else broke this session: no failed gates, no re-tags, no checksum
incidents, no force-pushes, no dirty-tree fights beyond the daemon's
expected mid-flight commits.

## e) WHAT WE SHOULD IMPROVE

1. **Make the Releases page a gate, not a memory.** AGENTS.md prose decays
   exactly the way this gap proved: the 2026-11-07 cuts had the skill
   available and still skipped the step. Concrete: `release-verify` grows
   a `gh release view vX.Y.Z` step (and optionally a set-equality sweep);
   or a companion `release-create` app generates notes from the CHANGELOG
   section and creates the Release with explicit flags.
2. **Set-equality over list-head sampling.** Every "are all releases
   present" check should be `git tag` ∩ `gh release list` diffed, never
   "the list looks fine at the top". Cheap command, should be in the
   post-release checklist (and (e)(1)'s gate).
3. **Explicit latest flags on every `gh release create`.** Auto-pick
   demonstrated unreliable semantics. Mechanical rule: `--latest` on the
   new release, `--latest=false` on every backfill.
4. **A wire-correctness SLA.** The FK class (silent success on a broken
   wire shape) deserves a cadence rule — e.g. cut within 24h of such a fix
   landing on main. Batched cuts are fine for internal/QA work; they are
   expensive for consumer-visible correctness. This session's (d)(4) is
   the quantified argument.
5. **Add the proxy `.info` probe to release-verify** — commit-exact and
   available minutes before pkg.go.dev indexes; today the flow discovers
   propagation via a 404 and a retry.
6. **Document the backfill note convention** (`*Backfilled release entry —
   …*` italic footer) in AGENTS.md if manual backfills remain the path;
   drop the convention entirely if (e)(1) automates them.

## f) Up to 50 things to get done next

Ranked by impact. Effort: S <30min, M 30min–2h, L >2h. Category in
brackets. **This section is HARVEST input** (`docs-health` → HARVEST):
items 1–2 and 15 are already in TODO_LIST (row 37 retarget, row 36);
items gated on (g) answers should wait for them; the rest route to
TODO_LIST or ROADMAP per rigor.

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| ~~1~~ | ~~Backfill the 5 missing Releases (v0.1.0, v0.3.2, v0.4.0, v0.4.1, v0.4.2) from their CHANGELOG sections, explicit `--latest=false`~~ DONE 2026-11-14 02:20 — 13/13 verified, v0.5.3 Latest intact | ~~High~~ | ~~S~~ | Cleanup |
| 2 | Bump bank-sync to v0.5.3: go.mod, `go mod vendor` + `buildflow -s nix-hash-fix --fix`, re-prove suite | High | M | Feature |
| 3 | Bump journal/InboxClean to v0.5.3, same two-step tail + suite re-proof | High | M | Feature |
| 4 | Extend `release-verify` with a Releases-page existence check (fails when `gh release view vX.Y.Z` 404s) | High | S | Quality |
| 5 | Drop papersync's `retryafter.Clamp` for `MaxDelay: 60s` during the bank-sync bump (v0.5.2 made it redundant) | Medium | S | Cleanup |
| 6 | Add tag↔Release set-equality sweep to the release checklist (one `comm` command, documented in AGENTS.md release bullet) | Medium | S | Quality |
| 7 | Resolve consumer Actions billing (TODO row 36) — owner action; unblocks suite re-proof in CI | High | S | Bug |
| 8 | Add proxy `.info` step to `release-verify` before the pkg.go.dev step | Medium | S | Quality |
| 9 | Add a `release-create` flake app: draft notes from the `[X.Y.Z]` CHANGELOG section, `gh release create --latest`, run set-equality | Medium | M | Feature |
| 10 | Confirm CI green on docs commit `cf83641` (run 43538242206) and archive the evidence in this report's follow-up | Low | S | Documentation |
| 11 | Decide + record the wire-correctness SLA (immediate patch cut for consumer-visible wire bugs) in AGENTS.md | Medium | S | Documentation |
| 12 | Carry-over: art-dupl baseline gate policy — `.art-dupl-baseline.json` + CI job at `-t 1` vs prose sweeps (owner question open since 01-32 report) | Medium | M | Quality |
| 13 | Carry-over: record the dedup scope-floor decision (scanner-flagged only vs proactive near-miss like `writeNotFound`) in AGENTS.md | Medium | S | Documentation |
| 14 | Carry-over: savedviews/sharelinks prose de-dup (leftover from the dedup session) | Low | S | Cleanup |
| 15 | Carry-over: `rejectWrongMethod` negative test | Medium | S | Quality |
| 16 | Carry-over: full buildflow loop on this repo (last full run predates the docs commits) | Medium | M | Quality |
| 17 | Watch fuzz schedule: next scheduled run after the 2026-11-08 corpus additions (runs have been green; confirm the new seeds exercise) | Low | S | Quality |
| 18 | Triage the two Dependabot update runs from 2026-11-08 (go_modules + github_actions both succeeded — check what they bumped) | Low | S | Cleanup |
| 19 | CHANGELOG placeholder convention: decide bare `[Unreleased]` (current repo style) vs skill-style placeholders, write the answer into CONTRIBUTING | Low | S | Documentation |
| 20 | Document the backfill note format in AGENTS.md if manual backfills persist (superseded if #9 lands) | Low | S | Documentation |
| 21 | Carry-over: erraudit hermetic flake pin — re-check whether erraudit reached the public proxy (unblocks CI job; currently host-profile-resolved) | Medium | S | Quality |
| 22 | Carry-over: remove the govalid-generate skip once the machine rebuilds on Go ≥ 1.27 | Low | S | Cleanup |
| 23 | Carry-over: branching-flow v2 revisit — re-run dispositions when line-suppression ships for dupe/composition (v0.6.7 honors panic/phantom only) | Low | L | Quality |
| 24 | Carry-over: v2 typed-API migration design (ROADMAP "Growing with the API"; decides FK-clearing contract, branded string params) | Low | L | Feature |
| 25 | Re-run `nix flake check` locally post-release for a fresh local all-gates artifact (CI is green; this is belt-and-suspenders for the release commit) | Low | M | Quality |
| 26 | Add a release smoke test to CONTRIBUTING: the exact command sequence this session used (gorelease → bump → CI watch → tag → proxy .info → gh release → release-verify) | Medium | S | Documentation |
| 27 | Verify MIGRATION.md renders correctly on pkg.go.dev for v0.5.3 (file is repo-root; confirm it's served in the doc set) | Low | S | Documentation |
| 28 | Consider tagging `v0.5.3` in consumers' changelogs/notes when bumps land (part of #2/#3, easy to forget) | Low | S | Documentation |
| 29 | Audit `gh release` titles for consistency (`vX.Y.Z — headline` em-dash style is now used by 5 of 9; normalize the older five during #1) | Low | S | Cleanup |
| 30 | Decide whether `release-verify` should also assert the tag is SSH-signed (currently implicit via git config; a config regression would go unnoticed) | Low | S | Quality |

Honest stop at 30: the remaining known items are already represented above
(consumer work #2/#3/#5/#7, release tooling #4/#6/#8/#9, carry-overs
#12–#16, #21–#24). Padding to 50 would mean inventing work no session
observed — the 01-32 report made the same call and it was the right one.

## g) Questions I cannot answer myself

1. **Consumer bump timing vs billing.** Should I run the bank-sync and
   journal/InboxClean bumps to v0.5.3 now (TODO row 37, High/M) — and if
   yes, is local suite re-proof acceptable while consumer Actions is
   billing-dead (row 36), or does the billing fix land first? I cannot
   decide consumer priorities, and I cannot pay the GitHub bill. This
   gates the only consumer-visible reason v0.5.3 exists.
2. ~~**Releases-page intent for the old five.**~~ ANSWERED 2026-11-14 by owner directive: backfill ALL, including v0.1.0 — no intentional-era carve-out exists.
3. **Carry-over, still unanswered from the 01-32 report:** enforce the
   paperlesstest 0-actionable state mechanically (`.art-dupl-baseline.json`
   + CI `art-dupl check` at `-t 1`) vs keep prose+manual sweeps, and is
   the dedup scope floor scanner-flagged-only or does it include proactive
   near-miss consolidations? Both are owner-preference questions I
   explicitly cannot settle from the codebase; they gate f#12/#13.

---

**HARVEST note:** (f) is the input for `docs-health` → HARVEST. The
TODO_LIST retarget this session (row 37) already covers f#2/#3/#5/#7 and
row 36 covers f#7; items f#1, #4, #6, #8–#11, #19, #20, #26, #29, #30 are
new and TODO_LIST-eligible; the watch items (#17, #21, #22) belong in
ROADMAP or the AGENTS.md watch lists. Per the standing instruction this
report ends in WAIT — harvest runs when you say so or answer (g).

---

## Appendix: backfill execution (2026-11-14 ~02:20 CET, owner directive)

All five missing Releases created from their CHANGELOG sections, every one
with explicit `--latest=false` (per the flag-steal lesson in (e)(3)):

| Tag | Title | CHANGELOG section |
|-----|-------|-------------------|
| v0.1.0 | initial release: Paperless-ngx REST client (API v10) | `[0.1.0]` |
| v0.3.2 | EnsureTag self-heal PATCH no longer rejected with 400 | `[0.3.2]` (incl. the recovered-verbatim blockquote) |
| v0.4.0 | paperlesstest fake, drift guards, fuzzers, typed saved-view rules | `[0.4.0]` (Added/Changed/Fixed/Verified) |
| v0.4.1 | paperlesstest fixture seeding and in-place reshaping | `[0.4.1]` |
| v0.4.2 | paperlesstest introspection for consumer migrations | `[0.4.2]` |

Evidence: `gh release create` returned all five URLs; the per-tag
`gh release view` loop over `git tag` resolves **13/13** with zero
"STILL MISSING" lines; `gh release list` shows v0.5.3 still marked
`Latest` (no steal this time — explicit flags on every create).
Releases page: https://github.com/LarsArtmann/go-paperless/releases
