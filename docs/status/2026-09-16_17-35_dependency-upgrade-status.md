# Status Report — Dependency Upgrade (go-error-family, go-retry, toolchain)

|                      |                                                                                                                                                                                       |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Date**             | 2026-09-16 17:35 CEST                                                                                                                                                                 |
| **Repo / Branch**    | `go-paperless` @ `main`, HEAD `057e3a0` (daemon-swept, moving)                                                                                                                        |
| **Session**          | Crush (`go-ecosystem-upgrade` + `buildflow` skills loaded; repo NOT BuildFlow-covered)                                                                                                |
| **Scope**            | Single task: "make sure we use the latest version superbly" — deps + toolchain of go-paperless only                                                                                   |
| **Format note**      | User explicitly requested `.md`; the status-report skill's canonical format is HTML. Override honored per skill contract.                                                             |
| **Parallel session** | A sibling Crush session worked this same repo concurrently (integration test scaffold + CI). Its uncommitted files at report time: `.github/workflows/ci.yml`, `integration_test.go`. |

---

## Executive Summary

Both direct dependencies bumped to latest and verified by the full hermetic battery. Go toolchain floor was already latest (go1.27.1 is the newest stable release; `go 1.27` floor correct). One commit (`5f8d07b`) contains the bump — tangled with a sibling session's vendorHash fix and two doc edits by the auto-commit daemon. All checks green at check time, but the green certifies a moving tree. No consumer impact until a release is cut.

---

## a) FULLY DONE

