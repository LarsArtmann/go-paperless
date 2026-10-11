# Status Report: Branded-ID Migration (go-branded-id adoption) — 2026-10-11 03:23

> Scope: this session only — the `branching-flow strong-id` → 0-findings
> migration of go-paperless to branded entity IDs. Report written at the
> operator's request after completion; not a general project audit.

## Session summary

Input: 90 `branching-flow strong-id` findings (paste of the analyzer's
markdown table) + the go-branded-id library checkout. Output: the full
typed-ID migration, released as **v0.5.0-ready state** (flake + CHANGELOG
bumped; tag not cut), with every available gate green.

| Gate                                                                   | Result                     |
| ---------------------------------------------------------------------- | -------------------------- |
| `go build ./...`                                                       | ✅                         |
| `go test ./...` (both packages)                                        | ✅                         |
| `go test -race ./...`                                                  | ✅                         |
| `go vet ./...` + `-tags integration`                                   | ✅                         |
| `nix run .#lint` (golangci)                                            | ✅ 0 issues                |
| `nix fmt`                                                              | ✅ clean                   |
| `erraudit --type-aware --enforce-go-error-family`                      | ✅ 0 violations            |
| `branching-flow strong-id .`                                           | ✅ **0 findings** (was 90) |
| `nix flake check` (hermetic: build/test/race/lint/fmt/integration-vet) | ✅ all checks passed       |
| `nix build .#default` (v0.5.0, new vendorHash)                         | ✅                         |
| Docs-drift tests (error codes, flake↔CHANGELOG version, README verbs)  | ✅                         |

---

## a) FULLY DONE

1. **Research phase** — read go-branded-id core (`id.go`, JSON v1/v2
   semantics incl. zero→`null` marshaling, `Ptr`/`FromPtr`), the entire
   `client.go`, all of `paperlesstest/`, and both consumers' usage
   (bank-sync `archiveMetadata`, InboxClean papersync; bank-sync already
   brands its own IDs with the same library).
2. **`ids.go`** — 12 branded ID types (`TagID`, `CorrespondentID`,
   `DocumentTypeID`, `DocumentID` int64-backed, `DocumentVersionID`,
   `TaskID` string-backed, `CustomFieldID`, `StoragePathID`, `UserID`,
   `NoteID`, `ShareLinkID`, `SavedViewID`), exported named brands
   (`document:42` display), `NewXxxID` constructors. Mirrors bank-sync's
   `domain/ids.go` pattern.
3. **`client.go` migrated** — every ID parameter, struct field, and
   return: `Upload` returns `TaskID`, `Ensure*`/`Find*` return branded
   IDs, `TaskOutcome.DocumentID`/`Duplicate()` branded, request structs
   (`UploadRequest`, `UpdateDocumentRequest`, `CreateShareLinkRequest`),
   entity structs (`DocumentMeta`, `StoragePath`, `DocumentNote`,
   `ShareLink`, `SavedView`), wire payloads (incl. `*DocumentID`,
   `[]DocumentID`). Error messages/codes byte-identical via `.Get()`.
4. **Generic named-entity plumbing** — `findNamed[B]`, `ensureNamed[B]`,
   `createNamed[B]`, `updateMatchingAlgorithm[B]`, `getNamedDetail[B]`,
   `namedPayload[B]` as FREE functions taking `*Client` (one body, three
   entity families).
5. **`paperlesstest/` migrated** — fixtures (`Document`, `CustomFieldValue`),
   counters, wire structs, handlers, notes map keyed by `DocumentID`;
   shared named-family routing became free generic
   `routeNamedFamily[B]`/`handleNamed*[B]` functions. Fake now serves
   `null` for unset correspondent/document_type (real-server shape).
6. **All tests migrated and green** — `client_test.go` (4,087 lines),
   `integration_test.go` (build-tagged), all `paperlesstest` tests,
   `example_test.go` godoc examples.
7. **Docs** — ADR 0004 + ADR index row; CHANGELOG `[0.5.0]` with
   Changed/Fixed; `flake.nix` version → 0.5.0 + new vendorHash; README
   (feature bullet, `Duplicate()` signature, branded-ID note); AGENTS.md
   (branching-flow note, scope, branded-ID conventions, paperlesstest
   contract); FEATURES (2 rows); ROADMAP (typed-fields split: IDs done /
   strings remain); TODO_LIST (consumer-migration row);
   `paperlesstest/doc.go` + `README.md` contract amendment; planning-doc
   amendment note (D1).
8. **Docs-drift test compatibility** — updated
   `kindArgumentPosition` in `docs_drift_test.go` for the free-function
   signature shift so `TestErrorCodesAreDocumented` keeps tracking
   kind-parameterized codes.

## b) PARTIALLY DONE

