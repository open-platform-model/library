# kernel-runtime Delta

## MODIFIED Requirements

### Requirement: Phase-Explicit Methods on Kernel

The `Kernel` SHALL expose two phase-explicit methods, each accepting a phase-specific input struct and returning a phase-appropriate result: `Match` (pairing without execution) and `Compile` (the full pipeline). The Kernel SHALL NOT expose a plan-only verb that runs the full pipeline and discards its output, nor a values-validation phase verb; values are validated where they are applied, by `ProcessModuleInstance`.

#### Scenario: Match phase method

- **WHEN** a caller invokes `k.Match(ctx, MatchInput{ModuleInstance, Platform})`
- **THEN** the kernel produces a `*MatchPlan` describing matched and non-matched component/transformer pairs, unresolved demands and unify failures
- **AND** does not execute any transformer

#### Scenario: Compile phase method

- **WHEN** a caller invokes `k.Compile(ctx, CompileInput{ModuleInstance, Platform, RuntimeName})`
- **THEN** the kernel runs Match then Execute against the instance's already-processed value and returns a `*CompileResult` containing `Compiled []*core.Compiled`, the `MatchPlan`, component summaries, and warnings
- **AND** the `CompileResult` carries no top-level unmatched-components field: an unmatched component fails `Compile` through the typed gate, and the plan inside the result still records `MatchPlan.Unmatched`
- **AND** performs no values validation of its own; the instance is rendered as processed

#### Scenario: Validate phase method

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/kernel`
- **THEN** neither `Validate` nor `ValidateInput` exists
- **AND** values are validated by `ProcessModuleInstance` at the point they are filled

#### Scenario: Plan phase method

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/kernel`
- **THEN** none of `Plan`, `PlanInput`, `PlanResult` exists
- **AND** a caller wanting a dry run calls `Match` for the pairing diagnosis or `Compile` and discards `Compiled`
