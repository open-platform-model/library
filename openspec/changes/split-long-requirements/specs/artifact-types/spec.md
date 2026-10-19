## MODIFIED Requirements

### Requirement: An overlay source can be written to a directory
`module.Source` SHALL expose `WriteTo(dir string) ([]string, error)`, which writes every entry of an overlay-mode source under `dir` at the entry's path relative to `Root`, creating parent directories as needed, and returns the dir-relative paths it wrote, sorted.

#### Scenario: A fetched module is written to disk

- **WHEN** a frontend acquires a module from a registry and calls `Source.WriteTo(dest)` on it
- **THEN** every file of the fetched module is written under `dest` at its module-relative path, the returned list names each written path relative to `dest` in sorted order, and no registry fetch happens

#### Scenario: On-disk source refused

- **WHEN** `WriteTo` is called on a source with a nil `Overlay`
- **THEN** it returns an error and writes nothing

#### Scenario: Entry outside the root refused

- **WHEN** an overlay entry's absolute path is not under `Root`
- **THEN** `WriteTo` returns an error naming the entry and writes nothing, including entries that were valid

#### Scenario: Frontends stop copying fetched modules themselves

- **WHEN** a frontend scaffolds a new module from a published template
- **THEN** it writes the acquired module's `Source` into place with `WriteTo`
- **AND** it performs no second registry fetch and no filesystem walk of its own

#### Scenario: The render stage writes nothing for an overlay input

- **WHEN** `Kernel.Render` stages an overlay-mode instance or platform
- **THEN** `WriteTo` is not called and no entry of the overlay appears on disk

## REMOVED Requirements

### Requirement: Uniform Artifact Shape
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Every artifact type is a struct with exactly three exported fields" and "The Metadata, Package and Source fields hold a projection, the source of truth and the staged source".

### Requirement: Kernel Artifact Type Set
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The kernel accepts exactly four artifact types", "ModuleDebug is not a kernel artifact type" and "Catalog is admitted on the terms of ADR-009 and a catalog is never rendered".

### Requirement: Artifacts carry their staged source
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "module.Source describes a staged source tree in two modes", "module.Source carries a Pkg field", "Every instance the kernel returns carries a non-nil Source" and "A module's Source is populated by acquisition and nil for a bare value".

### Requirement: Instance acquisition from a directory
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "AcquireInstanceFromDir loads a package through the shape gate and processes it", "An acquired instance's Source describes the directory", "A directory with no enclosing module is its own root", "The registry mapping for an instance's imports is the kernel's", "An instance acquired without values has an on-disk source", "Values sources are unified in order", "The unified values are rendered as a values file beside the package's files", "The package is built in one pass through the instance shape gate", "A layered instance's Source is in overlay mode" and "Instance acquisition writes nothing into the caller's directory".

### Requirement: Internal call sites use schema paths
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Kernel-internal call sites read an artifact's Package through the path inventory" and "The render path reads no artifact sub-value in Go".

### Requirement: Module acquisition from a directory
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "AcquireModuleFromDir loads a module package through the shape gate and constructs the typed module", "An acquired module's Source has a Root and a Pkg", "An acquired module's Source Overlay holds every cue file under Root", "An acquired module can be passed to SynthesizeInstance without further staging", "The registry mapping for a module's imports is the kernel's" and "Module acquisition writes nothing into the caller's directory".

### Requirement: Acquisition shape gate
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A shape gate runs right after the package is built", "The gate rejects a non-struct root, a wrong kind and more than one instance", "The gate rejects a package whose required identity field is absent or not concrete", "The required identity fields are those the schema never defaults", "A registry entry with no embedded catalog fails as a missing required field", "A defaulted identity field is not concrete for the gate", "A directory-acquired and a registry-acquired artifact are gated identically" and "opm/errors exports the three gate sentinels".

### Requirement: Directory acquisition reads its tree once and builds from that read
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A directory acquire verb checks the directory and a missing path fails", "A directory acquire verb refuses a path that is not a directory", "An overlay-mode verb reads the cue files once and builds from that overlay", "A failed layered build is attributed from the overlay already read", "Platform acquisition and instance acquisition without values keep the on-disk mode" and "A package that embeds a non-CUE file acquires from a directory".

### Requirement: Module and platform constructors from cue.Value
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Constructor helpers build module and platform artifacts from a cue.Value", "A constructed artifact carries no Source", "The kernel does not wrap the constructors" and "No instance constructor is exported".

### Requirement: Instance exposes its components, config schema, values and module metadata
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Instance exposes Components and Values", "Instance exposes ConfigSchema", "Instance exposes ModuleMetadata", "ConfigSchema and Values return the zero value rather than an error", "ModuleMetadata returns nil when it cannot be read and decodes afresh", "No accessor consults a version or mirrors a metadata field" and "The render build reads none of the instance accessors".

