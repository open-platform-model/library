## MODIFIED Requirements

### Requirement: Constructor Helpers from cue.Value

The library SHALL provide constructor helpers that build the module and platform artifacts from a raw `cue.Value`: `module.NewModuleFromValue(v)` and `platform.NewPlatformFromValue(v)`. Each constructor SHALL take only the artifact value, decode `Metadata` from the value's `metadata` field, and set the `Package` field to the supplied `cue.Value` unmodified. No instance constructor SHALL be exported: an instance is produced only by the kernel's acquisition paths, which stamp its staged source.

#### Scenario: NewModuleFromValue success path

- **WHEN** a caller invokes `module.NewModuleFromValue(v)` with a `cue.Value` carrying a valid module
- **THEN** the returned `*Module` has `Metadata.Name` matching the value's `metadata.name`
- **AND** `Package` is the supplied `cue.Value` unchanged

#### Scenario: NewModuleFromValue with missing metadata

- **WHEN** a caller invokes `module.NewModuleFromValue(v)` with a `cue.Value` that has no `metadata` field
- **THEN** the function returns a non-nil error stating that the metadata field is required
- **AND** no partial `*Module` is returned

#### Scenario: NewModuleFromValue with unknown apiVersion

- **WHEN** a caller invokes `module.NewModuleFromValue(v)` with a `cue.Value` that carries no `apiVersion` field, or one the library has never heard of
- **THEN** the constructor performs no version detection and no binding lookup; the value is decoded as the single OPM schema the library consumes
- **AND** the result depends only on the `metadata` field being present and decodable

#### Scenario: NewInstanceFromValue success path

- **WHEN** a consumer inspects the exported identifiers of `opm/module` and the exported methods of `Kernel`
- **THEN** no `NewInstanceFromValue` exists in either
- **AND** the only ways to obtain a `*module.Instance` are `Kernel.AcquireInstanceFromDir` and `Kernel.SynthesizeInstance`

#### Scenario: NewPlatformFromValue hoists the platform type

- **WHEN** a caller invokes `platform.NewPlatformFromValue(v)` with a `#Platform` value whose root has `type: "kubernetes"` alongside its `metadata` block
- **THEN** the returned `*Platform` has `Metadata.Type == "kubernetes"` and `Package == v`

#### Scenario: Constructors take no context owner

- **WHEN** a consumer inspects the exported identifiers of `opm/module` and `opm/platform`
- **THEN** `NewModuleFromValue` and `NewPlatformFromValue` each take a single `cue.Value` argument
- **AND** no `CueContextOwner` interface exists in either package

#### Scenario: No instance constructor

- **WHEN** a consumer inspects the exported identifiers of `opm/module`
- **THEN** `NewInstanceFromValue` does not exist

### Requirement: Artifacts carry their staged source

`module.Source` SHALL describe the staged source tree an artifact was loaded or synthesized from, in two modes: an in-memory tree (`Overlay` non-empty, keyed under the synthetic absolute `Root`) or an on-disk tree (`Overlay` nil, `Root` a real directory). `module.Source` SHALL carry a `Pkg` field naming the package directory relative to `Root`; empty means the root package (`.`). `module.Instance` SHALL expose `Source *module.Source`; every instance the kernel returns carries a non-nil `Source`, since instances are constructed only by `AcquireInstanceFromDir` and `SynthesizeInstance`. `module.Module.Source` is nil for a module built from a bare value via `NewModuleFromValue`; the existing `Module.HasSource()` gate semantics SHALL be unchanged.

#### Scenario: Value-constructed instance has no source

- **WHEN** a consumer looks for a way to build an instance from a bare `cue.Value`
- **THEN** none exists: the instance constructor is not exported, so no instance without a `Source` can be created
- **AND** an instance obtained from `AcquireInstanceFromDir` or `SynthesizeInstance` has a non-nil `Source` with a non-empty `Root`

#### Scenario: Value-constructed module has no source

- **WHEN** a caller builds a module via `NewModuleFromValue`
- **THEN** the returned `*Module` has `Source == nil` and `HasSource()` reports false

#### Scenario: On-disk source mode

- **WHEN** an artifact's `Source` has `Overlay == nil` and a non-empty `Root`
- **THEN** consumers treat `Root` as a real filesystem directory holding the tree, and `Pkg` as the package directory within it

#### Scenario: Synth gate unchanged

- **WHEN** `synth.Instance` is invoked with a module whose `Source` is nil or overlay-empty
- **THEN** it still fails with `ErrMissingSource`, exactly as before this change

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
