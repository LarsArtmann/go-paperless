# ADR 0002: Error model — coded at source, uncoded context wraps

- Status: accepted
- Date: 2026-09-14
- Deciders: Lars Artmann

## Context

The SDK reports every failure through [go-error-family]: errors carry a
`paperless.*` dot-notation code, a family (Rejection / Transient /
Corruption / Infrastructure), and structured context. Two structural
questions surfaced when driving `erraudit --enforce-go-error-family`
toward zero:

1. Call sites wrap inner errors with plain `fmt.Errorf("verb object:
   %w", err)` before returning — a stdlib constructor, which the
   enforcement flag reports as a violation.
2. `erraudit --enforce-generic-return` (opt-in) flags every method that
   returns the bare `error` interface.

## Decision

- **Errors get their code at the source of failure** (`classifyStatus`,
  decode sites, validation guards). One failure, one code, one family.
- **Call-site wraps stay uncoded `fmt.Errorf` with `%w`.** They add
  human context (which upload, which tag, which page) without a
  competing outer code. `errorfamily.Code(err)` and `Classify(err)`
  return the **outermost** classified error in the chain, so an
  `errorfamily.Wrap*` at the call site would shadow the inner HTTP
  classification (`paperless.rate_limited`, `paperless.auth_failed`, …)
  and could flip its family — a tested contract
  (`TestUpload429CarriesRetryAfterHint` asserts the wrapped family stays
  Transient through the call-site wrap).
- **Deliberate exception sites carry `//nolint:erraudit // reason`** so
  the gate `erraudit ./... --type-aware --enforce-go-error-family
  --disable-extensions` stays green while still catching genuinely new
  stdlib error construction.
- **Methods keep returning the `error` interface.** Per-method concrete
  error types (an `UploadError`, a `GetTaskError`, …) would bloat the
  API, break the drop-in compatibility contract with bank-sync, and buy
  nothing the code + family + context triple does not already provide.
  `--enforce-generic-return` stays off (the tool itself documents bare
  `error` returns as standard Go practice).
- **One code, one meaning, one family.** A code must never collide
  across failure sources (the original `paperless.empty_task_id` served
  both "caller passed an empty task ID" (Rejection) and "server returned
  no task ID" (Corruption) — the latter is now
  `paperless.missing_task_id`).

## Consequences

- Consumers switch on stable inner codes; human context rides the
  message; structured context rides `WithContext` (`%+v` renders it).
- `errors.Is`/`errors.AsType` keep working through every wrap, and the
  `ErrInvalidConfig` sentinel stays matchable even where a coded error
  wraps it (`paperless.invalid_retry`).
- Adding an operation-level code later would be a deliberate catalog
  change (new codes may appear; existing codes never change meaning),
  not a mechanical lint fix.

[go-error-family]: https://github.com/larsartmann/go-error-family

