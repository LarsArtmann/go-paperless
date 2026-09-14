# Status Report — 18-Skill Full Audit & Hardening

**Date:** 2026-09-13 18:37 CEST
**Scope:** One mega-session running architecture-review, architecture-visualization,
bdd-testing, brutal-self-review, code-quality-scan, data-model-review,
deduplicate-code, full-code-review, go-ecosystem-upgrade, go-error-modernization,
go-modularize, go-release, how-to-golang, library-deep-dive, naming-review,
nix-private-go-repos, nix-review, samber-do-best-practices against the whole tree.
(`improve-codebase-architecture` was requested but does not exist as a skill; its
intent was covered by architecture-review + go-modularize.)

**Starting state:** tag `v0.2.0` at HEAD, clean tree, all gates green.
**End state:** all gates green again, 6 code/config files changed, 2 new tests+
2 extended tests, 7 HTML reports, 2 D2/SVG diagrams, 4 living docs updated.

---

## Verification snapshot (all re-run at 18:33, exit codes checked not output tails)

| Gate                         | Result                                                                      |
| ---------------------------- | --------------------------------------------------------------------------- |
| `nix flake check`            | **all checks passed** (build/test/lint/fmt)                                 |
| `nix run .#test-race`        | ok                                                                          |
| `nix run .#vet`              | 0 findings                                                                  |
| `nix run .#lint`             | 0 issues                                                                    |
| `nix run .#coverage`         | **90.1%** (was 88.4%)                                                       |
| `art-dupl -t 5`              | 10 groups, 10 suppressed, **0 actionable** (all reviewed: test scaffolding) |
| `erraudit lint --type-aware` | **0 findings** (Go 1.27 dev shell)                                          |
| `go mod verify`              | all modules verified                                                        |
| LSP (gopls/golangci-lint-ls) | **still broken in-session** (see §e-7)                                      |

---

## a) FULLY DONE

1. **Every file visited and reviewed** (full-code-review): `client.go` (1,878 lines,
   pre-refactor), `client_test.go` (2,679), `example_test.go` (105), `fuzz_test.go`
   (153), `flake.nix` (229), `ci.yml` (43), `go.mod`/`go.sum`, `.golangci.yml`,
   `.crushrc`, dependabot.yml, all living docs.
2. **`go` directive normalized**: `go 1.27.1` → `go 1.27` (patch floors are a
   documented ecosystem trap). Confirmed stable through `go mod tidy` — no
   dependency re-bumps it.
3. **Pagination deduplicated**: three copy-pasted page loops
   (`ListDocumentChecksums`, `ListDocumentMetas`, `ListStoragePaths`) replaced by
   one generic `fetchAllPages[T]` (~70 lines removed). Page size, the 100-page
   cap, and the short-page stop condition now exist exactly once. All wire-level
   assertions (paginate test, page-cap test, concurrency test) pass unchanged.
4. **Double error-body read removed**: `doRequestDetail` used to cap the error
   snippet, wrap it in a NopCloser, and let `classifyStatus` re-read it. The
   snippet is now a parameter; the NopCloser dance is gone.
5. **Doc honesty restored**: `WithHTTPClient`'s comment claimed `WithTimeout`
   "has no effect" — false (options apply in order, last writer wins). The stale
   duplicated `doRequest` doc comment is merged into one accurate comment.
6. **Four lookup paths harmonized**: `FindCustomField` now sends `page_size=1`
   like `findNamed`, `FindStoragePath`, and the pagination helper.
7. **Test gaps closed** (TODO T06–T08 gaps, verified missing before writing):
   - `TestWithTimeoutBoundsSlowResponses` — server holds the request open, client
     deadline cuts it (no sleeps; waits on request-context cancellation)
   - `TestWithHTTPClientRoutesRequestsThroughSuppliedClient` — recording
     RoundTripper proves routing + token propagation
   - `TestNegotiatedAPIVersionReadsContentType` — 4-case table
   - `TestListDocumentMetasReturnsFields` extended: custom-fields round-trip
     (was `customFieldsFromPayload` 33% → now 100%; total 88.4% → 90.1%, the
     TODO's ≥90% goal met)
8. **CI supply-chain pin**: `govulncheck@latest` → `@v1.8.0`. Version verified
   against the module proxy first (`go list -m -versions golang.org/x/vuln`) —
   my unverified first guess (v1.1.4) would have pinned a 2-year-old release.
   gosec was already pinned (@v2.29.0); now consistent.
9. **flake.nix hardening**: `GOTOOLCHAIN = "local"` in default + ci devShells,
   so the flake's Go 1.27 is the single toolchain authority (nix-review
   hermeticity checklist).