## ADDED Requirements

### Requirement: Every artifact type is a struct with exactly three exported fields
Every OPM artifact type accepted by the kernel (`module.Module`, `module.Instance`, `platform.Platform`, `catalog.Catalog`) SHALL be a Go struct with exactly three exported fields: `Metadata *<Type>Metadata`, `Package cue.Value` and `Source *Source`. No artifact struct SHALL carry an API version field (`schema-dispatch`, "Module, Instance, Platform structs do not carry APIVersion").

#### Scenario: Module type shape

- **WHEN** a developer reads the `module.Module` struct definition
- **THEN** the struct has exactly three exported fields: `Metadata` (typed `*ModuleMetadata`), `Package` (typed `cue.Value`) and `Source` (typed `*Source`)
- **AND** there are no `APIVersion`, `Spec` or `Config` exported fields

### Requirement: The Metadata, Package and Source fields hold a projection, the source of truth and the staged source
The `Metadata` pointer SHALL hold a decoded ergonomic projection; the `Package` field carries the source-of-truth CUE value; `Source` is the staged source tree the artifact was acquired or synthesized from, nil for an artifact built from a bare value.

#### Scenario: ModuleInstance type shape

- **WHEN** a developer reads the `module.Instance` struct definition
- **THEN** the struct has exactly three exported fields: `Metadata` (typed `*InstanceMetadata`), `Package` (typed `cue.Value`) and `Source` (typed `*Source`)
- **AND** there are no `APIVersion`, `Module`, `Spec` or `Values` exported fields

### Requirement: The kernel accepts exactly four artifact types
The kernel SHALL accept exactly four artifact types: `Module`, `ModuleInstance`, `Platform` and `Catalog`. The set SHALL NOT grow again without the candidate meeting, in writing, the four-part test ADR-009 states.

#### Scenario: The enumerated set is stated once and agrees everywhere

- **WHEN** a developer reads the kernel's accepted-kinds list in `README.md`, `AGENTS.md` and this spec
- **THEN** all three enumerate the same four types
- **AND** each names ADR-009 as the record of why the fourth was admitted and of the test a fifth must pass

### Requirement: ModuleDebug is not a kernel artifact type
`#ModuleDebug` SHALL NOT be a kernel artifact type. Debug values are carried as a `debugValues` field within `Module.Package`; whether they participate in the values stack is a frontend policy decision, not a kernel concern.

#### Scenario: No top-level ModuleDebug type

- **WHEN** a developer searches the kernel public API for `ModuleDebug`
- **THEN** no exported Go type with that name exists in any `opm/` package
- **AND** no `opm/` package exports a `DecodeModuleDebugMetadata` or equivalent decoder

#### Scenario: debugValues accessible via Module.Package

- **WHEN** a frontend reads debug overlays from a Module
- **THEN** the read goes through `Module.DebugValues()`, which returns `Module.Package.LookupPath(corepath.DebugValues)`
- **AND** the kernel never receives `debugValues` as a separate parameter

#### Scenario: Documentation explicitly retires the construct

- **WHEN** a developer reads `library/README.md` or `opm/module/` godoc
- **THEN** at least one prose section states that `#ModuleDebug` is not a kernel artifact and that debug overlays are a frontend layering concern

### Requirement: Catalog is admitted on the terms of ADR-009 and a catalog is never rendered
`Catalog` SHALL be admitted on the terms ADR-009 records and on no others: the kernel acquires it, reads it and derives from it, and every verdict about what it reads stays with the caller. A catalog is never rendered and never executed; the transformers it carries reach a render only through a platform that subscribes to it.

#### Scenario: The fourth type is a read, never a render

- **WHEN** a developer searches the kernel public API for a way to render or execute a `Catalog`
- **THEN** none exists: `*catalog.Catalog` is produced by the two catalog acquire verbs or `catalog.NewCatalogFromValue`, and no kernel method renders or executes one; `Kernel.Render` takes an instance and a platform
- **AND** what the catalog provides and what it requires are reported by methods on the artifact, which return data and refuse nothing

### Requirement: module.Source describes a staged source tree in two modes
`module.Source` SHALL describe the staged source tree an artifact was loaded or synthesized from, in two modes: an in-memory tree (`Overlay` non-empty, keyed under the absolute `Root`, which may be a synthetic path or a real directory) or an on-disk tree (`Overlay` nil, `Root` a real directory). `Overlay` SHALL map each absolute file path to that file's contents as bytes.

#### Scenario: Overlay entries are bytes

- **WHEN** a consumer reads `Source.Overlay` on any acquired or synthesized artifact
- **THEN** each entry is the file's contents as a byte slice keyed by its absolute path under `Root`
- **AND** no entry is an opaque loader source type

#### Scenario: On-disk source mode

