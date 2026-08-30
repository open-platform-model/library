## MODIFIED Requirements

### Requirement: Phase-Explicit Methods on Kernel

The `Kernel` SHALL expose two phase-explicit methods, each accepting a phase-specific input struct and returning a phase-appropriate result: `Match` (pairing without execution) and `Compile` (the full pipeline). The Kernel SHALL NOT expose a plan-only verb that runs the full pipeline and discards its output, nor a values-validation phase verb; values are validated where they are applied, by `ProcessModuleInstance`.

#### Scenario: Match phase method

- **WHEN** a caller invokes `k.Match(ctx, MatchInput{ModuleInstance, Platform})`
- **THEN** the kernel produces a `*MatchPlan` describing matched and non-matched component/transformer pairs, unresolved demands and unify failures
- **AND** does not execute any transformer

#### Scenario: Compile phase method

- **WHEN** a caller invokes `k.Compile(ctx, CompileInput{ModuleInstance, Platform, RuntimeName})`
- **THEN** the kernel runs Match then Execute against the instance's already-processed value and returns a `*CompileResult` containing `Compiled []*core.Compiled`, the `MatchPlan`, component summaries, unmatched components, and warnings
- **AND** performs no values validation of its own; the instance is rendered as processed

#### Scenario: Validate phase method

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/kernel`
- **THEN** neither `Validate` nor `ValidateInput` exists
- **AND** values are validated by `ProcessModuleInstance` at the point they are filled

#### Scenario: Plan phase method

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/kernel`
- **THEN** none of `Plan`, `PlanInput`, `PlanResult` exists
- **AND** a caller wanting a dry run calls `Match` for the pairing diagnosis or `Compile` and discards `Compiled`

### Requirement: Phase Input Structs

Each phase method SHALL accept a phase-specific input struct rather than positional arguments. Input structs SHALL be additive: new fields SHALL be addable without breaking existing call sites. Phases that operate on a constructed `*module.Instance` SHALL NOT carry a parallel `*module.Module` field; the source module is reachable via the instance's `Package` at `schema.Module`. The matcher-facing input structs (`MatchInput`, `CompileInput`) SHALL carry the platform as a `*materialize.MaterializedPlatform` (the realized form), not a raw `*platform.Platform`; callers MUST `Materialize` before invoking these phases. Neither struct SHALL carry a user-values field: values enter through `ProcessModuleInstance`, and a field the pipeline validates but does not apply SHALL NOT exist.

#### Scenario: ValidateInput shape

- **WHEN** a developer searches `opm/kernel` for `ValidateInput`
- **THEN** no such struct exists; the values-validation phase verb it fed was removed

#### Scenario: MatchInput shape

- **WHEN** a developer reads the `MatchInput` struct
- **THEN** the struct has exactly `ModuleInstance *module.Instance` and `Platform *materialize.MaterializedPlatform` as required artifact fields
- **AND** the struct has no `Module` field

#### Scenario: PlanInput shape

- **WHEN** a developer searches `opm/kernel` for `PlanInput`
- **THEN** no such struct exists; the plan verb it fed was removed

#### Scenario: CompileInput shape

- **WHEN** a developer reads the `CompileInput` struct
- **THEN** the struct has `ModuleInstance *module.Instance`, `Platform *materialize.MaterializedPlatform`, and `RuntimeName string`
- **AND** the struct has no `Module`, `Provider` or `Values` field

#### Scenario: Compile sources its #config schema from the instance

- **WHEN** `kernel.Compile` runs on a `CompileInput`
- **THEN** it performs no `#config` validation; the instance's `#config` schema was applied by `ProcessModuleInstance`, reachable via `in.ModuleInstance.ConfigSchema()`
- **AND** no `*module.Module` is required on the `CompileInput`

#### Scenario: Match does not require module metadata

- **WHEN** `kernel.Match` is invoked with a `MatchInput`
- **THEN** matching consumes `in.ModuleInstance.MatchComponents()`, the instance name for diagnostics, and `in.Platform` (a `*materialize.MaterializedPlatform`) only
- **AND** the operation completes without reading any `*module.Module` field

### Requirement: Tier-2 Validation Always Runs

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema at the point they are filled into the instance, regardless of whether a Tier-1 helper validated them upstream. The Tier-2 entry point is `Kernel.ValidateConfig`, invoked by `Kernel.ProcessModuleInstance`. `Kernel.Compile` SHALL NOT perform a second validation pass: it renders the instance `ProcessModuleInstance` produced, which is already concrete.

#### Scenario: Kernel re-validates after Detailed

- **WHEN** a frontend that uses `k.ValidateConfigDetailed` supplies the resulting unified value to `k.ProcessModuleInstance`
- **THEN** the kernel performs full schema validation on the unified value via `ValidateConfig` before filling it
- **AND** any schema violation produces a CUE-native error walkable via `cueerrors.Errors`, wrapped with the instance name

#### Scenario: Kernel validates without Detailed

- **WHEN** a frontend skips `ValidateConfigDetailed` and feeds raw unified values to `ProcessModuleInstance` directly
- **THEN** the kernel still produces correct schema-validation errors
- **AND** the only loss is per-source attribution in error positions (`pos.Filename()` is empty unless the caller compiled with `cue.Filename(...)` themselves)

#### Scenario: Compile does not re-validate

- **WHEN** a caller invokes `k.Compile` with an instance returned by `ProcessModuleInstance`
- **THEN** no `#config` validation runs inside `Compile`; the pipeline is Match then Execute
