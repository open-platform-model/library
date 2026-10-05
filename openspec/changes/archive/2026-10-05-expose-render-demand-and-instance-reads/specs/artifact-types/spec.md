## ADDED Requirements

### Requirement: Module and platform constructors from cue.Value

The library SHALL provide constructor helpers that build the module and platform artifacts from a raw `cue.Value`: `module.NewModuleFromValue(v)` and `platform.NewPlatformFromValue(v)`. Each constructor SHALL take only the artifact value, decode `Metadata` from the value's `metadata` field, and set the `Package` field to the supplied `cue.Value` unmodified. The constructed artifact carries no `Source`. The kernel SHALL NOT wrap these constructors: a frontend holding a value calls the package constructor directly; a frontend that wants a source-carrying artifact uses an acquire verb. No instance constructor SHALL be exported: an instance is produced only by the kernel's acquisition paths, which stamp its staged source.

#### Scenario: NewModuleFromValue success path

- **WHEN** a caller invokes `module.NewModuleFromValue(v)` with a `cue.Value` carrying a valid module
- **THEN** the returned `*Module` has `Metadata.Name` matching the value's `metadata.name`
- **AND** `Package` is the supplied `cue.Value` unchanged
- **AND** `Source` is nil

#### Scenario: NewModuleFromValue with missing metadata

- **WHEN** a caller invokes `module.NewModuleFromValue(v)` with a `cue.Value` that has no `metadata` field
- **THEN** the function returns a non-nil error stating that the metadata field is required
- **AND** no partial `*Module` is returned

#### Scenario: NewModuleFromValue with unknown apiVersion

- **WHEN** a caller invokes `module.NewModuleFromValue(v)` with a `cue.Value` that carries no `apiVersion` field, or one the library has never heard of
- **THEN** the constructor performs no version detection and no binding lookup; the value is decoded as the single OPM schema the library consumes
- **AND** the result depends only on the `metadata` field being present and decodable

#### Scenario: NewPlatformFromValue hoists the platform type

- **WHEN** a caller invokes `platform.NewPlatformFromValue(v)` with a `#Platform` value whose root has `type: "kubernetes"` alongside its `metadata` block
- **THEN** the returned `*Platform` has `Metadata.Type == "kubernetes"`, `Package == v` and `Source == nil`

#### Scenario: Constructors take no context owner

- **WHEN** a consumer inspects the exported identifiers of `opm/module` and `opm/platform`
- **THEN** `NewModuleFromValue` and `NewPlatformFromValue` each take a single `cue.Value` argument
- **AND** no `CueContextOwner` interface exists in either package

#### Scenario: No kernel constructor wrappers

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** neither `NewModuleFromValue` nor `NewPlatformFromValue` exists on it

#### Scenario: No instance constructor

- **WHEN** a consumer inspects the exported identifiers of `opm/module` and the exported methods of `Kernel`
- **THEN** `NewInstanceFromValue` does not exist in either
- **AND** the only ways to obtain a `*module.Instance` are `Kernel.AcquireInstanceFromDir` and `Kernel.SynthesizeInstance`

### Requirement: Instance exposes its components, config schema, values and module metadata

`*module.Instance` SHALL expose the following accessors:

- `Components()`: the instance's evaluated components value, definition fields included, read through `schema.Components`.
- `ConfigSchema()`: the embedded module's `#config`, read as `Package.LookupPath(schema.Module)` followed by `LookupPath(schema.Config)`.
- `Values()`: the instance's merged values as evaluated, read through `schema.Values`.
- `ModuleMetadata()`: the metadata of the module the instance was built from, decoded from `Package.LookupPath(schema.Module)` on each call.

`ConfigSchema()` and `Values()` SHALL return the zero `cue.Value` (not an error) when the receiver is `nil` or the field they read is absent. `ConfigSchema()` returns it as well when the instance carries no `#module` or the embedded module declares no `#config`. `ModuleMetadata()` SHALL return nil when the receiver is `nil`, when the instance carries no `#module`, or when the embedded module's metadata does not decode. Each call decodes afresh from `Package`; the returned struct is a copy, and mutating it does not change `Package`. No accessor SHALL consult a version or binding.

The instance SHALL expose no accessor that mirrors a single field of its own decoded `Metadata`, because `Metadata` is that projection. The render build reads none of these accessors: the transformer context is projected by core (0019:D12), and the glue reads the instance by import.

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


#### Scenario: Values accessor