- **WHEN** an artifact's `Source` has `Overlay == nil` and a non-empty `Root`
- **THEN** consumers treat `Root` as a real filesystem directory holding the tree, and `Pkg` as the package directory within it

### Requirement: module.Source carries a Pkg field
`module.Source` SHALL carry a `Pkg` field naming the package directory relative to `Root`; empty means the root package (`.`).

#### Scenario: Pkg names the package directory

- **WHEN** a developer reads `module.Source`
- **THEN** it carries a `Pkg` field naming the package directory relative to `Root`, and empty means the root package (`.`)

### Requirement: Every instance the kernel returns carries a non-nil Source
`module.Instance` SHALL expose `Source *module.Source`; every instance the kernel returns carries a non-nil `Source`, since instances are constructed only by `AcquireInstanceFromDir` and `SynthesizeInstance`.

#### Scenario: Value-constructed instance has no source

- **WHEN** a consumer looks for a way to build an instance from a bare `cue.Value`
- **THEN** none exists: the instance constructor is not exported, so no instance without a `Source` can be created
- **AND** an instance obtained from `AcquireInstanceFromDir` or `SynthesizeInstance` has a non-nil `Source` with a non-empty `Root`

### Requirement: A module's Source is populated by acquisition and nil for a bare value
`module.Module.Source` SHALL be populated in overlay mode by both `AcquireModuleFromRegistry` and `AcquireModuleFromDir`, and SHALL be nil for a module built from a bare value via `NewModuleFromValue`; the `Module.HasSource()` gate (non-nil `Source`, non-empty `Root`, non-empty `Overlay`) SHALL be unchanged.

#### Scenario: Value-constructed module has no source

- **WHEN** a caller builds a module via `NewModuleFromValue`
- **THEN** the returned `*Module` has `Source == nil` and `HasSource()` reports false

#### Scenario: Directory-acquired module has an overlay source

- **WHEN** a caller acquires a module via `AcquireModuleFromDir`
- **THEN** the returned `*Module` has a non-nil `Source` whose `Root` is a real directory and whose `Overlay` holds the module's `.cue` files, and `HasSource()` reports true

#### Scenario: Synth gate unchanged

- **WHEN** `SynthesizeInstance` is invoked with a module whose `Source` is nil or overlay-empty
- **THEN** it fails with an error wrapping `oerrors.ErrMissingSource`

### Requirement: AcquireInstanceFromDir loads a package through the shape gate and processes it
The kernel SHALL expose `AcquireInstanceFromDir(ctx, dir, values ...Source)`, which loads a `#ModuleInstance` CUE package from a directory through the acquisition shape gate, processes it through the kernel's instance processing (concreteness enforced, metadata decoded).

#### Scenario: Validation failures propagate

- **WHEN** the directory's instance package is not fully concrete
- **THEN** `AcquireInstanceFromDir` returns the concreteness error framed `instance "<name>": …`, and no partial `*Instance`

#### Scenario: Loader failures propagate

- **WHEN** the directory does not exist, holds no CUE package, or fails the instance shape gate
- **THEN** the error wraps the corresponding `opm/errors` sentinel (`ErrInvalidPackage`, `ErrWrongKind`, or `ErrMissingRequiredField`)

### Requirement: An acquired instance's Source describes the directory
The kernel SHALL expose `AcquireInstanceFromDir(ctx, dir, values ...Source)`, which returns a `*module.Instance` whose `Source` describes the directory: `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/module.cue`, the directory itself when it is the root), and `Pkg` = the package directory relative to `Root` (empty for the root package).

#### Scenario: Acquired instance carries its source

- **WHEN** a caller invokes `AcquireInstanceFromDir(ctx, dir)` on a module root directory holding a valid, concrete instance package
- **THEN** the returned `*Instance` has decoded `Metadata`, `Source.Root` equal to the directory's absolute path, an empty `Source.Pkg`, and a nil `Overlay`

### Requirement: A directory with no enclosing module is its own root
A directory with no enclosing module SHALL be its own root with an empty `Pkg`.

#### Scenario: Acquired subpackage names its module root

- **WHEN** a caller invokes `AcquireInstanceFromDir` on a subdirectory of a module (its `cue.mod/module.cue` lives in an ancestor)
- **THEN** the returned `*Instance` has `Source.Root` equal to the module root's absolute path and `Source.Pkg` equal to the subdirectory's slash-separated path relative to it

### Requirement: The registry mapping for an instance's imports is the kernel's
The registry mapping used for the instance's imports SHALL be the kernel's (`WithRegistry`), applied through the load configuration's environment and never through `os.Setenv`.

#### Scenario: Registry mapping is the kernel's

- **WHEN** a kernel constructed with `WithRegistry(mapping)` acquires an instance whose module imports a catalog under `opmodel.dev`
- **THEN** the import resolves through `mapping`
- **AND** the process environment is not mutated

### Requirement: An instance acquired without values has an on-disk source
With no values, `Source.Overlay` SHALL be nil (on-disk mode).

