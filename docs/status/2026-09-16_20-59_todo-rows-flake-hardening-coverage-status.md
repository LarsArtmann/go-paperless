# Status Report — TODO-List Rows, Flake Hardening, Coverage (2026-09-16 20:59)

_Report written per status-report skill; format override honored: user explicitly requested `.md`, skill default is styled HTML. Section (f) is the docs-health HARVEST input — do not let it die in this timestamped file._

Session scope: resume from the 17:57 summary's post-execution state, then execute
every actionable row in `TODO_LIST.md` (3 TODO rows remained: ci.yml audit,
`--all-systems`, GOEXPERIMENT simd) plus the highest-value items from the 17-57
harvest list. The concurrent dependency-upgrade session is confirmed FINISHED
(its 18-18 report ends with final verification on clean HEAD `ebd50d4`).

Final state: **`nix flake check` exits 0** (build, test, test-race, lint,
format, integration-vet), golangci-lint **0 issues**, dprint clean on all
touched files, suite green, coverage **90.3% → 91.2%**.

## (a) FULLY DONE

Code and gates (all verified in-session):

1. **ci.yml audit** (TODO row 1): govulncheck ✓ (pinned v1.8.0), gosec ✓
   (fail-closed `Files > 0` pattern), test-race ✓ (deliberate redundancy, AGENTS
   documented). **Fuzz coverage was missing** — added:
   - `.#fuzz` flake app: runs all six fuzz targets for a per-target duration
     (`nix run .#fuzz -- 5m`, default 30s). Verified 6/6 targets execute, exit 0.
   - `.github/workflows/fuzz.yml`: weekly scheduled (Mon 04:00 UTC) + manual
     dispatch with a budget input. Deliberately NOT a per-push gate, matching
     the recorded verdict. Dependabot coverage verified (github-actions on `/`).
2. **`nix flake check --all-systems`** (TODO row 2): ran it; it exposed a real
   landmine — **nixpkgs 26.11 dropped x86_64-darwin**, so the flake could not
   even EVALUATE on Intel Macs (verified error, not guessed). Fixed by dropping
   x86_64-darwin from `systems` (inlined the list, removed the nix-systems
   input; aarch64-darwin and aarch64-linux evaluate fine). Post-fix plain
   `nix flake check`: green; the omitted-systems warning now lists only the
   aarch64 systems, whose builds need remote builders (documented, expected).
   flake.lock verified free of the stale input entry.
