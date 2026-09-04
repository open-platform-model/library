## ADDED Requirements

### Requirement: Internal call sites use schema paths

Every kernel-internal Go call site that reads a sub-value of an artifact's `Package` SHALL read it through the `opm/schema` path variables (`schema-dispatch`, "Path inventory exposed as package-level vars"), never through a removed struct field or an ad-hoc path literal. The render path reads no artifact sub-value in Go: the instance and the platform enter the render build by import, and the generated glue reads `components` and `#composedTransformers` in CUE.

#### Scenario: Render build reads artifacts by import

- **WHEN** `Kernel.Render` runs
- **THEN** no Go code looks up the instance's components or the platform's transformers by path; the staged render module imports both packages and the glue reads them

#### Scenario: Instance processing uses schema paths

- **WHEN** `Kernel.ProcessModuleInstance` reads a module's `#config` schema or fills validated values
- **THEN** the reads and the fill go through `schema.Module`, `schema.Config` and `schema.Values`
- **AND** there is no direct dereference of a removed field

#### Scenario: Metadata reads go through the metadata path

- **WHEN** Go code outside `opm/schema` reads a field of an artifact's `metadata` (the loaders' identity checks, the instance name for a diagnostic, the module's snake-case name)
- **THEN** it navigates to the metadata value through `schema.Metadata` and reads the field off that value, never through a dotted path literal rooted at the artifact

### Requirement: Instance exposes its components and config schema

`*module.Instance` SHALL expose `Components()` (the instance's evaluated components value, definition fields included, read through `schema.Components`) and `ConfigSchema()` (the embedded module's `#config`, read through `schema.Module` then `schema.Config`). It SHALL expose no accessor that mirrors a decoded metadata field (`Metadata` is the projection) and no accessor over the module-metadata projection: the transformer context that read those is projected by core (0019 D12).

#### Scenario: Components accessor

- **WHEN** a caller invokes `inst.Components()` on an acquired instance
- **THEN** the returned value is `inst.Package.LookupPath(schema.Components)` with `#names`, `#resources`, `#traits` and `#blueprints` intact

#### Scenario: No metadata-mirroring accessors

- **WHEN** a developer inspects the exported methods of `*module.Instance`
- **THEN** none of `InstanceName`, `Namespace`, `InstanceUUID`, `InstanceFQN`, `ModuleVersion`, `Labels`, `Annotations`, `MatchComponents` exists

## REMOVED Requirements

### Requirement: Internal Call Sites Use Binding Paths

**Reason**: binding-era text (`binding.Paths()`, the `opm/compile/` and `opm/validate/` pipelines) whose referents are all gone: bindings were retired by `schema-dispatch`, `opm/validate/` by the kernel consolidation, and `opm/compile/` by this change. Restated as "Internal call sites use schema paths".

**Migration**: none.

### Requirement: Instance-Side Module Paths On Binding

**Reason**: binding-era text (`api.Paths`, `b.Paths().ModuleMetadata`, `api.Lookup(rel.APIVersion)`). `schema.Module` survives as the instance's reference to its source module (read by `ConfigSchema()` and `ProcessModuleInstance`); `ModuleMetadataPath` and the `ModuleVersion()` accessor are retired with the Go transformer-context builder, their only production reader. Restated as "Instance exposes its components and config schema".

**Migration**: read the source module's identity off `inst.Package.LookupPath(schema.Module)`.
