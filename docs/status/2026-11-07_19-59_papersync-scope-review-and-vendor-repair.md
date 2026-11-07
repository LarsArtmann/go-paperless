# Status: papersync scope review + vendor-drift repair (go-paperless)

**Session task:** Review `../InboxClean/internal/papersync/` — is there something that
belongs in go-paperless? Plus full self-review and next-work inventory.

**Date:** 2026-11-07 19:59 CET · **Repo:** go-paperless @ `main` (daemon `c0e05ff` on top of `97340ef`)

---

## Scope line — what this report verified vs. did NOT verify

**Verified here (fresh commands/artifacts):**

- Read every non-test file of `InboxClean/internal/papersync/` (8 files, ~1,650 LOC:
  pipeline.go, pipeline_upload.go, pipeline_tasks.go, pipeline_ledger.go,
  pipeline_results.go, filter.go, pdf.go + fuzz/testdata listing; test files counted
  via `wc -l`, NOT read line-by-line).
- Inventoried `go-paperless/client.go` API surface (`rg "^func|^type"` over ~2,700
  lines) and read `WithRetry` (client.go:183–295), `WaitForTask` (client.go:632–702),
  `Upload` → `doRequest` retry path (client.go:425–480, 1816–1875) directly.
- Vendor drift healed and re-synchronized: `go mod vendor` in `nix develop`
  (modules.txt v0.7.0 → v0.7.1), then `buildflow -s nix-hash-fix --fix` repaired the
  stale `vendorHash` in flake.nix (daemon-committed as `c0e05ff`, content verified via
  `git show --stat c0e05ff` = flake.nix only).
- `nix run .#check` → **"all checks passed!"** (build, treefmt, test-race variants) —
  but see caveat below: first green run predated my TODO_LIST/ROADMAP edits; a second
  run over the final tree was started and its result is recorded in the Verification
  appendix.
- `buildflow -s gomod-check` → passes (only the pre-existing info finding
  "directory vendor exists but is not ignored in go.mod" remains, non-gating).
- Consumer pins read directly: InboxClean `go.mod:41` and bank-sync `go.mod:57` both
  pin `github.com/larsartmann/go-paperless v0.5.0`.
- `vendor/` is gitignored: `.gitignore:60` read directly (this is why `git status`
  never showed the drift).
- InboxClean production wiring: `rg` over non-test files found three `paperless.New`
  call sites (main.go:609, doctor_command.go:541, paperless_command.go:318), none
  passing `WithRetry`.
- qpdf/`/Encrypt` logic exists only in InboxClean: `rg -l "qpdf|MaybeEncryptedPDF|/Encrypt"`
  across `~/projects` (Go files) returned InboxClean-only hits; bank-sync has none.
- dprint-formatted my two markdown edits (`dprint fmt TODO_LIST.md ROADMAP.md`, 1 file
  reformatted, padding-only diff).

**NOT verified here (declared gaps):**

- **InboxClean's own build/test suite never executed** — the "both consumers migrated
  to v0.5.0" claim rests on go.mod pins + branded-API usage in source, not on a green
  InboxClean run.
- **bank-sync's paperless call patterns not audited** for its own hand-rolled
  retry/poll duplication (only the qpdf grep touched bank-sync).
- **papersync test files not reviewed line-by-line** (sizes only) — coverage claims
  about retry/poll paths are unverified.
- **`paperless --verify` / `--decrypt-repair` commands read only via grep** of their
  papersync-symbol usage, not audited in full.
- **golangci_lint_ls kept replaying the stale vendor error after `lsp_restart`** —
  declared stale-cache noise with `nix flake check` as ground truth, root cause not
  diagnosed.
- **The v0.5.0 tag's existence on the proxy** inferred from consumers' go.sum
  resolving, not probed directly.

---

## a) FULLY DONE