3. **GOEXPERIMENT simd verification** (TODO row 3): researched in-toolchain
   (`go doc internal/goexperiment.Flags` + grep of go1.27.1's flags.go): the
   `SIMD` experiment only enables the `simd` stdlib package and its intrinsics;
   `rg -i simd --type go` proves this module never imports it. **Dropped** →
   `goExperiment = "jsonv2"`. Also removed the hardcoded `GOEXPERIMENT=jsonv2,simd`
   re-set inside the release-verify app. AGENTS.md and README updated.
4. **Coverage recovery**: `writeUploadMetadata` 68.8% → **100%** (all five
   multipart field-write failure branches, code+family asserted per branch);
   `writeCustomFieldsField` 81.8% → **90.9%** (write-failure branch covered; the
   remainder is the structurally unreachable `json.Marshal` branch — Value is a
   `string`, marshal of `{Field int, Value string}` cannot fail). Total
   **90.3% → 91.2%**.
5. **Verb-drift test extension**: `TestREADMEListsLookupVerbs` now pins 10
   prefixes (Add/Create/Delete/Download/Ensure/Find/Get/List/Update/Upload).
   It **immediately caught 4 undocumented methods** — `DownloadDocument`,
   `ListDocumentChecksums`, `ListDocumentMetas`, `UpdateDocument` were prose
   only ("list checksums/metadata, update metadata, download, delete") — now
   name-checked in README's features list. Test green.
6. **Harvest quick wins**: README retry-policy defaults paragraph (the four
   `DefaultRetry*` constants); `TaskOutcome.Duplicate()` return order documented
   `(documentID int64, inTrash bool, refused bool)` — verified against
   client.go:493-504, not guessed; `ExampleClient_ProbeCapabilities`;
   `BenchmarkListChecksums` (100-doc page payload; verified running, ~13.6ms/op);
   FEATURES.md "Development and CI" section (drift tests, hermetic gates,
   fuzzing); SECURITY.md slug-entropy caveat + fuzz-scan sentence.
7. **Docs kept honest**: CHANGELOG [Unreleased] extended (fuzz app + workflow,
   coverage numbers with the unreachable-branch caveat, GOEXPERIMENT slim,
   x86_64-darwin drop); TODO_LIST.md reduced to exactly the 5 genuinely BLOCKED
   rows (the 3 done rows deleted per the list's own legend); README Development
   block now lists every app (`.#coverage`, `.#fuzz`, `.#integration`,
   `.#release-verify` were missing).
8. **Final verification**: `nix flake check` exit 0 on the final code state,
   golangci 0 issues after fixing its 5 findings (all in my new code), dprint
   clean on every file I touched.

## (b) PARTIALLY DONE

1. **Harvest items considered and dismissed this session** (recorded here, not
   yet in ROADMAP as dismissal notes): #19 error-family representative test
   (families already asserted in behavior tests; the catalog organizes by
   family sections and links ADR 0002 — a per-section Classify test duplicates
   existing coverage); #37 erraudit placeholder check (`nix flake check` has no
   "skip" concept — an always-pass placeholder is noise; the AGENTS.md note is
   the durable record). Effort to route: S.
2. **Fuzz depth**: the new `.#fuzz` app makes deeper budgets one command away,
   but no long run has happened yet (15s/target smoke runs only this session).
3. **The 5 BLOCKED rows** are verified still blocked (see (c)) — the release
   prep itself (version bump + tag plan) is unstarted pending the user's
   versioning call.

## (c) NOT STARTED (all deliberately blocked or demand-gated)

1. **Cut the release** (v0.3.2 vs v0.4.0 — user decision): CHANGELOG is
   written; consumers (bank-sync pins v0.3.0) wait.
2. **Push consumer bumps** (bank-sync, InboxClean): pushing is excluded without
   explicit instruction.
3. **erraudit upstream rebuild + hermetic flake pin**: external repo; needs
   go-ahead (its v0.4.0 bundles go1.26-era x/tools and rejects this module).
4. **bank-sync fork retirement**: belongs to bank-sync.
5. **Live integration run** (`nix run .#integration`): no
   PAPERLESS_INTEGRATION_URL/TOKEN available in this session; e2e remains
   compile-verified only.
6. All demand-gated ROADMAP items (ProbeCapabilities caching, share-link
   conveniences, saved-view PATCH) — no consumer request observed.

## (d) TOTALLY FUCKED UP (honest list)

1. **Wrote garbage Nix syntax into flake.nix TWICE while escaping awk** —
   backtick-wrapped `` `''$0` `` nonsense, then "fixed" it by introducing nix
   string concatenation inside an indented string (also nonsense). Both caught
   by `view` before any build, but I fumbled the same line three times because
   I kept typing backticks instead of thinking character-by-character. The real
   fix (delete the awk, single package + `grep '^Fuzz'`) took one attempt once I
   stopped fighting escaping and simplified the construct.
2. **The fuzz app shipped two latent bugs before working** (three run cycles
   instead of one): (i) `go test` consumed the loop's stdin — only 1 of 6
   targets ran; (ii) my awk (`{t=$0; next} /^ok/ {print $2, t}`) kept only the
   LAST target name per package — the stdin "fix" didn't address this because I
   misdiagnosed the cause from the symptom; a standalone pipeline debug revealed
   the awk. The counting check (`rg -c "=== fuzz"` → 6) is what proves the app
   correct; I should have designed the loop around that check from the start.
3. **Trusted unverified library behavior**: assumed `WriteField` on a closed
   `multipart.Writer` errors (memory of old Go behavior) — the first test run
   failed on go1.27 (no error). My own AGENTS.md lesson says never trust
   unverified claims; a 30-second experiment would have saved a cycle. Fixed
   with an always-failing `io.Writer` (deterministic, honest).
4. **`BenchmarkListChecksums` first draft called `server.Close()` immediately
   after creating the server** — it would have benchmarked a closed server.
   Caught by self-reviewing the diff BEFORE running (the one catch that worked
   as designed), but the draft had the bug.
5. **Pipeline masking, AGAIN**: `dprint check … | tail; echo DPRINT=$?` printed
   `DPRINT=0` while dprint had just reported "Found 1 not formatted file" —
   `$?` was `tail`'s. This is the exact failure class BOTH sibling reports
   documented as their cardinal sin and my own AGENTS.md warns about. Third
   session in a row. Only the visible tail output saved the verdict.
6. **Two wasted round trips on my own `nix fmt` rewrites**: the edit tool's
   mtime guard tripped twice because I formatted flake.nix between read and
   edit — a self-inflicted variant of the concurrent-session race.

## (e) WHAT WE SHOULD IMPROVE

1. **Pipeline discipline is still not absolute** (three sessions running).
   Concrete fix: never attach `$?` to a pipeline — `cmd > log 2>&1; e=$?` or
   `PIPESTATUS`, every gate, every time. Better: a Crush hook that rewrites or
   rejects `$?` after a pipe in Bash tool calls. This should become tooling,
   not a memory note.
2. **Verify library behavior with a micro-experiment before writing tests
   against it.** 30 seconds in a scratch test beats a fail-cycle.
3. **If an edit needs three attempts, replace the construct, not the escape.**
   The awk-in-nix fiasco was avoidable at attempt 2 by choosing grep/simpler
   shell. Rule: third try on the same line = redesign that line.
4. **Flake apps need a smoke checklist**: run once with a tiny budget AND count
   the iterations (targets processed, not just exit 0). Exit 0 over a loop that
   did one-third of its work is a false green — exactly the gosec
   `Files > 0` lesson, now for my own script.
5. **Markdown has no format gate**: treefmt carries no markdown formatter and
   dprint isn't in the flake, so unformatted md survives every gate (the
   sibling's 18-18 report proves it). Either wire dprint into treefmt/nix fmt
   or formally accept ungated markdown; the current half-state needs the
   AGENTS.md workaround note forever.
6. **Dismissed ideas need dismissal notes in the repo**, not just in a
   timestamped report (see (b)1) — otherwise the next session re-litigates them.
7. **The verb-drift extension paid for itself instantly** (4 real README gaps).
   Generalize: any README line that describes methods in prose is a drift risk;
   prefer name-checkable rows and let tests enforce them.

## (f) UP TO 50 NEXT THINGS (harvest-ready; impact-ordered)

In-repo, actionable (impact / effort S<30min M<2h L>2h / category):

1. Cut the release carrying the typed enum + dep floors + this session's work
   (v0.3.2 vs v0.4.0 per user call; CHANGELOG ready) — Critical / S / Release
2. Push this session's work, then push consumer bumps (explicit go-ahead) —
   Critical / S / Release
3. bank-sync pre-flight: confirm `RuleType: int` call sites survive the typed
   enum (`var x int = rule.RuleType` breaks; untyped constants don't) before
   rollout — High / S / Quality
4. Rebuild erraudit upstream on go1.27 x/tools; restore the dev-shell gate —
   High / M / Tooling (blocked on go-ahead)
5. Publish erraudit to the public module proxy; add the CI job per the recorded
   two-condition decision — High / M / Tooling
6. Live-run `TestIntegrationUploadReconcile` (`nix run .#integration`) and
   record results — High / M / Quality (needs server env)
7. First live `nix run .#release-verify -- <tag>` exercise post-release —
   Medium / S / Release
8. Verify the scheduled fuzz workflow fires (manual dispatch once now; first
   cron run Monday 04:00 UTC) and red-fails correctly on a planted bug —
   Medium / S / Quality
9. CI fuzz job: upload the crash corpus + repro test on failure so a red fuzz
   job is diagnosable — Medium / S / Quality
10. Overnight fuzz budget (`nix run .#fuzz -- 1h`) before the release; record
    exec counts in CHANGELOG — Medium / M / Quality
11. Coverage: `GetTask`/`classifyTask` bare-array + envelope edge mix —
    Medium / S / Quality
12. Coverage: `doRequestDetail` hook-failure + header-echo branches —
    Medium / S / Quality
13. Coverage: `Upload`-level `build_multipart`/`write_document` entry branches
    (CreateFormFile failure path) — Medium / S / Quality
14. Wire govulncheck + gosec as hermetic flake checks (currently GitHub-runner
    only) — Medium / M / Tooling
15. Seed-corpus growth for the fuzzers: feed real paperless-ngx response
    shapes as `f.Add` seeds — Medium / M / Quality
16. CONTRIBUTING.md: mention `.#fuzz`, `checks.integration-vet`,
    `.#integration`, `.#release-verify` — Medium / S / Docs
17. Route the sibling 17-57 report's un-routed idea items 45–50 (saved-view
    value typing, WaitForTask jitter, token bucket, retry override, fixture
    suite) into ROADMAP — Medium / S / Docs
18. Write dismissal notes for (b)1's two dismissed items into ROADMAP so they
    are not re-litigated — Medium / S / Docs
19. AGENTS.md: record the writeShellApplication loop pattern (stdin consumption
    - count-iterations check) as a gotcha — Medium / S / Docs
20. Options-table drift test: pin the README `WithX` options list against
    `New`'s variadic options — Medium / S / Quality
21. ADR 0001 consequence test: assert the `Accept: application/json; version=…`
    header on every request (currently implied by Ping only) — Medium / S /
    Quality
22. Research whether `GOEXPERIMENT=jsonv2` itself is still needed on go1.27
    (json/v2 is default there) or droppable like simd — Low / S / Cleanup
23. Decide the markdown-gate question: dprint into treefmt, or accept ungated
    md and delete the workaround note — Medium / S / Tooling (decision)
24. `dprint fmt` the sibling session's 18-18 status report (1 file, pre-existing
    finding) — requires permission to touch their file — Low / S / Cleanup
