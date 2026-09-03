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

### Requirement: Internal Call Sites Use Binding Paths

All kernel-internal call sites that previously read `Module.Spec`, `Module.Config`, `Instance.Spec`, `Instance.Values`, or `Instance.Module` SHALL be migrated to read sub-values from `Package` using `binding.Paths()` from the version binding.

#### Scenario: Compile pipeline uses binding paths

- **WHEN** the compile pipeline (`opm/compile/`) reads the components subtree of a Module
- **THEN** the read goes through `mod.Package.LookupPath(binding.Paths().Components)`
- **AND** there is no direct dereference of a removed field

#### Scenario: Validate pipeline uses binding paths

- **WHEN** the validate pipeline (`opm/validate/`) reads the `#config` schema of a Module
- **THEN** the read goes through `mod.Package.LookupPath(binding.Paths().Config)`

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

### Requirement: Instance-Side Module Paths On Binding

The `api.Paths` inventory SHALL expose two CUE paths for instance-side lookup of the embedded source module:

- `Paths.Module` — the path under which the instance's CUE package carries its `#Module` reference (v1alpha2: `#module`).
- `Paths.ModuleMetadata` — the path under which the instance's CUE package carries the projected module metadata (v1alpha2: `#moduleMetadata`).

These paths SHALL be populated by every concrete binding so that kernel-internal call sites and `*Instance` accessor methods can read module identity from `Instance.Package` without ad-hoc path strings. The accessor set reading `Paths.ModuleMetadata` SHALL be `ModuleVersion()` alone; `Instance` SHALL NOT expose a `ModuleFQN()` accessor (no production code or consumer read it, and the transformer context carries the instance's own FQN, not the source module's).

#### Scenario: Instance reaches its source module via the binding

- **WHEN** a caller holds a `*Instance` whose `Package` carries a `#module` field
- **THEN** `rel.Package.LookupPath(b.Paths().Module).Exists()` is true (where `b` is the binding for `rel.APIVersion`)

#### Scenario: Instance accessors read module metadata via the binding

- **WHEN** a caller invokes `rel.ModuleVersion()`
- **THEN** the returned value is read from `rel.Package.LookupPath(b.Paths().ModuleMetadata)` via `api.Lookup(rel.APIVersion)`
- **AND** there is no cached `*Module` field on `Instance` carrying the same data

#### Scenario: ModuleFQN accessor removed

- **WHEN** a developer searches `opm/module` for `ModuleFQN`
- **THEN** no such accessor exists on `Instance`; the source module's identity remains reachable through `Instance.Package` at the module-metadata path

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

The kernel SHALL expose `AcquireInstanceFromDir`, which loads a `#ModuleInstance` CUE package from a directory through the existing shape-gated loader path, processes it through the validated entry point (concreteness enforced, metadata decoded, no extra values supplied), and returns a `*module.Instance` whose `Source` describes the directory in on-disk mode: `Overlay` nil, `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/module.cue`, the directory itself when it is the root), and `Pkg` = the package directory relative to `Root` (empty for the root package). A directory with no enclosing module is its own root with an empty `Pkg`. This is what lets a package in a subdirectory of its module be imported correctly by a follow-on build such as `Kernel.Render`.

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

