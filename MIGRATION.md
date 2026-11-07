# Migrating from v0.4.x to v0.5.x

v0.5.0 replaced every raw entity reference with a branded ID type
(ADR 0004). Error codes, messages, families, and endpoint behavior are
unchanged — what changed is the Go types at the API surface and two
JSON shapes on the wire. Consumers pin module versions, so nothing
breaks until you upgrade; the compiler then walks you through every
site. This guide is the human-readable version of that walk.

## The one-sentence version

Everywhere you passed an `int` (or a `string` task UUID), pass a
branded ID (`paperless.DocumentID`, `paperless.TagID`, …); construct
one from a raw value with `paperless.NewTagID(3)` and read the raw
value back with `.Get()`.

## Branded-ID idioms

```go
// v0.4: raw ints flow everywhere; any int fits any slot.
tagID, err := client.EnsureTag(ctx, "inbox")        // int
err = client.DeleteDocument(ctx, 42)                 // any int compiles

// v0.5: each entity family has its own type; mixing is a compile error.
tagID, err := client.EnsureTag(ctx, "inbox")        // paperless.TagID
err = client.DeleteDocument(ctx, paperless.NewDocumentID(42))
```

### Constructors

`NewTagID(3)`, `NewDocumentID(42)`, `NewTaskID("uuid")` — one per
entity family (`ids.go` lists them all). Zero values are valid and
mean "unset": a zero `DocumentID` marshals as JSON `null` and guards
with `.IsZero()`. Constructors from an empty string or a negative
number panic — construct from data you trust, or guard first.

### Reading raw values back

`.Get()` returns the underlying `int64`/`string` — that is what goes
into `strconv.Itoa`, SQL parameters, log fields, and `fmt` verbs:

```go
// v0.4
log.Printf("uploaded document %d", docID)
// v0.5
log.Printf("uploaded document %d", docID.Get())
```

**Never print a branded ID with `%v` and never store `.String()` as
the canonical form.** `.String()` renders `"Brand:value"`
(`"document:42"` — a debug aid for test failures and logs), while
error messages and error codes produced by the SDK keep using
`.Get()`-style raw values, so what you paste into a Paperless-ngx URL,
a task ID query, or a database column must come from `.Get()`. The
`.Equal` method compares two IDs of the same family; `==` also works
(struct equality) but `.Equal` reads better in diffs.

### Guarding

`if docID.IsZero()` replaces `if docID == 0`. The paperlesstest fake's
fixtures and introspection speak the same branded types, so fake state
compares equal to SDK results without conversions (ADR 0004
amendment).

## The two wire deltas

Both are pinned by the wire-shape golden tests (`wire_golden_test.go`)
and both matter only if you mirror the SDK's HTTP payloads yourself
(for example in a hand-rolled fake — prefer paperlesstest):

1. **Create bodies send `"id":null`, not `"id":0`.** A zero-value
   branded ID marshals as JSON `null` (Paperless-ngx serializers
   ignore read-only IDs, so behavior on the server is unchanged).
2. **FK clearing sends explicit nulls, not `0`.**
   `UpdateDocumentRequest.CorrespondentID`/`DocumentTypeID` set to a
   pointer to a zero ID PATCH `"correspondent":null` /
   `"document_type":null` — the server's documented way to clear a
   nullable FK. Sending `0` was a DRF invalid-primary-key rejection;
   sending nothing was a silent no-op (that silent no-op was itself a
   v0.5.2-era bug under encoding/json/v2, fixed and pinned — see
   CHANGELOG [Unreleased]).

## paperlesstest fixtures

`AddDocument`/`AddTag`/`SeedTask` and friends take and return branded
IDs (`Document.ID paperless.DocumentID`, `AddTag` →
`paperless.TagID`). If your tests built fixtures from raw ints,
wrap: `paperless.NewDocumentID(42)`.

## Upgrade checklist

- [ ] `go get github.com/larsartmann/go-paperless@v0.5.2` (or newer)
- [ ] compile: fix every "cannot use X (variable of type int) as
      paperless.DocumentID" by wrapping with the matching
      `New…ID` constructor
- [ ] grep your repo for `.String()` on branded IDs used outside
      debug logs; canonical forms use `.Get()`
- [ ] grep for `== 0` / `== ""` on former ID variables; replace with
      `.IsZero()`
- [ ] if you hand-rolled a fake, swap to
      `github.com/larsartmann/go-paperless/paperlesstest` (see its
      package doc and ADRs 0005/0006) or at minimum apply the two
      wire deltas above
- [ ] run your suite; if anything wire-shaped asserts bytes, compare
      against `wire_golden_test.go`'s pinned bodies
