# artifact-types Specification

## Purpose
TBD - created by syncing change unify-artifact-shape. Update Purpose after archive.
## Requirements

### Requirement: Uniform Artifact Shape

Every OPM artifact type accepted by the kernel SHALL be a Go struct with exactly three exported fields: `APIVersion apiversion.Version`, `Metadata *<Type>Metadata`, and `Package cue.Value`. The `Metadata` pointer holds a decoded ergonomic projection; the `Package` field carries the source-of-truth CUE value.

#### Scenario: Module type shape

- **WHEN** a developer reads the `module.Module` struct definition
- **THEN** the struct has exactly three exported fields: `APIVersion`, `Metadata` (typed `*ModuleMetadata`), and `Package` (typed `cue.Value`)
- **AND** there are no `Spec` or `Config` exported fields

#### Scenario: ModuleInstance type shape

- **WHEN** a developer reads the `module.Instance` struct definition
- **THEN** the struct has exactly three exported fields: `APIVersion`, `Metadata` (typed `*InstanceMetadata`), and `Package` (typed `cue.Value`)
- **AND** there are no `Module`, `Spec`, or `Values` exported fields

### Requirement: Constructor Helpers from cue.Value

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

#### Scenario: NewInstanceFromValue success path

- **WHEN** a consumer inspects the exported identifiers of `opm/module` and the exported methods of `Kernel`
- **THEN** no `NewInstanceFromValue` exists in either
- **AND** the only ways to obtain a `*module.Instance` are `Kernel.AcquireInstanceFromDir` and `Kernel.SynthesizeInstance`

#### Scenario: No instance constructor

- **WHEN** a consumer inspects the exported identifiers of `opm/module` and the exported methods of `Kernel`
- **THEN** `NewInstanceFromValue` does not exist in either
- **AND** the only ways to obtain a `*module.Instance` are `Kernel.AcquireInstanceFromDir` and `Kernel.SynthesizeInstance`

### Requirement: Package Is Source of Truth

When the typed `Metadata` field and the corresponding subtree of `Package` carry conflicting values, the `Package` value SHALL be authoritative. Documentation SHALL state that `Metadata` is an ergonomic cache, not a parallel source of truth.

#### Scenario: Documentation states authority

- **WHEN** a developer reads the godoc for `Module.Metadata` or `Instance.Metadata`
- **THEN** the doc comment states that `Package` is authoritative and `Metadata` is a decoded cache
- **AND** the doc comment warns against mutating `Package` after construction without re-running the constructor

### Requirement: APIVersion Field Stamped at Construction

The `APIVersion` field SHALL be set by the constructor based on detection of the `apiVersion` field in `Package`. The field SHALL NOT be settable directly through a public constructor argument.

#### Scenario: APIVersion matches Package

- **WHEN** a constructor returns a typed artifact
- **THEN** `artifact.APIVersion` equals the value extracted from `artifact.Package` via `apiversion.Detect`

### Requirement: Kernel Artifact Type Set

The kernel SHALL accept exactly three artifact types: `Module`, `ModuleInstance`, and `Platform`. `#ModuleDebug` SHALL NOT be a kernel artifact type. Debug values are carried as a `debugValues` field within `Module.Package`; whether they participate in the values stack is a frontend policy decision, not a kernel concern.

#### Scenario: No top-level ModuleDebug type

- **WHEN** a developer searches the kernel public API for `ModuleDebug`
- **THEN** no exported Go type with that name exists in any `opm/` package
- **AND** the version binding (`opm/api/<version>/`) exposes no `DecodeModuleDebugMetadata` or equivalent

#### Scenario: debugValues accessible via Module.Package

- **WHEN** a frontend reads debug overlays from a Module
- **THEN** the read goes through `Module.Package.LookupPath(binding.Paths().DebugValues)` (or directly through CUE if binding does not enumerate the path)
- **AND** the kernel never receives `debugValues` as a separate parameter

#### Scenario: Documentation explicitly retires the construct

- **WHEN** a developer reads `library/README.md` or `opm/module/` godoc
- **THEN** at least one prose section states that `#ModuleDebug` is not a kernel artifact and that debug overlays are a frontend layering concern

### Requirement: Instance Config Schema Accessor

