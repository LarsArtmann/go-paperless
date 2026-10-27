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

## Amendments

### 2026-10-27 — retry outcomes, coded sentinel, 3xx classification

Trigger: the 2026-10-27 brutal review of the error surfaces
([report](../reviews/2026-10-27_17-58_brutal-self-review.html); plan:
[superb error management](../planning/2026-10-27_18-35_SUPERB-ERROR-MANAGEMENT.md)).

- **go-retry's terminal outcomes surface their `retry.*` codes
  outermost** (`retry.exhausted` / `retry.canceled` / `retry.deadline`,
  family Infrastructure). This is the one sanctioned coded wrap above
  SDK errors: the fail-closed Infrastructure family keeps an outer
  retry loop from multiplying attempts, while the last attempt's
  `paperless.*` error stays reachable via `errors.Is`/`errors.AsType`.
  The SDK adds only the usual uncoded "request failed after retries"
  message wrap on top. Consumers programming against persistent
  rate-limit exhaustion read the cause chain (`*RetryAfterError` →
  `paperless.rate_limited`), not `Code()`. Without `WithRetry` these
  outcomes cannot occur: transport failures and cancellation surface as
  `paperless.request_failed` (Transient) directly.
- **The `ErrInvalidConfig` sentinel is coded at its return site.** The
  sentinel stays a bare `errors.New` so `errors.Is` keeps matching, but
  `New` returns it wrapped in `paperless.invalid_config` (Rejection) —
  never bare, so `Classify` no longer falls back to its fail-open
  Transient default for a construction rejection. The catalog-header
  claim ("every error this SDK constructs carries a `paperless.*`
  code") is mechanically enforced by `TestEveryConstructedErrorIsCoded`
  (`docs_drift_test.go`): every construction is coded or a `%w` wrap.
- **Unfollowable 3xx responses classify as `paperless.redirect`**
  (Rejection) instead of falling into the 5xx/Transient default —
  retrying a redirect cannot succeed. Only 3xx responses the
  http.Client hands back as-is (e.g. a redirect without a Location
  header) reach this branch; followable redirects stay transparent.
  The Location header rides along as context.

[go-error-family]: https://github.com/larsartmann/go-error-family
