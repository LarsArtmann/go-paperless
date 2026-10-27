## Why

`errorfamily.Code()` sees the outermost coded layer only (`classify.go:39` — first `errors.AsType[Coded]` hit in the chain wins), and on exhaustion the outermost coded layer is `retry.exhausted`. A consumer switching on the code therefore sees `retry.exhausted` even when the operationally relevant code (`paperless.rate_limited`, a provider quota code, ...) sits one `WithCause` deeper — exactly after all attempts burned, which is when that classification matters most for alerting and routing.

Hit for real in go-paperless: persistent 429 × MaxAttempts through `retry.Do` surfaces `retry.exhausted`/Infrastructure, and `paperless.rate_limited` is only reachable via `errors.AsType` digging. Our consumer docs currently teach that digging as the workaround.

## Source-level cause

```go
// retry.go:123-124 (Do), v0.8.0
return errorfamily.WrapInfrastructure(ErrExhausted, codeExhausted,
	msgExhausted).WithCause(err)
```

The coded terminal is built outermost; the last attempt error goes in as cause. Same shape at `retry.go:214-219` (`contextEnded`) for `retry.canceled`/`retry.deadline`. Caller-side can't fix it: `Config.OnExhausted` (`retry.go:119-121`) has no return value.

## Proposed contract

When `errorfamily.Code(lastAttemptErr)` is non-empty:

- `Code(terminal)` → the last attempt's code (outermost layer)
- `Classify(terminal)` → stays `Infrastructure` — fail-closed nesting must not regress (README "Exhaustion and nesting": an inner Rejection/Transient must never look retryable to an outer loop)
- `errors.Is(terminal, ErrExhausted)` → still true (exhaustion stays answerable)
- `errors.Is`/`errors.AsType` reachability of the last attempt error → unchanged
- Empty inner code → today's behavior (`retry.exhausted` outermost)

Sketch, mechanics only (family forced; inner error stays in the chain through the retained terminal):

```go
terminal := errorfamily.WrapInfrastructure(ErrExhausted, codeExhausted,
	msgExhausted).WithCause(err)
if code := errorfamily.Code(err); code != "" {
	terminal = errorfamily.WrapInfrastructure(err, code,
		msgExhausted).WithCause(terminal)
}
return terminal
```

## Design questions

- [ ] Breaking-in-minor vs opt-in? `Code()` output on exhaustion changes (`retry.exhausted` → inner code). In a 0.x repo that is a minor by convention, but an opt-in `Option` would keep code-switchers safe. I lean breaking-in-minor: `errors.Is(err, ErrExhausted)` remains the exhaustion check, and code-switching on `retry.exhausted` is exactly the pattern this retires.
- [ ] Same treatment for `contextEnded`? The outermost-code question is identical there; the multi-`%w` chain already carries both `context.Canceled`/`DeadlineExceeded` and the last attempt error.

## To verify

- [ ] `retry_test.go`: exhaustion with a coded last error → `Code()` = inner code, `Classify` = Infrastructure, `errors.Is(err, ErrExhausted)` true, `errors.AsType` still reaches the typed cause
- [ ] exhaustion with a plain (uncoded) last error → unchanged surface
- [ ] nested `Do`: outer loop still treats inner exhaustion as terminal

Filed 2026-10-27: https://github.com/LarsArtmann/go-retry/issues/3

Found while executing the go-paperless superb-error-management plan (finding F1, `docs/planning/2026-10-27_18-35_SUPERB-ERROR-MANAGEMENT.md`). Versions: go-retry v0.8.0, go-error-family v0.12.0.

💘 Generated with Crush
