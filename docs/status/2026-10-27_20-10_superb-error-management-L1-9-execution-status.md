# Superb-Error-Management Execution Status — L1-9 and Close-Out

- **Date:** 2026-10-27 20:10 CET
- **Scope:** the superb-error-management run triggered by the plan
  [`docs/planning/2026-10-27_18-35_SUPERB-ERROR-MANAGEMENT.md`](../planning/2026-10-27_18-35_SUPERB-ERROR-MANAGEMENT.md):
  L1-1…L1-8 executed earlier today (commits `41d6e29`…`a033822`), L1-9 executed
  in this continuation (commit `4817f3d`). Session-scoped observations only;
  no unrelated research.
- **Repo state at writing:** branch `main`, **10 commits ahead of origin, NOT
  pushed** (push needs explicit authorization), working tree clean at
  `4817f3d`, `nix flake check` all checks passed.
- **Format note:** the status-report skill is HTML-canonical; the operator
  explicitly requested `.md` at this path, so `.md` it is (flagged override,
  not propagated into the skill).

---

## a) FULLY DONE

| #   | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | Evidence                                                                        |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| a1  | **L1-1 Catalog truth pass** — `docs/ERROR_CODES.md` header claim scoped, `request_failed` family fixed Infrastructure→Transient (load-bearing: default retry predicate is Transient-only), Retry-outcomes section added                                                                                                                                                                                                                                                                                                                            | commit `41d6e29`; `TestErrorCodesAreDocumented` green                           |
| a2  | **L1-2 Surface pinning tests** — exhaustion (`retry.exhausted`), cancel×2 (`retry.canceled` vs `paperless.request_failed`), 3xx-as-is, all pinned before any behavior change                                                                                                                                                                                                                                                                                                                                                                       | commit `f55e19b`; suite green                                                   |
| a3  | **L1-3 Coded sentinel** — `New` returns `errorfamily.WrapRejection(ErrInvalidConfig, "paperless.invalid_config", …)`; `errors.Is(err, ErrInvalidConfig)` still matches (pinned)                                                                                                                                                                                                                                                                                                                                                                    | commit `f383160`                                                                |
| a4  | **L1-4 3xx branch** — unfollowable redirects classify as `paperless.redirect` (Rejection, `location` context) instead of `paperless.server_error`/Transient; full `TestClassifyStatusMatrix`                                                                                                                                                                                                                                                                                                                                                       | commit `e31aaa9`                                                                |
| a5  | **L1-5 Meta-test** — `TestEveryConstructedErrorIsCoded` AST-scans all non-test root files; mutation probe verified all three violation classes are caught before trusting it                                                                                                                                                                                                                                                                                                                                                                       | commit `1203254`; probe trashed after green                                     |
| a6  | **L1-6 ADR/AGENTS sync** — ADR 0002 "Amendments / 2026-10-27" section; AGENTS.md error-model bullet carries the three decisions + wrapcheck rationale                                                                                                                                                                                                                                                                                                                                                                                              | commit `7cef7ae`                                                                |
| a7  | **L1-7 Bookkeeping** — TODO_LIST rewritten (umbrella row deleted per convention), CHANGELOG `[Unreleased]` carries Added/Changed/Fixed                                                                                                                                                                                                                                                                                                                                                                                                             | commit `ccc2b47`                                                                |
| a8  | **L1-8 Full gates + unplanned lint-gate repair** — discovered `nix flake check` was red BEFORE the session (daemon commit `43419c8` lifted wrapcheck caps, unmasking 51 pre-existing false positives, proven via baseline worktree); fixed own 6 findings, taught wrapcheck the error model, left limits un-capped                                                                                                                                                                                                                                 | commit `a033822`; fmt/vet/test/erraudit×2/golangci-0/flake-check all green      |
| a9  | **L1-9 Verify-then-file** — all 5 verify-before-filing gates passed against go-retry v0.8.0 source (gap real at `retry.go:123-124`; no existing mechanism — `OnExhausted` has no return value; zero duplicate issues/PRs; mechanics verified against go-error-family v0.12.0 `classify.go:39` + `error.go:238`); draft passed the voice checker (0 FAIL / 0 WARN); **filed [LarsArtmann/go-retry#3](https://github.com/LarsArtmann/go-retry/issues/3)**; landing byte-verified (content identical modulo GitHub trailing-blank-line normalization) | commit `4817f3d`; draft at `docs/drafts/2026-10-27_go-retry_exhaustion-code.md` |
| a10 | **Operational recovery** — daemon-raced `.git/index.lock` (6 min old, no owning process) diagnosed as stale and trashed; commit landed clean afterward                                                                                                                                                                                                                                                                                                                                                                                             | `git log` + `ps` output in session                                              |
| a11 | **Todo ledger** — all 9 L1 tasks completed and tracked through execution                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | todo tool: 9/9 completed                                                        |

## b) PARTIALLY DONE

| #  | Item                                                                              | Works now                                                                                                          | Open                                                                                                                                                                                                | Blocker                                                                                             | Effort to finish   |
| -- | --------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- | ------------------ |
| b1 | **Finding F1 root fix** (exhaustion buries the last attempt's `paperless.*` code) | Consumer-side mitigation is complete and pinned: docs teach the `errors.AsType` dig, tests pin the current surface | The upstream change itself: filed as go-retry#3, awaiting ruling → implementation → release; then the consumer flip (2 pinned tests + `ERROR_CODES.md` retry-outcomes section + ADR 0002 amendment) | Upstream maintainer decision (same person, but it is a design ruling the issue explicitly requests) | M after acceptance |
| b2 | **The header claim "every error carries a code"**                                 | Mechanically true inside the SDK since L1-3/L1-5 (meta-test enforces it)                                           | `Code()` on exhaustion still returns `retry.exhausted` — by documented design until go-retry#3 moves                                                                                                | Same as b1                                                                                          | —                  |
| b3 | **v0.5.0 release** (pre-existing TODO_LIST row, not this session's work)          | Three-place version contract agrees (flake/CHANGELOG/README), CI green                                             | Only the tag + `nix run .#release-verify -- v0.5.0` ritual; open question whether the 2026-10-27 behavior changes ride as v0.5.1 instead                                                            | Owner decision (question g2)                                                                        | S                  |
| b4 | **Error-catalog reverse gate**                                                    | Code→docs direction enforced (`TestErrorCodesAreDocumented`); meta-test enforces coded constructors                | Docs→code direction unenforced: a catalog row whose code vanishes from `client.go` fails nothing today. Not started; listed in (f)                                                                  | —                                                                                                   | S                  |

## c) NOT STARTED

Planned and tracked, deliberately untouched this run (priority as tracked in
TODO_LIST unless noted):

- **Owner-gated:** push the 10 commits (g1); release cut (b3/g2); consumer CI
  billing restoration (GitHub Actions spending failure on InboxClean/bank-sync).
- **Consumer migrations:** bank-sync + InboxClean to v0.5.0 branded IDs;
  delete bank-sync's legacy `testhelpers.NewFakePaperlessServer`.
- **Verification tail (2026-10-11 review):** drift-guard negative-verify,
  `pageBounds` overflow regression test, 30s fuzz smoke, coverage vs ~88.6%.
- **Structure:** split `client.go` (2,675 lines) / `client_test.go` (4,098).
- **Pins:** golden-bytes wire shapes (`"id":null` etc.).
- **Gates:** erraudit hermetic flake pin; api-diff gate; branching-flow CI
  wiring; strong-id CI gate.
- **Fake (`paperlesstest/`):** pagination ADR; store==serve ADR; created-string
  parser consolidation; `FilterRules` nil-vs-empty contract.
- **Docs:** MIGRATION.md v0.4→v0.5; CONTRIBUTING flake-app names; README
  common-tasks refresh; runnable examples.
- **Low tail:** test tail (GetTask/classifyTask edges, hook failures, multipart);
  `BenchmarkEnsureTag`; fuzz depth; small-cleanups batch; govulncheck pin /
  lychee re-run / go.mod indirects audit.
- **Blocked (external):** govalid-generate unskip (nixpkgs govalid is
  go1.26-built); erraudit CI (not on public proxy); gosec un-park (no
  go1.27-capable release).
- **Session-observed, not started:** go-retry#3 follow-through row in
  TODO_LIST (see e1); branching-flow re-run after this run's Go changes (see
  e2); gopls/golangci LSP restart to clear ghost `zz_metacheck_probe.go`
  diagnostics (stale cache — the file was trashed pre-L1-5, diagnostics
  persisted all session); auto-commit daemon health check (see e7).

## d) TOTALLY FUCKED UP

**Nothing at that severity — stated plainly rather than padded.** The repo is
green at HEAD, nothing is broken for users, no data was lost. The two
incidents that could have escalated, with honest severity:

| Incident                | Severity            | What exactly happened                                                                                                                | Root cause                                                                                                                               | Mitigation                                                                                                       |
| ----------------------- | ------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Stale `.git/index.lock` | Low (workflow only) | First `git add`/`git commit` failed twice with "index.lock exists"; lock was 6 min old with **no git process alive** (`ps` verified) | Auto-commit daemon raced the manual commit and left a stale lock (or died mid-operation — no daemon process was found afterward, see e7) | Verified staleness (age + no owner via `ps`) before `trash .git/index.lock`; commit then landed; no data at risk |
| First-commit failure    | None (retried OK)   | Consequence of the row above                                                                                                         | Same                                                                                                                                     | Retry after lock recovery                                                                                        |
| Ghost LSP diagnostics   | Cosmetic            | `zz_metacheck_probe.go` (trashed in the earlier phase) kept producing `unused` warnings in every tool result all session             | Crush's golangci LSP cache did not invalidate on file deletion                                                                           | Confirmed file absent via `ls` + clean `git status`; LSP restart is the fix (f25)                                |

Pre-existing, observed, NOT caused by this run (documented for the record):
`nix flake check` omits aarch64 systems (warning); machine govalid and gosec
cannot process Go 1.27 modules (both parked with documented rationale in
AGENTS.md / `.buildflow.yml`).

## e) WHAT WE SHOULD IMPROVE

1. **The filed issue's sketch is correct but under-annotated.**
   `WrapInfrastructure(err, code, msg).WithCause(terminal)` re-points the cause
   from `err` to `terminal` (so `err` stays reachable only _through_
   `terminal`). I reasoned this through but did not write one sentence of
   explanation into go-retry#3 — the maintainer (us) gets a double-take. Fix:
   comment on the issue.
2. **The gate battery is plan-dependent.** The plan's L1-8 listed
   fmt/vet/test/erraudit/flake — branching-flow was not re-run even though
   L1-3/L1-4/L1-5 added Go code, and it is tracked at "0 active findings" in
   AGENTS.md. Impact: a dupe/composition regression could sit unnoticed.
   Fix: re-run now (f6) and bundle the canonical battery into one flake app
   (f42) so plans cannot under-list gates.
3. **The go-retry#3 follow-through lives nowhere in the living docs.** The
   TODO_LIST row that pointed at L1-9 was deleted on completion (correct per
   convention), so if go-retry#3 is accepted, nothing in the repo reminds the
   next session to flip the pinned tests, the ERROR_CODES retry-outcomes
   section, and the ADR amendment. Fix: HARVEST row f8.
4. **Draft-vs-issue is a small split brain.** `docs/drafts/…` is now a frozen
   copy of a live, editable issue; they can drift. Deliberate (provenance),
   but needs a policy: annotate the draft as "issue is canonical" or trash it
   post-landing (f26).
5. **Posted-body verification was eyeballed.** The draft-vs-posted diff was
   judged by hand (byte-count 3317 vs codepoint 3291 = UTF-8 em dashes;
   trailing blank line = GitHub normalization). Correct conclusion, fragile
   method. Fix: strip trailing whitespace before diffing, compare NFC.
6. **Ghost LSP diagnostics waste attention.** Every tool result carried three
   false warnings for ~90 minutes. Fix: restart the LSP the moment a trashed
   file haunts diagnostics (f25).
7. **Auto-commit daemon liveness is unknown.** No daemon process found in
   `ps` while a stale lock existed; this report is deliberately NOT manually
   committed (harness contract), so it relies on that daemon picking it up.
   If it is dead, the report stays untracked. Fix: host-level health check
   (f24), and record the stale-lock recovery pattern in AGENTS.md (f47).
8. **Self-review scope.** The opening questions (forget/better/improve) were
   answered session-scoped inside this report; a full standalone
   brutal-self-review (HTML, whole codebase, ghost systems/split-brain sweep)
   was NOT run this session (f50).

## f) Top 50 things we should get done next

Ranked by impact; brainstorm, not commitment — `docs-health` HARVEST routes
these into TODO_LIST/ROADMAP. (Critical/High/Medium/Low · S <30min, M
30min–2h, L >2h · category)

| #  | Task                                                                                                                                                                                                             | Impact | Effort | Category      |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Push the 10 local commits to `origin/main` (g1)                                                                                                                                                                  | High   | S      | Process       |
| 2  | Rule on the release: tag `v0.5.0` now vs cut `v0.5.1` carrying the 2026-10-27 changes; run `nix run .#release-verify` either way (g2)                                                                            | High   | S      | Process       |
| 3  | Rule on go-retry#3: breaking-in-minor vs opt-in `Option`; decide whether `contextEnded` gets the same treatment in the same change (g3)                                                                          | High   | S      | Feature       |
| 4  | HARVEST this report: land (f) items into TODO_LIST/ROADMAP                                                                                                                                                       | High   | S      | Documentation |
| 5  | Add the go-retry#3 flip-plan TODO row: on acceptance, flip `TestWithRetryExhaustionSurfacesRetryOutcome`, the ERROR_CODES retry-outcomes section, and the ADR 0002 amendment                                     | High   | S      | Documentation |
| 6  | Re-run `branching-flow all . --format finding` to confirm 0-active after this run's Go changes                                                                                                                   | High   | S      | Quality       |
| 7  | Restore consumer CI (GitHub Actions billing) so InboxClean/bank-sync jobs start again                                                                                                                            | High   | S      | Infra         |
| 8  | Migrate bank-sync + InboxClean to go-paperless v0.5.0 branded IDs; delete bank-sync's legacy fake                                                                                                                | High   | M      | Feature       |
| 9  | Comment on go-retry#3 explaining the sketch's cause-chain dance (e1)                                                                                                                                             | Medium | S      | Documentation |
| 10 | Verification tail: negative-verify the drift guard (temp unknown `findByName` kind → red → revert)                                                                                                               | Medium | S      | Quality       |
| 11 | Add the `pageBounds` overflow regression test                                                                                                                                                                    | Medium | S      | Bug           |
| 12 | Run `nix run .#fuzz -- 30s` smoke                                                                                                                                                                                | Medium | S      | Quality       |
| 13 | Run `nix run .#coverage` vs the ~88.6% baseline                                                                                                                                                                  | Medium | S      | Quality       |
| 14 | Add a catalog→code reverse gate (a documented `ERROR_CODES.md` row whose code no longer exists fails the suite) (b4)                                                                                             | Medium | S      | Quality       |
| 15 | Split `client.go`/`client_test.go` by domain (upload/tasks/documents/named/notes/sharelinks/savedviews/transport)                                                                                                | Medium | L      | Cleanup       |
| 16 | Pin the deliberate wire shapes with golden-bytes tests (`"id":null`, FK-clearing null-PATCH, `"filter_rules":[]`)                                                                                                | Medium | M      | Quality       |
| 17 | Pin erraudit hermetically in the flake (today it resolves via host PATH)                                                                                                                                         | Medium | S      | Quality       |
| 18 | Add an api-diff gate protecting the bank-sync drop-in contract                                                                                                                                                   | Medium | M      | Quality       |
| 19 | Wire branching-flow into CI/`nix flake check` (probe `--exit-code` semantics first)                                                                                                                              | Medium | M      | Quality       |
| 20 | Write the fake's pagination-policy ADR                                                                                                                                                                           | Medium | S      | Documentation |
| 21 | Write the fake's store==serve entity-family ADR                                                                                                                                                                  | Medium | S      | Documentation |
| 22 | Consolidate `parseServedCreated`/`parseUploadDate`                                                                                                                                                               | Medium | S      | Cleanup       |
| 23 | Decide + test the `FilterRules` nil-vs-empty contract                                                                                                                                                            | Medium | S      | Bug           |
| 24 | Investigate auto-commit daemon health: why the stale lock, is it still running? (e7)                                                                                                                             | Medium | S      | Infra         |
| 25 | Restart gopls/golangci LSP to clear the ghost probe-file diagnostics (e6)                                                                                                                                        | Low    | S      | Cleanup       |
| 26 | Decide the draft policy: annotate `docs/drafts/…` as "issue is canonical" or trash it post-landing (e4)                                                                                                          | Low    | S      | Documentation |
| 27 | Bundle the canonical gate battery into one flake app (fmt/vet/test/erraudit×2/lint/branching-flow) so plans cannot under-list gates (e2)                                                                         | Medium | M      | Quality       |
| 28 | Write MIGRATION.md for v0.4→v0.5 consumers                                                                                                                                                                       | Medium | M      | Documentation |
| 29 | After the go-retry#3 ruling: sync the ERROR_CODES reachability wording if the decision differs from today's docs                                                                                                 | Medium | S      | Documentation |
| 30 | Run `nix flake check --all-systems` once (aarch64 currently omitted)                                                                                                                                             | Medium | S      | Quality       |
| 31 | Draft v0.5.x release notes from CHANGELOG `[Unreleased]` once cut (2) is decided                                                                                                                                 | Medium | S      | Documentation |
| 32 | CONTRIBUTING: name the flake apps (`.#fuzz`, `.#integration`, `.#release-verify`, checks) + branching-flow conventions                                                                                           | Low    | S      | Documentation |
| 33 | README: refresh "Common tasks" snippets + link the review series                                                                                                                                                 | Low    | M      | Documentation |
| 34 | Test tail: `GetTask`/`classifyTask` edge mix, hook-failure branches, multipart entry branches                                                                                                                    | Low    | L      | Quality       |
| 35 | Runnable examples: `UpdateDocument`/`ListDocumentMetas`/`EnsureStoragePath`, download/update, hooks                                                                                                              | Low    | M      | Documentation |
| 36 | `BenchmarkEnsureTag` + branded hot-path benchmarks                                                                                                                                                               | Low    | M      | Quality       |
| 37 | Fuzz depth: `Retry-After` overflow seeds, overnight budget, CI crash-corpus upload                                                                                                                               | Low    | L      | Quality       |
| 38 | Small cleanups batch: `tbStub` `Fatalf`, dead `return`, `Requests()` Header deep-clone, fold `newTestClientWithToken`, options-table drift test, apiVersion header test, family-label test, code-uniqueness test | Low    | L      | Cleanup       |
| 39 | Verify govulncheck CI pin (v1.8.0); re-run lychee over the 2026-10-11 HTML reports; go.mod indirects audit                                                                                                       | Low    | S      | Quality       |
| 40 | README: show the `errors.AsType` retry-outcome dig pattern in the consumer example (today only ERROR_CODES.md has it)                                                                                            | Low    | S      | Documentation |
| 41 | Unskip `govalid-generate` when nixpkgs ships a Go ≥ 1.27 govalid (blocked)                                                                                                                                       | Low    | S      | Quality       |
| 42 | erraudit CI job once it is on the public module proxy (blocked)                                                                                                                                                  | Low    | S      | Quality       |
| 43 | Restore the gosec CI scan when a go1.27-capable release ships (blocked, parked 2026-10-11)                                                                                                                       | Low    | S      | Quality       |
| 44 | Record the stale-lock recovery + daemon-race pattern in AGENTS.md if it recurs (f47 lesson)                                                                                                                      | Low    | S      | Documentation |
| 45 | Probe the treefmt-vs-dprint markdown lag (plugin bump follow-through)                                                                                                                                            | Low    | S      | Cleanup       |
| 46 | Annotate this report as items complete (docs-health ANNOTATE, never rewrite)                                                                                                                                     | Low    | S      | Documentation |
| 47 | go-retry: backport an ERROR_CODES-style "retry outcomes" table to its README if #3 is accepted                                                                                                                   | Low    | S      | Documentation |
| 48 | Wire `branching-flow strong-id --exit-code` as a CI gate (strong-id at zero, enforced by nothing)                                                                                                                | Medium | S      | Quality       |
| 49 | Post-tag: bump consumer pins (rides 8)                                                                                                                                                                           | Medium | S      | Process       |
| 50 | Schedule a full standalone brutal-self-review (whole codebase, HTML at `docs/reviews/`) — this session ran session-scoped only (e8)                                                                              | Medium | M      | Quality       |

## g) Questions I cannot figure out myself

Tried and blocked on each: (1) push is authorization-gated by the harness;
(2) the TODO_LIST row itself marks the cut as an owner decision; (3) the
design ruling is the open question the filed issue ends with.

1. **Push now or after the release decision?** 10 commits sit local on
   `main`. Pushing first unblocks consumer migrations and CI visibility;
   waiting keeps the release cut atomic. Which order do you want?
2. **`v0.5.0` now, or `v0.5.1` carrying the 2026-10-27 behavior changes**
   (3xx→`paperless.redirect`, coded `invalid_config`)? The three-place
   contract currently agrees on 0.5.0 without them.
3. **go-retry#3 ruling:** breaking-in-minor (my lean, stated in the issue) or
   an opt-in `Option`? And should `contextEnded` (`retry.canceled`/
   `retry.deadline`) get the same outermost-code treatment in the same
   change?
