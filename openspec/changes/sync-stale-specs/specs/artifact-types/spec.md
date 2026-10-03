## MODIFIED Requirements

### Requirement: Uniform Artifact Shape

Every OPM artifact type accepted by the kernel (`module.Module`, `module.Instance`, `platform.Platform`, `catalog.Catalog`) SHALL be a Go struct with exactly three exported fields: `Metadata *<Type>Metadata`, `Package cue.Value` and `Source *Source`. The `Metadata` pointer holds a decoded ergonomic projection; the `Package` field carries the source-of-truth CUE value; `Source` is the staged source tree the artifact was acquired or synthesized from, nil for an artifact built from a bare value. No artifact struct SHALL carry an API version field (`schema-dispatch`, "Module, Instance, Platform structs do not carry APIVersion").

#### Scenario: Module type shape

- **WHEN** a developer reads the `module.Module` struct definition
- **THEN** the struct has exactly three exported fields: `Metadata` (typed `*ModuleMetadata`), `Package` (typed `cue.Value`) and `Source` (typed `*Source`)
- **AND** there are no `APIVersion`, `Spec` or `Config` exported fields

#### Scenario: ModuleInstance type shape

- **WHEN** a developer reads the `module.Instance` struct definition
- **THEN** the struct has exactly three exported fields: `Metadata` (typed `*InstanceMetadata`), `Package` (typed `cue.Value`) and `Source` (typed `*Source`)
- **AND** there are no `APIVersion`, `Module`, `Spec` or `Values` exported fields

### Requirement: Kernel Artifact Type Set

The kernel SHALL accept exactly four artifact types: `Module`, `ModuleInstance`, `Platform` and `Catalog`. `#ModuleDebug` SHALL NOT be a kernel artifact type. Debug values are carried as a `debugValues` field within `Module.Package`; whether they participate in the values stack is a frontend policy decision, not a kernel concern.

`Catalog` is admitted on the terms ADR-009 records and on no others: the kernel acquires it, reads it and derives from it, and every verdict about what it reads stays with the caller. A catalog is never rendered and never executed; the transformers it carries reach a render only through a platform that subscribes to it. The set SHALL NOT grow again without the candidate meeting, in writing, the four-part test ADR-009 states.

#### Scenario: No top-level ModuleDebug type

- **WHEN** a developer searches the kernel public API for `ModuleDebug`
- **THEN** no exported Go type with that name exists in any `opm/` package
- **AND** no `opm/` package exports a `DecodeModuleDebugMetadata` or equivalent decoder

#### Scenario: debugValues accessible via Module.Package

- **WHEN** a frontend reads debug overlays from a Module
- **THEN** the read goes through `Module.Package.LookupPath(schema.DebugValues)`
- **AND** the kernel never receives `debugValues` as a separate parameter

#### Scenario: Documentation explicitly retires the construct

- **WHEN** a developer reads `library/README.md` or `opm/module/` godoc
- **THEN** at least one prose section states that `#ModuleDebug` is not a kernel artifact and that debug overlays are a frontend layering concern

#### Scenario: The fourth type is a read, never a render

- **WHEN** a developer searches the kernel public API for a way to render or execute a `Catalog`
- **THEN** none exists: `*catalog.Catalog` is produced by the two catalog acquire verbs or `catalog.NewCatalogFromValue`, and no kernel method renders or executes one; `Kernel.Render` takes an instance and a platform
- **AND** what the catalog provides and what it requires are reported by methods on the artifact, which return data and refuse nothing

#### Scenario: The enumerated set is stated once and agrees everywhere

- **WHEN** a developer reads the kernel's accepted-kinds list in `README.md`, `AGENTS.md` and this spec
- **THEN** all three enumerate the same four types
- **AND** each names ADR-009 as the record of why the fourth was admitted and of the test a fifth must pass

### Requirement: Instance exposes its components and config schema

`*module.Instance` SHALL expose `Components()` (the instance's evaluated components value, definition fields included, read through `schema.Components`) and `ConfigSchema()` (the embedded module's `#config`, read as `Package.LookupPath(schema.Module)` followed by `LookupPath(schema.Config)`). `ConfigSchema()` SHALL return the zero `cue.Value` (not an error) when the receiver is `nil`, when the instance carries no `#module`, or when the embedded module declares no `#config`, and SHALL consult no version or binding. The instance SHALL expose no accessor that mirrors a decoded metadata field (`Metadata` is the projection) and no accessor over the module-metadata projection: the transformer context that read those is projected by core (0019:D12).

#### Scenario: Components accessor

- **WHEN** a caller invokes `inst.Components()` on an acquired instance
- **THEN** the returned value is `inst.Package.LookupPath(schema.Components)` with `#names`, `#resources`, `#traits` and `#blueprints` intact

#### Scenario: No metadata-mirroring accessors

- **WHEN** a developer inspects the exported methods of `*module.Instance`
- **THEN** none of `InstanceName`, `Namespace`, `InstanceUUID`, `InstanceFQN`, `ModuleVersion`, `Labels`, `Annotations`, `MatchComponents` exists

#### Scenario: Config schema reachable on a well-formed instance

- **WHEN** a caller invokes `inst.ConfigSchema()` on a `*Instance` whose `Package` carries an embedded `#module` with a `#config` definition
- **THEN** the returned `cue.Value` exists (`v.Exists() == true`)
- **AND** it is the value `inst.Package.LookupPath(schema.Module).LookupPath(schema.Config)`

#### Scenario: Config schema is the zero value on a missing #module

- **WHEN** a caller invokes `inst.ConfigSchema()` on a `*Instance` whose `Package` has no `#module` field
- **THEN** the returned `cue.Value` is the zero value (`v.Exists() == false`)
- **AND** no error is returned

#### Scenario: Config schema is the zero value on a missing #config

- **WHEN** a caller invokes `inst.ConfigSchema()` on a `*Instance` whose embedded `#module` does not declare a `#config` definition
- **THEN** the returned `cue.Value` is the zero value (`v.Exists() == false`)

#### Scenario: Config schema on a nil instance is the zero value

- **WHEN** a caller invokes `(*Instance)(nil).ConfigSchema()`
- **THEN** the returned `cue.Value` is the zero value
- **AND** no panic occurs

## REMOVED Requirements

### Requirement: APIVersion Field Stamped at Construction

**Reason**: No artifact struct has an `APIVersion` field and the `opm/apiversion` package no longer exists. The library consumes one OPM schema and detects no version. `schema-dispatch` ("Module, Instance, Platform structs do not carry APIVersion") states the current rule, and this requirement contradicted it.
**Migration**: None. There is no field to read. A caller that needs the schema release reads `Kernel.SchemaCache().ResolvedVersion()` or the module's own `cue.mod/module.cue`.

### Requirement: Instance Config Schema Accessor

**Reason**: It looked the schema up through a version binding keyed by `r.APIVersion`, and neither exists. Its scenarios "Schema reachable on a well-formed instance" and "Zero value on unregistered binding" name that binding. The accessor itself is unchanged; "Instance exposes its components and config schema" already specified it through `schema.Module` and `schema.Config`, and now carries its zero-value cases too.
**Migration**: None for callers. `(*module.Instance).ConfigSchema()` keeps its signature and its zero-value contract.
