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

The library SHALL provide constructor helpers that build each typed artifact from a raw `cue.Value`. Each constructor SHALL:

1. Detect `apiVersion` via `apiversion.Detect`.
2. Look up the binding for that version.
3. Decode `Metadata` using the binding's metadata decoder.
4. Stamp the `APIVersion` field on the returned struct from the detected version.
5. Set the `Package` field to the supplied `cue.Value` unmodified.

#### Scenario: NewModuleFromValue success path

- **WHEN** a caller invokes `module.NewModuleFromValue(k, v)` with a `cue.Value` carrying a valid v1alpha2 module
- **THEN** the returned `*Module` has `APIVersion == apiversion.V1alpha2`
- **AND** `Metadata.Name` matches the value's `metadata.name`
- **AND** `Package` is the supplied `cue.Value` unchanged

#### Scenario: NewModuleFromValue with unknown apiVersion

- **WHEN** a caller invokes `module.NewModuleFromValue(k, v)` with a `cue.Value` whose `apiVersion` field is not registered
- **THEN** the function returns a non-nil error wrapping `apiversion.ErrUnknownAPIVersion`
- **AND** no partial `*Module` is returned

#### Scenario: NewInstanceFromValue success path

- **WHEN** a caller invokes `module.NewInstanceFromValue(k, v)` with a `cue.Value` carrying a valid instance
- **THEN** the returned `*Instance` has `APIVersion`, `Metadata`, and `Package` populated
- **AND** the instance's referenced module is reachable via `Package.LookupPath(binding.Paths().Module)`

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

`module.Source` SHALL describe the staged source tree an artifact was loaded or synthesized from, in two modes: an in-memory tree (`Overlay` non-empty, keyed under the synthetic absolute `Root`) or an on-disk tree (`Overlay` nil, `Root` a real directory). `module.Source` SHALL carry a `Pkg` field naming the package directory relative to `Root`; empty means the root package (`.`). `module.Instance` SHALL expose `Source *module.Source`, nil when the instance was constructed from a bare `cue.Value`. The existing `Module.HasSource()` gate semantics SHALL be unchanged.

#### Scenario: Value-constructed instance has no source

- **WHEN** a caller builds an instance via `NewInstanceFromValue`
- **THEN** the returned `*Instance` has `Source == nil`

#### Scenario: On-disk source mode

- **WHEN** an artifact's `Source` has `Overlay == nil` and a non-empty `Root`
- **THEN** consumers treat `Root` as a real filesystem directory holding the tree, and `Pkg` as the package directory within it

#### Scenario: Synth gate unchanged

- **WHEN** `synth.Instance` is invoked with a module whose `Source` is nil or overlay-empty
- **THEN** it still fails with `ErrMissingSource`, exactly as before this change

### Requirement: Instance acquisition from a directory

The kernel SHALL expose `AcquireInstanceFromDir`, which loads a `#ModuleInstance` CUE package from a directory through the existing shape-gated loader path, processes it through the validated entry point (concreteness enforced, metadata decoded), and returns a `*module.Instance` whose `Source` describes the directory: `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/module.cue`, the directory itself when it is the root), and `Pkg` = the package directory relative to `Root` (empty for the root package). A directory with no enclosing module is its own root with an empty `Pkg`. This is what lets a package in a subdirectory of its module be imported correctly by a follow-on build such as `Kernel.Render`.

With no option, no extra values are supplied and `Source.Overlay` is nil (on-disk mode). With the extra-values option, the caller supplies one or more values sources (the `Source` type `ValidateConfigDetailed` accepts); the kernel SHALL unify them, render the result as a values file (`opm-values.cue`, a reserved name so it can never shadow a file the package authored) declaring the package's own package name and the top-level `values` field, place it beside the package's on-disk files in an overlay, and build the package in one pass through the instance shape gate, so the schema's own values unification performs the merge in CUE. The returned instance's `Source` SHALL then be overlay mode: same `Root` and `Pkg`, `Overlay` carrying every on-disk `.cue` file under the module root (the files `cue/load` reads; the module's own `cue.mod/module.cue` included) plus the rendered values file, exactly as `load.Config.Overlay` expects, so `Kernel.Render` imports the layered package by source. The kernel MUST NOT write into the caller's directory and MUST NOT fill values into the evaluated value from Go.

#### Scenario: Acquired instance carries its source

- **WHEN** a caller invokes `AcquireInstanceFromDir` on a module root directory holding a valid, concrete instance package
- **THEN** the returned `*Instance` has decoded `Metadata`, a `Package` identical to what `LoadInstancePackage` returns for that directory, `Source.Root` equal to the directory's absolute path, an empty `Source.Pkg`, and a nil `Overlay`

#### Scenario: Acquired subpackage names its module root

- **WHEN** a caller invokes `AcquireInstanceFromDir` on a subdirectory of a module (its `cue.mod/module.cue` lives in an ancestor)
- **THEN** the returned `*Instance` has `Source.Root` equal to the module root's absolute path and `Source.Pkg` equal to the subdirectory's slash-separated path relative to it

#### Scenario: Validation failures propagate

- **WHEN** the directory's instance package is not fully concrete
- **THEN** `AcquireInstanceFromDir` returns the same validation error the validated entry point produces, and no partial `*Instance`

#### Scenario: Loader failures propagate

- **WHEN** the directory does not exist, holds no CUE package, or fails the instance shape gate
- **THEN** the error wraps the same sentinel the file loader reports today (`ErrInvalidPackage`, `ErrWrongKind`, or `ErrMissingRequiredField`)

#### Scenario: Extra values layer onto the package

- **WHEN** a caller invokes `AcquireInstanceFromDir` with two values sources on an instance package whose `values` leave a field unset
- **THEN** the returned instance's `values` carry the unified sources merged with the package's own values, `Source.Overlay` is non-nil and contains the on-disk `.cue` files plus the rendered values file, and the caller's directory is unchanged

#### Scenario: Layered instance renders

- **WHEN** an instance acquired with extra values is passed to `Kernel.Render`
- **THEN** the render build imports the layered package and the rendered objects reflect the extra values

#### Scenario: Conflicting extra values fail at acquisition

- **WHEN** an extra values source conflicts with the package's own values or its module's `#config` schema
- **THEN** `AcquireInstanceFromDir` returns the validation error naming the conflicting path, with source positions attributable to the values source (a conflict with the package's own values is re-validated as layered validation does; a `#config` violation is checked on the sources after the build), and no partial `*Instance`

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
