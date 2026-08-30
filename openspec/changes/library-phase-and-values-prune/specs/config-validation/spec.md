## REMOVED Requirements

### Requirement: Phase Method Wraps With Module Name
**Reason**: `Kernel.Validate(ctx, ValidateInput)` had no caller in `cli` or `opm-operator`; its one library caller was `Kernel.Compile`, which validated a `Values` field it never applied. The phase verb and its input struct are removed; the instance-name wrapping performed by `ProcessModuleInstance`, which this requirement also described, moves to its own requirement below.
**Migration**: Validate values where they are applied: call `k.ProcessModuleInstance(ctx, spec, mod, values)`, whose error is wrapped with `instance "<name>": `. A caller that only wants the check without the fill calls `k.ValidateConfig(m.ConfigSchema(), values)` and frames the error itself.

## ADDED Requirements

### Requirement: ProcessModuleInstance Wraps With Instance Name

`Kernel.ProcessModuleInstance(ctx, spec, mod, values)` SHALL route its values validation through `Kernel.ValidateConfig` (no per-call options) and SHALL wrap any returned validation error with `fmt.Errorf("instance %q: %w", instanceName, err)`, where `instanceName` is the spec's `metadata.name` when concrete, else the module name, else `"<unknown>"`. The subsequent concreteness assertion on the filled spec is CUE's own `Validate(cue.Concrete(true))`, unchanged.

#### Scenario: Validation error is framed with the instance name

- **WHEN** `k.ProcessModuleInstance` is called with values that violate the module's `#config`
- **THEN** the returned error's text begins `instance "<name>": ` followed by the CUE error message
- **AND** the underlying CUE diagnostics are reachable via `errors.As` and `cueerrors.Errors`

#### Scenario: No values supplied

- **WHEN** `k.ProcessModuleInstance` is called with the zero `cue.Value` for values
- **THEN** no validation runs and no fill is performed; the spec must already be concrete on every required field

## MODIFIED Requirements

### Requirement: Module and Instance Typed Convenience Methods

`*Module` and `*Instance` SHALL each expose a `ConfigSchema()` accessor returning the `#config` schema reachable on the artifact's `Package` (for an instance, through its embedded `#module`), or the zero `cue.Value` when absent. The Kernel SHALL NOT expose per-artifact wrappers over the validation primitives: a caller composes `ConfigSchema()` with `ValidateConfig`, `ValidateConfigPartial` or `ValidateConfigDetailed` directly.

#### Scenario: Module.ConfigSchema accessor

- **WHEN** a caller invokes `m.ConfigSchema()`
- **THEN** the result is the `cue.Value` at `schema.Config` inside `m.Package`
- **AND** the accessor returns a zero value if the module has no `#config` field

#### Scenario: Instance.ConfigSchema accessor

- **WHEN** a caller invokes `r.ConfigSchema()`
- **THEN** the result is the `cue.Value` at `schema.Config` inside the instance's embedded module at `schema.Module`
- **AND** the accessor returns a zero value if the instance has no embedded module or the module has no `#config`

#### Scenario: Kernel.ValidateModuleValues delegates without name wrapping

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** neither `ValidateModuleValues` nor `ValidateInstanceValues` exists
- **AND** `k.ValidateConfig(m.ConfigSchema(), values)` is the spelling for a concrete check against a module, with no name wrapping

#### Scenario: Kernel.ValidateModuleValuesPartial delegates

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** neither `ValidateModuleValuesPartial` nor `ValidateInstanceValuesPartial` exists
- **AND** `k.ValidateConfigPartial(m.ConfigSchema(), values)` is the spelling for a partial check against a module

#### Scenario: Kernel.ValidateModuleValuesDetailed delegates

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** neither `ValidateModuleValuesDetailed` nor `ValidateInstanceValuesDetailed` exists
- **AND** `k.ValidateConfigDetailed(m.ConfigSchema(), sources, opts...)` is the spelling for layered validation against a module

#### Scenario: Instance equivalents

- **WHEN** a caller holds a `*module.Instance` rather than a `*module.Module`
- **THEN** it composes `r.ConfigSchema()` with the same three primitives; no instance-typed wrapper exists on the Kernel
