## ADDED Requirements

### Requirement: Canonical Implementations Live on Kernel

The canonical Go implementation of values validation (full and partial) and module-release processing SHALL live on the `*Kernel` receiver in `opm/kernel/`. No standalone `validate.Config` / `validate.ConfigPartial` / `module.ParseModuleRelease` free functions SHALL remain in the library; the `opm/validate/` package SHALL NOT exist after this change.

#### Scenario: ValidateConfig is a kernel method

- **WHEN** a caller invokes `k.ValidateConfig(schema, values, contextLabel, name)`
- **THEN** the method runs the full Tier-2 schema validation directly (no delegation to a `opm/validate` free function) and returns the validated `cue.Value` on success or a `*oerrors.ConfigError` on failure
- **AND** no `opm/validate/` import is required by callers

#### Scenario: ValidateConfigPartial is a kernel method

- **WHEN** a caller invokes `k.ValidateConfigPartial(schema, values, contextLabel, name)`
- **THEN** the method runs the partial-validation entry point (catches type errors, disallowed fields, and pattern violations on fields that ARE set; does not flag missing fields) and returns the value on success or a `*oerrors.ConfigError` on failure

#### Scenario: ProcessModuleRelease is a kernel method

- **WHEN** a caller invokes `k.ProcessModuleRelease(ctx, spec, mod, values)`
- **THEN** the method validates `values` via the kernel's own validation impl, fills the validated value into `spec`, asserts concreteness, decodes release metadata via the binding, and returns a `*module.Release`
- **AND** the method does not delegate to any deprecated free function

#### Scenario: opm/validate package is gone

- **WHEN** a developer runs `ls opm/validate/` after this change ships
- **THEN** the directory does not exist

#### Scenario: module.ParseModuleRelease free function is gone

- **WHEN** a developer searches `opm/module/` for `ParseModuleRelease`
- **THEN** no free function with that name exists
- **AND** the only `ParseModuleRelease` symbol in the library is the deprecated method on `*Kernel` (see the deprecation requirement below)

#### Scenario: compile.CompileModuleRelease free function is gone

- **WHEN** a developer searches `opm/compile/` for `CompileModuleRelease`
- **THEN** no free function with that name exists
- **AND** the canonical compile entry point is `(*Kernel).Compile`

### Requirement: ParseModuleRelease Deprecated Alias

`*Kernel` SHALL expose a deprecated `ParseModuleRelease` method that delegates to `ProcessModuleRelease` for one cycle to soften the rename for downstream callers.

#### Scenario: Alias delegates to canonical method

- **WHEN** a caller invokes `k.ParseModuleRelease(ctx, spec, mod, values)`
- **THEN** the result is identical to invoking `k.ProcessModuleRelease(ctx, spec, mod, values)`

#### Scenario: Alias carries deprecation marker

- **WHEN** a developer reads the godoc for `(*Kernel).ParseModuleRelease`
- **THEN** the comment begins with `// Deprecated:` and points to `(*Kernel).ProcessModuleRelease`

## MODIFIED Requirements

### Requirement: Backward-Compatible Method Wrappers

For every existing exported function in `opm/helper/loader/file/`, `opm/helper/platform/`, `opm/helper/values/`, and the `*FromValue` constructors in `opm/module/` and `opm/platform/` that takes a `*cue.Context` (directly or via a `CueContextOwner` / `KernelOwner` interface), the Kernel SHALL provide a method wrapper that sources `*cue.Context` from itself. The Kernel SHALL NOT wrap functions whose canonical implementation now lives on the Kernel itself (validation and module-release processing); those are direct kernel methods, not wrappers.

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
- **THEN** the file contains the canonical implementation of `ValidateConfig` and `ValidateConfigPartial` directly, with no `//nolint:staticcheck // SA1019: ... wraps the deprecated free function` exemption

### Requirement: Single Pre-Unified Values Input

The kernel SHALL accept a single, pre-unified `cue.Value` for the values argument on every public method that takes user values. The kernel SHALL NOT accept `[]cue.Value` as a values argument on any public method.

#### Scenario: ValidateConfig takes a single value

- **WHEN** a caller invokes `k.ValidateConfig(schema, values, contextLabel, name)` with `values` as a `cue.Value`
- **THEN** the method validates the supplied `values` against `schema` and returns the validated value or a `*oerrors.ConfigError`
- **AND** there is no internal merge loop; the method consumes `values` as-is

#### Scenario: ProcessModuleRelease takes a single value

- **WHEN** a caller invokes `k.ProcessModuleRelease(ctx, spec, mod, values)` with `values` as a single `cue.Value`
- **THEN** the method validates `values` via the kernel's own `ValidateConfig` implementation, fills the validated value into `spec`, and returns a `*module.Release`
- **AND** the method does not accept a slice form

#### Scenario: Empty values is the zero value

- **WHEN** a caller passes a zero-value `cue.Value{}` to `k.ValidateConfig` or `k.ProcessModuleRelease`
- **THEN** the call succeeds (no validation errors, no fill operation)
- **AND** the behavior matches the previous slice's "no values supplied" path

### Requirement: Tier-2 Validation Always Runs

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema regardless of whether a Tier-1 helper validated them upstream.

#### Scenario: Kernel re-validates after Tier-1

- **WHEN** a frontend that uses `opm/helper/values` (slice 05) supplies a unified value to `k.ValidateConfig`
- **THEN** the kernel performs full schema validation on the unified value
- **AND** any schema violation produces a `*oerrors.ConfigError`

#### Scenario: Kernel validates without Tier-1

- **WHEN** a frontend skips Tier-1 helper validation and feeds raw unified values directly
- **THEN** the kernel still produces correct schema-validation errors via `*oerrors.ConfigError`
- **AND** the only loss is per-source attribution in error messages

### Requirement: Compile Rename

The compile pipeline's terminal verb SHALL be `Compile`. The canonical entry point is `(*Kernel).Compile`. The free function `compile.CompileModuleRelease` SHALL NOT exist after this change. The earlier `opm/render/process_module.go` / `render.ProcessModuleRelease` names SHALL NOT reappear inside `opm/compile/`.

Note: `(*Kernel).ProcessModuleRelease` (added by this change) names a different operation — module-release validation, value-filling, and metadata decoding — distinct from the compile pipeline. The two names occupy different concepts and do not conflict.

#### Scenario: Canonical compile entry

- **WHEN** a caller invokes `k.Compile(ctx, in)` on a `*Kernel`
- **THEN** the call performs the full compile pipeline against `in.Platform` and returns a `*CompileResult`

#### Scenario: compile.CompileModuleRelease symbol gone

- **WHEN** a developer searches for `CompileModuleRelease` in `opm/compile/`
- **THEN** the symbol does not exist
- **AND** callers MUST use `(*Kernel).Compile`

#### Scenario: ModuleResult aliased

- **WHEN** a caller references `*render.ModuleResult`
- **THEN** the type resolves to `*render.CompileResult` via a Go type alias

## REMOVED Requirements

### Requirement: Temporary Migration Helper

**Reason**: Already removed in slice 05 (`introduce-tiered-validation`); the requirement only existed because `validate.UnifyAndValidate` was a transitional helper inside `opm/validate/`. With `opm/validate/` deleted in this change, the requirement is moot.

**Migration**: Use `opm/helper/values.ValidateAndUnify` (or the `(*Kernel).ValidateAndUnify` ergonomic shortcut) for layered values handling.
