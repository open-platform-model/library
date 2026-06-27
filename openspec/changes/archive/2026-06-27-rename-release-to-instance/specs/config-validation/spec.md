## ADDED Requirements

### Requirement: Module and Instance Typed Convenience Methods

`*Module` SHALL expose a `ConfigSchema()` accessor (`*Instance` already exposes one). The Kernel SHALL expose typed convenience methods `ValidateModuleValues`, `ValidateModuleValuesPartial`, `ValidateModuleValuesDetailed`, `ValidateInstanceValues`, `ValidateInstanceValuesPartial`, `ValidateInstanceValuesDetailed` — each a 1-line schema-lookup wrapper that delegates to the corresponding primitive.

The convenience methods live on `*Kernel` rather than on `*Module`/`*Instance` because `opm/kernel` already imports `opm/module`; placing methods that take a `*kernel.Kernel` on `*Module`/`*Instance` would close the import cycle.

#### Scenario: Module.ConfigSchema accessor

- **WHEN** a caller invokes `m.ConfigSchema()`
- **THEN** the result is the `cue.Value` at `b.Paths().Config` inside `m.Package`, where `b` is the binding for `m.APIVersion`
- **AND** the accessor returns a zero value if the module has no `#config` field

#### Scenario: Kernel.ValidateModuleValues delegates without name wrapping

- **WHEN** a caller invokes `k.ValidateModuleValues(m, values)`
- **THEN** the result is identical to `k.ValidateConfig(m.ConfigSchema(), values)`
- **AND** the method does NOT wrap the error with the module name (caller wraps if needed; the phase method `Kernel.Validate` is the wrapping entry point)

#### Scenario: Kernel.ValidateModuleValuesPartial delegates

- **WHEN** a caller invokes `k.ValidateModuleValuesPartial(m, values)`
- **THEN** the result is identical to `k.ValidateConfigPartial(m.ConfigSchema(), values)`

#### Scenario: Kernel.ValidateModuleValuesDetailed delegates

- **WHEN** a caller invokes `k.ValidateModuleValuesDetailed(m, sources, opts...)`
- **THEN** the result is identical to `k.ValidateConfigDetailed(m.ConfigSchema(), sources, opts...)`

#### Scenario: Instance equivalents

- **WHEN** a caller invokes any of `k.ValidateInstanceValues(r, values)`, `k.ValidateInstanceValuesPartial(r, values)`, or `k.ValidateInstanceValuesDetailed(r, sources, opts...)`
- **THEN** the behavior mirrors the Module equivalents, sourcing the schema from `r.ConfigSchema()` (which resolves the embedded `#module` reference at `b.Paths().Module` then `b.Paths().Config`)

## MODIFIED Requirements

### Requirement: Phase Method Wraps With Module Name

`Kernel.Validate(ctx, ValidateInput)` SHALL retain its public signature and SHALL internally call `Kernel.ValidateConfig` then wrap any returned error with `fmt.Errorf("module %q: %w", name, err)` where `name` is derived from `ValidateInput.ModuleInstance.Metadata.Name` (or a "<unknown>" fallback).

#### Scenario: Validate phase signature unchanged

- **WHEN** a caller invokes `k.Validate(ctx, ValidateInput{Module, ModuleInstance, Values})`
- **THEN** the method returns nil on success or a wrapped `error` on failure
- **AND** the wrapped error is walkable via `errors.As` and `cueerrors.Errors` to reach the underlying CUE diagnostics
- **AND** the textual prefix on `Error()` is `module "<name>": ` followed by the CUE error message

#### Scenario: ProcessModuleInstance uses ValidateConfig and wraps with instance name

- **WHEN** `k.ProcessModuleInstance(ctx, spec, mod, values)` performs its values validation step
- **THEN** the call routes through `k.ValidateConfig(schema, values)` (no per-call options)
- **AND** any returned error is wrapped with `fmt.Errorf("instance %q: %w", instanceName, err)`
- **AND** the subsequent `spec.Validate(cue.Concrete(true))` call (CUE stdlib) is unchanged

## REMOVED Requirements

### Requirement: Module and Release Typed Convenience Methods

**Reason**: Renamed for Release→Instance vocabulary (enhancement 0002 D11/D12).

**Migration**: See the ADDED requirement of the new name.