`*module.Instance` SHALL expose a `ConfigSchema() cue.Value` accessor that returns the embedded source module's `#config` schema. The accessor SHALL look up the schema via the binding registered for `r.APIVersion` using `Paths().Module` followed by `Paths().Config`. The accessor SHALL return the zero `cue.Value` (not an error) when the binding is unregistered, the receiver is `nil`, or the path does not exist.

#### Scenario: Schema reachable on a well-formed instance

- **WHEN** a caller invokes `rel.ConfigSchema()` on a `*Instance` whose `Package` carries an embedded `#module` with a `#config` definition
- **THEN** the returned `cue.Value` exists (`v.Exists() == true`)
- **AND** the returned value is identical to `rel.Package.LookupPath(b.Paths().Module).LookupPath(b.Paths().Config)` where `b` is the binding for `rel.APIVersion`

#### Scenario: Zero value on unregistered binding

- **WHEN** a caller invokes `rel.ConfigSchema()` on a `*Instance` whose `APIVersion` has no registered binding
- **THEN** the returned `cue.Value` is the zero value (`v.Exists() == false`)
- **AND** no error is returned

#### Scenario: Zero value on missing #config path

- **WHEN** a caller invokes `rel.ConfigSchema()` on a `*Instance` whose embedded `#module` does not declare a `#config` definition
- **THEN** the returned `cue.Value` is the zero value (`v.Exists() == false`)

#### Scenario: Nil receiver safety

- **WHEN** a caller invokes `(*Instance)(nil).ConfigSchema()`
- **THEN** the returned `cue.Value` is the zero value
- **AND** no panic occurs

### Requirement: Artifacts carry their staged source

`module.Source` SHALL describe the staged source tree an artifact was loaded or synthesized from, in two modes: an in-memory tree (`Overlay` non-empty, keyed under the absolute `Root`, which may be a synthetic path or a real directory) or an on-disk tree (`Overlay` nil, `Root` a real directory). `Overlay` SHALL map each absolute file path to that file's contents as bytes. `module.Source` SHALL carry a `Pkg` field naming the package directory relative to `Root`; empty means the root package (`.`). `module.Instance` SHALL expose `Source *module.Source`; every instance the kernel returns carries a non-nil `Source`, since instances are constructed only by `AcquireInstanceFromDir` and `SynthesizeInstance`. `module.Module.Source` SHALL be populated in overlay mode by both `AcquireModuleFromRegistry` and `AcquireModuleFromDir`, and SHALL be nil for a module built from a bare value via `NewModuleFromValue`; the `Module.HasSource()` gate (non-nil `Source`, non-empty `Root`, non-empty `Overlay`) SHALL be unchanged.

#### Scenario: Overlay entries are bytes

- **WHEN** a consumer reads `Source.Overlay` on any acquired or synthesized artifact
- **THEN** each entry is the file's contents as a byte slice keyed by its absolute path under `Root`
- **AND** no entry is an opaque loader source type

#### Scenario: Value-constructed instance has no source

- **WHEN** a consumer looks for a way to build an instance from a bare `cue.Value`
- **THEN** none exists: the instance constructor is not exported, so no instance without a `Source` can be created
- **AND** an instance obtained from `AcquireInstanceFromDir` or `SynthesizeInstance` has a non-nil `Source` with a non-empty `Root`

#### Scenario: Value-constructed module has no source

- **WHEN** a caller builds a module via `NewModuleFromValue`
- **THEN** the returned `*Module` has `Source == nil` and `HasSource()` reports false

#### Scenario: Directory-acquired module has an overlay source

- **WHEN** a caller acquires a module via `AcquireModuleFromDir`
- **THEN** the returned `*Module` has a non-nil `Source` whose `Root` is a real directory and whose `Overlay` holds the module's `.cue` files, and `HasSource()` reports true

#### Scenario: On-disk source mode

- **WHEN** an artifact's `Source` has `Overlay == nil` and a non-empty `Root`
- **THEN** consumers treat `Root` as a real filesystem directory holding the tree, and `Pkg` as the package directory within it

#### Scenario: Synth gate unchanged

- **WHEN** `SynthesizeInstance` is invoked with a module whose `Source` is nil or overlay-empty
- **THEN** it fails with an error wrapping `oerrors.ErrMissingSource`

### Requirement: Instance acquisition from a directory

