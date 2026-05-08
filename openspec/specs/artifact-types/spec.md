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

#### Scenario: ModuleRelease type shape

- **WHEN** a developer reads the `module.Release` struct definition
- **THEN** the struct has exactly three exported fields: `APIVersion`, `Metadata` (typed `*ReleaseMetadata`), and `Package` (typed `cue.Value`)
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

#### Scenario: NewReleaseFromValue success path

- **WHEN** a caller invokes `module.NewReleaseFromValue(k, v)` with a `cue.Value` carrying a valid release
- **THEN** the returned `*Release` has `APIVersion`, `Metadata`, and `Package` populated
- **AND** the release's referenced module is reachable via `Package.LookupPath(binding.Paths().Module)`

### Requirement: Package Is Source of Truth

When the typed `Metadata` field and the corresponding subtree of `Package` carry conflicting values, the `Package` value SHALL be authoritative. Documentation SHALL state that `Metadata` is an ergonomic cache, not a parallel source of truth.

#### Scenario: Documentation states authority

- **WHEN** a developer reads the godoc for `Module.Metadata` or `Release.Metadata`
- **THEN** the doc comment states that `Package` is authoritative and `Metadata` is a decoded cache
- **AND** the doc comment warns against mutating `Package` after construction without re-running the constructor

### Requirement: APIVersion Field Stamped at Construction

The `APIVersion` field SHALL be set by the constructor based on detection of the `apiVersion` field in `Package`. The field SHALL NOT be settable directly through a public constructor argument.

#### Scenario: APIVersion matches Package

- **WHEN** a constructor returns a typed artifact
- **THEN** `artifact.APIVersion` equals the value extracted from `artifact.Package` via `apiversion.Detect`

### Requirement: Release-Side Module Paths On Binding

The `api.Paths` inventory SHALL expose two CUE paths for release-side lookup of the embedded source module:

- `Paths.Module` — the path under which the release's CUE package carries its `#Module` reference (v1alpha2: `#module`).
- `Paths.ModuleMetadata` — the path under which the release's CUE package carries the projected module metadata (v1alpha2: `#moduleMetadata`).

These paths SHALL be populated by every concrete binding so that kernel-internal call sites and `*Release` accessor methods can read module identity from `Release.Package` without ad-hoc path strings.

#### Scenario: Release reaches its source module via the binding

- **WHEN** a caller holds a `*Release` whose `Package` carries a `#module` field
- **THEN** `rel.Package.LookupPath(b.Paths().Module).Exists()` is true (where `b` is the binding for `rel.APIVersion`)

#### Scenario: Release accessors read module metadata via the binding

- **WHEN** a caller invokes `rel.ModuleFQN()` or `rel.ModuleVersion()`
- **THEN** the returned value is read from `rel.Package.LookupPath(b.Paths().ModuleMetadata)` via `api.Lookup(rel.APIVersion)`
- **AND** there is no cached `*Module` field on `Release` carrying the same data

### Requirement: Internal Call Sites Use Binding Paths

All kernel-internal call sites that previously read `Module.Spec`, `Module.Config`, `Release.Spec`, `Release.Values`, or `Release.Module` SHALL be migrated to read sub-values from `Package` using `binding.Paths()` from the version binding.

#### Scenario: Render pipeline uses binding paths

- **WHEN** the render pipeline (`pkg/render/`) reads the components subtree of a Module
- **THEN** the read goes through `mod.Package.LookupPath(binding.Paths().Components)`
- **AND** there is no direct dereference of a removed field

#### Scenario: Validate pipeline uses binding paths

- **WHEN** the validate pipeline (`pkg/validate/`) reads the `#config` schema of a Module
- **THEN** the read goes through `mod.Package.LookupPath(binding.Paths().Config)`

### Requirement: Kernel Artifact Type Set

The kernel SHALL accept exactly three artifact types: `Module`, `ModuleRelease`, and `Platform`. `#ModuleDebug` SHALL NOT be a kernel artifact type. Debug values are carried as a `debugValues` field within `Module.Package`; whether they participate in the values stack is a frontend policy decision, not a kernel concern.

#### Scenario: No top-level ModuleDebug type

- **WHEN** a developer searches the kernel public API for `ModuleDebug`
- **THEN** no exported Go type with that name exists in any `pkg/` package
- **AND** the version binding (`pkg/api/<version>/`) exposes no `DecodeModuleDebugMetadata` or equivalent

#### Scenario: debugValues accessible via Module.Package

- **WHEN** a frontend reads debug overlays from a Module
- **THEN** the read goes through `Module.Package.LookupPath(binding.Paths().DebugValues)` (or directly through CUE if binding does not enumerate the path)
- **AND** the kernel never receives `debugValues` as a separate parameter

#### Scenario: Documentation explicitly retires the construct

- **WHEN** a developer reads `library/README.md` or `pkg/module/` godoc
- **THEN** at least one prose section states that `#ModuleDebug` is not a kernel artifact and that debug overlays are a frontend layering concern
