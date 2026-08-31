## MODIFIED Requirements

### Requirement: Compile Rename

The compile pipeline's terminal verb SHALL be `Compile`. The canonical entry point is `(*Kernel).Compile`. The free function `compile.CompileModuleInstance` SHALL NOT exist. The earlier `opm/render/process_module.go` / `render.ProcessModuleInstance` names SHALL NOT reappear inside `opm/compile/`. The result type is `compile.CompileResult` (aliased as `kernel.CompileResult`); no other alias of it exists.

Note: `(*Kernel).ProcessModuleInstance` names a different operation, module-instance validation, value-filling, and metadata decoding, distinct from the compile pipeline. The two names occupy different concepts and do not conflict.

#### Scenario: Canonical compile entry

- **WHEN** a caller invokes `k.Compile(ctx, in)` on a `*Kernel`
- **THEN** the call performs the full compile pipeline against `in.Platform` and returns a `*CompileResult`

#### Scenario: compile.CompileModuleInstance symbol gone

- **WHEN** a developer searches for `CompileModuleInstance` in `opm/compile/`
- **THEN** the symbol does not exist
- **AND** callers MUST use `(*Kernel).Compile`

#### Scenario: ModuleResult aliased

- **WHEN** a developer searches `opm/compile` for `ModuleResult`
- **THEN** the identifier does not exist; the result type is referenced as `CompileResult`

### Requirement: Canonical Implementations Live on Kernel

The canonical Go implementation of values validation (full, partial, and detailed) and module-instance processing SHALL live on the `*Kernel` receiver in `opm/kernel/`. No standalone `validate.Config` / `validate.ConfigPartial` / `module.ParseModuleInstance` free functions SHALL remain in the library; the `opm/validate/` and `opm/helper/values/` packages SHALL NOT exist.

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

#### Scenario: ProcessModuleInstance is a kernel method

- **WHEN** a caller invokes `k.ProcessModuleInstance(ctx, spec, mod, values)`
- **THEN** the method validates `values` via the kernel's own `ValidateConfig`, fills the validated value into `spec`, asserts concreteness via `spec.Validate(cue.Concrete(true))` (CUE stdlib), decodes instance metadata, and returns a `*module.Instance`
- **AND** the method does not delegate to any deprecated free function

#### Scenario: opm/validate package is gone

- **WHEN** a developer runs `ls opm/validate/`
- **THEN** the directory does not exist

#### Scenario: opm/helper/values package is gone

- **WHEN** a developer runs `ls opm/helper/values/`
- **THEN** the directory does not exist

#### Scenario: module.ParseModuleInstance free function is gone

- **WHEN** a developer searches `opm/` for `ParseModuleInstance`
- **THEN** no free function and no method with that name exists; `(*Kernel).ProcessModuleInstance` is the only spelling

#### Scenario: compile.CompileModuleInstance free function is gone

- **WHEN** a developer searches `opm/compile/` for `CompileModuleInstance`
- **THEN** no free function with that name exists
- **AND** the canonical compile entry point is `(*Kernel).Compile`

## REMOVED Requirements

### Requirement: LoadModuleFromRegistry Method on Kernel
**Reason**: `Kernel.LoadModuleFromRegistry` returned a source-free `cue.Value` that `synth.Instance` can no longer consume (it requires staged source since `synth-instance-in-module-root`). Both consumers moved to `Kernel.AcquireModuleFromRegistry`; the value-only wrapper had no caller.
**Migration**: Call `k.AcquireModuleFromRegistry(ctx, modPath, version)` and read `mod.Package` when only the value is wanted.

### Requirement: ParseModuleInstance Deprecated Alias
**Reason**: The alias was specified for one cycle and never implemented; no `ParseModuleInstance` symbol has existed in `opm/kernel` since the rename landed. The requirement described surface that did not exist.
**Migration**: None; callers already use `(*Kernel).ProcessModuleInstance`.
