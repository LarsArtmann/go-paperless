# ADR 0005: Fake pagination policy — bounded client walks, envelope-shaped fake

- Status: accepted
- Date: 2026-11-08
- Deciders: Lars Artmann

## Context

Every Paperless-ngx list endpoint (documents, metas, checksums, tags,
correspondents, document types, custom fields, storage paths, share links,
saved views) is DRF-paginated. The SDK walks pages itself; the paperlesstest
fake must answer those walks in a way that keeps fake state and SDK results
comparable (the ADR 0004 contract). Two policies had to be fixed, because a
drift in either silently corrupts every list-derived result:

1. How many pages the SDK will walk, at what page size, and when it stops.
2. What the fake serves per page and what terminates a walk.

## Decision

**Client walk (the SDK side):** every paginated list scan requests
`page_size=100` (`documentListPageSize`, documents.go:107) and walks at most
100 pages (`maxDocumentListPages`, documents.go:111) — a hard cap of 10,000
items per call. The walk terminates on the **short page**: a page serving
fewer than `page_size` results ends the scan. `Next` links are never
followed; the page counter is the only cursor. This keeps the walk
deterministic, immune to hostile or malformed link values, and trivially
fuzzable (the fuzz targets feed arbitrary page bodies).

**Fake serving (the paperlesstest side):** the fake serves the DRF envelope
shape (`{"count": N, "next": null, "previous": null, "results": [...]}`) with
`Next`/`Previous` always null — the paperlesstest envelope type pins this
(paperlesstest/documents.go:222): "the SDK walks pages via the page query
parameter and terminates on the short page, never on the links". The fake
slices its store by the requested `page`/`page_size` (`pageBounds`,
paperlesstest/documents.go:238) with `page_size` defaulting to 100 —
mirroring the SDK's scan size — and clamps overflowing page arithmetic
(absurd page values serve an empty page, never panic; regression-pinned by
`TestPageBoundsClampsOverflowingPagination`).

**Why the fake serves exactly one page per request and never needs
pagination fixtures:** a walk against the fake hits `page=1` with the full
collection (or a short page when the fixture count < 100), which exercises
the same terminate-on-short-page code path production uses for small
servers. Cap behavior is tested at the unit level
(`TestListDocumentMetasCapStopsAtMaxPages` and siblings) with a fake that
serves exactly `maxDocumentListPages` full pages plus one.

## Consequences

- A consumer with more than 10,000 items of one kind silently truncates at
  the cap. This is a documented bound, not a bug; raising it is a
  one-constant change plus a benchmark.
- The fake cannot simulate mid-walk pagination (page 2 of 3). When a test
  needs it, the cap-stop siblings show the pattern: serve full pages and
  assert the walk stops at the cap, or add fixture counts that cross one
  page boundary.
- Terminating on the short page means a server that always serves exactly
  full pages with one final empty page still terminates; a server that
  serves full pages forever stops at the cap. Both behaviors are pinned.

## Verification

- `TestListDocumentChecksumsPaginates`, `TestListStoragePathsPaginatesAndMaps`,
  `TestListShareLinksPaginates` pin multi-page walks against page-shaped
  fakes.
- The `*CapStopsAtMaxPages` sibling tests pin the cap termination.
- `TestPageBoundsClampsOverflowingPagination` pins the fake's overflow
  clamp.