10. **erraudit verdict**: zero findings. The codebase already uses
    `errors.AsType[*RetryAfterError]` (the Go 1.26+ preferred API) and `errors.Is`
    only for the `ErrInvalidConfig` sentinel — exactly the correct split per the
    go-error-modernization decision tree. Nothing to migrate, no suppressions.
11. **Clone-detection judgment**: art-dupl reported 10 groups, all suppressed by
    actionability filters. Each suppressed group was individually reviewed — all
    are test scaffolding (`t.Parallel()` preambles, `server.Close` cleanups).
    Zero harmful duplication. The two `// art-dupl:accept` markers validated.
12. **7 HTML reports** written from the kit template (byte-identical CSS via
    scripted splice, never hand-transcribed):
    - `docs/architecture-understanding/2026-09-13_18-05_modularity.html`
    - `docs/reviews/2026-09-13_18-05_code-quality-scan.html`
    - `docs/reviews/2026-09-13_18-05_full-code-review.html`
    - `docs/reviews/2026-09-13_18-05_brutal-self-review.html`
    - `docs/reviews/2026-09-13_18-05_naming-review.html`
    - `docs/reviews/2026-09-13_data-model-review.html`
    - `docs/research/2026-09-13_go-error-family-go-retry-deep-dive.html`
13. **2 D2 diagrams** rendered (exit-code gated) to
    `docs/architecture-understanding/2026-09-13_18-05-sdk-architecture{,-improved}.svg`.
