## ADDED Requirements

### Requirement: Phase-Explicit Methods on Kernel

The `Kernel` SHALL expose four phase-explicit methods, each accepting a phase-specific input struct and returning a phase-appropriate result.

#### Scenario: Validate phase method

- **WHEN** a caller invokes `k.Validate(ctx, ValidateInput{Module, ModuleRelease, Values})`
- **THEN** the kernel performs Tier-2 schema validation of `Values` against `Module.Package`'s `#config`
- **AND** returns nil on success or a `*oerrors.ConfigError` on failure
- **AND** does not perform matching, execution, or finalization

#### Scenario: Match phase method

- **WHEN** a caller invokes `k.Match(ctx, MatchInput{Module, ModuleRelease, Provider})`
- **THEN** the kernel produces a `*MatchPlan` describing matched and non-matched component/transformer pairs
- **AND** does not execute any transformer

#### Scenario: Plan phase method

- **WHEN** a caller invokes `k.Plan(ctx, PlanInput{Module, ModuleRelease, Values, Provider, RuntimeName})`
- **THEN** the kernel runs Validate + Match, executes transformers in dry-run mode (no finalization commits), and returns a `*PlanResult` containing component summaries, unmatched FQNs, ambiguous FQNs, and warnings
- **AND** does not return rendered values

#### Scenario: Compile phase method

- **WHEN** a caller invokes `k.Compile(ctx, CompileInput{Module, ModuleRelease, Values, Provider, RuntimeName})`
- **THEN** the kernel runs the full pipeline (Validate + Match + Execute + Finalize) and returns a `*CompileResult` containing `Rendered []*core.Rendered`, component summaries, unmatched FQNs, ambiguous FQNs, and warnings

### Requirement: Phase Input Structs

Each phase method SHALL accept a phase-specific input struct rather than positional arguments. Input structs SHALL be additive — new fields SHALL be addable without breaking existing call sites.

#### Scenario: ValidateInput shape

- **WHEN** a developer reads the `ValidateInput` struct
- **THEN** the struct has at minimum `Module *module.Module`, `ModuleRelease *module.Release`, and `Values cue.Value`
- **AND** each field has godoc explaining its role

#### Scenario: CompileInput shape

- **WHEN** a developer reads the `CompileInput` struct
- **THEN** the struct has at minimum `Module *module.Module`, `ModuleRelease *module.Release`, `Values cue.Value`, `Provider *provider.Provider` (substituted by `Platform *Platform` after slice 08 lands), and `RuntimeName string`

### Requirement: Compile Rename

The render pipeline's terminal verb SHALL be `Compile`. `pkg/render/process_module.go` SHALL be renamed to `pkg/render/compile_module.go`. `render.ProcessModuleRelease` SHALL be renamed to `render.CompileModuleRelease`. The old name SHALL remain as a `// Deprecated:` alias delegating to the new name.

#### Scenario: New name available

- **WHEN** a caller invokes `render.CompileModuleRelease(ctx, rel, p, runtimeName)`
- **THEN** the call performs the full render pipeline and returns a `*CompileResult`

#### Scenario: Deprecated alias still works

- **WHEN** a caller invokes `render.ProcessModuleRelease(ctx, rel, p, runtimeName)`
- **THEN** the call succeeds with the same behavior as `CompileModuleRelease`
- **AND** the function carries a `// Deprecated:` doc comment pointing to `CompileModuleRelease`

#### Scenario: ModuleResult aliased

- **WHEN** a caller references `*render.ModuleResult`
- **THEN** the type resolves to `*render.CompileResult` via a Go type alias

### Requirement: Utility Methods on Kernel

The Kernel SHALL expose `DetectAPIVersion(v cue.Value) (apiversion.Version, error)` and `Finalize(v cue.Value) (cue.Value, error)` as methods.

#### Scenario: DetectAPIVersion delegates to apiversion package

- **WHEN** a caller invokes `k.DetectAPIVersion(v)`
- **THEN** the result is identical to calling `apiversion.Detect(v)` directly
- **AND** the method exists for discovery purposes (callers find the operation through the Kernel anchor)

#### Scenario: Finalize uses kernel cue.Context

- **WHEN** a caller invokes `k.Finalize(v)`
- **THEN** the function performs schema-constraint stripping (existing `render.FinalizeValue` behavior) using the Kernel's `cue.Context`