The kernel SHALL expose `AcquireInstanceFromDir(ctx, dir, values ...Source)`, which loads a `#ModuleInstance` CUE package from a directory through the acquisition shape gate, processes it through the kernel's instance processing (concreteness enforced, metadata decoded), and returns a `*module.Instance` whose `Source` describes the directory: `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/module.cue`, the directory itself when it is the root), and `Pkg` = the package directory relative to `Root` (empty for the root package). A directory with no enclosing module is its own root with an empty `Pkg`. The registry mapping used for the instance's imports SHALL be the kernel's (`WithRegistry`), applied through the load configuration's environment and never through `os.Setenv`.

With no values, `Source.Overlay` is nil (on-disk mode). With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts), the kernel SHALL unify them in order, render the result as a values file (`opm-values.cue`, a reserved name so it can never shadow a file the package authored) declaring the package's own package name and the top-level `values` field, place it beside the package's on-disk files in an overlay, and build the package in one pass through the instance shape gate, so the schema's own values unification performs the merge in CUE. The returned instance's `Source` SHALL then be overlay mode: same `Root` and `Pkg`, `Overlay` carrying every on-disk `.cue` file under the module root (the module's own `cue.mod/module.cue` included) plus the rendered values file, as bytes. The kernel MUST NOT write into the caller's directory and MUST NOT fill values into the evaluated value from Go.

#### Scenario: Acquired instance carries its source

- **WHEN** a caller invokes `AcquireInstanceFromDir(ctx, dir)` on a module root directory holding a valid, concrete instance package
- **THEN** the returned `*Instance` has decoded `Metadata`, `Source.Root` equal to the directory's absolute path, an empty `Source.Pkg`, and a nil `Overlay`

#### Scenario: Acquired subpackage names its module root

- **WHEN** a caller invokes `AcquireInstanceFromDir` on a subdirectory of a module (its `cue.mod/module.cue` lives in an ancestor)
- **THEN** the returned `*Instance` has `Source.Root` equal to the module root's absolute path and `Source.Pkg` equal to the subdirectory's slash-separated path relative to it

#### Scenario: Validation failures propagate

- **WHEN** the directory's instance package is not fully concrete
- **THEN** `AcquireInstanceFromDir` returns the concreteness error framed `instance "<name>": …`, and no partial `*Instance`

#### Scenario: Loader failures propagate

- **WHEN** the directory does not exist, holds no CUE package, or fails the instance shape gate
- **THEN** the error wraps the corresponding `opm/errors` sentinel (`ErrInvalidPackage`, `ErrWrongKind`, or `ErrMissingRequiredField`)

#### Scenario: Registry mapping is the kernel's

- **WHEN** a kernel constructed with `WithRegistry(mapping)` acquires an instance whose module imports a catalog under `opmodel.dev`
- **THEN** the import resolves through `mapping`
- **AND** the process environment is not mutated

#### Scenario: Extra values layer onto the package

- **WHEN** a caller invokes `AcquireInstanceFromDir(ctx, dir, a, b)` with two values sources on an instance package whose `values` leave a field unset
- **THEN** the returned instance's `values` carry the unified sources merged with the package's own values, `Source.Overlay` is non-nil and contains the on-disk `.cue` files plus the rendered values file, and the caller's directory is unchanged

#### Scenario: Layered instance renders

- **WHEN** an instance acquired with extra values is passed to `Kernel.Render`
- **THEN** the render build imports the layered package and the rendered objects reflect the extra values

#### Scenario: Conflicting extra values fail at acquisition

- **WHEN** an extra values source conflicts with the package's own values or its module's `#config` schema
- **THEN** `AcquireInstanceFromDir` returns the validation error naming the conflicting path, with source positions attributable to the values source, and no partial `*Instance`

#### Scenario: No option type for values

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`
- **THEN** neither `AcquireOption` nor `WithValues` exists; values are the variadic trailing argument

### Requirement: Internal call sites use schema paths

Every kernel-internal Go call site that reads a sub-value of an artifact's `Package` SHALL read it through the `opm/schema` path variables (`schema-dispatch`, "Path inventory exposed as package-level vars"), never through a removed struct field or an ad-hoc path literal. The render path reads no artifact sub-value in Go: the instance and the platform enter the render build by import, and the generated glue reads `components` and `#composedTransformers` in CUE.

#### Scenario: Render build reads artifacts by import

- **WHEN** `Kernel.Render` runs
- **THEN** no Go code looks up the instance's components or the platform's transformers by path; the staged render module imports both packages and the glue reads them

#### Scenario: Instance processing uses schema paths

- **WHEN** the kernel's internal instance processing decodes a built instance's metadata, or `Kernel.AcquireInstanceFromDir` with extra values checks the sources against the module's `#config`
- **THEN** the reads go through `schema.Metadata`, and `schema.Module` then `schema.Config`
- **AND** no Go code fills values into the evaluated instance; the merge happens in the build through the package's own `values` field

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

### Requirement: Module acquisition from a directory

The kernel SHALL expose `AcquireModuleFromDir(ctx, dir) (*module.Module, error)`, which loads a `#Module` CUE package from a directory through the acquisition shape gate, constructs the typed module (metadata decoded, `Package` set to the built value) and stamps `Source` in overlay mode: `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/module.cue`, the directory itself when it is the root, or the directory itself when no ancestor holds one), `Pkg` = the package directory relative to `Root` (empty for the root package), and `Overlay` = every `.cue` file under `Root` keyed by its absolute path, the module's own `cue.mod/module.cue` included. The acquired module SHALL satisfy the synthesis precondition (`Module.HasSource()` true) so it can be passed to `SynthesizeInstance` without further staging. The registry mapping used for the module's imports SHALL be the kernel's (`WithRegistry`), applied through the load configuration's environment and never through `os.Setenv`. The kernel MUST NOT write into the caller's directory.

