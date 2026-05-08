## MODIFIED Requirements

### Requirement: Phase-Explicit Methods on Kernel

The `Kernel` SHALL expose four phase-explicit methods, each accepting a phase-specific input struct and returning a phase-appropriate result.

#### Scenario: Validate phase method

- **WHEN** a caller invokes `k.Validate(ctx, ValidateInput{Module, ModuleRelease, Values})`
- **THEN** the kernel performs Tier-2 schema validation of `Values` against `Module.Package`'s `#config`
- **AND** returns nil on success or a `*oerrors.ConfigError` on failure
- **AND** does not perform matching, execution, or finalization

#### Scenario: Match phase method

- **WHEN** a caller invokes `k.Match(ctx, MatchInput{Module, ModuleRelease, Platform})`
- **THEN** the kernel produces a `*MatchPlan` describing matched and non-matched component/transformer pairs
- **AND** does not execute any transformer

#### Scenario: Plan phase method

- **WHEN** a caller invokes `k.Plan(ctx, PlanInput{Module, ModuleRelease, Values, Platform, RuntimeName})`
- **THEN** the kernel runs the full Compile pipeline (Validate + Match + Execute + Finalize) and returns a `*PlanResult` containing component summaries, unmatched FQNs, ambiguous FQNs, and warnings
- **AND** does not return rendered values

#### Scenario: Compile phase method

- **WHEN** a caller invokes `k.Compile(ctx, CompileInput{Module, ModuleRelease, Values, Platform, RuntimeName})`
- **THEN** the kernel runs the full pipeline (Validate + Match + Execute + Finalize) and returns a `*CompileResult` containing `Rendered []*core.Rendered`, component summaries, unmatched FQNs, ambiguous FQNs, and warnings

### Requirement: Phase Input Structs

Each phase method SHALL accept a phase-specific input struct rather than positional arguments. Input structs SHALL be additive — new fields SHALL be addable without breaking existing call sites.

#### Scenario: ValidateInput shape

- **WHEN** a developer reads the `ValidateInput` struct
- **THEN** the struct has at minimum `Module *module.Module`, `ModuleRelease *module.Release`, and `Values cue.Value`
- **AND** each field has godoc explaining its role

#### Scenario: CompileInput shape

- **WHEN** a developer reads the `CompileInput` struct
- **THEN** the struct has at minimum `Module *module.Module`, `ModuleRelease *module.Release`, `Values cue.Value`, `Platform *platform.Platform`, and `RuntimeName string`
- **AND** the struct has no `Provider` field

### Requirement: Compile Rename

The render pipeline's terminal verb SHALL be `Compile`. `pkg/render/process_module.go` SHALL be renamed to `pkg/render/compile_module.go`. `render.ProcessModuleRelease` SHALL be renamed to `render.CompileModuleRelease`.

#### Scenario: New name available

- **WHEN** a caller invokes `compile.CompileModuleRelease(ctx, rel, plat, runtimeName)`
- **THEN** the call performs the full render pipeline against the supplied `*platform.Platform` and returns a `*CompileResult`

#### Scenario: ProcessModuleRelease alias removed

- **WHEN** a developer searches for `ProcessModuleRelease` in `pkg/compile/` or `pkg/kernel/`
- **THEN** the symbol does not exist
- **AND** callers MUST use `CompileModuleRelease` (free function) or `(*Kernel).Compile` (method)