#### Scenario: Without values the overlay is nil

- **WHEN** a caller invokes `AcquireInstanceFromDir` with no values
- **THEN** `Source.Overlay` is nil (on-disk mode)

### Requirement: Values sources are unified in order
With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts), the kernel SHALL unify them in order.

#### Scenario: Extra values layer onto the package

- **WHEN** a caller invokes `AcquireInstanceFromDir(ctx, dir, a, b)` with two values sources on an instance package whose `values` leave a field unset
- **THEN** the returned instance's `values` carry the unified sources merged with the package's own values, `Source.Overlay` is non-nil and contains the on-disk `.cue` files plus the rendered values file, and the caller's directory is unchanged

#### Scenario: No option type for values

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`
- **THEN** neither `AcquireOption` nor `WithValues` exists; values are the variadic trailing argument

### Requirement: The unified values are rendered as a values file beside the package's files
With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts), the kernel SHALL render the result as a values file (`opm-values.cue`, a reserved name so it can never shadow a file the package authored) declaring the package's own package name and the top-level `values` field, place it beside the package's on-disk files in an overlay.

#### Scenario: Layered instance renders

- **WHEN** an instance acquired with extra values is passed to `Kernel.Render`
- **THEN** the render build imports the layered package and the rendered objects reflect the extra values

### Requirement: The package is built in one pass through the instance shape gate
With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts), the kernel SHALL build the package in one pass through the instance shape gate, so the schema's own values unification performs the merge in CUE.

#### Scenario: Conflicting extra values fail at acquisition

- **WHEN** an extra values source conflicts with the package's own values or its module's `#config` schema
- **THEN** `AcquireInstanceFromDir` returns the validation error naming the conflicting path, with source positions attributable to the values source, and no partial `*Instance`

### Requirement: A layered instance's Source is in overlay mode
The values sources are those of requirement "Values sources are unified in order". The returned instance's `Source` SHALL then be overlay mode: same `Root` and `Pkg`, `Overlay` carrying every on-disk `.cue` file under the module root (the module's own `cue.mod/module.cue` included) plus the rendered values file, as bytes.

#### Scenario: A layered instance carries an overlay source

- **WHEN** `AcquireInstanceFromDir` is invoked with one or more values sources
- **THEN** the returned instance's `Source` is overlay mode: same `Root` and `Pkg`, `Overlay` carrying every on-disk `.cue` file under the module root plus the rendered values file, as bytes

### Requirement: Instance acquisition writes nothing into the caller's directory
The kernel MUST NOT write into the caller's directory and MUST NOT fill values into the evaluated value from Go.

#### Scenario: Acquisition writes nothing and fills nothing from Go

- **WHEN** a caller invokes `AcquireInstanceFromDir`
- **THEN** the kernel does not write into the caller's directory and does not fill values into the evaluated value from Go

### Requirement: Kernel-internal call sites read an artifact's Package through the path inventory
Every kernel-internal Go call site that reads a sub-value of an artifact's `Package` SHALL read it through the path inventory (`schema-dispatch`, "Path inventory exposed as package-level vars"): the three variables `opm/schema` exports and the rest in the internal package `opm/internal/corepath`, never through a removed struct field or an ad-hoc path literal.

#### Scenario: Instance processing uses schema paths

- **WHEN** the kernel's internal instance processing decodes a built instance's metadata, or `Kernel.AcquireInstanceFromDir` with extra values checks the sources against the module's `#config`
- **THEN** the reads go through `schema.Metadata`, and `schema.Module` then `corepath.Config`
- **AND** no Go code fills values into the evaluated instance; the merge happens in the build through the package's own `values` field

#### Scenario: Metadata reads go through the metadata path

- **WHEN** Go code outside `opm/schema` reads a field of an artifact's `metadata` (the loaders' identity checks, the instance name for a diagnostic, the module's snake-case name)
- **THEN** it navigates to the metadata value through `schema.Metadata` and reads the field off that value, never through a dotted path literal rooted at the artifact

### Requirement: The render path reads no artifact sub-value in Go
The render path SHALL read no artifact sub-value in Go: the instance and the platform enter the render build by import, and the generated glue reads `components` and `#composedTransformers` in CUE.

#### Scenario: Render build reads artifacts by import

- **WHEN** `Kernel.Render` runs
- **THEN** no Go code looks up the instance's components or the platform's transformers by path; the staged render module imports both packages and the glue reads them

### Requirement: AcquireModuleFromDir loads a module package through the shape gate and constructs the typed module
The kernel SHALL expose `AcquireModuleFromDir(ctx, dir) (*module.Module, error)`, which loads a `#Module` CUE package from a directory through the acquisition shape gate, constructs the typed module (metadata decoded, `Package` set to the built value).

#### Scenario: Shape-gate failures propagate