- **WHEN** a caller invokes `inst.Values()` on an acquired instance
- **THEN** the returned value is `inst.Package.LookupPath(schema.Values)`, the instance's merged values as evaluated

#### Scenario: Values on a nil instance is the zero value

- **WHEN** a caller invokes `(*Instance)(nil).Values()`
- **THEN** the returned `cue.Value` is the zero value
- **AND** no panic occurs

#### Scenario: Module metadata of an acquired instance

- **WHEN** a caller invokes `inst.ModuleMetadata()` on an instance returned by `SynthesizeInstance` or `AcquireInstanceFromDir`
- **THEN** it returns a non-nil `*ModuleMetadata` whose `Name`, `ModulePath`, `Version`, `FQN` and `UUID` equal the embedded `#module`'s `metadata`

#### Scenario: Module metadata is nil when it cannot be read

- **WHEN** a caller invokes `ModuleMetadata()` on a nil `*Instance`, on an instance whose `Package` has no `#module`, or on one whose embedded module metadata does not decode
- **THEN** it returns nil
- **AND** no panic occurs and no error is raised

#### Scenario: Module metadata reads the package

- **WHEN** a caller builds an `Instance` struct literal whose `Package` carries an embedded `#module`
- **THEN** `ModuleMetadata()` returns that module's metadata, decoded from `Package`, with no processing step having run

### Requirement: Module exposes its debug values

`*module.Module` SHALL expose `DebugValues()`, which returns `Package.LookupPath(schema.DebugValues)`: the module's author-supplied `debugValues`. It SHALL return the zero `cue.Value` for a nil receiver or a module that declares no `debugValues`, so callers test `Exists()`. The kernel itself SHALL NOT read it. Whether a frontend layers it into its values stack remains the frontend's policy. No `InitValues()` accessor is exposed.

#### Scenario: Debug values present

- **WHEN** a caller invokes `mod.DebugValues()` on a module that declares `debugValues`
- **THEN** the returned value exists and is `mod.Package.LookupPath(schema.DebugValues)`

#### Scenario: Debug values absent

- **WHEN** a caller invokes `mod.DebugValues()` on a module that declares no `debugValues`, or on a nil `*Module`
- **THEN** the returned value is the zero value (`Exists() == false`)
- **AND** no panic occurs

## MODIFIED Requirements

### Requirement: Kernel Artifact Type Set

The kernel SHALL accept exactly four artifact types: `Module`, `ModuleInstance`, `Platform` and `Catalog`. `#ModuleDebug` SHALL NOT be a kernel artifact type. Debug values are carried as a `debugValues` field within `Module.Package`; whether they participate in the values stack is a frontend policy decision, not a kernel concern.

`Catalog` is admitted on the terms ADR-009 records and on no others: the kernel acquires it, reads it and derives from it, and every verdict about what it reads stays with the caller. A catalog is never rendered and never executed; the transformers it carries reach a render only through a platform that subscribes to it. The set SHALL NOT grow again without the candidate meeting, in writing, the four-part test ADR-009 states.

#### Scenario: No top-level ModuleDebug type

- **WHEN** a developer searches the kernel public API for `ModuleDebug`
- **THEN** no exported Go type with that name exists in any `opm/` package
- **AND** no `opm/` package exports a `DecodeModuleDebugMetadata` or equivalent decoder

#### Scenario: debugValues accessible via Module.Package

- **WHEN** a frontend reads debug overlays from a Module
- **THEN** the read goes through `Module.DebugValues()`, which returns `Module.Package.LookupPath(schema.DebugValues)`
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

## REMOVED Requirements

### Requirement: Constructor Helpers from cue.Value

**Reason**: Its scenario "NewInstanceFromValue success path" duplicates the scenario "No instance constructor": the same WHEN, and a THEN that differs only in wording. OpenSpec refuses a MODIFIED requirement that drops a scenario.

**Migration**: The same requirement text and every other scenario are re-added as "Module and platform constructors from cue.Value".

### Requirement: Instance exposes its components and config schema

**Reason**: It said that the instance exposes no accessor over the module-metadata projection. ADR-013, decision d2 adds `ModuleMetadata()` and `Values()` for frontends, so the requirement's name and that clause no longer hold.

**Migration**: It is re-added as "Instance exposes its components, config schema, values and module metadata", with every scenario kept. A caller that read `inst.Package.LookupPath(schema.Module)` for module metadata, or `schema.Values` for values, can call the accessors. The raw reads keep working, because `Package` stays public.
