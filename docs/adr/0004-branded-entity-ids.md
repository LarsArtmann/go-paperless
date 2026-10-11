# ADR 0004: Branded entity IDs via go-branded-id

- Status: accepted
- Date: 2026-10-11
- Deciders: Lars Artmann

## Context

Every Paperless-ngx entity reference in the SDK was a raw `int` (or
`string` for task UUIDs): `DeleteDocument(ctx, documentID int)`,
`UploadRequest.CorrespondentID int`, `EnsureTag` returning `int`, ….
Nothing stopped a caller from passing a tag ID where a correspondent ID
was expected — exactly the mixing bug bank-sync guards against in its own
domain (`internal/domain/ids.go` brands its IDs with
`github.com/larsartmann/go-branded-id` already). The
`branching-flow strong-id` analyzer reported 90 such sites across
`client.go` and `paperlesstest/`.

The ROADMAP had deferred typed public fields to "v2 scope" because typed
fields break the public API contract (bank-sync consumes this module as a
drop-in replacement for its trimmed fork). Two facts moved the ID subset
ahead of the full phantom-type migration:

1. Consumers pin module versions; a breaking **v0.5.0** breaks nobody
   until they upgrade, and the compiler walks them through every site.
2. bank-sync already depends on go-branded-id, so the dependency adds no
   new module to any consumer graph.

String phantom types (`Slug`, `Checksum`, `SortField`, … —
branching-flow's PHANTOM_TYPE findings) remain v2 scope; this ADR covers
IDs only.

## Decision

- **Every entity reference is a branded ID type** defined in `ids.go`:
  `TagID`, `CorrespondentID`, `DocumentTypeID`, `DocumentID`,
  `DocumentVersionID`, `TaskID` (string-valued), `CustomFieldID`,
  `StoragePathID`, `UserID`, `NoteID`, `ShareLinkID`, `SavedViewID` —
  all aliases of `id.ID[Brand, V]` from go-branded-id.
- **Brand types are exported** (`TagBrand`, …) with `Name()` methods, so
  IDs render as `tag:7` / `document:42` in logs and test failures, and
  paperlesstest can instantiate the same ID types in its generic
  plumbing. `NewTagID(v)`-style constructors cover construction from
  config/DB integers.
- **`DocumentID` is int64-backed**; every other entity ID is int-backed.
  Int64 matches the task-poll payloads (`result_data.document_id`,
  `related_document_ids`) and bank-sync's ledger, which already stores
  document IDs as int64.
- **The named-entity plumbing is generic over the brand**:
  `findNamed[B]`, `ensureNamed[B]`, `createNamed[B]`,
  `updateMatchingAlgorithm[B]`, `getNamedDetail[B]`, and the payload
  `namedPayload[B]` — one body, three families, distinct ID types.
  (Go 1.27 methods accept type parameters; this codebase verified it.)
- **paperlesstest shares the SDK's ID vocabulary.** Its non-test files
  import the root `paperless` package for the ID types (fixtures,
  counters, handlers) so a fake `Document.ID` compares equal to a
  `DocumentMeta.ID` without conversions. The stdlib-only contract is
  amended accordingly: no SDK *logic* is reused, only the ID types; the
  wire protocol remains the fake's source of truth, and the fake's
  shared named-family routing became free generic functions
  (`routeNamedFamily[B]`) because Go methods cannot carry type
  parameters onto free-standing handler reuse.
- **Wire format is preserved** except where noted: zero-valued IDs in
  create bodies now serialize as `"id":null` instead of `"id":0`
  (paperless-ngx serializers treat `id` as read-only and ignore both),
  and a zero `UpdateDocumentRequest.CorrespondentID` pointer now sends
  `"correspondent":null` (the server's way to clear an FK) instead of
  `0` (which DRF would reject). Round-trip behavior is unchanged; the
  fake serves `null` for unset correspondent/document_type, matching
  the real server more closely than the previous `0`.
- **Error messages and codes are byte-identical** to pre-migration
  output: every `WithContext`/`fmt.Errorf` formats `id.Get()`, never
  the branded `String()`.
- **`branching-flow strong-id` gates at zero** from now on; the tool's
  PHANTOM_TYPE analysis stays skipped (string phantom types remain v2
  scope, see `.buildflow.yml`).

## Consequences

- **Breaking public-API change**, released as v0.5.0 (pre-1.0 SemVer
  allows breaking minor bumps). Consumers (bank-sync, InboxClean) keep
  compiling on their pinned v0.4.x until they migrate; the compiler
  then enumerates every conversion site. bank-sync's `archiveMetadata`
  struct and `TaskOutcome` handling get *simpler* (no more int↔int64
  juggling at the `Duplicate()` boundary).
- Cross-entity ID mixing is now a compile error, not a live incident.
- `Get()`/`IsZero()`/`Equal()` replace `> 0` guards and `==`/`!=`
  comparisons at every boundary — slightly more verbose, explicitly
  zero-aware.
- `paperlesstest` non-test files depend on the root package; the module
  graph for consumers gains `github.com/larsartmann/go-branded-id`
  (zero-dependency, already in bank-sync's graph via the SDK itself).
- A later string-phantom migration (Slug, Checksum, …) remains possible
  as another 0.x breaking step or as v2; nothing here forecloses it.