- **WHEN** the directory does not exist, holds no CUE package, holds a package whose `kind` is not `"Module"`, or omits a required identity field
- **THEN** the error wraps the corresponding `opm/errors` sentinel (`ErrInvalidPackage`, `ErrWrongKind` or `ErrMissingRequiredField`) and no partial `*Module` is returned

### Requirement: An acquired module's Source has a Root and a Pkg
The kernel SHALL expose `AcquireModuleFromDir(ctx, dir) (*module.Module, error)`, which stamps `Source` in overlay mode: `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/module.cue`, the directory itself when it is the root, or the directory itself when no ancestor holds one), `Pkg` = the package directory relative to `Root` (empty for the root package).

#### Scenario: Acquired module carries an overlay source

- **WHEN** a caller invokes `AcquireModuleFromDir` on a module root directory holding a valid module package
- **THEN** the returned `*Module` has decoded `Metadata`, `Source.Root` equal to the directory's absolute path, an empty `Source.Pkg`, and `Source.Overlay` holding every `.cue` file under the directory and nothing else
- **AND** `HasSource()` reports true

### Requirement: An acquired module's Source Overlay holds every cue file under Root
The kernel SHALL expose `AcquireModuleFromDir(ctx, dir) (*module.Module, error)`, which stamps `Source` in overlay mode: `Overlay` = every `.cue` file under `Root` keyed by its absolute path, the module's own `cue.mod/module.cue` included.

#### Scenario: The overlay holds every cue file under Root

- **WHEN** `AcquireModuleFromDir` stamps `Source` in overlay mode
- **THEN** `Overlay` is every `.cue` file under `Root` keyed by its absolute path, the module's own `cue.mod/module.cue` included

### Requirement: An acquired module can be passed to SynthesizeInstance without further staging
The acquired module SHALL satisfy the synthesis precondition (`Module.HasSource()` true) so it can be passed to `SynthesizeInstance` without further staging.

#### Scenario: Acquired module synthesizes like a registry module

- **WHEN** the same module is acquired once from a directory and once from a registry serving the same published version, and each is passed to `SynthesizeInstance` with the same name, namespace and values
- **THEN** both synthesized instances render to the same set of objects

#### Scenario: Frontends stop staging module sources themselves

- **WHEN** a frontend renders from a module directory
- **THEN** it acquires the module with `AcquireModuleFromDir` and passes it to `SynthesizeInstance`
- **AND** it constructs no `module.Source` of its own

### Requirement: The registry mapping for a module's imports is the kernel's
The registry mapping used for the module's imports SHALL be the kernel's (`WithRegistry`), applied through the load configuration's environment and never through `os.Setenv`.

#### Scenario: The module's imports use the kernel's mapping

- **WHEN** `AcquireModuleFromDir` loads a module
- **THEN** the registry mapping used for the module's imports is the kernel's (`WithRegistry`), applied through the load configuration's environment and never through `os.Setenv`

### Requirement: Module acquisition writes nothing into the caller's directory
The kernel MUST NOT write into the caller's directory.

#### Scenario: Module acquisition writes nothing

- **WHEN** a caller invokes `AcquireModuleFromDir`
- **THEN** the kernel does not write into the caller's directory

