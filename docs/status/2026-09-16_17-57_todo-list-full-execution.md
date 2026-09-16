# TODO-List Full Execution — Status Report (2026-09-16 17:57)

_Report written per status-report skill; format override honored: user explicitly requested `.md`, skill default is styled HTML. Section (f) is the docs-health HARVEST input — do not let it die in this timestamped file._

Session scope: execute the ENTIRE `TODO_LIST.md` (10 Medium + 10 Low TODO rows,
2 High BLOCKED rows) end to end — READ/UNDERSTAND/RESEARCH → execute → verify,
per item. A second concurrent session was active the whole time (BuildFlow
adoption, dependency bumps, its own status reports) — its work is referenced
only where it collided with mine, not researched.

Final state: **`nix flake check` exits 0** (build, test, test-race, lint,
format, integration-vet), dprint clean, erraudit gate freshly discovered
broken (environmental, upstream).

## (a) FULLY DONE

Code (all verified by the suite + `nix flake check`):

1. **Typed enum for saved-view `rule_type`** — `SavedViewRuleType` distinct
   type + 16 constants, every ID verified against the official
   `filter-rule-type.ts` on paperless-ngx main (fetched; 6 = has-tags-ALL,
   correcting the old comment's vaguer "has tag"). Unknown IDs pass through
   (tested). `ShareLinkFileVersion` turned out to already exist (pre-done);
   the enum task's remaining half was rule_type.
2. **Doc-drift tests** (`docs_drift_test.go`): error-code catalog test
   (AST-parses client.go, resolves dynamic code construction at
   findNamed/ensureNamed/createNamed/getNamedDetail/updateMatchingAlgorithm/
   fetchAllPages call sites, requires a literal kind arg — a non-literal
   fails the test so new sites can't hide codes), flake-version ↔ CHANGELOG
   test, README lookup-verb test (reflection over exported methods).
3. **Real bug found by the drift test**: `findNamed` emitted
   `paperless.decode_document types` (space in the code!) for multi-word
   kinds. Fixed to `decode_document_types`.
4. **Catalog drift fixed**: nine live codes were missing from
   ERROR_CODES.md (decode_tags/correspondents/document_types, decode_tag/
   correspondent/document_type, marshal_tag/correspondent/document_type,
   marshal_tag_update) — all added.
5. **Cap-parity tests** for `ListDocumentMetas` + `ListStoragePaths` — all
   five listings now have `CapStopsAtMaxPages`.
6. **Tests tail**: `defaultTransport` honors exported idle-pool constants;
   `WithHTTPClient(nil)` keeps the default transport (defensive branch +
   live request); `UpdateDocument` asserts custom_fields wire shape AND the
   empty-`TagIDs`-omitted contract (the wipe-tags side effect guard).
7. **Fuzzers** for savedView/shareLink payload decoding — 15s each, ~2.7M
   execs apiece, zero failures, mapping invariants asserted.
8. **Coverage recovery**: named the top uncovered blocks (delete-family
   error paths, self-heal PATCH failure), closed them; **88.6% → 90.3%**,
   above the old 90.1% baseline. DeleteShareLink/DeleteSavedView/
   DeleteDocument now at 100%.
9. **Examples batch**: `ExampleClient_WaitForTask` (interval param +
   correct `Duplicate()` destructuring — my first draft used a wrong
   signature; the compiler caught it), `ExampleWithRetry`;
   `ExampleNew`/`ExampleNew_withOptions` are now runnable with `// Output:`
   and their `testableexamples` suppressions dropped.
10. **Integration e2e** `TestIntegrationUploadReconcile`: upload (unique
    timestamped content) → WaitForTask → duplicate-refusal check →
    checksum-must-appear-in-listing (with three-way failure diagnosis) →
    title/tag assertions → `t.Cleanup` delete. Compiles via vet+build with
    the `integration` tag (type error `int64`→`int` caught and fixed by
    that gate — exactly what it exists for).
11. **gosec CI fail-closed**: step asserts `Files > 0`. **The original
    grep pattern was wrong** — local gosec run showed the summary format is
    `Files  : N` (spaces BEFORE the colon), which my first pattern would
    not match (false red). Pattern corrected and verified against real
    gosec output. Caught by refusing to trust an unverified format.
12. **Flake wiring**: `checks.integration-vet` (in `nix flake check`), and
    opt-in apps `.#integration` (live run) + `.#release-verify -- vX.Y.Z`
    (the whole post-release ritual: tag/CHANGELOG/flake agreement, push
    status, CI green, pkg.go.dev indexing, race run on the tagged
    worktree).
13. **Lint-fix audit (commit 3f058b2, all 8 read as hunks)**: dupl ×5 =
    documented nolint suppressions with reasons (2× 429/503 pair, 3× cap
    tests — correct call); errorlint = `%v last poll error` → dual `%w`
    (semantically right: lastErr's coded chain becomes errors.Is-reachable,
    nothing relied on non-matching; test asserts the message); wrapcheck =
    real contextual wrap `"request failed after retries: %w"` in doRequest;
    nonamedreturns = `Duplicate()` unnamed returns, no clarity lost;
    modernize-wg = `wg.Add/Done` → `wg.Go(func(){})`, live in tree.
14. **Tagged-release verification**: fresh worktrees at v0.2.0, v0.3.0,
    v0.3.1; v0.3.1 `nix build` clean; v0.2.0/v0.3.0 `checks.build/test/lint`
    ALL GREEN in-sandbox (6/6 explicit exit codes). **Verdict: no red gate
    shipped inside any tag** — the failed `go-paperless-0.2.0` derivation
    was the v0.3.0 commit's drift-named build and it passes. Worktrees
    removed after.
15. **Docs**: SECURITY.md share-link threat-model paragraph; README
    "Common tasks" snippets (notes/share links/saved views, signatures
    checked against code) + custom-fields row (satisfying the verb test);
    ADR index + template + ADR 0003 (streaming upload NOT promised, with
    Content-Length/replay constraints); ROADMAP cross-link + integration
    line updated; FEATURES.md typed-enum rows; CHANGELOG [Unreleased]
    rewritten (merged with the concurrent session's entries);
    TODO_LIST.md reduced to the 3 genuinely blocked rows.
16. **.crushrc GOTOOLCHAIN item**: settled without a restart —
    `crush_info` shows gopls + golangci_lint_ls both `ready` with the
    project `.crushrc` loaded this session, and the skill docs confirm
    `--env KEY VALUE` (space form) is the canonical syntax. Removed as done.

## (b) PARTIALLY DONE

1. **Integration e2e live run** — code is complete and compiles in the
   flake gate, but never EXECUTED against a real Paperless-ngx (no
   PAPERLESS_INTEGRATION_URL/TOKEN available in this session). It remains
   compile-verified only.
2. **erraudit gate** — intended to just "record the CI decision", ended up
   diagnosing a fresh breakage instead: erraudit v0.4.0 bundles go1.26-era
   x/tools ("This application uses version go1.26 of the source-processing
   packages...") and rejects this module outright — reproduced on the
   pristine v0.3.1 worktree, so NOT caused by this session's code. AGENTS.md
   corrected (the "exits 0" claim was stale), CI decision updated to two
   unblock conditions, TODO row added. The gate itself is down until
   erraudit upstream rebuilds — outside this repo.
3. **Fuzz depth** — 15s per fuzzer is smoke-depth, not overnight depth. New
   fuzzers pass; longer budgets would be the next increment.

## (c) NOT STARTED

1. Nothing from the TODO_LIST's TODO rows — all 20 are done or re-homed
   (the erraudit-rebuild row is new and BLOCKED by design).
2. The 2 pre-existing High BLOCKED rows (push consumer bumps; bank-sync
   fork retirement) were verified as still-blocked and intentionally not
   touched — pushing is excluded without explicit instruction, and the
   fork decision belongs to bank-sync.

## (d) TOTALLY FUCKED UP (honest list)

1. **`rm -f /dev/null` in a command line** — a careless cleanup attempt
   after a throwaway build. It failed (non-root) and `/dev/null` was
   verified intact immediately, but writing that at all was a lapse.
   Should never appear in any command.
2. **First gosec CI pattern was wrong** (`Files:` vs the real
   `Files  :` format) — would have shipped a gate that false-fails. Caught
   pre-push by running gosec locally, which is the exact lesson, but the
   initial pattern was written from memory instead of from evidence.
3. **Race with the concurrent session** — my first flake.nix edits were
   written against stale content and got overwritten wholesale by the
   other session's rewrite; I detected it (file-mtime guard tripped) and
   re-applied, but cleaner would have been re-reading immediately before
   every multi-block edit in a known-concurrent repo. Same class: one
   CHANGELOG edit initially clobbered the other session's dependency-bump
   entry; re-read and merged before writing.
4. **Small code mistakes caught by gates (fine in aggregate, sloppy at
   source)**: struct-conversion compile error after the enum change;
   `WaitForTask` example using a wrong signature + wrong `Duplicate()`
   destructure order; `int64`→`int` in the e2e; two leftover lines in the
   first docs_drift_test draft; tagged-switch/wsl findings in the new
   self-heal test (the ONLY lint findings in the final gate — in my code).
5. **`echo exit=$?` after pipelines** — twice printed 0 for failed builds
   because `tail` masked the exit (my own memory file warns about exactly
   this pipeline-masking class). Both times the real failure was caught by
   re-running with explicit exit capture.

## (e) WHAT WE SHOULD IMPROVE

1. **In a multi-agent repo, re-read files immediately before every edit**
   (mtime-guard is reactive; a strict read-just-before-edit loop is
   proactive).
2. **Every format string / summary pattern in CI must be verified against
   real tool output** before landing — no pattern from memory.
3. **Exit codes must be captured before any pipe** (`cmd > log; e=$?` then
   inspect), never after `| tail`.
4. The erraudit breakage argues for **hermetic tool pinning**: the gate's
   toolchain dependency (x/tools version inside the binary) is invisible
   until it breaks; `go version -m $(which erraudit)` belongs in the
   gate's failure message.
5. Coverage numbers flatter the delete-family now but `writeUploadMetadata`
   (68.8%) and `writeCustomFieldsField` (81.8%) are the next real gaps.
6. ADR 0003's callback-shape sketch should be the seed of a design doc IF
   the streaming idea ever leaves ROADMAP.

## (f) UP TO 50 NEXT THINGS (harvest-ready)

In-repo, actionable (impact-ordered):

1. Live-run `TestIntegrationUploadReconcile` against a real Paperless-ngx
   (`nix run .#integration`) and record results.
2. Push this session's work + the consumer bumps (explicit go-ahead
   required), then run `nix run .#release-verify -- v0.3.1` end to end as
   the ritual's first live exercise.
3. Rebuild erraudit upstream on current x/tools; restore the dev-shell
   gate to exit 0; then close the TODO row.
4. Publish erraudit to the public module proxy; add the CI job per the
   recorded decision (conditions a+b).
5. Coverage: `writeUploadMetadata` error branches (68.8% — correspondent/
   created/custom-fields/tags/title write failures).
6. Coverage: `writeCustomFieldsField` (81.8%) malformed-field paths.
7. Coverage: `GetTask`/`classifyTask` bare-array + envelope edge mix.
8. Coverage: `doRequestDetail` hook-failure and header-echo branches.
9. Overnight/30-min fuzz budgets for the four payload fuzzers (CI opt-in
   app, not per-push).
10. Release v0.3.2 (or fold into v0.4.0) so consumers can pick up the
    space-in-code fix + typed enum; CHANGELOG is already written.
11. bank-sync side: confirm `RuleType: int` call sites survive the typed
    enum (untyped constants compile; `var x int = rule.RuleType` does not)
    before the v0.3.2 rollout.
12. Add a `GetShareLink`/`ListShareLinks(documentID)` convenience only if
    a consumer actually asks (ROADMAP demand-gate holds).
13. Saved-view Update/PATCH — same demand-gate.
14. `ProbeCapabilities` caching per client lifetime (ROADMAP item) if
    consumers feel the per-call cost.
15. Consider `expath`-style context helpers? NO — rejected: scope creep
    beyond client-only. (Kept here to show the idea was considered and
    dismissed, not missed.)
16. `dprint fmt` hook into `nix fmt` decision: either pin dprint plugins
    inside treefmt or drop the duplicate formatter — the current two-formatter
    setup needs the AGENTS.md workaround note.
17. Wire `govulncheck` + `gosec` into `nix flake check` as hermetic checks
    (currently only GitHub-hosted runners run them).
18. Add `.#fuzz` flake app wrapping the four fuzzers with a duration flag.
19. Error-code test: assert catalog rows' family labels against
    `errorfamily.Classify` at least for one representative code per family.
20. Verb drift test: extend to `List*`/`Create*`/`Delete*`/`Add*`/
    `Update*`/`Upload*` prefixes (currently only Find/Get/Ensure per the
    original task).
21. README: retry-policy constants table (DefaultRetry* values are
    documented in code, invisible in README).
22. README: document `TaskOutcome.Duplicate()` return order explicitly
    (named at call sites only).
23. ADR 0001 consequence check: add a test asserting `apiVersion` is sent
    on every request (currently implied via Ping test only).
24. Split `docs/status/` annotated-but-unarchived reports (two 2026-09-16
    files from the concurrent session are candidates for annotation).
25. FEATURES.md: rows for the doc-drift tests + release-verify app (CI/
    DX features are absent from the inventory).
26. CONTRIBUTING.md: mention `checks.integration-vet` and the
    `.#integration` opt-in app.
27. `nix flake check --all-systems` in CI matrix (aarch64-darwin/linux
    currently omitted everywhere).
28. Dependabot: verify it covers the new GitHub Actions pins after the
    concurrent session's ci.yml test-race addition.
29. `.golangci.yml`: add `testifylint`-equivalent? NO — no testify here;
    rejected. (Consideration recorded.)
30. Consider `paralleltest` for `docs_drift_test.go`'s three tests — done
    in this session; verify golangci_lint_ls LSP warnings (stale) clear
    after a Crush restart — the ONE `.crushrc` item that still needs the
    restart-cycle proof.
31. Benchmarks: `BenchmarkUpload` exists; add `BenchmarkListChecksums` for
    the pagination hot path.
32. Example: `ExampleClient_ProbeCapabilities` (capability probe has no
    example; FEATURES advertises it).
33. Example: `ExampleClient_Upload` still non-runnable; a runnable variant
    using `httptest` is possible inside `client_test.go` (not example_test).
34. Error docs: link ERROR_CODES.md rows back to ADR 0002 family rules.
35. SECURITY.md: mention share-link slug entropy is server-controlled
    (SDK cannot guarantee unpredictability — upgrade note for very
    sensitive documents).
36. Consider `ShareLink.URL(baseURL string)` helper — currently README
    does string concat; demand-gate first (ROADMAP line exists).
37. flake: `checks.erraudit` placeholder that SKIPS with the upstream
    pointer instead of silently being absent, so the gate's absence is
    visible in `nix flake check` output.
38. Make `TODO_LIST.md` "Status legend" table survive dprint reformat
    (emoji-width churn observed twice today).
39. Decide release cadence: CHANGELOG [Unreleased] now carries a
    breaking-ish field-type change; consumers need a tag soon (see 10).
40. Post-release: verify pkg.go.dev re-indexing after the next tag (the
    release-verify app checks it; first run pending).

Upstream/external (their repos, tracked here):

41. erraudit: rebuild + release with go1.27 x/tools (see 3, 4).
42. BuildFlow: GOTOOLCHAIN caller-wins contract feedback already filed by
    the concurrent session — verify it landed upstream.
43. paperless-ngx: nothing owed; upstream rule-type table already matched
    (no drift to report).
44. go-error-family/go-retry: concurrent session bumped floors; nothing
    owed from this session.

Backlog ideas (not refined — belongs in ROADMAP if kept):

45. Saved-view rule VALUE typing (dates/tags have string values today).
46. `WaitForTask` jitter on the poll interval for fleets of syncs.
47. Rate-limit backoff shared across concurrent calls (per-client token
    bucket) — only if a consumer hits 429 storms.
48. Retry-policy per-call override (opt-in option param).
49. Structured request/response hook context via generics? — rejected
    shape; hooks are value snapshots deliberately.
50. Generate a lightweight API-shape fixture suite from a live server
    (record/replay) to extend the e2e without a container.

## (g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Release intent**: should the current [Unreleased] (typed enum =
   compile-visible change + code-catalog fixes) ship as **v0.3.2**
   (patch-flavored, per ADR 0001's stability story) or fold into a later
   **v0.4.0**? I can cut either; I cannot know which you want consumers
   to swallow first, given bank-sync's drop-in contract.
2. **erraudit ownership**: the v0.4.0 binary's x/tools is too old for this
   module — do you want me to go fix erraudit upstream NOW (its repo,
   rebuild + release + publish), or is erraudit mid-refactor elsewhere and
   I should leave the TODO row BLOCKED until you say go?
3. **Concurrent-session protocol**: a second session was rewriting
   flake.nix/ci.yml/CHANGELOG while I worked; my rule was
   read-before-edit + merge-don't-clobber. Is that session still active
   (I will keep merging), or is it done (I can then do one reconciling
   pass over its BuildFlow/dependency work to make sure nothing of mine
   or its got lost)?
