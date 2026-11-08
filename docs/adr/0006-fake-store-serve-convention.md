# ADR 0006: Fake store==serve — one wire-tagged struct per entity family

- Status: accepted
- Date: 2026-11-08
- Deciders: Lars Artmann

## Context

paperlesstest is the consumer-testing fake (plan:
`docs/planning/2026-09-23_16-19_paperlesstest-consumer-testing-sdk.md`). Its
original design kept internal store structs (state fixtures) separate from
wire structs (HTTP serialization), five entity families at a time: `namedEntity`

- wire payload, `customField` + payload, `storagePath` + payload, `savedView`
- `savedViewRule`, `shareLink` + payload. Every read paid a conversion, every
  new field had to be threaded through two structs, and the branching-flow
  review (2026-10-11 triage) flagged the pairs as DUPLICATE_TYPE findings.

The alternative — one struct per family, serving the store struct directly —
had one catch: unexported store state (mutex-adjacent fields, future
counters) must never leak onto the wire.

## Decision

**One struct per entity family, wire-tagged, served as-is.** The store
struct IS the wire struct: `Document`, `SavedView`, `ShareLink`, the named
entities, and the custom-field definition carry their JSON tags on the
single definition, the store keeps `[]*Document` (etc.), and serving hands
the same values to the encoder (paperlesstest/documents.go `documentWireFor`
builds the wire view only where the store struct carries fields the wire
must not show — today that is exactly one case, `Document.Content` bytes).

Rules that keep this honest:

1. **A field is either always-served or never-served.** Per-caller wire
   shapes are not expressible; if an endpoint must hide a field, that field
   belongs in a separate struct (the escape hatch, used for `Content`).
2. **Fixtures speak the SDK's branded vocabulary** (ADR 0004 amendment):
   `Document.ID` is a `paperless.DocumentID`, `AddTag` returns a
   `paperless.TagID`. Fake state compares equal to SDK results without
   conversion — that contract is WHY store==serve works: both sides decode
   and encode the same shape.
3. **The wire is the truth, the route set is enforced.** Every route the
   client speaks must be registered in the fake's `servedRoutes` and
   round-trip through the real `Client`; `TestFakeCoversEveryClientRoute`
   (paperlesstest/coverage_test.go:76) enforces set equality with the
   client's path constants in both directions. A new SDK endpoint fails the
   suite until the fake speaks it.
4. **Unexported types cannot be shared across packages** — the fake
   re-declares the SDK's unexported wire mirrors (e.g. `namedPayload`);
   those declarations carry the "SDK↔fake wire-shape mirror is deliberate"
   comment (documents.go:295) and are pinned by the round-trip suites.
   Duplicating a PRIVATE shape is the mirror contract, not drift.

## Consequences

- The 2026-10-11 consolidation merged the five pairs, deleting ~10 internal
  structs and every conversion (CHANGELOG [0.5.0]); consumed faults disarm
  via pointer identity (`slices.DeleteFunc`) instead of index arithmetic.
- Adding a field to a family = one struct edit + fixture default + (if
  served) a round-trip assertion. Before, it was two struct edits plus a
  conversion.
- `faults.go` disarms consumed faults by pointer identity — possible only
  because store and wire share one struct identity.
- The route-coverage test is the completeness gate: a fake that under-serves
  the client is a red suite, not a silently green lie.

## Verification

- `TestFakeCoversEveryClientRoute` (both directions).
- The round-trip suites in `paperlesstest/*_test.go` drive the REAL client
  against the fake for every family.
- `TestDocumentListServesFullMetaShape`-style shape pins assert the served JSON carries
  every field the SDK decodes (DocumentMeta.Content was added exactly this
  way, CHANGELOG [0.5.1]).
