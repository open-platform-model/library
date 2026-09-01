## Why

The kernel's three dependency-injection slots (logger, tracer, clock) have had no reader since they were added: `k.logger`, `k.tracer` and `k.clock` are stored by `New` and never consulted by any kernel operation (verified 2026-09-01; enhancement 0009 D9's rationale records the same measurement). The tracer slot is also the sole reason `go.opentelemetry.io/otel/trace` is a direct dependency. Enhancement 0009 D9 previously reserved the slots for the execution half; the owner has reversed that reservation (D9 is being revised in place in `enhancements/0009`): the kernel carries no write-only surface, and the execution half re-introduces the injection surface it actually needs when it lands.

## What Changes

- **BREAKING** `opm/kernel`: remove the `Clock` interface, the unexported `systemClock`, and the options `WithClock`, `WithTracer` and `WithLogger`. The `Kernel` struct drops its `logger`, `tracer` and `clock` fields and keeps `cueCtx`, `schemaLoader`, `schemaCache` and `registry`; the remaining options are `WithSchemaLoader` and `WithRegistry`.
- `go.mod`: `go.opentelemetry.io/otel/trace` (and the indirect `go.opentelemetry.io/otel`) leave the dependency set; the `log/slog` discard logger in `New` goes with the logger field.
- Docs: rewrite the `New` defaults and options prose in `opm/kernel/doc.go`, `docs/getting-started.md` (the `WithLogger` construction example) and the `CLAUDE.md`/`README.md` lines that describe the kernel as threading logger/tracer/clock through every operation.
- Landing order: the `enhancements/0009` D9 revision (the reserved-slots half becomes "removed now, re-introduced by the execution half"; the cancellation half of D9 is untouched) lands before this change is implemented.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `kernel-runtime`: the dependency-injection requirement ("SHALL provide at minimum `WithLogger`, `WithTracer`, and `WithClock`") is removed; the `New` defaults requirement and its scenarios no longer name a logger or clock; the capability's purpose statement no longer describes logger/tracer/clock as Kernel-owned cross-cutting dependencies.

## Impact

- Packages: `opm/kernel` (`kernel.go`, `kernel_test.go`), `go.mod`/`go.sum`.
- Consumers: MAJOR under SemVer (Principle VI). `opm-operator` is the one caller: `cmd/main.go` drops its `kernel.WithLogger(slog.New(logr.ToSlogHandler(...)))` argument (and the then-unused logr/slog bridge imports); the removal changes no observable behavior because nothing ever read the logger. `cli` passes none of the three options and keeps compiling unchanged. Consumers migrate in the same PR wave (pre-GA); recorded per the migrations policy (pre-GA: changelog and archive, no fragment).
- Enhancements: implements the revised 0009 D9 (declared in `enhancement.yaml`). The execution half re-adds logger/tracer/clock injection in whatever shape the planner and runner need; that re-introduction is 0009's to design, not a restoration of these symbols.
