# Tasks: library-slot-prune

## 1. Prerequisite (enhancements repo, outside this change's edits)

- [x] 1.1 Confirm the `enhancements/0009` D9 in-place revision has landed (the reserved-slots half now states removal-now with re-introduction by the execution half; cancellation half unchanged). Do not start 2.x before it exists.

## 2. opm/kernel

- [x] 2.1 Remove the `Clock` interface, `systemClock`, and the `logger`, `tracer`, `clock` fields from `Kernel` in `opm/kernel/kernel.go`; drop the corresponding defaults from `New` (including the `log/slog` discard logger and the noop tracer) and the now-unused imports.
- [x] 2.2 Remove `WithLogger`, `WithTracer`, `WithClock`; keep `WithSchemaLoader` and `WithRegistry` and update the `Option` doc comment.
- [x] 2.3 Update `opm/kernel/kernel_test.go`: delete `fakeClock`, `TestNew_WithClock`, `TestNew_WithTracer`, the nil-option acceptance cases for the removed options, and any logger-option assertions; keep the remaining option tests green.
- [x] 2.4 Rewrite the construction/options prose in `opm/kernel/doc.go` (defaults list, "threads cross-cutting dependencies" framing).

## 3. Dependencies

- [x] 3.1 `task tidy`: verify `go.opentelemetry.io/otel/trace` and the indirect `go.opentelemetry.io/otel` leave `go.mod`/`go.sum`.

## 4. Docs

- [x] 4.1 `docs/getting-started.md`: drop the `kernel.New(kernel.WithLogger(myLogger))` example and the logger/tracer/clock options sentence.
- [x] 4.2 `CLAUDE.md` and `README.md`: remove the logger/tracer/clock wording from the kernel descriptions (including the Constitution-adjacent "logging is caller-passed via parameter" phrasing where it names the kernel slots rather than the principle).
- [x] 4.3 Update `openspec/specs/kernel-runtime/spec.md`'s Purpose line (drop "cross-cutting dependencies (logger, tracer, clock)") per the delta.

## 5. Consumer migration (same PR wave)

- [x] 5.1 `opm-operator/cmd/main.go`: delete the `kernel.WithLogger(...)` argument and the then-unused `log/slog` + `logr` bridge imports; `task dev:fmt dev:vet` in `opm-operator`.
- [x] 5.2 Verify `cli` builds unchanged against the pruned kernel (no source edit expected).

## 6. Validation

- [x] 6.1 `task check` in `library` (fmt, vet, lint, test).