#### Scenario: Acquired module carries an overlay source

- **WHEN** a caller invokes `AcquireModuleFromDir` on a module root directory holding a valid module package
- **THEN** the returned `*Module` has decoded `Metadata`, `Source.Root` equal to the directory's absolute path, an empty `Source.Pkg`, and `Source.Overlay` holding every `.cue` file under the directory and nothing else
- **AND** `HasSource()` reports true

#### Scenario: Acquired module synthesizes like a registry module

- **WHEN** the same module is acquired once from a directory and once from a registry serving the same published version, and each is passed to `SynthesizeInstance` with the same name, namespace and values
- **THEN** both synthesized instances render to the same set of objects

#### Scenario: Shape-gate failures propagate

- **WHEN** the directory does not exist, holds no CUE package, holds a package whose `kind` is not `"Module"`, or omits a required identity field
- **THEN** the error wraps the corresponding `opm/errors` sentinel (`ErrInvalidPackage`, `ErrWrongKind` or `ErrMissingRequiredField`) and no partial `*Module` is returned

#### Scenario: Frontends stop staging module sources themselves

- **WHEN** a frontend renders from a module directory
- **THEN** it acquires the module with `AcquireModuleFromDir` and passes it to `SynthesizeInstance`
- **AND** it constructs no `module.Source` of its own

### Requirement: Acquisition shape gate