25. Consolidate the four same-day status reports (sibling suggestion) into a
    per-day annotated record — Low / M / Docs
26. Split `docs/status/` annotated-vs-unarchived (two candidates for
    annotation/archival) — Low / S / Docs
27. Benchmarks: `BenchmarkEnsureTag` (name-resolution hot path) — Low / S /
    Quality
28. ADR 0003 seed: turn the streaming-upload sketch into a design doc IF the
    ROADMAP item ever leaves demand-gate — Low / M / Design
29. ProbeCapabilities caching per client lifetime (ROADMAP, demand-gated) —
    Low / M / Feature
30. `GetShareLink`/`ListShareLinks(documentID)` convenience (demand-gated) —
    Low / S / Feature
31. Saved-view Update/PATCH (demand-gated) — Low / M / Feature
32. `ShareLink.URL(baseURL)` helper (demand-gated) — Low / S / Feature
33. Examples: `ExampleClient_DownloadDocument`, `ExampleClient_UpdateDocument` —
    Low / S / Docs
34. README "Common tasks": download + update snippets — Low / S / Docs
35. `ExampleClient_Upload` runnable variant via httptest inside client_test.go —
    Low / M / Docs
36. release-verify app: add a fuzz smoke step (30s/target) to the post-release
    ritual — Low / S / Release