14. **Living docs synced**: AGENTS.md (go-floor wording, art-dupl line-reference
    de-staled, LSP restart caveat), TODO_LIST.md (v0.2.0 row BLOCKED→DONE with
    evidence, coverage row DONE at 90.1%, T06–T08 row DONE, +1 new erraudit-CI
    row, LSP row updated with today's failed-restart evidence), CHANGELOG.md
    (`[Unreleased]` Added/Changed/Filled filled).
15. **Library deep-dive grounded, not guessed**: both dependencies verified at
    their latest releases via pkg.go.dev (fetched live today): go-error-family
    v0.10.0, go-retry v0.5.0. Adoption 88/100; the notable synergy documented
    (go-retry's default `IsRetryable` is `errorfamily.IsRetryable`, so the
    transient-only retry contract costs zero glue).
16. **Not-applicable skills dispositioned with evidence**: samber-do (zero
    imports), nix-private-go-repos (no private deps, no vendor/, hermetic via
    buildGoModule), go-modularize (2+ High "don't split" signals → single module
    is correct), go-release Phase 0 (v0.2.0 already tagged at HEAD; nothing new
    to cut).

## b) PARTIALLY DONE

1. **bdd-testing**: behavior-driven coverage gaps were closed (see a-7), but the
   suite intentionally stays stdlib table-driven + httptest — no Ginkgo/Gomega
   conversion. Deliberate Verschlimmbesserung-avoidance (a 2,700-line
   conversion adds a dependency and churn for zero behavioral gain), but it
   means the skill's letter (Ginkgo specs) is not implemented.
2. **library-deep-dive**: usage-vs-capability audit done for the two real
   dependencies; the "full potential" research leaned on pkg.go.dev summaries.
   The jitter-≤50% claim was NOT verified against go-retry source code.
3. **go-release**: Phase 0 assessment done (tag v0.2.0 == HEAD; `[Unreleased]`
   now holds today's work → next cut would be v0.2.1). No tag created (not
   requested; tags are immutable).
4. **TODO_LIST harvest**: findings that were fixed are logged; the three
   remaining forward items (integration scaffold, erraudit CI gate, consumer Go
   bump) are tracked — but the erraudit item lacks the reasoning that CI
   installability of erraudit is itself unverified (the binary may be
   private-only; see g-1).
5. **Data-model review fixes**: the report documents accepted deviations
   (raw `int` IDs, pointer PATCH semantics) but the optional typed
   `MatchingAlgorithm` enum was left as a recommendation, not implemented.

## c) NOT STARTED

1. Integration test scaffold (`//go:build integration`, env-driven) — still the
   single biggest testing gap; httptest cannot prove real pagination headers,
   HTML-only API roots, or async task persistence.
2. CI `test-race` job (race is verified locally + in flake test check, but CI
   never runs it).
3. ADR 0001 — API-version policy beyond v10.
4. `ExampleClient_WaitForTask` + retry example (README↔code drift guard).
5. T23 streaming-upload spike note.
6. Consumer flakes (bank-sync, InboxClean) bump to Go 1.27 — cross-repo work,
   unowned by this repo.
7. gosec job status investigation (TODO_LIST references a red gosec job on the
   v0.1.1 release; I never checked whether it is still red or why).
8. Timed fuzz sessions (`go test -fuzz=... -fuzztime=...`); the four fuzz
   targets only run their seed corpora in normal `go test`.

## d) TOTALLY FUCKED UP

Nothing destructive — but three real self-inflicted wounds, honestly:

1. **Claimed gates green, then flake check failed on formatting.** After adding
   tests I ran build+test+lint but not `nix fmt`/treefmt; the flake check later
   failed on one >120-char line in my new test code. Caught it and fixed within
   minutes, but the sequence was wrong: format should have preceded the green
   claim. (Lesson reinforced: the gate is `nix flake check`, not "the parts I
   remembered".)
2. **Two wasted round trips on edit hygiene**: one `edit` failed because I had
   only `cat`'d ci.yml instead of `view`ing it; one multiedit failed because the
   auto-commit daemon touched client_test.go between my read and my edit. Both
   recovered, both preventable by reading the tool rules before acting.
3. **The `.crushrc` LSP pin did not survive an in-session `lsp_restart`** —
   diagnostics still cite the host's go 1.26.7 / `GOTOOLCHAIN=local`. I updated
   the TODO row to say a full Crush restart is required, but I did not root-cause
   whether `--env GOTOOLCHAIN auto` (space-separated) is even valid Crush syntax
   vs `--env GOTOOLCHAIN=auto`. Possibly the pin itself is malformed — unverified.

## e) WHAT WE SHOULD IMPROVE

1. **Format-then-verify order**: always `nix fmt` before any green claim.
2. **FEATURES.md test-line citations are stale again** — it cites
   `client_test.go:864` etc.; my insertions shifted lines. Either de-line-number
   the citations (test names are stable, line numbers rot) or re-verify per
   docs-health VERIFY.
3. **TODO_LIST "BLOCKED" needs a staleness check on read**: the v0.2.0 row sat
   BLOCKED while the tag existed. Rule: verify the blocking condition, not the
   note, before trusting any 🔵 row.
4. **Error-code catalog**: `paperless.*` codes are scattered through client.go;
   a table in FEATURES/docs would let consumers switch on codes without reading
   the SDK.
5. **Coverage honesty**: the remaining 9.9% is dominated by unreachable
   `bytes.Buffer` write-error branches. Stop chasing the number; chase the
   integration scaffold.
6. **`GetTask` bare-array fallback**: on the array path it wraps the FIRST
   unmarshal error, not the array parse error — works, but the choice is
   implicit; deserves one comment or a deliberate error merge. Also the fallback
   double-parses every array-shaped response (two allocations per poll).
7. **LSP toolchain pin**: root-cause the `.crushrc` env syntax at next session
   start (g-3); a broken LSP makes every future session fly half-blind.
8. **Report series cross-linking**: the 7 HTML reports cross-reference by name;
   a tiny index (`docs/reviews/INDEX.md`) would make the series navigable.
9. **Manual commits were skipped on purpose** (critical rule: never commit unless
   asked) — the auto-daemon committed everything with generic
   "auto-commit N changed file(s)" messages, losing the detailed messages the
   review skills asked for. If detailed history matters, say "commit" and I'll
   write proper messages going forward.

## f) UP TO 50 THINGS TO GET DONE NEXT (impact-ordered, evidence-backed)

**High impact**

1. Bump bank-sync flake to `go_1_27` (blocks all `go get` of v0.1.1+)
2. Bump InboxClean flake to `go_1_27`
3. Decide: retire bank-sync's trimmed fork in favor of this module
4. Integration test scaffold (`//go:build integration`, `PAPERLESS_URL/TOKEN`)
5. CI: add `test-race` job
6. Verify `govulncheck@v1.8.0` actually runs green in CI (next push)
7. Investigate the red gosec job (still red? why? fix or document)
8. ADR 0001: API-version policy beyond v10

**Medium impact**
9. erraudit CI gate — first verify a public install path exists (g-1)
10. `ExampleClient_WaitForTask` + `WithRetry` example
11. Error-code catalog (`paperless.*`) in FEATURES.md
12. De-line-number FEATURES.md test citations (or re-verify them)
13. `TestListStoragePathsCapStopsAtMaxPages` (parity with the checksums cap test)
14. `TestListDocumentMetasCapStopsAtMaxPages` (same parity)
15. Context-cancellation test for a plain request (non-`WaitForTask` path)
16. Timed fuzz sessions on all four targets (`-fuzztime=30s` each)
17. ProbeCapabilities hook-coverage assertion (probes go through
`doRequestDetail`, so hooks fire — assert it)
18. Comment or fix the `GetTask` bare-array error-wrap choice
19. Avoid the double allocation in `GetTask`'s array fallback
20. Typed `MatchingAlgorithm` (unexported enum)
21. Harmonize `t.Context()` vs `context.Background()` across 6 older tests
22. Harmonize `defer server.Close()` vs `t.Cleanup(server.Close)`
23. Pin GitHub Actions by SHA (actions/checkout@v7 etc.)
24. dependabot: add `github-actions` ecosystem
25. SECURITY.md: confirm token-redaction guidance covers response Set-Cookie
26. Consider gitleaks + trivy in CI (how-to-golang security stack)
27. v0.2.1 release cut once the next batch lands
28. Release checklist section in CONTRIBUTING.md (tag discipline)
29. `docs/reviews/INDEX.md` — navigate the report series
30. ADR directory convention (`docs/adr/`) for ADR 0001

**Lower impact / polish**
31. Append the red-gosec note to the v0.1.1 GitHub release body (blocked on g-2)
32. T23: streaming-upload spike note (io.Reader body + Content-Length)
33. `BenchmarkUpload` (multipart construction cost)
34. Inline `documentListPage` into `ProbeCapabilities` (only remaining user)
35. Retry-observer hook via go-retry `OnRetry` (only when a consumer asks)
36. `retry.ErrExhausted` fast-path consideration (only when needed)
37. README: add a `WithRetry(RetryPolicy)` snippet to the Options section
38. README: document `Default*` constants for retry/poll tuning
39. ROADMAP.md harvest + review (not touched this session)
40. PR template (.github/PULL_REQUEST_TEMPLATE.md)
41. Fuzz corpus: commit any interesting crashers found in f-16
42. `ExampleNew_withOptions` — extend to show hooks (redaction reminder)
43. Consider `t.Chdir`-free lint? N/A — skip; replace with: review
`.golangci.yml` linter set against how-to-golang defaults (e.g. add
`gochecksumtype`, `err113`-style rules if desired)
44. flake.nix: `version` is hardcoded "0.2.0" — either derive from a release
file or document the accepted deviation in a comment
45. Add `checks.test-race` to flake.nix so `nix flake check` runs race in CI
46. Docs: explain the Find/Get/Ensure verb contract in FEATURES or README
47. Consider publishing the report series as a docs site (website-launch skill)
48. `client.go` file split (transport/tasks/named/documents) at ~3k lines
49. Post-restart: confirm `.crushrc` clears the LSP failures, then close the
TODO row
50. After 3–5: re-run this session's full gate battery as the regression check

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **erraudit in CI**: the binary exists locally but its public installability is
   unverified (the go-error-modernization skill itself lists erraudit as
   "unverified, likely private"). Should the CI gate (a) wait until erraudit is
   publicly installable, or (b) be wired now with private-runner auth, or (c) be
   dropped in favor of a golangci-lint custom analyzer?
2. **v0.1.1 release body**: append the red-gosec note (supersede posture), or
   leave the release body untouched? The tag itself stays immutable either way —
   this is purely a communication decision you owned and never answered.
3. **Consumer repo ownership**: do you want me to go bump the bank-sync and
   InboxClean flakes to `go_1_27` from here (cross-repo edits in their checkouts),
   or is that work owned by dedicated sessions in those repos?

---

_Everything above is from this session's runs; no claims copied from older
reports without re-verification. Point-in-time snapshot: 2026-09-13 18:37 CEST._