Every kernel acquisition from a directory or a registry (`AcquireModuleFromDir`, `AcquireModuleFromRegistry`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`) SHALL run a shape gate immediately after the CUE package is built and before any typed artifact is constructed. The gate validates structural identity only; it SHALL NOT perform full schema validation of the artifact's configuration fields, which remains the contract of validation and rendering.

The gate SHALL reject a package when any of the following hold, returning an error that wraps the corresponding sentinel declared in `opm/errors`:

- The built root value is not a struct, or the package load resolved other than exactly one instance: wraps `ErrInvalidPackage`.
- The concrete `kind` literal does not match the artifact the verb acquires (`"Module"`, `"ModuleInstance"`, `"Platform"`): wraps `ErrWrongKind`.
- A required identity field is absent or not concrete: wraps `ErrMissingRequiredField`.

Required identity fields are those the schema never defaults:

- Module: `metadata.name`, `metadata.modulePath`, `metadata.version`.
- Instance: `metadata.name`, `metadata.namespace`, and `#module` present with `#module.kind == "Module"`.
- Platform: `metadata.name`, `type`, and every `#registry` entry complete under `cue.Concrete(true)`. Core derives an entry's `version` from its embedded catalog's stamped `metadata.version` (0019 D5), so an entry with no embedded catalog, or one embedding an unstamped catalog, fails here as a missing required field naming the entry. This is the refusal `single-build-render` relies on for a subscription-shaped platform.

A required identity field declared as a disjunction with a default (for example `#VersionType | *"1.0.1"`) is NOT concrete for the purpose of this gate: the gate judges the value as authored, before default finalization. When the gate rejects such a field, the error SHALL name the default arm's value and state that identity fields must be concrete literals.

A directory-acquired artifact and a registry-acquired artifact SHALL be gated identically and fail with the same sentinel values. `opm/errors` SHALL export `ErrInvalidPackage`, `ErrWrongKind` and `ErrMissingRequiredField` so frontends can branch programmatically; no other package SHALL export a sentinel of the same meaning.

#### Scenario: Wrong artifact type rejected

- **WHEN** a caller invokes `AcquireModuleFromDir` on a directory containing a package whose `kind` is `"Platform"`
- **THEN** the call returns an error wrapping `oerrors.ErrWrongKind` and no artifact
- **AND** the error message names both the expected and the actual `kind`

#### Scenario: Missing identity field rejected

- **WHEN** a caller invokes `AcquireModuleFromDir` on a module package that omits `metadata.name`
- **THEN** the call returns an error wrapping `oerrors.ErrMissingRequiredField`
- **AND** the error identifies the field path `metadata.name`

#### Scenario: Defaulted identity field rejected with the default named

- **WHEN** a caller invokes `AcquireModuleFromDir` on a module package that declares `metadata.version` as a disjunction with a default (for example `#VersionType | *"1.0.1"`) rather than a concrete literal
- **THEN** the call returns an error wrapping `oerrors.ErrMissingRequiredField`
- **AND** the error identifies the field path `metadata.version`, names the default value `"1.0.1"`, and states that identity fields must be concrete literals
- **AND** a concrete `metadata.version` referencing an identity package whose `Version` is a plain literal passes the gate

#### Scenario: Instance embedding a non-module rejected

- **WHEN** a caller invokes `AcquireInstanceFromDir` on an instance whose `#module` field carries a value whose `kind` is not `"Module"`
- **THEN** the call returns an error wrapping `oerrors.ErrWrongKind`

#### Scenario: Registry entry with no embedded catalog rejected

- **WHEN** a caller invokes `AcquirePlatformFromDir` on a platform whose `#registry` entry declares `enable` and a `version` scalar but embeds no `#catalog`
- **THEN** the call returns an error wrapping `oerrors.ErrMissingRequiredField` that names the entry and its `version` field
- **AND** the same entry with the catalog imported and embedded passes the gate

#### Scenario: Non-struct root rejected

- **WHEN** a caller invokes any acquire verb on a package that evaluates to a scalar or list rather than a struct
- **THEN** the call returns an error wrapping `oerrors.ErrInvalidPackage`

#### Scenario: Conflicting package clauses rejected

- **WHEN** a directory contains two `.cue` files declaring different `package` names and a caller invokes any directory acquire verb on it
- **THEN** the call returns a non-nil error originating from the CUE loader

#### Scenario: Registry and directory gate identically

- **WHEN** the same module content is acquired once from a directory and once from a registry, and its `kind` is not `"Module"`
- **THEN** both calls return an error wrapping the same `oerrors.ErrWrongKind` value

#### Scenario: Sentinels live in one package

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`, `opm/module`, `opm/platform` and `opm/helper/**`
- **THEN** none of them declares `ErrInvalidPackage`, `ErrWrongKind` or `ErrMissingRequiredField`; the three are declared in `opm/errors` only

### Requirement: An overlay source can be written to a directory

`module.Source` SHALL expose `WriteTo(dir string) ([]string, error)`, which writes every entry of an overlay-mode source under `dir` at the entry's path relative to `Root`, creating parent directories as needed, and returns the dir-relative paths it wrote, sorted. The method SHALL validate before it writes: a nil receiver, an on-disk source (`Overlay` nil, since `Root` already is the directory) and any entry whose path is not under `Root` SHALL be refused with a plain error and nothing written. `WriteTo` SHALL be the library's one overlay writer, for frontends that need a fetched tree on disk; the kernel's render stage SHALL serve an overlay-mode input from memory and SHALL NOT write it.

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