37. ROADMAP demand-gate re-check against bank-sync's actual usage before the
    v0.3.2 rollout (which conveniences do consumers really want?) — Medium / S /
    Planning
38. e2e scaffold: extend with a share-link roundtrip (create/list/delete) —
    Low / M / Quality
39. e2e scaffold: extend with a note roundtrip — Low / M / Quality
40. Fuzz: property assertion that unknown `SavedViewRuleType` IDs round-trip at
    the fuzz level (unit-tested today) — Low / S / Quality
41. `.golangci.yml`: review err113 scope (it fired on a test helper sentinel;
    confirm tests should be held to the same bar) — Low / S / Tooling
42. CI: `nix flake check --all-systems` with an aarch64 remote builder, or
    formally accept + document the warning — Low / M / Infra (needs hardware)
43. CHANGELOG "Verified" section: record this session's all-systems run and
    simd research (fold into release prep) — Low / S / Docs
44. AGENTS.md: record the empirical `multipart.Writer.WriteField`-after-Close
    behavior (no error on go1.27) — Low / S / Docs
45. ROADMAP: WaitForTask poll jitter (fleet syncs) — Low / S / Idea
46. ROADMAP: per-client shared rate-limit backoff (only on 429 storms) — Low /
    M / Idea
47. ROADMAP: record/replay API fixture suite from a live server to extend the
    e2e without a container — Low / M / Idea
48. ROADMAP: retry-policy per-call override (opt-in option param) — Low / S /
    Idea
49. Docs: link ERROR_CODES.md rows to their asserting tests (drift test already
    pins existence; test links aid debugging) — Low / M / Docs
50. Consider a `docs-health` HARVEST pass right after this report (per the
    skill's handoff rule) — Medium / S / Docs

## (g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Release version**: ship the accumulated work (typed enum = compile-visible
   change, dep floors, fuzz CI, x86_64-darwin drop, catalog fixes) as **v0.3.2**
   or fold into a later **v0.4.0**? I read ADR 0001's stability story and the
   CHANGELOG; I cannot know what you want consumers (bank-sync pins v0.3.0) to
   swallow first. My lean: v0.3.2 per the drop-in contract, since untyped
   constants keep compiling — but the call is yours.
2. **erraudit ownership**: do you want me to go fix erraudit upstream NOW
   (rebuild on go1.27 x/tools, release, publish), or is that repo mid-work and
   the TODO rows should stay BLOCKED until you say go? I cannot see what else
   is happening in that repo.
3. **Live server access**: is there a PAPERLESS_INTEGRATION_URL /
   PAPERLESS_INTEGRATION_TOKEN I may use to run `nix run .#integration` for
   real? The upload-reconcile e2e is complete and compile-verified but has
   never executed against a live Paperless-ngx — I found no credentials in the
   session environment and will not guess endpoints.
