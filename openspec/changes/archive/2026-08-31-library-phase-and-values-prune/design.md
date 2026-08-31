## Context

See proposal.md, Why. The phase verbs date from `add-phase-methods-and-rename-compile` (2026-05-08), which mapped them onto frontend subcommands and an operator admission webhook that were designed on paper. Measured 2026-08-30 against `cli` and `opm-operator` `main`: `Compile` has five call sites, `Match` none (its `MatchPlan` is read off `CompileResult`), `Plan` none, `Validate` none, the six typed wrappers none, and no caller sets `CompileInput.Values`. `ProcessModuleInstance` is the one place values are validated and applied.

Constraints:

- `library-dead-symbol-sweep` lands first and touches no phase surface; this change touches nothing that one removes.
- Enhancement 0019 D7/D18 adds a skew-policy field to `CompileInput`, and D10 moves matching into the build. `CompileInput` therefore stays a struct, and `Match` stays until D10 replaces the Go matcher.
- The `cli` sibling `cli-layered-values` adopts `Source`, `LoadSourceFromFile` and `ValidateConfigDetailed`; those stay, and so do `Partial()` and `ValidateConfigPartial` (the `vet` primitive).

## Goals / Non-Goals

**Goals:**

- `Kernel` exposes two phase verbs, `Match` and `Compile`, and one processing verb, `ProcessModuleInstance`, with validation happening in the last.
- `Compile` renders exactly the instance it is given; no hidden re-validation, no input it validates but ignores.
- The validation surface is the three primitives plus `Source` and its loaders; the six typed wrappers are gone.

**Non-Goals:**

- Making `Match` cheaper or moving it into CUE (0019 D10).
- Changing `MatchPlan`, `CompileResult` or `ComponentSummary` shapes; the CLI prints them.
- Removing `Match`. It is the only phase-only diagnostic verb, and 0019 D10 owns its retirement.

## Decisions

### D1: Delete `Plan` rather than make it stop after Match

**Context**: `Plan` runs the full pipeline and drops `Compiled`. The originating design chose that over a stop-after-match shortcut because "no current frontend needs the speedup" and a green Plan should imply a green Compile.

**Explored**: Keeping `Plan` as `Match` plus summaries (cheap). Rejected: it would be a third verb over the same two inputs whose only difference from `Match` is a summary slice `CompileResult` already carries, and it would still have no caller.

**Decision**: Delete `Plan`, `PlanInput`, `PlanResult`. A caller wanting a dry run calls `Match` for the pairing verdict or `Compile` and discards `Compiled`; both are documented on `Kernel`.

**Rationale**: Principle VII. A verb with no consumer that costs the same as the verb it wraps is surface without a job.

### D2: Remove `Kernel.Validate` and `CompileInput.Values` together

**Context**: `Compile` calls `k.Validate(ValidateInput{Module: moduleFromInstance, Values: in.Values})` and then renders `in.ModuleInstance.Package` unchanged. The validated value is never filled. `ProcessModuleInstance` already validates and fills.

**Explored**: Keeping `CompileInput.Values` and making `Compile` fill it (turning the assertion into a real knob). Rejected: it duplicates `ProcessModuleInstance`, and every consumer processes before compiling; a second fill path is a second place for a values bug.

**Decision**: Remove both. `Compile` becomes Match then Execute; the Tier-2 requirement is restated on `ProcessModuleInstance`, where it has always actually run.

**Rationale**: One place validates, one place fills, and the input struct only carries fields the pipeline reads.

### D3: Keep the three primitives and `Source`; drop the six typed wrappers

**Context**: The typed wrappers are one-line delegations. `Module.ConfigSchema()` and `Instance.ConfigSchema()` are the only logic they carry.

**Decision**: Keep the accessors, delete `validate_typed.go`. Keep `ValidateConfig`, `ValidateConfigPartial`, `ValidateConfigDetailed`, `Source`, `ValidateOption`, `Partial()`, `LoadSourceFromFile`, `LoadSourceFromBytes`, `LoadSourceFromString`.

**Rationale**: The CLI sibling is the first real consumer of layered validation and needs the primitive plus the loaders, not the wrappers. Six ways to spell one line is cost without a reader.

### D4: Retire the validate-flow design note instead of rewriting it

**Context**: `docs/design/kernel-validate-flow.md` documents `Kernel.Validate` and a `Binding` / `api.Lookup` layer that no longer exists.

**Decision**: Delete it. The surviving surface is described by the `config-validation` spec and the `Kernel` package doc.

## Risks / Trade-offs

- [A consumer branch calls `Plan` or `Validate`] → Both consumers' `main` were grepped 2026-08-30; `MIGRATIONS.md` gives the one-line replacement for each.
- [Removing `Compile`'s re-validation drops a safety net for hand-built instances] → A hand-built `*module.Instance` (tests, `NewInstanceFromValue`) was never validated by that path either, because `in.Values` was always zero. `ProcessModuleInstance` remains the validated entry; `NewInstanceFromValue` stays documented as the raw one.
- [`Match` looks like the next candidate on the same logic] → It stays by explicit decision (proposal, Non-Goals); 0019 D10 owns it.
- [The CLI sibling names surface this change keeps] → The two proposals were written together; the kept set is listed in both.

## Migration Plan

1. Land after `library-dead-symbol-sweep`.
2. `MIGRATIONS.md` under `## Unreleased — Breaking`: `Plan` → `Match` or `Compile`; `Validate` / `CompileInput.Values` → `ProcessModuleInstance`; `ValidateModuleValues*` / `ValidateInstanceValues*` → `k.ValidateConfig*(m.ConfigSchema(), ...)`.
3. Rollback is a revert.