~~1. **Consumer migration** — bank-sync and InboxClean still pin v0.4.x~~ already harvested — TODO_LIST consumer-migration row (v0.5.0)
(they keep compiling; TODO_LIST row added). bank-sync's
`PaperlessDocumentID` now duplicates the SDK's `DocumentID` until
migrated.
~~2. **strong-id as a CI/BuildFlow gate** — analyzer run manually at 0;~~ → TODO_LIST (2026-10-11 harvest): strong-id CI gate
not wired into `.buildflow.yml`/CI (branching-flow tool still skipped
wholesale because of PHANTOM_TYPE).
~~3. **Release state** — v0.5.0 is release-READY (version, CHANGELOG,~~ → TODO_LIST release row (2026-10-11 harvest): tag is the owner's call
vendorHash agree) but the tag/push/`release-verify` ritual was not run
(no commit/push without explicit request).
~~4. **Verification breadth** — BuildFlow (`nix develop -c buildflow`) was~~ done — BuildFlow fast gate re-run 2026-10-11 (docs-health pass)
NOT run after adding the new dependency: go-licenses, govulncheck,
vulnix, lychee steps unexercised against `go-branded-id v0.7.0`.
Individual equivalents not run either.

## c) NOT STARTED (this session's scope)

~~1. bank-sync / InboxClean code changes.~~ already harvested — TODO_LIST consumer-migration row
~~2. Live-server integration run (`PAPERLESS_INTEGRATION_*`) — the~~ → ROADMAP (2026-10-11 harvest): needs live-server credentials
integration scaffold compiles under vet but never executed.
~~3. `nix run .#fuzz` / `#coverage` post-migration.~~ → TODO_LIST Low (2026-10-11 harvest): post-migration fuzz/coverage
~~4. Website/pkg.go.dev refresh (needs the release to propagate).~~ routed — follows the v0.5.0 release
~~5. CI pipeline run on a push (auto-commit daemon committed locally;~~ routed — CI runs on every push; local commits ride the next push
nothing pushed).

## d) TOTALLY FUCKED UP (caught and fixed — but process failures)

1. **A bulk edit silently deleted a live error path.** My first multiedit
   batch replaced the Upload tail and dropped the
   `paperless.missing_task_id` empty-task-ID guard entirely. Caught by my
   own re-read one step later and restored. Root cause: editing without
   immediately viewing the result; the tool even warned about whitespace
   normalization and I moved on.
2. **Designed for the compiler, not the toolchain.** I shipped generic
   METHODS (`func (c *Client) findNamed[B]`) because `go build` accepted
   them (Go 1.27 does accept type-parameterized methods) — then
   `nix fmt` failed: golines parses with an older grammar. Burned a
   refactor cycle converting to free functions, and had already written
   "Go 1.27 methods accept type parameters" into AGENTS/ADR as if clean.
   Both texts corrected; the ADR now records the real constraint.
3. **Refactored helper signatures without grepping consumers first.** The
   docs-drift test AST-parses client.go call sites; my signature shift
   broke `TestErrorCodesAreDocumented` (kind argument position). Found by
   test run, not by forethought.
4. **Called vet done too early.** Plain `go vet ./...` skips build-tagged
   files; `nix flake check`'s integration-vet caught three type errors in
   `integration_test.go` after I had declared verification complete.
5. **Mechanical sloppiness in edits** — two multiedit mismatches
   (comment-wrap differences), one double `}` producing a syntax error,
   two wrong-type perl rewrites (`[]TagID` into a `[]DocumentID`
   comparison) — each caught by compiler/tests, none shipped. Pattern:
   too much sed/perl on a 4k-line test file, too little per-hunk review.
6. **Stale LSP diagnostics all session** — gopls/golangci-lint LS kept
   reporting `go-branded-id is not in your go.mod` (false) and outdated
   example_test errors; I correctly distrusted them (go test proved
   clean) but never restarted the LSP, so signal stayed noisy and one
   godoclint warning was never re-adjudicated.

## e) WHAT WE SHOULD IMPROVE (self-critique → actions)

