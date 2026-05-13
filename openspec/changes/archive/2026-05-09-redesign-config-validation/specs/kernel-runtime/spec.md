## MODIFIED Requirements

### Requirement: Single Pre-Unified Values Input

The kernel SHALL accept a single, pre-unified `cue.Value` for the values argument on every public method that takes user values. The kernel SHALL NOT accept `[]cue.Value` as a values argument on any public method, with the sole exception of `ValidateConfigDetailed` which accepts `[]Source` for layered input.

#### Scenario: ValidateConfig takes a single value

- **WHEN** a caller invokes `k.ValidateConfig(schema, values)` with `values` as a `cue.Value`
- **THEN** the method validates the supplied `values` against `schema` and returns the validated `cue.Value` and a CUE-native `error`
- **AND** there is no internal merge loop; the method consumes `values` as-is

#### Scenario: ProcessModuleRelease takes a single value

- **WHEN** a caller invokes `k.ProcessModuleRelease(ctx, spec, mod, values)` with `values` as a single `cue.Value`
- **THEN** the method validates `values` via the kernel's own `ValidateConfig` implementation, fills the validated value into `spec`, and returns a `*module.Release`
- **AND** the method does not accept a slice form

#### Scenario: ValidateConfigDetailed takes a slice of Source

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources, opts...)` with `sources` as `[]Source`
- **THEN** the method unifies the sources in order then validates the merged value against `schema`
- **AND** this is the only public method that accepts a multi-value input

#### Scenario: Empty values is the zero value

- **WHEN** a caller passes a zero-value `cue.Value{}` to `k.ValidateConfig` / `k.ValidateConfigPartial` / `k.ProcessModuleRelease`, or an empty `[]Source` to `k.ValidateConfigDetailed`
- **THEN** the call succeeds (no validation errors, no fill operation)
- **AND** the behavior is documented as "no values supplied"

### Requirement: Tier-2 Validation Always Runs

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema regardless of whether a Tier-1 helper validated them upstream. The Tier-2 entry point is `Kernel.ValidateConfig`.

#### Scenario: Kernel re-validates after Detailed

- **WHEN** a frontend that uses `k.ValidateConfigDetailed` supplies the resulting unified value to `k.ValidateConfig`
- **THEN** the kernel performs full schema validation on the unified value
- **AND** any schema violation produces a CUE-native error walkable via `cueerrors.Errors`

#### Scenario: Kernel validates without Detailed

- **WHEN** a frontend skips `ValidateConfigDetailed` and feeds raw unified values directly
- **THEN** the kernel still produces correct schema-validation errors
- **AND** the only loss is per-source attribution in error positions (`pos.Filename()` is empty unless the caller compiled with `cue.Filename(...)` themselves)

### Requirement: Phase-Explicit Methods on Kernel

The `Kernel` SHALL expose four phase-explicit methods, each accepting a phase-specific input struct and returning a phase-appropriate result.

#### Scenario: Validate phase method

- **WHEN** a caller invokes `k.Validate(ctx, ValidateInput{Module, ModuleRelease, Values})`
- **THEN** the kernel performs Tier-2 schema validation of `Values` against `Module.Package`'s `#config` by calling `k.ValidateConfig` internally
- **AND** returns nil on success or a CUE-native error wrapped with `fmt.Errorf("module %q: %w", name, err)` on failure
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
- **THEN** the kernel runs the full pipeline (Validate + Match + Execute + Finalize) and returns a `*CompileResult` containing `Compiled []*core.Compiled`, component summaries, unmatched FQNs, ambiguous FQNs, and warnings

### Requirement: Canonical Implementations Live on Kernel

The canonical Go implementation of values validation (full, partial, and detailed) and module-release processing SHALL live on the `*Kernel` receiver in `opm/kernel/`. No standalone `validate.Config` / `validate.ConfigPartial` / `module.ParseModuleRelease` free functions SHALL remain in the library; the `opm/validate/` and `opm/helper/values/` packages SHALL NOT exist after this change.

#### Scenario: ValidateConfig is a kernel method

