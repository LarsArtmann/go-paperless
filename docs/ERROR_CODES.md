# Error-code catalog

Every error this SDK returns carries a `paperless.*` dot-notation code (via
[go-error-family](https://github.com/larsartmann/go-error-family)). Consumers
can switch on codes with `errorfamily.Code(err)` (or `errors.AsType`) without
reading SDK source. Codes are a compatibility contract: new codes may appear,
existing codes never change meaning.

The error model is [ADR 0002](adr/0002-error-model.md): codes and families
are assigned at the source of failure; call-site wraps add human message
context without a competing code, so `Code`/`Classify` always surface the
inner classification.

## Validation (family: Rejection — fail fast, never retry)

| Code                            | Meaning                                                                                                                                     |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `paperless.invalid_url`         | Base URL passed to `New` is not a valid absolute URL                                                                                        |
| `paperless.empty_document_id`   | Document ID argument was zero                                                                                                               |
| `paperless.empty_task_id`       | Task ID argument was empty                                                                                                                  |
| `paperless.empty_note`          | Document note text was empty                                                                                                                |
| `paperless.empty_note_id`       | Note ID argument was zero                                                                                                                   |
| `paperless.empty_share_link_id` | Share link ID argument was zero                                                                                                             |
| `paperless.empty_saved_view`    | Saved view name was empty                                                                                                                   |
| `paperless.empty_saved_view_id` | Saved view ID argument was zero                                                                                                             |
| `paperless.empty_storage_path`  | Storage path name or directory template was empty                                                                                           |
| `paperless.empty_update`        | `UpdateDocument` called with no fields set                                                                                                  |
| `paperless.invalid_retry`       | `WithRetry` passed a negative `MaxAttempts` (wraps `ErrInvalidConfig`, so `errors.Is` still matches; the offending value is in the message) |

## HTTP classification (family depends on status)

| Code                       | Meaning                                               | Family         | Retryable |
| -------------------------- | ----------------------------------------------------- | -------------- | --------- |
| `paperless.auth_failed`    | 401/403 from the server                               | Rejection      | no        |
| `paperless.client_error`   | any other 4xx (body snippet attached)                 | Rejection      | no        |
| `paperless.rate_limited`   | 429; wraps `RetryAfterError` when a hint is parseable | Transient      | yes       |
| `paperless.server_error`   | 5xx; wraps `RetryAfterError` on 503 with a hint       | Transient      | yes       |
| `paperless.request_failed` | network/transport failure before a response           | Infrastructure | yes       |

## Task outcomes

| Code                            | Meaning                                                                                                                |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `paperless.task_failed`         | Consumption task reached a failed terminal state                                                                       |
| `paperless.missing_task_id`     | Upload accepted (2xx) but the response carried no task ID (family: Corruption — the server misbehaved, not the caller) |
| `paperless.task_poll_abandoned` | `WaitForTask` context deadline expired (wraps `context.DeadlineExceeded`; last poll error attached)                    |

## Wire/encoding (family: Corruption or Infrastructure)

| Code                                                                                                                                                                                                            | Meaning                                                                                                                                                                                                           |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `paperless.decode_documents`                                                                                                                                                                                    | Document list page did not decode                                                                                                                                                                                 |
| `paperless.decode_document_metas`                                                                                                                                                                               | Document-meta list page did not decode                                                                                                                                                                            |
| `paperless.decode_storage_paths`                                                                                                                                                                                | Storage-path list page did not decode                                                                                                                                                                             |
| `paperless.decode_share_links`                                                                                                                                                                                  | Share-link list page did not decode                                                                                                                                                                               |
| `paperless.decode_saved_views`                                                                                                                                                                                  | Saved-view list page did not decode                                                                                                                                                                               |
| `paperless.decode_tags` / `paperless.decode_correspondents` / `paperless.decode_document_types`                                                                                                                 | Named-object search result (find) did not decode                                                                                                                                                                  |
| `paperless.decode_notes`                                                                                                                                                                                        | Document-notes response did not decode                                                                                                                                                                            |
| `paperless.decode_task`                                                                                                                                                                                         | Task payload did not decode                                                                                                                                                                                       |
| `paperless.decode_custom_fields` / `paperless.decode_custom_field`                                                                                                                                              | Custom-field payload did not decode                                                                                                                                                                               |
| `paperless.decode_tag` / `paperless.decode_correspondent` / `paperless.decode_document_type` / `paperless.decode_storage_path` / `paperless.decode_share_link` / `paperless.decode_saved_view`                  | Single-object payload did not decode                                                                                                                                                                              |
| `paperless.marshal_document`                                                                                                                                                                                    | Upload multipart document section could not be encoded                                                                                                                                                            |
| `paperless.marshal_tag` / `paperless.marshal_correspondent` / `paperless.marshal_document_type` / `paperless.marshal_document_update` / `paperless.marshal_custom_field` / `paperless.marshal_storage_path` / `paperless.marshal_note` / `paperless.marshal_share_link` / `paperless.marshal_saved_view`          | Request body JSON could not be encoded                                                                                                                                                                            |
| `paperless.marshal_tag_update`                                                                                                                                                                                  | Tag matching-algorithm self-heal PATCH could not be encoded                                                                                                                                                       |
| `paperless.build_request`                                                                                                                                                                                       | HTTP request could not be constructed                                                                                                                                                                             |
| `paperless.build_multipart` / `paperless.close_multipart` / `paperless.buffer_request_body`                                                                                                                     | Multipart/body preparation failed                                                                                                                                                                                 |
| `paperless.write_correspondent` / `paperless.write_created` / `paperless.write_custom_fields` / `paperless.write_document` / `paperless.write_document_type` / `paperless.write_tags` / `paperless.write_title` | A multipart form part could not be written                                                                                                                                                                        |
| `paperless.read_response`                                                                                                                                                                                       | 2xx response body could not be read                                                                                                                                                                               |
| `body_read_error` context key                                                                                                                                                                                   | Not a code: when the best-effort read of an error-response body fails mid-body, HTTP-classification errors attach the read failure under this context key so a missing/truncated snippet is explained, not silent |

## Sentinel

| Error                        | Meaning                                                               |
| ---------------------------- | --------------------------------------------------------------------- |
| `paperless.ErrInvalidConfig` | `New` called with an empty base URL or token (match with `errors.Is`) |