| What                                                                                                                                                                              | Evidence                                                                                                                                                                                                                   |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Full read of papersync package + classification of every piece (SDK-generic vs consumer-domain)                                                                                   | 8 files read in session; triage table in session summary                                                                                                                                                                   |
| Verdict: nothing must move into go-paperless now; `pdf.go` (PDFDecryptor/MaybeEncryptedPDF) is the one demand-gated candidate                                                     | Cross-checked against AGENTS.md scope ("Client only. No pipeline abstractions, no ledger/projection code") and ROADMAP "Growing with the API"                                                                              |
| Vendor drift healed: `go mod vendor` (go-branded-id v0.7.0 → v0.7.1 in modules.txt)                                                                                               | `rg go-branded-id vendor/modules.txt` = v0.7.1 post-fix                                                                                                                                                                    |
| Stale `vendorHash` repaired via `buildflow -s nix-hash-fix --fix`                                                                                                                 | `git diff flake.nix` showed old→new sha; daemon commit `c0e05ff`                                                                                                                                                           |
| Repo gate green over repaired state                                                                                                                                               | `nix run .#check` run 1 → "all checks passed!"; run 2 over the final tree FAILED on one pre-existing load-sensitive flake (see (d)); solo rebuilds of the failed drv and the targeted test are green — details in appendix |
| `buildflow -s gomod-check` clean (error finding gone)                                                                                                                             | `BUILDFLOW_NO_RESULT_CACHE=1` run: "passed with warnings (1 finding)" = info-only                                                                                                                                          |
| TODO_LIST row 25 de-staled: both consumers verified on v0.5.0, row rewritten to surviving scope (bank-sync fake consolidation)                                                    | Edit applied; go.mod reads for both consumers                                                                                                                                                                              |
| ROADMAP raw idea added: encrypted-PDF helper (`pdfdecrypt`) as demand-gated sibling-package extraction                                                                            | ROADMAP.md "Growing with the API", +7 lines                                                                                                                                                                                |
| AGENTS.md lesson recorded: dependency bumps need `go mod vendor` + `nix-hash-fix` tail; vendor/ gitignored makes drift git-invisible; golangci LSP can replay stale vendor errors | AGENTS.md Critical section, new bullet                                                                                                                                                                                     |
| Markdown edits dprint-formatted (padding-only)                                                                                                                                    | `dprint fmt` output: 1 file formatted; diff stat 1 insertion, 1 deletion (plus ROADMAP addition)                                                                                                                           |

## b) PARTIALLY DONE

| Item                                  | Works                                                         | Open                                                                                                                       | Effort |
| ------------------------------------- | ------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- | ------ |
| Final-tree gate re-run                | Started in background mid-session                             | Result to be folded into appendix before hand-off                                                                          | S      |
| Review-question answer                | Verdict delivered for papersync package                       | InboxClean cmd-layer (`paperless_verify.go`, `paperless_decrypt_repair.go`) only grep-audited, not fully read              | M      |
| Consumer-adoption finding (WithRetry) | Three `paperless.New` sites identified; duplication confirmed | Not written into any InboxClean TODO (cross-repo policy undecided — question g1); bank-sync unaudited for the same pattern | S/M    |

## c) NOT STARTED

| Item                                                             | Why                                                                                                                                          |
| ---------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Any code extraction of `pdf.go` into go-paperless                | Deliberately demand-gated (single consumer today; root package stays exec-free); ROADMAP idea only                                           |
| WithRetry adoption in InboxClean/bank-sync                       | Consumer-repo decision, behavior change (500ms-base hand-rolled → SDK 100ms-base + jitter + hint precedence); needs owner sign-off on values |
| bank-sync `NewFakePaperlessServer` → paperlesstest consolidation | Surviving scope of rewritten TODO row 25; untouched this session                                                                             |
| papersync `resolveTaskOutcome` → `WaitForTask` simplification    | Optional; current attempt-budget shape is deliberate and correct; only worth it if consumer wants less code                                  |
| Everything else in section (f) below                             | Brainstorm inventory, routed to HARVEST                                                                                                      |

## d) TOTALLY FUCKED UP

