# Status — paperlesstest Execution (Plan M1–M24)

_Point-in-time report, 2026-09-23 18:12 CEST. Written mid-execution: the
paperlesstest package is released; the InboxClean migration is half-landed
with one test under active debugging; bank-sync not started._

> **[2026-09-23 22:45] SUPERSEDED** by
> `2026-09-23_22-45_paperlesstest-consumers-done-status.md` — the plan
> finished the same day; all three open questions below are answered there.

## Session summary

Executed the `paperlesstest` consumer-testing-SDK plan
(`docs/planning/2026-09-23_16-19_paperlesstest-consumer-testing-sdk.md`)
end-to-end on the go-paperless side and started the consumer migrations.
The fake is complete, round-trip-verified against the real `Client`,
lint/race/buildflow green, and shipped as **v0.4.0 → v0.4.1 → v0.4.2**
(same hour — see "what went wrong" for why three tags). The InboxClean
migration (M21/M22) is mid-flight; bank-sync (M23) not started.

## a) FULLY DONE

**paperlesstest package (plan M1–M6, M8–M19)**

- Scaffold: `NewServer(tb, opts...)`, `t.Cleanup` shutdown, 404 +
  `t.Errorf` on unexpected routes, request recording (`Requests()`).
- Document list: DRF pagination (`page`/`page_size`, short-page stop),
  both checksum shapes via `WithChecksumShape(Flat|Versions)` — kills
  InboxClean's hand-encoded 3.1.0 shape.
- Upload + tasks: `post_document` multipart capture (`Uploads()`),
  `/api/tasks/` polling with the v10 TaskSerializer shape; natural path
  refuses duplicate checksums (failure + `duplicate_of`) and consumes
  everything else into a stored document carrying form metadata.
- Task scripting: FIFO `TaskPlan` queue (`WithTaskSuccess/Failure/
  Duplicate/PendingThenSuccess` + runtime `ScriptTask`); scripted success
  with `DocumentID == 0` consumes naturally while keeping pending polls.
- Fixtures: `WithDocuments`, `AddDocument` (runtime), `EditDocument`
  (mid-test mutation under lock), `ChecksumOf` (SHA-256, mirrors server).
- Runtime introspection: `SeedTask` (fixed-ID planted tasks),
  `CorrespondentCount`, `DocumentTypeID`, `CustomFieldID`.
- Faults: `Fault{Method,PathPrefix,Times,Status,Body,RetryAfter}` +
  `WithFaults`/`InjectFault`; round-tripped 5xx-then-success under
  `WithRetry`, 429 + Retry-After (seconds AND HTTP-date) →
  `RetryAfterError`, malformed-body → decode corruption.
- Auth: `WithToken` enforcement (401 DRF body, endpoints never run),
  `RequireAuthorized` / `RequireUploadCount` assertions.
- Entities: tags / correspondents / document_types / custom_fields /
  storage_paths with `name__iexact` filtering, creates, detail-by-ID,
  and the matching-algorithm PATCH (self-heal flow, `AddTag(name, 6)`
  plants legacy tags).
- Document detail: PATCH (applied + recorded via `Patches()`), DELETE
  (recorded via `DeletedDocuments()`), GET, download (original bytes).
- Notes (bare-array endpoint, newest-first, delete by `?id=`), share
  links (generated slugs, archive default, expiration), saved views
  (filter-rule round-trips), `WithAPIVersion` for probe testing.
- **Drift guard (M16)**: `coverage_test.go` parses client.go `path*`
  constants via go/ast and enforces two-way set equality against
  `servedRoutes()` — new SDK endpoints fail the suite until the fake
  speaks them.
- Docs: runnable godoc example, `paperlesstest/README.md`, root README
  section, CHANGELOG, FEATURES rows, AGENTS.md conventions bullet.

**Quality gates**

- `nix run .#check` green; package lint 0 issues; `-race` green; full
  `buildflow` run green.
- **erraudit gate restored**: AGENTS.md said it was broken upstream
  (2026-09-16) — it works again (verified exit-0) and it caught a real
  error-discard in my upload handler (fixed via defer + nolint with
  reason). Stale AGENTS.md note corrected; two obsolete TODO rows
  removed.

**Releases**

- v0.4.0 (paperlesstest + previously-unreleased [Unreleased] content),
  v0.4.1 (AddDocument/EditDocument), v0.4.2 (SeedTask + entity
  lookups) — all tagged and pushed; module proxy serves all three.
- Recovered the LOST `[0.3.2]` changelog section verbatim from the pushed
  v0.3.2 tag: main had regressed (flake said 0.3.1 while v0.3.2 existed
  on origin — the three-way version contract was silently broken).

**InboxClean (M21/M22) — infrastructure part**

- go-paperless bumped v0.3.2 → v0.4.2.
- `fakePaperlessServer` (fake 1) fully replaced by a paperlesstest
  wrapper; all `.URL` call sites migrated.
- `fakeBackfillPaperless` (fake 2, ~500 lines of hand-encoded wire
  protocol incl. the drift-hazard 3.1.0 shape comment) reduced to a thin
  wrapper over paperlesstest — zero HTTP handling left; `go vet` clean.

