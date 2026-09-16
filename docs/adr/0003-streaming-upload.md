# ADR 0003: Streaming upload is not promised

- Status: accepted
- Date: 2026-09-16
- Deciders: Lars Artmann

## Context

`Upload` takes `Content []byte` and buffers the whole multipart body in
memory before sending. ROADMAP carries a raw idea: "streaming multipart
upload for sources larger than the memory-safe envelope". The risk of the
raw idea is that consumers assume it is a promise — it is not, and this
record exists so nobody plans around one.

The technical constraints a streaming design must satisfy:

- **HTTP requires a known length on this path.** Paperless-ngx's
  `post_document` endpoint consumes a multipart body; without chunked
  transfer support end-to-end, `Content-Length` must be known up front.
  A bare `io.Reader` has no length, so the client would have to buffer
  (what we do today), stat a file (`*os.File` special case), or use
  chunked encoding — which the server side may not accept.
- **Retries replay bodies byte-for-byte.** `WithRetry` re-sends the
  request on transient failures. A stream, once read, is drained; replay
  would need a re-readable source (`*os.File` seek, or re-opening a
  callback), a breaking signature change.
- **The consumers do not need it.** InboxClean and bank-sync upload
  statement PDFs and scans — kilobytes to a few megabytes, far inside the
  memory-safe envelope. SECURITY.md already tells users to size uploads
  accordingly.

## Decision

- The `Upload` signature stays `Content []byte`. Streaming upload is
  **not** on the SDK's promise list.
- If a real consumer arrives with documents that stress memory, the shape
  to evaluate first is a body-builder callback (`func() (io.Reader, int64,
  error)`) — it preserves Content-Length and replay, unlike a bare
  `io.Reader` parameter.
- Until then, large-document callers own the memory sizing, as documented
  in SECURITY.md.

## Consequences

- No false expectations: the ROADMAP idea stays an idea, and any future
  design starts from the callback shape, not from `io.Reader`.
- Cost: a future streaming API will be a new method or a v2 signature
  break — accepted, because pretending today's signature could grow a
  stream is worse.