| What                                                                                                                                                                                                                                                                                                                         | Severity                                                                                                                         | Root cause                                                                                                                                                                                                                                         | Mitigation                                                                                                                                                                                                                     |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Repo arrived broken from the previous session's commit `97340ef` ("chore(deps): update Go dependencies", 2026-11-06 05:25): go.mod bumped to go-branded-id v0.7.1 without `go mod vendor`, and flake `vendorHash` left stale                                                                                                 | Blocker-grade for the next session/build: gopls+golangci red ("inconsistent vendoring"), `nix build` would fail on hash mismatch | The deps commit changed only go.mod/go.sum (verified via `git show --stat`); `vendor/` is gitignored (`.gitignore:60`) so the drift is invisible in `git status`; BuildFlow's gomod-check detects the mismatch but has NO auto-fix for vendor sync | Both tails executed this session (vendor resync + nix-hash-fix); lesson written into AGENTS.md; upstream auto-fix idea filed as task (f27)                                                                                     |
| This session's first `rg -rn "pattern"` invocation was the replace-flag trap again — the hook message says THIRD strike of this exact trap on this host                                                                                                                                                                      | Process smell, no damage (hook blocked)                                                                                          | Muscle memory from grep's `-rn` recursion syntax                                                                                                                                                                                                   | Reran correctly; consider a host-level alias/abbreement or hook that rewrites instead of blocks                                                                                                                                |
| **Pre-existing load-sensitive flaky test discovered**: `TestWithRetryCancelDuringBackoffSurfacesRetryCanceled` (client_test.go:2106) — run 2 of `nix run .#check` failed it under concurrent load (lint + check builds sharing GOCACHE on /mnt/buildcache): `code = "paperless.rate_limited", want paperless.request_failed` | Gates CI/local runs intermittently red; nondeterministic                                                                         | Cancellation racing the backoff window: whether the cancel lands mid-backoff or after the next attempt determines which coded error surfaces outermost (go-retry v0.9.0 behavior, test last touched by commit 241f8ef's v0.9.0 adaptation)         | Verified NOT a regression: identical Go inputs passed run 1; targeted `-race -count=10` run solo: ok 10/10; failed drv rebuilt solo: exit 0. Fix needs determinism (inject clock / synchronize the cancel hook) — task (f) #47 |
| BuildFlow result cache replayed the already-fixed gomod finding on the 2nd and 3rd check runs, costing extra cycles                                                                                                                                                                                                          | Wasted ~2 min                                                                                                                    | Skill's failure-triage ("finding replays after its cause was deleted → `BUILDFLOW_NO_RESULT_CACHE=1`") was applied only after the repeat                                                                                                           | Follow the triage table on FIRST suspicion, not third                                                                                                                                                                          |
| `edit` on AGENTS.md rejected once ("read the file first") despite full content being in session context                                                                                                                                                                                                                      | One wasted round                                                                                                                 | Tool requires an in-session View; project_context inclusion doesn't count                                                                                                                                                                          | None needed — View-then-edit worked                                                                                                                                                                                            |
| `lsp_restart` cleared gopls but NOT golangci_lint_ls's stale vendor error                                                                                                                                                                                                                                                    | Cosmetic (ground truth green), but it would mislead the next session's diagnostics                                               | Unknown — stale cache inside golangci-lint's loader, not diagnosed                                                                                                                                                                                 | Declared in AGENTS.md that `nix flake check` is ground truth; diagnosis filed as task (f25)                                                                                                                                    |

## e) WHAT WE SHOULD IMPROVE

1. **Dep-bump tail automation.** The 97340ef incident is the second recent commit in
   this exact family (241f8ef's message literally says "…and fix vendor hash"). The
   two-step tail (`go mod vendor` + `nix-hash-fix`) should be one BuildFlow-owned,
   pre-commit-enforced step — upstream it (f27) instead of re-teaching every session
   via AGENTS.md.
2. **Filter-time MIME fallback contradicts its own determinism goal.** papersync
   `filter.go` curates `mimeExtensionByType` "so the mapping is deterministic", but
   `extensionForMimeType` still falls back to stdlib `mime.ExtensionsByType`, which
   reads the host's mime.types — nondeterministic across machines for unmapped types.
   Either extend the curated table or document the accepted nondeterminism (InboxClean).
3. **Review-first sessions should end with an execution verdict file**, not just chat:
   the pdfdecrypt trigger condition lives in ROADMAP (good), but the InboxClean-side
   simplifications currently live only in this report — entombed-in-timestamped-file
   risk that HARVEST (f17) must close.
4. **Consumer pin verification should include a build/test run**, not just go.mod
   reads — "both consumers on v0.5.0" was accepted from pins; a 2-minute
   `nix build`/test in each consumer repo would have made it a measured claim.
5. **Cross-repo findings need a routing decision up front** (write into the other
   repo's TODO vs. report-only). I chose report-only and flagged it as g1; decide the
   policy once so sessions stop improvising.
6. **Golangci LSP cache staleness** after vendor repair burned trust in diagnostics
   twice this session; the cheap fix is documenting "flake check is ground truth"
   (done) plus a diagnosis task (f25).

## f) NEXT TASKS (brainstorm ≤50, sorted by impact; HARVEST input — route Critical/High → TODO_LIST, speculative/demand-gated → ROADMAP)

| #  | Task                                                                                                                                                                                                                                                                                                      | Impact      | Effort | Category      |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | ------ | ------------- |
| 1  | Run `nix run .#release-verify -- v0.5.0` and close TODO row 24 — the "consumers still on v0.4.x" blocker is gone (both pin v0.5.0, verified)                                                                                                                                                              | High        | S      | Release       |
| 2  | Harvest this report's (f) into TODO_LIST/ROADMAP per docs-health routing rigor                                                                                                                                                                                                                            | High        | S      | Documentation |
| 3  | Audit bank-sync's paperless call sites for hand-rolled upload retry/poll duplication (fleet-wide WithRetry adoption scope)                                                                                                                                                                                | High        | M      | Quality       |
| 4  | Wire `WithRetry(RetryPolicy{…})` into InboxClean's three `paperless.New` sites (main.go:609, doctor_command.go:541, paperless_command.go:318) and delete papersync's `uploadWithRetry`; decide backoff values (hand-rolled was 500ms-base/3; SDK default 100ms-base/3/5s-cap/2x/jitter + hint precedence) | High        | M      | Quality       |
| 5  | Build + test InboxClean against the v0.5.0 pin to convert "migrated" from pin-inference into a measured claim                                                                                                                                                                                             | High        | S      | Quality       |
| 6  | Replace bank-sync's `testhelpers.NewFakePaperlessServer` with shared `paperlesstest` fake and delete it (rewritten TODO row 25)                                                                                                                                                                           | High        | M      | Cleanup       |
| 7  | Restore consumer-repo CI (GitHub Actions billing) — TODO row 23, owner action                                                                                                                                                                                                                             | High        | owner  | Infra         |
| 8  | Deep-audit InboxClean `paperless_verify.go` reconcile loop against SDK `ListDocumentChecksums`/`TaskOutcome.DocumentID` contracts (grep-only so far)                                                                                                                                                      | Medium      | M      | Quality       |
| 9  | Split client.go (~2.7k lines) and client_test.go by domain — TODO_LIST Medium row                                                                                                                                                                                                                         | Medium      | M–L    | Cleanup       |
| 10 | Golden-bytes wire-shape tests: `"id":null` create bodies, FK-clearing null-PATCH, saved-view `"filter_rules":[]` — TODO_LIST row                                                                                                                                                                          | Medium      | S–M    | Quality       |
| 11 | Evaluate papersync `resolveTaskOutcome` → `WaitForTask` + per-upload deadline ctx; keep the non-blocking run shape                                                                                                                                                                                        | Medium      | S      | Cleanup       |
| 12 | Write the two fake ADRs: pagination policy + store==serve convention — TODO_LIST row                                                                                                                                                                                                                      | Medium      | S      | Documentation |
| 13 | Consolidate fake's `parseServedCreated`/`parseUploadDate`; decide + test nil-vs-empty `FilterRules` contract — TODO_LIST row                                                                                                                                                                              | Medium      | S      | Cleanup       |
| 14 | papersync `lastDotSegment` (pdf.go:169) → stdlib `path.Ext` dedupe                                                                                                                                                                                                                                        | Low         | S      | Cleanup       |
| 15 | Extend papersync's curated MIME→extension table or document the `mime.ExtensionsByType` fallback's host-nondeterminism (see e2)                                                                                                                                                                           | Medium      | S      | Quality       |
| 16 | Close the 2026-10-11 verification tail: negative-verify drift guard, pageBounds overflow regression test, fuzz smoke, coverage vs ~88.6% — TODO_LIST row                                                                                                                                                  | Medium      | S–M    | Quality       |
| 17 | Test tail: `GetTask`/`classifyTask` edge mix, hook-failure branches, multipart branches — TODO_LIST row                                                                                                                                                                                                   | Low         | M      | Quality       |
| 18 | `BenchmarkEnsureTag` + branded hot-path benchmarks — TODO_LIST row                                                                                                                                                                                                                                        | Low         | S      | Quality       |
| 19 | Fuzz depth: Retry-After overflow seeds, overnight budget, diagnosable CI fuzz failures (crash corpus upload) — TODO_LIST row                                                                                                                                                                              | Low         | M      | Quality       |
| 20 | README: download/update common-tasks snippets + review-series links — TODO_LIST Low row                                                                                                                                                                                                                   | Low         | S      | Documentation |
| 21 | README/examples: verify `WithRetry`/`WaitForTask` are documented and exampled (TestREADMEListsLookupVerbs only covers Find*/Get*/Ensure*)                                                                                                                                                                 | Low         | S      | Documentation |
| 22 | branching-flow v2 revisit: typed-API migration, drop disabled_rules + inline nolints when line-suppression ships — TODO_LIST row                                                                                                                                                                          | Low         | M      | Quality       |
| 23 | Pin erraudit hermetically in the flake (currently resolves via host system profile) — AGENTS.md open item                                                                                                                                                                                                 | Low         | S      | Infra         |
| 24 | Re-verify erraudit's proxy.golang.org 404 (AGENTS.md CI decision says re-verify before revisiting; not re-checked since 2026-09-16)                                                                                                                                                                       | Low         | S      | Infra         |
| 25 | Diagnose why golangci_lint_ls replays the stale vendor error after `lsp_restart` while gopls clears                                                                                                                                                                                                       | Low         | S      | Infra         |
| 26 | Rebuild/reinstall BuildFlow binary (built at e35e902; repo HEAD moved to 940aeae — advisory warn every run)                                                                                                                                                                                               | Low         | S      | Infra         |
| 27 | Upstream to BuildFlow: a step/hook that runs `go mod vendor` + `nix-hash-fix` when go.mod/go.sum change (gomod-check detects; nothing auto-fixes; this is the 97340ef-class root cause)                                                                                                                   | High        | M      | Infra         |
| 28 | pdfdecrypt extraction WHEN bank-sync (or any second consumer) needs encrypted-PDF ingestion — ROADMAP idea; extraction spec must keep `TaskStateFor` consumer-side (emailcqrs-coupled)                                                                                                                    | Low (today) | M      | Feature       |
| 29 | Review papersync test files line-by-line (only sizes counted this session) before any retry/poll refactor lands                                                                                                                                                                                           | Medium      | M      | Quality       |
| 30 | Decide whether `encryptedTagName` ("encrypted") stays hardcoded in papersync or becomes config (InboxClean call)                                                                                                                                                                                          | Low         | S      | Cleanup       |
| 31 | Verify DocumentTitle's 128-char cap against actual paperless-ngx title handling (server-side truncation behavior never confirmed)                                                                                                                                                                         | Low         | S      | Quality       |
| 32 | Run papersync fuzz targets (`pipeline_fuzz_test.go` + corpus) on a schedule in InboxClean CI, if not already                                                                                                                                                                                              | Low         | S      | Quality       |
| 33 | Run go-paperless integration scaffold against a live paperless-ngx instance periodically (real-server tier is scaffold-only)                                                                                                                                                                              | Low         | M      | Quality       |
| 34 | ROADMAP (existing): WaitForTask poll jitter for fleets; per-client shared rate-limit backoff; per-call retry override                                                                                                                                                                                     | Low         | M      | Feature       |
| 35 | ROADMAP (existing): capabilities-informed retry tuning (probe feeds backoff)                                                                                                                                                                                                                              | Low         | M      | Feature       |
| 36 | ROADMAP (existing): streaming-upload design doc (demand-gated, ADR 0003 gate)                                                                                                                                                                                                                             | Low         | L      | Feature       |
| 37 | ROADMAP (existing): typed public STRING fields (phantom types) — v2 scope at the earliest                                                                                                                                                                                                                 | Low         | L      | Feature       |
| 38 | ROADMAP (existing): saved-view Update/Retrieve methods if a consumer needs editing                                                                                                                                                                                                                        | Low         | S      | Feature       |
| 39 | ROADMAP (existing): share-link conveniences (full-URL helper, document-scoped listing)                                                                                                                                                                                                                    | Low         | S      | Feature       |
| 40 | ROADMAP (existing): ProbeCapabilities caching if consumers feel the per-call cost                                                                                                                                                                                                                         | Low         | S      | Feature       |
| 41 | Remove govalid skip in `.buildflow.yml` when govalid is adopted or machine rebuilds on Go ≥ 1.27 — AGENTS.md parked item                                                                                                                                                                                  | Low         | S      | Infra         |
| 42 | Restore gosec CI scan when a go1.27-capable gosec ships (verify locally first, per AGENTS.md recipe)                                                                                                                                                                                                      | Low         | S      | Infra         |
| 43 | Decide whether parked review fixes ride `[Unreleased]` as v0.5.1 (TODO row 24 parenthetical)                                                                                                                                                                                                              | Low         | S      | Release       |
| 44 | Verify InboxClean AGENTS.md documents papersync's shared-vocabulary role (sync/verify/decrypt-repair all reuse it); add if missing                                                                                                                                                                        | Low         | S      | Documentation |
| 45 | Carry papersync's `/Encrypt` fixture edge cases into the pdfdecrypt extraction spec's fuzz/seed list when f28 triggers                                                                                                                                                                                    | Low         | S      | Documentation |
| 46 | Consider host-level guard for the `rg -r` replace-flag trap (hook has struck 3×; rewrite instead of block)                                                                                                                                                                                                | Low         | S      | Infra         |
| 47 | **Fix the load-sensitive flake** `TestWithRetryCancelDuringBackoffSurfacesRetryCanceled` (client_test.go:2106): make cancellation-vs-backoff deterministic (injectable clock or synchronized cancel hook) so the surfaced coded error is order-independent                                                | High        | M      | Bug           |
| 48 | Merge stale-claim sweeps into one quarterly pass: `.golangci.yml` wrapcheck glob + issue limits (AGENTS.md 2026-10-18 note) AND `.buildflow.yml` known-tool-bug skips (branching-flow, govalid) against current tool versions                                                                             | Low         | S      | Documentation |
| 49 | Add "consumers pin: InboxClean vX, bank-sync vX" line to TODO row templates so consumer-pin staleness (this session's row-25 bug) is grep-detectable                                                                                                                                                      | Low         | S      | Documentation |
| 50 | TestFlakeVersionMatchesChangelog-style drift guard: consider a docs-drift test asserting README documents WithRetry/WaitForTask (pairs with f21, mechanically enforced)                                                                                                                                   | Low         | S      | Quality       |

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Cross-repo TODO policy:** May I write consumer-side findings (f4 WithRetry
   adoption, f11 WaitForTask simplification, f14/f15 papersync micro-items) directly
   into InboxClean's TODO_LIST, or do you hand-curate consumer-repo backlogs? I tried
   the existing rows of both consumer repos for a precedent and found none — this is
   the one decision blocking f4/f11 from being recorded anywhere durable.
2. **Does bank-sync ingest (or plan to ingest) password-protected PDFs?** The
   pdfdecrypt extraction (f28) is demand-gated on a second consumer; your bank's
   export format is knowledge I cannot obtain from any repo — the ROADMAP idea stays
   dormant until you answer this.
3. **BuildFlow binary freshness:** the binary is built at e35e902 while the BuildFlow
   repo HEAD is 940aeae (advisory warn on every run; a concurrent session appears
   active there). Reinstall now (`nix build . && nix run .#reinstall`) or wait until
   the other session lands its work?

---

## Verification appendix (command ledger)

| #  | Command / artifact                                                                     | Result                                                                                                          |
| -- | -------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| 1  | `rg -n "paperless.New                                                                  | WithRetry                                                                                                       | NewRunner"` (InboxClean, non-test) | 3 `paperless.New` sites, none with WithRetry |
| 2  | `nix develop -c buildflow -s gomod-check --fix` (1st)                                  | 2 findings: vendor-ignored (info) + v0.7.0≠v0.7.1 (error); no auto-fix exists                                   |
| 3  | `nix develop -c go mod vendor`                                                         | modules.txt v0.7.0 → v0.7.1                                                                                     |
| 4  | `BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s gomod-check`                                 | passed with warnings (info-only)                                                                                |
| 5  | `buildflow -s nix-hash-fix --fix`                                                      | flake.nix `vendorHash` old→new sha; committed by daemon as `c0e05ff` (verified content: flake.nix only)         |
| 6  | `nix run .#check` (run 1)                                                              | "all checks passed!" — predated final TODO/ROADMAP content                                                      |
| 7  | `nix run .#check` (run 2, final tree)                                                  | FAIL: `TestWithRetryCancelDuringBackoffSurfacesRetryCanceled` under concurrent load — see (d)                   |
| 13 | `nix build <race drv>` solo                                                            | exit 0 (silent success confirmed with `&& echo`)                                                                |
| 14 | `go test -race -run TestWithRetryCancelDuringBackoffSurfacesRetryCanceled -count=10 .` | `ok` — 10/10 solo                                                                                               |
| 15 | `nix develop -c buildflow -s golangci-lint`                                            | ✓ BuildFlow passed (3.8s) — canonical lint gate green; the golangci LSP's vendor error is confirmed stale cache |
| 8  | `dprint fmt TODO_LIST.md ROADMAP.md`                                                   | 1 file formatted (padding-only diff)                                                                            |
| 9  | `rg -n go-branded-id` both consumers' go.mod                                           | InboxClean v0.5.0 (line 41), bank-sync v0.5.0 (line 57)                                                         |
| 10 | `rg -ln "qpdf                                                                          | MaybeEncryptedPDF                                                                                               | /Encrypt" ~/projects` (Go) | InboxClean-only hits; bank-sync none |
| 11 | `git show --stat c0e05ff` / `git show --stat 97340ef`                                  | c0e05ff = flake.nix; 97340ef = go.mod+go.sum only (root cause confirmed)                                        |
| 12 | `.gitignore`                                                                           | line 60: `vendor/`                                                                                              |

**Run-2 result (final tree):** FAIL on the flaky race test under concurrent load
(see (d) and appendix rows 7/13/14); everything else green — lint gate passed,
solo drv rebuild exit 0, targeted test 10/10. The vendor/vendorHash repair itself is
proven by appendix rows 3–6 and 13: the failure is a pre-existing nondeterministic
test, not the repair.