- **WHEN** a caller invokes `k.ValidateConfig(schema, values)`
- **THEN** the method runs the full Tier-2 schema validation directly and returns the validated `cue.Value` on success or a CUE-native error on failure
- **AND** no `opm/validate/` import is required by callers

#### Scenario: ValidateConfigPartial is a kernel method

- **WHEN** a caller invokes `k.ValidateConfigPartial(schema, values)`
- **THEN** the method runs the partial-validation entry point (catches type errors, disallowed fields, and pattern violations on fields that ARE set; does not flag missing fields) and returns the value on success or a CUE-native error on failure

#### Scenario: ValidateConfigDetailed is a kernel method

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources, opts...)`
- **THEN** the method unifies the sources in order, then validates the merged value (full or partial depending on `Partial()` option) and returns the merged `cue.Value` plus a CUE-native error
- **AND** no `opm/helper/values/` import is required by callers

#### Scenario: ProcessModuleRelease is a kernel method

- **WHEN** a caller invokes `k.ProcessModuleRelease(ctx, spec, mod, values)`
- **THEN** the method validates `values` via the kernel's own `ValidateConfig`, fills the validated value into `spec`, asserts concreteness via `spec.Validate(cue.Concrete(true))` (CUE stdlib), decodes release metadata via the binding, and returns a `*module.Release`
- **AND** the method does not delegate to any deprecated free function

#### Scenario: opm/validate package is gone

- **WHEN** a developer runs `ls opm/validate/` after this change ships
- **THEN** the directory does not exist

#### Scenario: opm/helper/values package is gone

- **WHEN** a developer runs `ls opm/helper/values/` after this change ships
- **THEN** the directory does not exist

#### Scenario: module.ParseModuleRelease free function is gone

- **WHEN** a developer searches `opm/module/` for `ParseModuleRelease`
- **THEN** no free function with that name exists
- **AND** the only `ParseModuleRelease` symbol in the library is the deprecated method on `*Kernel` (see the deprecation requirement below)

#### Scenario: compile.CompileModuleRelease free function is gone

- **WHEN** a developer searches `opm/compile/` for `CompileModuleRelease`
- **THEN** no free function with that name exists
- **AND** the canonical compile entry point is `(*Kernel).Compile`

### Requirement: Backward-Compatible Method Wrappers

For every existing exported function in `opm/helper/loader/file/` and `opm/helper/platform/`, and the `*FromValue` constructors in `opm/module/` and `opm/platform/` that takes a `*cue.Context` (directly or via a `CueContextOwner` interface), the Kernel SHALL provide a method wrapper that sources `*cue.Context` from itself. The Kernel SHALL NOT wrap functions whose canonical implementation now lives on the Kernel itself (validation, layered values, and module-release processing); those are direct kernel methods, not wrappers. The Kernel SHALL NOT expose a `ValidateAndUnify` wrapper — the canonical replacement is `Kernel.ValidateConfigDetailed`.

#### Scenario: Loader method wrapper

- **WHEN** a caller invokes `k.LoadModulePackage(ctx, "./module")`
- **THEN** the result is identical to calling `helper/loader/file.LoadModulePackage(k.CueContext(), "./module")`
- **AND** any error returned is the same instance the underlying free function would return

#### Scenario: Helper-shaped functions remain callable

- **WHEN** existing downstream code calls `helper/loader/file.LoadModulePackage(cueCtx, dir)` directly
- **THEN** the call succeeds with the same behavior as before
- **AND** the helper signature continues to accept `*cue.Context` so non-kernel consumers can use it without importing `opm/kernel`

#### Scenario: Validation methods are not wrappers

- **WHEN** a developer reads `opm/kernel/validate.go`
- **THEN** the file contains the canonical implementation of `ValidateConfig`, `ValidateConfigPartial`, and `ValidateConfigDetailed` directly, with no `//nolint:staticcheck // SA1019:` exemptions for delegating to deleted helper packages

#### Scenario: ValidateAndUnify wrapper is gone

- **WHEN** a developer searches `opm/kernel/wrappers.go` (or the entire `opm/kernel/`) for `ValidateAndUnify`
- **THEN** no exported method or function with that name exists
- **AND** callers MUST use `k.ValidateConfigDetailed`