## b) PARTIALLY DONE

- ~~**M21/M22 test suite migration**~~ done — suite green; the three failures root-caused and fixed (00:08 report a-4/a-5).
  3 failures at last full run. Two root-caused and fixed but NOT yet
  re-verified (tests mutated `doc.CustomFields = nil` through what used
  to be pointers — now copies; converted to `EditDocument`). One under
  active debugging (below).
- ~~**WrongCorrespondent idempotency bug**~~ done — test-harness artifact (`patches()` ignored `seenPatches`); the command was always idempotent (00:08 a-1).
  `correspondent:1` again despite the fake state being verifiably correct
  after run 1 (doc.Correspondent=1; GetCorrespondentName(1) =
  "billing@example.com" via probe). The fake is provably serving the
  right data; the remaining suspects are in the command's second-pass
  comparison, not in paperlesstest. Debug instrumentation is currently
  IN the test file (must be removed).
- ~~**M20 release verification**~~ done — pkg.go.dev verified serving v0.4.2 (2026-09-23 22:40). pkg.go.dev
  indexing (404 at check time — proxy has the versions; indexing lags).
  Needs a re-run for v0.4.2 (and v0.4.0).
- ~~**TODO_LIST** row~~ done — flipped and drained (delete-on-done). still needs
  the final "done" flip and the v0.3.2-coordination row resolution
  (superseded by the v0.4.x cuts).
- ~~**Plan annotations**~~ done — plan annotated (header + all 8 DoD boxes ticked, 00:08 a-9).
  no annotations written yet.

## c) NOT STARTED

- ~~**M23**~~ done — bank-sync paperlesstest tests shipped (00:08 a-7).
- ~~**M24**~~ done — both consumers pinned, green, and pushed (00:08 a-11); CI blocked by billing (TODO_LIST).boxClean
  vendorHash heal (`nix run .#update-vendor-hash` after go.mod change);
  InboxClean full `nix run .#test-race` + lint + `nix flake check`;
  pushes of both consumers (row 22 was BLOCKED awaiting owner go — the
  "do the whole list" order was taken as that go).
- ~~Final plan definition-of-done checklist pass + plan annotations~~ done — all 8 DoD boxes ticked (00:08 a-9).
- ~~go-paperless TODO_LIST final flip~~ done — TODO_LIST drained to open rows only.ers land.

## d) TOTALLY FUCKED UP (honest list)

- **Three releases in one hour.** v0.4.0 was cut before the consumer
  migration had exercised the API; two follow-up releases (v0.4.1,
  v0.4.2) were needed for gaps a consumer-migration spike would have
  caught (AddDocument/EditDocument/SeedTask/entity lookups). The tags are
  consistent, but the cadence is sloppy.
- **Sloppy first-draft code repeatedly caught by compilers**: a phantom
  `testingStub`, an invented `httpRequest` type, a `for…else` (Go has no
  else), a wrong destructure of `TaskOutcome.Duplicate()` (order is
  docID, inTrash, refused), an inverted fault-recovery assertion, junk
  "anchor" vars (`var _ = mime.FormatMediaType` etc.) twice, and one edit
  string that accidentally contained a crab emoji (caught by the edit
  tool refusing the stale file — luck, not process).
- **Debug instrumentation got auto-committed** by InboxClean's daemon
  (3 heuristic commits) — the WrongCorrespondent t.Logf/probe lines are
  now in history and must be cleaned before M24.
- **Raced the auto-commit daemon repeatedly**: several "nothing to
  commit" surprises; two commits briefly carried heuristic messages
  until amended; one soft-reset squash was needed to restore readable
  history (M16–M19).
- **Committed before lint was green** (M8/M9): 7 findings (goconst/mnd/
  unused/cyclop) had to be fixed and folded via amend.
- **release-verify left red** (pkg.go.dev 404) without a follow-up run.

## e) WHAT WE SHOULD IMPROVE

1. **Spike consumers before tagging**: run the InboxClean/bank-sync
   migration against a LOCAL replace directive or pre-release, cut the
   tag once. Would have collapsed v0.4.0–v0.4.2 into one release.
2. **Annotate the plan file at every tier boundary** (its own header
   rule) instead of carrying state in my head.
3. **Run lint per increment, not per tier** — findings compound.
4. **Daemon awareness**: check `git status` before every commit; prefer
   amend-with-message on heuristic commits immediately.
5. **No placeholder code in `write` calls** — design the test doubles
   (tbStub etc.) before the tests that use them.
6. **Re-read AGENTS.md gotchas at gate time**: the stale erraudit note
   cost a buildflow cycle; stale docs actively mislead — fix on sight
   (done, but should be the reflex).
7. **Verify doc-drift contracts against tags, not just HEAD** — the
   v0.3.2 flake/CHANGELOG regression sat unnoticed until M20.

## f) NEXT ACTIONS (prioritized, ~30)