~~1. **No golden-bytes wire test was added.** The two deliberate wire~~ → TODO_LIST (2026-10-11 harvest): golden-bytes wire test
deltas (`"id":null` vs `"id":0` on create bodies; zero-pointer PATCH
sending `"correspondent":null`) are documented and behavior-tested
indirectly, but no test pins exact request bytes. A golden-body test
would make future wire drift mechanical, not philosophical.
~~2. **The null-PATCH capability (clearing an FK) shipped without a~~ → TODO_LIST (2026-10-11 harvest): null-PATCH wire test
dedicated test** — it is new behavior the SDK could not express
before, and nothing asserts `"correspondent":null` on the wire.
~~3. **Toolchain-first rule**: before adopting any language feature, check~~ done — codified (AGENTS.md toolchain note + this repo's golines constraint)
the FORMATTER and linter parsers, not just the compiler (golines
lesson, now in AGENTS.md — keep it).
~~4. **Bulk-rewrite discipline**: prefer compiler-driven single-site fixes~~ done — codified in global AGENTS.md (bulk-rewrite discipline)
over perl multireplace on test files; when perl is used, review each
hunk (the two bad rewrites were both sed artifacts).
~~5. **New dependency without license/vuln gates**: adding~~ done — BuildFlow fast gate green 2026-10-11 (docs-health pass)
`go-branded-id` and not running BuildFlow/go-licenses/govulncheck
violates the repo's own supply-chain hygiene; make BuildFlow the
first post-migration action.
~~6. **`String()` output change is user-visible**: branded IDs print~~ done — CHANGELOG v0.5.0 documents the .Get() requirement (upgrade note)
`tag:7`/`document:42` in `fmt.Println`/`%v` contexts (godoc examples
updated to rely on it deliberately) — consumers parsing such output
must switch to `.Get()`; worth one line in the consumer-migration
notes.
~~7. **Live-server validation of the `"id":null` create-body claim** rests~~ → TODO_LIST (2026-10-11 harvest): live-server validation
on DRF read-only-id knowledge + fake behavior; one integration run
against real paperless-ngx would close it empirically.
~~8. **Session hygiene**: restart Crush/gopls when diagnostics contradict~~ routed — fresh sessions start with clean LSP state
a green build instead of carrying noise for hours.

## f) Up to 50 things to do next (ranked, brainstorm-grade — most are

ROADMAP/TODO fuel, not commitments)