| Item                                                                         | Evidence                                                                                                                                                                                                                                                                        |
| ---------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Researched latest versions via module proxy + go.dev (not guessed)           | `go list -m -versions` for both deps; go.dev/dl JSON shows go1.27.1 latest stable                                                                                                                                                                                               |
| Reviewed both upstream release diffs for breaking changes before bumping     | `gh api compare v0.10.0...v0.10.1`, `v0.5.0...v0.6.0`; all root-module patches of error.go/family.go/handle.go/http.go/classify.go/example_test.go inspected                                                                                                                    |
| Baseline established BEFORE change                                           | `nix run .#build` + `.#test` green pre-bump (no pre-existing failures)                                                                                                                                                                                                          |
| `go-error-family` v0.10.0 → **v0.10.1**                                      | go.mod grep-verified; `go mod verify` = "all modules verified"                                                                                                                                                                                                                  |
| `go-retry` v0.5.0 → **v0.6.0**                                               | go.mod grep-verified; go.sum updated                                                                                                                                                                                                                                            |
| `go 1.27` floor preserved (major.minor only — no patch-floor leak from deps) | `go mod edit -go=1.27` normalization; file re-read after                                                                                                                                                                                                                        |
| Full verification battery                                                    | build ✅ test ✅ vet ✅ host `-race` ✅ sandboxed `checks.test-race` ✅ golangci-lint ✅ treefmt ✅ erraudit (printed "No violations found" 15:45; exit was pipeline-masked — §d1; gate BROKEN later 2026-09-16 by erraudit v0.4.0 upstream — AGENTS.md) ✅* `go mod verify` ✅ |
| `nix flake check` (CI-equivalent incl. sibling's new `integration-vet`)      | "all checks passed!"                                                                                                                                                                                                                                                            |
| vendorHash FOD updated for new module set                                    | `sha256-oLknr8l…` → `sha256-mk/dJNF…`, proven by two successful sandboxed builds (authored by sibling session — see d2)                                                                                                                                                         |
| No stale version references in living surfaces                               | swept go.mod/go.sum/flake.nix/README/AGENTS/CI/dprint/.golangci — none; AGENTS' "go 1.26.7" mentions describe the host machine, deliberately accurate                                                                                                                           |
| CHANGELOG `[Unreleased] → Changed` entry                                     | added, now committed by daemon                                                                                                                                                                                                                                                  |
| Version-surface inventory                                                    | CI uses `go-version-file: go.mod` (no drift possible); README badge "Go 1.27+" still correct; dependabot config noted                                                                                                                                                           |

## b) PARTIALLY DONE

1. ~~**Upstream diff review completeness.** I viewed 6 of 8 changed root-level `.go` files in error-family v0.10.1 and the retry.go patch only up to a truncation cut. Compensated by the full behavioral suite passing, but the review itself is incomplete (corrections in §Honesty).~~ done at continuation 2026-09-16 ~18:00 — full retry.go patch (78 lines: pure constants extraction, identical values) and both `examples/cmd/*` patches (comment/formatting + additive `WithContext`) viewed; all 8 changed files across both upgrades now reviewed.
2. ~~**Consumer impact verification.** Reasoned (sound: semver-compatible dep bumps don't change go-paperless's API) rather than measured; afterwards confirmed bank-sync pins `go-paperless v0.3.0`, so consumers see none of this until release+bump anyway. The pin check itself was one command I skipped during the session.~~ done at continuation — measured: bank-sync `go.mod:48` pins `v0.3.0`; TODO_LIST row 1 corroborates (consumer bumps exist locally, unpushed).
3. **Commit hygiene.** My work is committed, but not by me and not cleanly — see d2.
4. **Post-CHANGELOG-edit re-verification.** Docs-only change made after the battery; deliberately not re-run, but strictly the "verify after every change" loop wasn't closed on the literal last edit.

## c) NOT STARTED

- Release/tag of go-paperless carrying the dep bump (v0.3.2 or next) — user didn't request; nothing pushed. → TODO_LIST 2026-09-16 (BLOCKED on §g-2).
- Consumer sweep: bumping bank-sync's `go-paperless` pin (still v0.3.0) — needs release first. → TODO_LIST 2026-09-16 (rides the release row).
- ~~TODO_LIST.md reconciliation (harvest this report's §f; mark any deps item done) — user said WAIT, so parked.~~ done at continuation 2026-09-16 — HARVEST executed: TODO_LIST +5 rows, ROADMAP theme 4.
- ~~Porting bank-sync's `update-vendor-hash` flake app to go-paperless (manual got:-pasting is currently the only path).~~ Won't implement — BuildFlow coverage landed in-repo 2026-09-16 (`.buildflow.yml`); `buildflow -s nix-hash-fix` owns the dance.
- ~~Hermetic erraudit in this flake (the documented gate silently relies on the host's PATH erraudit — version unverified).~~ Superseded — erraudit v0.4.0 upstream breakage (go1.26-era x/tools vs go1.27 modules) discovered later 2026-09-16; hermetic pin BLOCKED on the upstream rebuild (TODO_LIST), not pursued into an actively-edited shared worktree.
- Sibling session's work (integration app, CI changes) — theirs, intentionally untouched by me.
- `nix flake check --all-systems` (check omitted aarch64-darwin/aarch64-linux/x86_64-darwin with a warning). → TODO_LIST 2026-09-16.

## d) TOTALLY FUCKED UP

1. **I nearly shipped a false "vendorHash" narrative — and my first verification attempt used a pipeline that could have masked a failure.** The first `nix build .#default` piped through `grep` printed "no output" and I moved on without checking the pipe's exit semantics; the second showed EXIT=0 with zero output, which I initially couldn't explain. Root cause: the sibling session had already fixed the vendorHash between my bump and my build — a shared-worktree race I know about from bank-sync's AGENTS ("Shared-worktree reverts are REAL") and still walked into. It cost three extra verification rounds and I only got the true story by cross-checking `git status` + `git show`. Pipeline-masking is a documented past lesson — I repeated a variant of it under time pressure.
2. **Daemon commit `5f8d07b` tangled three actors into one commit**: my go.mod/go.sum bump + the sibling's vendorHash line + AGENTS.md/docs/ERROR_CODES.md edits. History attribution is permanently muddy. Worse: I verified the flake.nix line of that commit but **never read the AGENTS.md/ERROR_CODES.md rider diffs** — unreviewed changes I implicitly certified by proceeding.
3. **Two claims in my final chat report exceeded the evidence** (see §Honesty corrections): "v0.10.1 is comment/nolint/test-only" was stated blanket-style while two `examples/cmd/*` patches were never viewed, and the retry.go patch was truncation-trimmed so "behavior-preserving" rested on the test suite, not on the complete diff. The conclusions happen to hold, but the certainty as phrased was not fully earned.
4. **Certified a moving tree.** `nix flake check` ran while the sibling had uncommitted, still-growing flake.nix changes (18 → 71 lines between observations). "All checks passed" is a snapshot of their half-done state, not of a settled tree. A mid-write read could equally have produced a spurious failure I'd have chased.

## Honesty corrections (did I lie?)

No deliberate lies. Two overbroad statements corrected:

- ~~"v0.10.1 is lint-comment/test-only" — **verified for all 6 root-module library files**; the 2 `examples/cmd/*` patches were unviewed (and don't affect consumers of the root package regardless — but the sentence didn't scope that).~~ resolved at continuation — examples patches viewed: comment/formatting + additive `WithContext` only; root package untouched. Claim now fully earned.
- ~~"go-retry v0.6.0 behavior-preserving" — the retry.go diff I inspected was head-truncated; one upstream test switched from `errors.Is` to pointer-identity on the non-retryable path, which _could_ indicate an unviewed behavior change there. The full go-paperless suite (including the `invalid_retry` rejection-identity contract) passes on v0.6.0, which is the real gate — but the diff-level claim should have been stated as "reviewed up to truncation; suite-verified".~~ resolved at continuation — full retry.go patch is pure constants-extraction with identical values; no non-retryable-path change. "Behavior-preserving" earned at diff level.

## e) WHAT WE SHOULD IMPROVE

1. **Parallel-session protocol.** Two agents + daemon on one worktree produced tangled commits and a near-miss on vendorHash attribution. Options: jj/worktree-per-session, or explicit "repo is busy" handoff. Biggest structural fix available.
2. ~~**Port `update-vendor-hash` from bank-sync's flake.** go-paperless still does the vendorHash dance by hand (exactly what bank-sync automation forbids there).~~ Won't implement — BuildFlow coverage landed in-repo 2026-09-16; `nix-hash-fix` owns it.
3. ~~**Hermetic erraudit.** The AGENTS-documented gate runs the _host's_ erraudit through PATH inside `nix develop`; flake ships no pinned binary (bank-sync pins `packages.erraudit` v0.3.1 hermetically). Unreproducible gate.~~ Superseded & sharpened — gate BROKEN 2026-09-16 by the host's erraudit v0.4.0 dev build (go1.26 x/tools fails ALL go1.27 modules; reproduced on pristine v0.3.1 by the sibling session, AGENTS.md erraudit bullet); hermetic pin BLOCKED on the upstream rebuild — TODO_LIST.
4. **Verify-rider-changes rule.** Any commit I build on must be diff-read in full (learned via 5f8d07b's AGENTS/ERROR_CODES riders).
5. **Complete-diff discipline.** When a compare patch is truncated, either paginate it or scope the claim explicitly to what was seen + what the tests prove.
6. **Claim only measured things.** The consumer-impact statement was reasoned, not measured; one grep would have converted it.
7. **Build immediately after dep changes** (bank-sync lesson) — here the sibling's parallel build beat mine and hid the expected hash-mismatch failure; acting before the window opened would have kept the causal chain clean.

## Brutal self-review (the 11 questions, short)

1. **Forgot:** reading 5f8d07b's AGENTS/ERROR_CODES riders; the bank-sync pin grep; TODO_LIST reconciliation; checking `examples/cmd/*` patches; pagination of the retry.go diff.
2. **Stupid we do anyway:** daemon heuristic commits mixing sessions' work; manual vendorHash pasting without the bank-sync-style app; erraudit gate by PATH-convention.
3. **Done better:** run `nix build` the second the bump landed (beat the race); measure consumer impact; verify the pipe, not the grep output.
4. **Still improve:** items e1–e7.
5. **Lied:** no; two overbroad claims corrected above.
6. **Less stupid:** worktree isolation per session; automate the hash dance; pin the gate binaries.
7. **Ghost systems:** none created; no split brains introduced (CHANGELOG single-source, README/AGENTS/version claims consistent). Pre-existing dual-update-path (dependabot + manual) noted as minor process split brain.
8. **Scope creep:** resisted — sibling's work untouched; no unrelated fixes folded in (F10 clean).
9. **Removed something useful:** nothing removed.
10. **Split brains:** none new; flagged pre-existing ones above.
11. **Tests:** suite is strong (93KB client_test, httptest, fuzz file, integration tag, race, hermetic). Gap: fuzz appears local-only; erraudit gate not a flake check; `--all-systems` not run.

## f) Next things to get done (brainstorm, impact-ordered — §f is HARVEST fuel, most items below the line are ROADMAP fuel)

1. ~~[NOW] Re-run `nix flake check` on the sibling's **final** tree — current green certified a mid-edit state.~~ done at continuation ~18:00 — re-ran on the then-current snapshot (client_test.go still mid-edit): **all checks passed**. A truly-final re-run is still owed once the sibling lands.
2. ~~[NOW] Read the AGENTS.md + docs/ERROR_CODES.md rider diffs inside `5f8d07b` (unreviewed, built upon).~~ done at continuation — riders read: BuildFlow usage notes, samber/lo rejection rationale, race-coverage rationale; docs-only, benign. ERROR_CODES.md was pure table re-alignment.
3. ~~[NOW] Finish the upstream diff review: paginate retry.go patch; view `examples/cmd/*` patches; then the "behavior-preserving" claim is fully earned.~~ done at continuation — see §Honesty resolutions.
4. [NOW] Decide + cut release carrying the bump (v0.3.2): CHANGELOG date, tag, proxy-resolution verification (go-release skill). → TODO_LIST (BLOCKED on §g-2).
5. [NEXT] Bump bank-sync's `go-paperless` pin from v0.3.0 → released version (+ its consumer battery). → TODO_LIST (rides the release row).
6. ~~[NEXT] Port `update-vendor-hash` app from bank-sync flake.~~ Won't implement — BuildFlow coverage landed in-repo 2026-09-16; `nix-hash-fix` owns it.
7. ~~[NEXT] Hermetic erraudit: pin in flake packages + wire as a real flake `checks.erraudit`.~~ Superseded — upstream v0.4.0 breakage first; pin BLOCKED on the rebuild (TODO_LIST).
8. [NEXT] Parallel-session protocol: jj/worktree convention or busy-handoff rule across the fleet. → ROADMAP (theme 4).
9. [NEXT] Document dependabot-vs-manual policy (who owns routine bumps; dependabot off for coordinated ones?). → ROADMAP (theme 4) + §g-3.
10. [NEXT] Verify CI actually runs govulncheck/fuzz (go-paperless ci.yml not audited this session — flagged, not researched per scope). → TODO_LIST.
11. [NEXT] `nix flake check --all-systems` once, to kill the omitted-systems warning. → TODO_LIST.
12. ~~[NEXT] docs-health HARVEST of this §f into TODO_LIST/ROADMAP; reconcile any "deps" TODO item.~~ done at continuation 2026-09-16 — TODO_LIST +5 rows (release, erraudit pin, ci.yml audit, --all-systems, simd), ROADMAP theme 4 added; deduped against the sibling's 3 existing rows.
13. [ROADMAP] CHANGELOG convention: dedicated "Dependencies" subsection instead of Changed-burial. → ROADMAP (theme 4).
14. [ROADMAP] Weekly freshness guard: `go list -m -u all` report job complementing dependabot. → ROADMAP (theme 4).
15. [ROADMAP] flake.lock ownership: who runs `nix flake update` and on what cadence (daemon did it today). → ROADMAP (theme 4).
16. [ROADMAP] Verify `GOEXPERIMENT=jsonv2,simd` — is `simd` still meaningful/current on go1.27? (noticed, not researched). → TODO_LIST.
17. [ROADMAP] Wire fuzz targets into CI or document local-only status. → TODO_LIST (folded into the ci.yml audit row).
18. ~~[ROADMAP] `coverage/` dir in repo root — inspect and clean or ignore.~~ NOT-DO — `coverage/` + `coverage.out` are gitignored local output; nothing to clean.
19. ~~[ROADMAP] AGENTS.md wording: name erraudit's actual source (host PATH) until item 7 lands.~~ Superseded — the sibling's AGENTS rewrite (erraudit bullet, 2026-09-16) documents the gate state far more thoroughly than the planned one-liner.
20. [ROADMAP] Add pre-push/CI assertion that `result` symlink & flake `version` stay in sync with the latest tag.
21. [ROADMAP] Consider `gomodGuard`-style check: fail CI if go.mod deps trail latest major by >1 minor (policy per repo).
22. [ROADMAP] Fleet-level: apply this same "latest version" sweep to bank-sync + siblings (same session pattern, per-repo commits).
23. ~~[ROADMAP] Record erraudit version used for this session's green gate in AGENTS (reproducibility).~~ Superseded — moot: the gate broke hours later (upstream v0.4.0); the rebuild + hermetic pin items in TODO_LIST subsume reproducibility.
24. [ROADMAP] Add `nix run .#integration` app docs cross-link if the sibling's scaffold lands without docs.
25. [ROADMAP] Ship-position check: does the repo want a release notes excerpt in GitHub Releases automation (GoReleaser) — bank-sync-class polish.

(Items 26–50 deliberately not fabricated: the honest backlog beyond these is in TODO_LIST/ROADMAP, not in this session's observations.)

## g) Questions only you can answer

1. **Commit authorization & protocol:** In go-paperless, may I commit explicitly per task (beating the daemon), so bump+hash+changelog land as one honest commit? Same for future repos — or is heuristic daemon history acceptable to you there?
2. **Coordination:** Is the sibling session's integration work still intended for the same release? Should I wait for its completion before cutting v0.3.2, or release the dep bump standalone?
3. **Dependency policy:** Keep weekly dependabot gomod PRs _and_ do manual sweeps like this one, or restrict dependabot (e.g. security-only) so bumps you care about stay deliberate?

---

_Point-in-time snapshot. The tree was actively changing while this was written (HEAD moved 4 daemon commits during the session; sibling files uncommitted). WAITING FOR INSTRUCTIONS._

## Continuation appendix (2026-09-16 ~18:00)

Session continued after this snapshot: §f NOW items executed (rider diffs
read, upstream diff review completed, `nix flake check` re-run green on
the then-current snapshot), §f harvested into TODO_LIST (+5 rows) and
ROADMAP (theme 4). The erraudit gate was found BROKEN upstream hours
after the battery ran: the host binary became an erraudit v0.4.0 dev
build (go1.26-era x/tools fails all go1.27 modules; environmental —
reproduced on pristine v0.3.1 by the sibling session, see AGENTS.md).
The "erraudit ✅*" battery row above refers to the 15:45 run's printed
output; its exit code was pipeline-masked (`$?` captured `tail`).
Update-vendor-hash porting was dropped: BuildFlow coverage landed
in-repo the same day. §§b–d verdicts above were annotated inline where
the continuation resolved them.