~~1. Remove DEBUG instrumentation from WrongCorrespondent test; rerun it.~~ done — instrumentation removed (00:08 a-4)
~~2. Root-cause the remaining run-2 repatch (suspect: command-side derived~~ done — root-caused: test-wrapper artifact (00:08 a-1)
name or a second reconcile pass), fix, verify.
~~3. Re-run the two provenance tests fixed via EditDocument conversion.~~ done — re-verified green (00:08 a-5)
~~4. Full InboxClean `go test ./cmd/inboxclean/` green.~~ done — InboxClean suite green (00:08 a-5)
~~5. InboxClean `nix run .#test-race` (AGENTS-required before push).~~ done — test-race green (00:08 a-5)
~~6. InboxClean `nix run .#lint` + `nix flake check`.~~ done — lint 0 issues + flake green (00:08 a-5)
~~7. Remove leftover unused imports (httptest/sync/strconv) if lint flags.~~ done — cleaned in the squash (cce98ed)
~~8. `nix run .#update-vendor-hash` (go.mod changed).~~ done — vendor hash updated (00:08 a-5)
~~9. Squash/rewrite InboxClean's 3 heuristic daemon commits into a clean~~ done — squashed into cce98ed (00:08 a-5)
M21/M22 commit message.
~~10. InboxClean: check docs/spec for a paperless spec needing an update.~~ done — spec updated (00:08 a-6)
~~11. InboxClean CHANGELOG entry for the fake migration (if it keeps one).~~ done — CHANGELOG entry written (00:08 a-6)
~~12. Update go-paperless TODO_LIST: paperlesstest row flip after consumers;~~ done — TODO_LIST drained
resolve the v0.3.2-coordination row (superseded by v0.4.x).
~~13. Annotate the plan file: tier checkboxes, M20 result, deviations~~ done — plan annotated (00:08 a-9)
(extra APIs, three tags).
~~14. Re-run `nix run .#release-verify -- v0.4.2` (and v0.4.0) once~~ → TODO_LIST Low (2026-10-11 harvest): release-verify formal exercise
pkg.go.dev indexes; confirm all-green.
~~15. M23: bank-sync — inventory paperless call sites.~~ done — inventory done during M23 (00:08 a-7)
~~16. M23: bump bank-sync to go-paperless v0.4.2.~~ done — bank-sync bumped to v0.4.2 (00:08 a-7)
~~17. M23: upload/task-path unit tests via paperlesstest.~~ done — shipped (00:08 a-7)
~~18. M23: checksum-reconcile tests (flat + versions).~~ done — shipped (00:08 a-7)
~~19. M23: task-poll/refusal tests.~~ done — shipped (00:08 a-7)
~~20. M23: error/retry-path tests (faults).~~ done — shipped (00:08 a-7)
~~21. M23: bank-sync suite green (+ race, + vendorHash if it vendors).~~ done — suite + race green (00:08 a-7)
~~22. M23: bank-sync commit (daemon-aware message).~~ done — committed 37e3ced6 (00:08 a-7)
~~23. M24: push InboxClean (branch was already ahead 3 pre-session).~~ done — pushed cce98ed (00:08 a-11)
~~24. M24: push bank-sync.~~ done — pushed 37e3ced6 (00:08 a-11)
~~25. M24: verify both consumers' CI (or note CI-blocked items like~~ done — CI checked; blocked by billing (TODO_LIST row)
InboxClean's armed-but-disabled CI).
~~26. go-paperless TODO_LIST: flip paperlesstest row to DONE.~~ done — TODO_LIST drained to open rows
~~27. Plan definition-of-done checklist: tick + annotate exceptions.~~ done — all 8 DoD boxes ticked (00:08 a-9)
~~28. Cross-project lesson candidate for crush-config `lessons.md`~~ routed — crush-config lessons.md (external repo, commit required)
("spike consumers before tagging" — commit, not in-session write).
~~29. Final status report superseding this one (docs-health ANNOTATE on~~ done — superseded by the 22:45 + 00:08 reports
this file).
~~30. Decide: Features/README claims about `Server.Client()` helper —~~ routed — consumer-repo scope (bank-sync's Client() assertion needs)
deliberately NOT built (stdlib-only contract); confirm that's the
standing call.

## g) QUESTIONS FOR LARS

~~1. **WrongCorrespondent run-2 repatch**: the fake is verified correct~~ done — answered: test-harness artifact (00:08 a-1)
(probe: correspondent 1 = "billing@example.com", doc.Correspondent=1)
yet the second backfill still PATCHes. Do you know an intended
command behavior that would make a re-run legitimately repatch (e.g.
a reconcile pass that re-detects from the ledger), or should I keep
digging into `backfillCorrespondent`'s run-2 inputs?
~~2. **Release cadence**: are three same-hour tags (v0.4.0–v0.4.2)~~ done — answered: accepted as-is (22:45 answers §2)
acceptable history for you, or should future consumer-driven API gaps
hold pushes until the consumer migration is green (one tag)?
~~3. **M24 pushes**: row 22 ("push consumer bumps") was BLOCKED awaiting~~ done — answered: the execution order was taken as the owner go (22:45 answers §3)
your go — I treated "do the whole list" as that go. Confirm InboxClean
(master, currently ahead 3 + this work) and bank-sync should be
pushed as the final step?