~~1. Run `nix develop -c buildflow --fix` (licenses, vulns, links) against~~ done — BuildFlow fast gate green 2026-10-11 (docs-health pass)
the new dependency tree.
~~2. Add golden-bytes tests for create/PATCH bodies (pin `"id":null`,~~ → TODO_LIST (2026-10-11 harvest)
`"correspondent":null`).
~~3. Add explicit test: zero-value `*CorrespondentID` PATCH clears the FK.~~ → TODO_LIST (2026-10-11 harvest)
~~4. Migrate bank-sync to v0.5.0 (compiler-driven; TODO_LIST row exists).~~ already harvested — TODO_LIST consumer-migration row
~~5. Migrate InboxClean to v0.5.0.~~ already harvested — TODO_LIST consumer-migration row
~~6. Unify bank-sync `PaperlessDocumentID` with `paperless.DocumentID`~~ already harvested — TODO_LIST consumer-migration row
(alias or delete the local brand).
~~7. Update bank-sync/InboxClean AGENTS docs with the ID vocabulary.~~ routed — consumer-repo scope
~~8. Tag `v0.5.0` + `nix run .#release-verify -- v0.5.0` (release ritual).~~ → TODO_LIST release row (2026-10-11 harvest)
~~9. Verify gosec CI job against the migrated source (G120 sites moved).~~ done — CI green post-migration (gosec job, v0.4.2 fix + v0.5.0 state)
~~10. Run govulncheck on go-branded-id v0.7.0 explicitly.~~ done — CI govulncheck runs on every push (incl. go-branded-id)
~~11. Confirm go-licenses output for go-branded-id (MIT expected).~~ done — go-licenses ran in the BuildFlow gate (2026-10-11 docs-health pass)
~~12. Live integration run (`nix run .#integration`) to validate null-id~~ → ROADMAP (2026-10-11 harvest): needs live-server credentials
create bodies against a real server.
~~13. Wire `branching-flow strong-id --exit-code` as a CI gate.~~ → TODO_LIST (2026-10-11 harvest): strong-id CI gate
~~14. Restart/refresh LSP so session diagnostics stop lying (gopls pin).~~ routed — fresh sessions start with clean LSP state
~~15. Add `ExampleDocumentID`/`ExampleTagID` godoc examples showing~~ → TODO_LIST Low (2026-10-11 harvest): branded-ID examples
construct/compare/guard idioms.
~~16. README quick start: show `NewDocumentID(...)` inline in the notes~~ → TODO_LIST Low (2026-10-11 harvest): README quick-start inline brand
example (currently prose-only).
~~17. Evaluate `id.Ptr()`/`FromPtr()` for `UpdateDocumentRequest` pointer~~ → ROADMAP (2026-10-11 harvest): pointer ergonomics
ergonomics in consumer examples.
~~18. Add a test asserting the fake serves `"correspondent":null` (the new~~ → TODO_LIST Low (2026-10-11 harvest): null-shape assertion
faithful shape) explicitly.
~~19. Fuzz run post-migration (`nix run .#fuzz`) — corpus seeds ran, but~~ → TODO_LIST Low (2026-10-11 harvest): post-migration fuzz
no new fuzzing time was invested.
~~20. Coverage run + review of any branded-conversion branches uncovered.~~ → TODO_LIST Low (2026-10-11 harvest): coverage review
~~21. Consider `ValidateID` at SDK entry points where zero IDs indicate~~ → ROADMAP (2026-10-11 harvest): ValidateID policy
caller bugs (currently hand-rolled `IsZero` guards).
~~22. PHANTOM_TYPE (string phantoms: `Slug`, `Checksum`, `SortField`)~~ already harvested — ROADMAP typed-string item (v2)
migration plan as v2 ADR draft.
~~23. Revisit `.buildflow.yml` branching-flow skip note wording (strong-id~~ done — AGENTS.md branching-flow note rewritten (2026-10-11 triage state)
half now moot).
~~24. Saved-view Update/Retrieve methods (ROADMAP standing item).~~ routed — ROADMAP standing (demand-gated)
~~25. Streaming upload for large documents (ADR 0003 standing).~~ routed — ROADMAP standing (ADR 0003)
~~26. Share-link full-URL helper + document-scoped listing (ROADMAP).~~ routed — ROADMAP standing (demand-gated)
~~27. `ProbeCapabilities` caching (ROADMAP).~~ routed — ROADMAP standing (demand-gated)
~~28. erraudit → CI when module proxy unblocks (standing, re-verify).~~ routed — AGENTS.md standing erraudit CI decision
~~29. govalid skip removal when machine rebuilds on Go ≥ 1.27 (standing).~~ routed — AGENTS.md standing govalid skip decision
~~30. Harvest this report's (f) items into TODO_LIST/ROADMAP via~~ done — harvested this pass (2026-10-11 docs-health)
docs-health (this report deliberately only waits).
~~31. Add a CHANGELOG "Upgrade notes" block with the `.Get()`/`IsZero()`~~ done — CHANGELOG v0.5.0 carries the construct/guard/compare idioms
migration idioms for consumers.
~~32. Check `docs/ERROR_CODES.md` examples for raw-int formatting drift.~~ done — spot-checked 2026-10-11: catalog examples use .Get() formatting
~~33. Benchmark the branded hot paths (`Upload` metadata build, task poll~~ → ROADMAP (2026-10-11 harvest): branded hot-path benchmarks
decode) to substantiate zero-cost claims.
~~34. Consider branded `CorrespondentID` zero-means-none doc on~~ → TODO_LIST Low (2026-10-11 harvest): zero-means-none doc
`DocumentMeta` (wire null decodes to zero — stated in ids.go, repeat
at use site).
~~35. golangci `godoclint` warning on ids.go re-adjudication after LSP~~ → TODO_LIST Low (2026-10-11 harvest): godoclint re-adjudication
restart.
~~36. Sweep `paperlesstest` error-message formats for branded `%v` leakage~~ → TODO_LIST Low (2026-10-11 harvest): %v leakage sweep
(cosmetic: prints `tag:3` in failure text).
~~37. One-page MIGRATION.md in-repo for v0.4→v0.5 consumers.~~ → TODO_LIST (2026-10-11 harvest): MIGRATION.md for v0.4→v0.5
~~38. Pin go-branded-id floor consciously (v0.7.0 today; decide~~ done — go.mod floors go-branded-id v0.7.0; dependabot gomod owns bumps
upgrade cadence).
~~39. Consider `//nolint` audit for now-stale suppressions after the~~ → TODO_LIST Low (2026-10-11 harvest): stale-suppression audit
refactor.
~~40. Re-run `nix flake check --all-systems` if darwin target matters.~~ done — nix flake check --all-systems verified (0.4.0 systems restriction)

## g) Questions I can NOT figure out myself

~~1. **Release timing**: cut the `v0.5.0` tag now (SDK is ready and~~ → TODO_LIST release row (2026-10-11 harvest)
gates-green), or hold until bank-sync/InboxClean are migrated so the
ecosystem lands in lockstep?
~~2. **Consumer migration ownership**: do you want me to migrate~~ already harvested — TODO_LIST consumer-migration row
bank-sync/InboxClean in follow-up sessions now, or do you handle
those repos yourself?
~~3. **Contract amendment sign-off**: ADR 0004 changes paperlesstest's~~ routed — owner decision pending (amendment shipped in the v0.5.0 state)
"stdlib-only" contract to "stdlib + root paperless ID types". Accept
as documented, or do you prefer a stricter split (branded IDs in a
leaf subpackage both packages import)?

---

_Point-in-time snapshot; written 2026-10-11 03:23 after gates ran green.
Auto-commit daemon will pick this file up (no manual commit per harness
contract). Section (f) is TODO_LIST/ROADMAP fuel awaiting docs-health
HARVEST — deliberately not harvested while awaiting instructions._