### Requirement: A shape gate runs right after the package is built
Every kernel acquisition from a directory or a registry (`AcquireModuleFromDir`, `AcquireModuleFromRegistry`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`) SHALL run a shape gate immediately after the CUE package is built and before any typed artifact is constructed. The gate validates structural identity only; it SHALL NOT perform full schema validation of the artifact's configuration fields, which remains the contract of validation and rendering.

#### Scenario: The gate runs after the build and judges structure only

- **WHEN** an acquisition verb builds a CUE package
- **THEN** the shape gate runs immediately after the package is built and before any typed artifact is constructed, and validates structural identity only

### Requirement: The gate rejects a non-struct root, a wrong kind and more than one instance
The gate SHALL reject a package when any of the following hold, returning an error that wraps the corresponding sentinel declared in `opm/errors`:
- The built root value is not a struct, or the package load resolved other than exactly one instance: wraps `ErrInvalidPackage`.
- The concrete `kind` literal does not match the artifact the verb acquires (`"Module"`, `"ModuleInstance"`, `"Platform"`): wraps `ErrWrongKind`.

#### Scenario: Wrong artifact type rejected

- **WHEN** a caller invokes `AcquireModuleFromDir` on a directory containing a package whose `kind` is `"Platform"`
- **THEN** the call returns an error wrapping `oerrors.ErrWrongKind` and no artifact
- **AND** the error message names both the expected and the actual `kind`

#### Scenario: Non-struct root rejected

- **WHEN** a caller invokes any acquire verb on a package that evaluates to a scalar or list rather than a struct
- **THEN** the call returns an error wrapping `oerrors.ErrInvalidPackage`

#### Scenario: Conflicting package clauses rejected

- **WHEN** a directory contains two `.cue` files declaring different `package` names and a caller invokes any directory acquire verb on it
- **THEN** the call returns a non-nil error originating from the CUE loader

### Requirement: The gate rejects a package whose required identity field is absent or not concrete
The gate SHALL reject a package when any of the following hold, returning an error that wraps the corresponding sentinel declared in `opm/errors`:
- A required identity field is absent or not concrete: wraps `ErrMissingRequiredField`.

#### Scenario: Missing identity field rejected

- **WHEN** a caller invokes `AcquireModuleFromDir` on a module package that omits `metadata.name`
- **THEN** the call returns an error wrapping `oerrors.ErrMissingRequiredField`
- **AND** the error identifies the field path `metadata.name`

### Requirement: The required identity fields are those the schema never defaults
Required identity fields SHALL be those the schema never defaults:
- Module: `metadata.name`, `metadata.modulePath`, `metadata.version`.
- Instance: `metadata.name`, `metadata.namespace`, and `#module` present with `#module.kind == "Module"`.
- Platform: `metadata.name`, `type`, and every `#registry` entry complete under `cue.Concrete(true)`.

#### Scenario: Instance embedding a non-module rejected

- **WHEN** a caller invokes `AcquireInstanceFromDir` on an instance whose `#module` field carries a value whose `kind` is not `"Module"`
- **THEN** the call returns an error wrapping `oerrors.ErrWrongKind`

### Requirement: A registry entry with no embedded catalog fails as a missing required field
Core derives an entry's `version` from its embedded catalog's stamped `metadata.version` (enhancement 0019, "A platform imports its catalog and embeds it whole into the registry entry, which derives everything else"), so an entry with no embedded catalog, or one embedding an unstamped catalog, SHALL fail here as a missing required field naming the entry. This is the refusal `single-build-render` relies on for a subscription-shaped platform.

#### Scenario: Registry entry with no embedded catalog rejected

- **WHEN** a caller invokes `AcquirePlatformFromDir` on a platform whose `#registry` entry declares `enable` and a `version` scalar but embeds no `#catalog`
- **THEN** the call returns an error wrapping `oerrors.ErrMissingRequiredField` that names the entry and its `version` field
- **AND** the same entry with the catalog imported and embedded passes the gate

### Requirement: A defaulted identity field is not concrete for the gate
A required identity field declared as a disjunction with a default (for example `#VersionType | *"1.0.1"`) is NOT concrete for the purpose of this gate: the gate judges the value as authored, before default finalization. When the gate rejects such a field, the error SHALL name the default arm's value and state that identity fields must be concrete literals.

#### Scenario: Defaulted identity field rejected with the default named

- **WHEN** a caller invokes `AcquireModuleFromDir` on a module package that declares `metadata.version` as a disjunction with a default (for example `#VersionType | *"1.0.1"`) rather than a concrete literal
- **THEN** the call returns an error wrapping `oerrors.ErrMissingRequiredField`
- **AND** the error identifies the field path `metadata.version`, names the default value `"1.0.1"`, and states that identity fields must be concrete literals
- **AND** a concrete `metadata.version` referencing an identity package whose `Version` is a plain literal passes the gate

### Requirement: A directory-acquired and a registry-acquired artifact are gated identically
A directory-acquired artifact and a registry-acquired artifact SHALL be gated identically and fail with the same sentinel values.

#### Scenario: Registry and directory gate identically

- **WHEN** the same module content is acquired once from a directory and once from a registry, and its `kind` is not `"Module"`
- **THEN** both calls return an error wrapping the same `oerrors.ErrWrongKind` value

### Requirement: opm/errors exports the three gate sentinels
`opm/errors` SHALL export `ErrInvalidPackage`, `ErrWrongKind` and `ErrMissingRequiredField` so frontends can branch programmatically; no other package SHALL export a sentinel of the same meaning.

#### Scenario: Sentinels live in one package

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`, `opm/module`, `opm/platform` and `opm/helper/**`
- **THEN** none of them declares `ErrInvalidPackage`, `ErrWrongKind` or `ErrMissingRequiredField`; the three are declared in `opm/errors` only

### Requirement: WriteTo validates before it writes
`WriteTo` SHALL validate before it writes: a nil receiver, an on-disk source (`Overlay` nil, since `Root` already is the directory) and any entry whose path is not under `Root` SHALL be refused with a plain error and nothing written.

#### Scenario: WriteTo refuses before it writes

- **WHEN** `WriteTo` is called on a nil receiver, an on-disk source or a source with an entry whose path is not under `Root`
- **THEN** it is refused with a plain error and nothing is written

### Requirement: WriteTo is the library's one overlay writer
`WriteTo` SHALL be the library's one overlay writer, for frontends that need a fetched tree on disk; the kernel's render stage SHALL serve an overlay-mode input from memory and SHALL NOT write it.

#### Scenario: WriteTo is the one overlay writer

- **WHEN** a frontend needs a fetched tree on disk
- **THEN** it uses `WriteTo`, and the kernel's render stage serves an overlay-mode input from memory and does not write it

### Requirement: A directory acquire verb checks the directory and a missing path fails
Every directory acquire verb (`AcquireModuleFromDir`, `AcquireCatalogFromDir`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`) SHALL check the directory before reading anything from it. A path that does not exist SHALL fail with `accessing <label> directory "<absolute path>"` wrapping the stat error (so it matches `fs.ErrNotExist`).

#### Scenario: A missing directory fails before any tree read

- **WHEN** a caller invokes any of the four directory acquire verbs on a path that does not exist
- **THEN** the error reads `accessing <label> directory "<absolute path>": ...` with the verb's label and matches `fs.ErrNotExist`
- **AND** it does not read `reading module tree`, and no artifact is returned

### Requirement: A directory acquire verb refuses a path that is not a directory
Every directory acquire verb (`AcquireModuleFromDir`, `AcquireCatalogFromDir`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`) SHALL check the directory before reading anything from it. A path that is not a directory SHALL fail with `<label> path "<absolute path>" is not a directory`, where `<label>` is `module`, `catalog`, `platform` or `instance`.

#### Scenario: A file path is refused as not a directory

- **WHEN** a caller invokes any of the four directory acquire verbs on the path of a regular file
- **THEN** the error reads `<label> path "<absolute path>" is not a directory`, and no artifact is returned

### Requirement: An overlay-mode verb reads the cue files once and builds from that overlay
A verb that stamps its `Source` in overlay mode (`AcquireModuleFromDir`, `AcquireCatalogFromDir`, and `AcquireInstanceFromDir` with values sources) SHALL read the `.cue` files under the module root once, before the build, and SHALL build the package from that overlay, so `Package` and `Source.Overlay` come from the same read.

#### Scenario: Module and catalog are built from the overlay they carry

- **WHEN** a caller acquires a module with `AcquireModuleFromDir` or a catalog with `AcquireCatalogFromDir`
- **THEN** the package is built from the `.cue` bytes held in the returned `Source.Overlay`, read once before the build

### Requirement: A failed layered build is attributed from the overlay already read
When the layered instance build fails, the kernel SHALL attribute the failure to the values sources by building the authored package from the overlay already read.

#### Scenario: Values attribution reuses the authored read

- **WHEN** an instance acquired with values sources fails to build because a source conflicts with the package's values or the module's `#config`
- **THEN** the error is attributed to the source, as before

### Requirement: Platform acquisition and instance acquisition without values keep the on-disk mode
Platform acquisition and instance acquisition without values sources SHALL keep the on-disk mode their own requirements state.

#### Scenario: Platform acquisition keeps the on-disk mode

- **WHEN** `AcquirePlatformFromDir` or `AcquireInstanceFromDir` without values sources runs
- **THEN** it keeps the on-disk mode its own requirement states

### Requirement: A package that embeds a non-CUE file acquires from a directory
A package that embeds a non-CUE file stored beside it SHALL acquire from a directory as it does from disk, because files outside the overlay are read from the directory.

#### Scenario: A module embedding a non-CUE file acquires from a directory

- **WHEN** a module package uses the embed attribute to embed a JSON file stored beside it, and a caller invokes `AcquireModuleFromDir` on its directory
- **THEN** acquisition succeeds and `Package` carries the embedded data
- **AND** `Source.Overlay` still holds only the `.cue` files under the module root

### Requirement: Constructor helpers build module and platform artifacts from a cue.Value
The library SHALL provide constructor helpers that build the module and platform artifacts from a raw `cue.Value`: `module.NewModuleFromValue(v)` and `platform.NewPlatformFromValue(v)`. Each constructor SHALL take only the artifact value, decode `Metadata` from the value's `metadata` field, and set the `Package` field to the supplied `cue.Value` unmodified.

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

### Requirement: A constructed artifact carries no Source
The constructed artifact SHALL carry no `Source`.

#### Scenario: A constructed artifact has no Source

- **WHEN** a constructor builds an artifact from a bare `cue.Value`
- **THEN** the constructed artifact carries no `Source`

### Requirement: The kernel does not wrap the constructors
The kernel SHALL NOT wrap the constructors `module.NewModuleFromValue(v)` and `platform.NewPlatformFromValue(v)`: a frontend holding a value calls the package constructor directly; a frontend that wants a source-carrying artifact uses an acquire verb.

#### Scenario: No kernel constructor wrappers

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** neither `NewModuleFromValue` nor `NewPlatformFromValue` exists on it

### Requirement: No instance constructor is exported
No instance constructor SHALL be exported: an instance is produced only by the kernel's acquisition paths, which stamp its staged source.

#### Scenario: No instance constructor

- **WHEN** a consumer inspects the exported identifiers of `opm/module` and the exported methods of `Kernel`
- **THEN** `NewInstanceFromValue` does not exist in either
- **AND** the only ways to obtain a `*module.Instance` are `Kernel.AcquireInstanceFromDir` and `Kernel.SynthesizeInstance`

### Requirement: Instance exposes Components and Values
`*module.Instance` SHALL expose the following accessors:
- `Components()`: the instance's evaluated components value, definition fields included, read through `corepath.Components`.
- `Values()`: the instance's merged values as evaluated, read through `corepath.Values`.

#### Scenario: Components accessor

- **WHEN** a caller invokes `inst.Components()` on an acquired instance
- **THEN** the returned value is `inst.Package.LookupPath(corepath.Components)` with `#names`, `#resources`, `#traits` and `#blueprints` intact

#### Scenario: Values accessor

- **WHEN** a caller invokes `inst.Values()` on an acquired instance
- **THEN** the returned value is `inst.Package.LookupPath(corepath.Values)`, the instance's merged values as evaluated

### Requirement: Instance exposes ConfigSchema
`*module.Instance` SHALL expose the following accessors:
- `ConfigSchema()`: the embedded module's `#config`, read as `Package.LookupPath(schema.Module)` followed by `LookupPath(corepath.Config)`.

#### Scenario: Config schema reachable on a well-formed instance

- **WHEN** a caller invokes `inst.ConfigSchema()` on a `*Instance` whose `Package` carries an embedded `#module` with a `#config` definition
- **THEN** the returned `cue.Value` exists (`v.Exists() == true`)
- **AND** it is the value `inst.Package.LookupPath(schema.Module).LookupPath(corepath.Config)`

### Requirement: Instance exposes ModuleMetadata
`*module.Instance` SHALL expose the following accessors:
- `ModuleMetadata()`: the metadata of the module the instance was built from, decoded from `Package.LookupPath(schema.Module)` on each call.

#### Scenario: Module metadata of an acquired instance

- **WHEN** a caller invokes `inst.ModuleMetadata()` on an instance returned by `SynthesizeInstance` or `AcquireInstanceFromDir`
- **THEN** it returns a non-nil `*ModuleMetadata` whose `Name`, `ModulePath`, `Version`, `FQN` and `UUID` equal the embedded `#module`'s `metadata`

### Requirement: ConfigSchema and Values return the zero value rather than an error
`ConfigSchema()` and `Values()` SHALL return the zero `cue.Value` (not an error) when the receiver is `nil` or the field they read is absent. `ConfigSchema()` returns it as well when the instance carries no `#module` or the embedded module declares no `#config`.

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

#### Scenario: Values on a nil instance is the zero value

- **WHEN** a caller invokes `(*Instance)(nil).Values()`
- **THEN** the returned `cue.Value` is the zero value
- **AND** no panic occurs

### Requirement: ModuleMetadata returns nil when it cannot be read and decodes afresh
`ModuleMetadata()` SHALL return nil when the receiver is `nil`, when the instance carries no `#module`, or when the embedded module's metadata does not decode. Each call decodes afresh from `Package`; the returned struct is a copy, and mutating it does not change `Package`.

#### Scenario: Module metadata is nil when it cannot be read

- **WHEN** a caller invokes `ModuleMetadata()` on a nil `*Instance`, on an instance whose `Package` has no `#module`, or on one whose embedded module metadata does not decode
- **THEN** it returns nil
- **AND** no panic occurs and no error is raised

#### Scenario: Module metadata reads the package

- **WHEN** a caller builds an `Instance` struct literal whose `Package` carries an embedded `#module`
- **THEN** `ModuleMetadata()` returns that module's metadata, decoded from `Package`, with no processing step having run

### Requirement: No accessor consults a version or mirrors a metadata field
No accessor SHALL consult a version or binding. The instance SHALL expose no accessor that mirrors a single field of its own decoded `Metadata`, because `Metadata` is that projection.

#### Scenario: No metadata-mirroring accessors

- **WHEN** a developer inspects the exported methods of `*module.Instance`
- **THEN** none of `InstanceName`, `Namespace`, `InstanceUUID`, `InstanceFQN`, `ModuleVersion`, `Labels`, `Annotations`, `MatchComponents` exists

### Requirement: The render build reads none of the instance accessors
The render build SHALL read none of the accessors of `*module.Instance`: the transformer context is projected by core (enhancement 0019, "`#TransformerContext` becomes a projection of the other two inputs; the kernel fills `#runtimeName` alone"), and the glue reads the instance by import.

#### Scenario: The render build reads no accessor

- **WHEN** the render build reads an instance
- **THEN** it reads none of the accessors of `*module.Instance`: the transformer context is projected by core, and the glue reads the instance by import
