## Context

See proposal.md, Why. The 2026-09-01 kernel review re-verified the 0009 D9 measurement: `k.logger`, `k.tracer` and `k.clock` are assigned in `New` and by their options and read by nothing; `kernel_test.go`'s option tests assert acceptance only ("Clock is internal; we exercise the option to confirm it is accepted"). The tracer slot is the sole reason `go.opentelemetry.io/otel/trace` is a direct dependency.

Constraints that shape the approach:

- Enhancement 0009 D9 currently reserves the slots and explicitly rejected this removal. 0009 is `draft`, so D9 MUST be revised in place (enhancements repo rules) before this change lands; the revision, not this change, is where the reversal is decided and recorded.
- Principle VI: removing the options is MAJOR. Pre-GA, consumers migrate in the same PR wave and the record is changelog plus archive (migrations policy, dormant until GA).
- Principle I is unaffected either way: the kernel never logged; removing the slots removes the appearance of an observability story, not an actual one.

## Goals / Non-Goals

**Goals:**

- Remove the three slots, their options and their tests in one change, leaving `WithSchemaLoader` and `WithRegistry` as the option set.
- Shrink the dependency set: `go.opentelemetry.io/otel/trace` (and indirect `go.opentelemetry.io/otel`) leave `go.mod`.
- Land the operator's one-line migration in the same wave.

**Non-Goals:**

- Any change to the cancellation half of 0009 D9 (`context.Context` parameters stay exactly as they are).
- Designing the execution half's future injection surface. That is 0009's, with its first reader.
- Any behavior change. Nothing read the slots, so no output, error or log line changes.

## Decisions

### D1: Remove all three slots, including the consumer-called `WithLogger`

**Context**: `WithLogger` has one caller (`opm-operator/cmd/main.go`), but the injected logger is never read, so the call has no observable effect. Clock and tracer have no callers at all.

**Explored**: Removing only clock and tracer and keeping `WithLogger` because it is "utilized". Rejected by the owner (2026-09-01): utilization without a reader is the exact write-only shape this change removes, and keeping one slot of three preserves the misleading suggestion that kernel-internal logging exists.

**Decision**: Remove `WithLogger`, `WithTracer`, `WithClock`, the `Clock` interface, `systemClock`, and the three fields. The operator deletes its `WithLogger` argument and the then-unused logr/slog bridge imports.

**Rationale**: One rule with no exception: the kernel exposes no injection slot without a kernel-side reader. The operator's log output is unchanged because the kernel never wrote to the logger.

### D2: Revise 0009 D9 in place first; this change implements the revised decision

**Context**: D9 (Kind: scope) reserved the slots for the execution half and listed removal as a rejected alternative. Enhancements rules allow in-place revision while an entry is `draft`, folding the superseded position into Alternatives considered.

**Explored**: Landing the removal and letting 0009 catch up later. Rejected: the library would be contradicting a live design decision for however long the gap lasted, and `enhancement.yaml` could not truthfully declare what the change implements.

**Decision**: The D9 revision lands in `enhancements/0009` before implementation starts (tasks 1.1). The revised D9 states: slots removed now; the execution half introduces the injection surface it needs together with its first reader; the cancellation half is unchanged. This change declares `implements: 0009/D9`.

**Rationale**: The reversal is a design decision and belongs in the design record, not in a library changelog. Ordering it first keeps every artifact truthful at every point.

### D3: No deprecation period, no replacement surface

**Context**: Pre-GA alpha line; the library's stated policy is same-PR-wave consumer migration with no shims.

**Explored**: Keeping the option functions as no-ops for one release. Rejected: a no-op `WithLogger` is strictly worse than a compile error; the caller believes logging is configured.

**Decision**: Plain removal. Compile error is the migration signal.

**Rationale**: The one consumer is in this workspace and migrates in the same wave; an external consumer does not exist yet.

## Risks / Trade-offs

- [0009 later re-adds similar options and the churn looks wasteful] → Accepted deliberately in the revised D9: the future surface is shaped by its first reader (planner/runner spans, step logs, deterministic time), which the current slots were guesses at. The otel dependency returning with a real tracer consumer is the correct time to pay for it.
- [A downstream embedder outside the workspace passes an option today] → None exists (the library is pre-1.0 and workspace-consumed); the compile error names the removed identifier.

## Migration Plan

1. `enhancements/0009`: revise D9 in place, append the history event.
2. `library`: implement tasks 2 through 4; `task check`.
3. `opm-operator` (same PR wave): drop the `WithLogger` argument and unused imports; `task dev:fmt dev:vet dev:test`.

Rollback is a revert of the library commit plus the operator line; no data or schema surface is involved.
