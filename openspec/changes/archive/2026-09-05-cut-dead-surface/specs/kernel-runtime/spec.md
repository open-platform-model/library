## MODIFIED Requirements

### Requirement: Backward-Compatible Method Wrappers

For every exported package loader in `opm/helper/loader/file/` (`LoadModulePackage`, `LoadInstancePackage`, `LoadPlatformPackage`) and for the module and platform constructors in `opm/module/` and `opm/platform/` (`NewModuleFromValue`, `NewPlatformFromValue`), the Kernel SHALL provide a method wrapper that sources `*cue.Context` from itself where the underlying function takes one. The constructors SHALL take only the artifact value: no context-owner argument and no interface describing one exists in `opm/module` or `opm/platform`. The Kernel SHALL NOT wrap functions whose canonical implementation lives on the Kernel itself (layered validation, instance processing, and the values-file source loader); those are direct kernel methods, not wrappers. The Kernel SHALL NOT expose an instance constructor wrapper: instances are constructed only by `AcquireInstanceFromDir` and `SynthesizeInstance`.

#### Scenario: Loader method wrapper for module packages

- **WHEN** a caller invokes `k.LoadModulePackage(ctx, "./module", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the result is identical to calling `helper/loader/file.LoadModulePackage(k.CueContext(), "./module", loaderfile.LoadOptions{Registry: "..."})`
- **AND** any error returned is the same instance the underlying free function would return

#### Scenario: Loader method wrapper for instance packages

- **WHEN** a caller invokes `k.LoadInstancePackage(ctx, "./instance", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the result is identical to calling `helper/loader/file.LoadInstancePackage(k.CueContext(), "./instance", loaderfile.LoadOptions{Registry: "..."})`
- **AND** any error returned is the same instance the underlying free function would return

#### Scenario: Constructors take only the value

- **WHEN** a caller invokes `module.NewModuleFromValue(v)` or `platform.NewPlatformFromValue(v)`
- **THEN** the constructor decodes metadata from `v` and returns the typed artifact with `Package == v`
- **AND** `k.NewModuleFromValue(v)` and `k.NewPlatformFromValue(v)` return the same result
- **AND** neither `opm/module` nor `opm/platform` exports a `CueContextOwner` interface

#### Scenario: No instance constructor is exported

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel` and `opm/module`
- **THEN** neither `Kernel.NewInstanceFromValue` nor `module.NewInstanceFromValue` exists

#### Scenario: Helper-shaped functions remain callable

- **WHEN** existing downstream code calls `helper/loader/file.LoadModulePackage(cueCtx, dir, opts)` or `helper/loader/file.LoadInstancePackage(cueCtx, dir, opts)` directly
- **THEN** the call succeeds with the documented behavior
- **AND** the helper signatures continue to accept `*cue.Context` so non-kernel consumers can use them without importing `opm/kernel`

#### Scenario: Validation methods are not wrappers

- **WHEN** a developer reads `opm/kernel/validate.go`
- **THEN** the file contains the canonical implementation of `ValidateConfigDetailed` directly, with no `//nolint:staticcheck // SA1019:` exemptions for delegating to deleted helper packages

#### Scenario: ValidateAndUnify wrapper is gone

- **WHEN** a developer searches `opm/kernel/wrappers.go` (or the entire `opm/kernel/`) for `ValidateAndUnify`
- **THEN** no exported method or function with that name exists
- **AND** callers MUST use `k.ValidateConfigDetailed`

### Requirement: Single Pre-Unified Values Input

User values SHALL reach the kernel through exactly three public inputs, each carrying values as `Source` values or a single pre-unified `cue.Value`: `ValidateConfigDetailed` accepts `[]Source` for layered validation; `AcquireInstanceFromDir` accepts `WithValues(sources ...Source)` and unifies the sources inside the package build; `SynthesizeInstance` accepts `synth.InstanceInput.Values`, a single `cue.Value`, and renders it into the synthesized package. The kernel SHALL NOT accept `[]cue.Value` as a values argument on any public method, and SHALL NOT expose a method that fills values into an already-built instance from Go.

#### Scenario: ValidateConfig takes a single value

- **WHEN** a caller holds a single pre-unified `cue.Value` to validate
- **THEN** it passes it as a one-element `[]Source` to `k.ValidateConfigDetailed`; no `ValidateConfig` method exists
- **AND** there is no internal merge loop for a single source; the value is consumed as-is

#### Scenario: ProcessModuleInstance takes a single value

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ProcessModuleInstance` does not exist
- **AND** values reach an instance only through `WithValues` on `AcquireInstanceFromDir` or `synth.InstanceInput.Values` on `SynthesizeInstance`

#### Scenario: ValidateConfigDetailed takes a slice of Source

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources)` with `sources` as `[]Source`
- **THEN** the method unifies the sources in order then validates the merged value against `schema`
- **AND** this is the only public method that accepts a multi-value input as a slice

#### Scenario: Extra values enter the instance build

- **WHEN** a caller invokes `k.AcquireInstanceFromDir(ctx, dir, opts, WithValues(a, b))`
- **THEN** the sources are unified and rendered into the package overlay so the schema's own values unification performs the merge in CUE
- **AND** no Go code fills the merged value into the evaluated instance

#### Scenario: Synthesized values are a single value

- **WHEN** a caller invokes `k.SynthesizeInstance(ctx, in)` with `in.Values` as a single `cue.Value`
- **THEN** the value is rendered into the synthesized package's values source and participates in the single build
- **AND** the method does not accept a slice form

#### Scenario: Empty values is the zero value

- **WHEN** a caller passes an empty `[]Source` to `k.ValidateConfigDetailed`, no `WithValues` option to `k.AcquireInstanceFromDir`, or a zero `cue.Value{}` as `in.Values` to `k.SynthesizeInstance`
- **THEN** the call succeeds with no values applied (the package or synthesized spec must already be concrete on every required field)
- **AND** the behavior is documented as "no values supplied"

### Requirement: Canonical Implementations Live on Kernel

The canonical Go implementation of layered values validation and of instance processing (concreteness assertion and metadata decoding on a built instance spec) SHALL live on the `*Kernel` receiver in `opm/kernel/`. Instance processing SHALL be kernel-internal: it is reached only through `AcquireInstanceFromDir` and `SynthesizeInstance` and is not an exported method. No standalone `validate.Config` / `validate.ConfigPartial` / `module.ParseModuleInstance` free functions SHALL remain in the library; the `opm/validate/` and `opm/helper/values/` packages SHALL NOT exist.

#### Scenario: ValidateConfigDetailed is a kernel method

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources)`
- **THEN** the method unifies the sources in order, then validates the merged value with concreteness enforced and returns the merged `cue.Value` plus a CUE-native error
- **AND** no `opm/validate/` or `opm/helper/values/` import is required by callers

#### Scenario: ValidateConfig is a kernel method

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ValidateConfig` does not exist; `ValidateConfigDetailed` is the one validation method and runs the full Tier-2 schema validation directly

#### Scenario: ValidateConfigPartial is a kernel method

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ValidateConfigPartial` does not exist
- **AND** the partial-validation pass (type errors, disallowed fields and pattern violations on fields that are set, missing fields not flagged) is an unexported kernel internal used by `AcquireInstanceFromDir` to attribute extra-values errors to their sources

#### Scenario: ProcessModuleInstance is a kernel method

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ProcessModuleInstance` does not exist; instance processing is an unexported kernel internal
- **AND** both `AcquireInstanceFromDir` and `SynthesizeInstance` assert concreteness on the built spec via `spec.Validate(cue.Concrete(true))` (CUE stdlib), decode instance metadata, and return a `*module.Instance`

#### Scenario: opm/validate package is gone

- **WHEN** a developer runs `ls opm/validate/`
- **THEN** the directory does not exist

#### Scenario: opm/helper/values package is gone

- **WHEN** a developer runs `ls opm/helper/values/`
- **THEN** the directory does not exist

#### Scenario: module.ParseModuleInstance free function is gone

- **WHEN** a developer searches `opm/` for `ParseModuleInstance`
- **THEN** no free function and no method with that name exists

### Requirement: Kernel.SynthesizeInstance method

The `*Kernel` type SHALL expose a method `SynthesizeInstance(ctx context.Context, in synth.InstanceInput) (*module.Instance, error)` that combines `synth.Instance` with the kernel's internal instance processing in a single call: it builds the instance spec by single-build CUE evaluation inside the module's staged source with the supplied values rendered into the package, then asserts concreteness on the built spec, decodes instance metadata, and returns the constructed `*module.Instance` with its `Source` populated.

The method SHALL use the Kernel's owned `*cue.Context` when calling `synth.Instance`. The method SHALL NOT consult any additional values source: `in.Values` is the only values input.

#### Scenario: SynthesizeInstance produces an end-to-end instance

- **WHEN** `k.SynthesizeInstance(ctx, synth.InstanceInput{Module: mod, Name: "demo", Namespace: "default", Values: concreteValues})` is called against an acquired module
- **THEN** the returned `*module.Instance` is non-nil
- **AND** `Instance.Metadata.Name` equals `"demo"`, `Instance.Metadata.Namespace` equals `"default"`
- **AND** `Instance.Metadata.UUID` equals `uuid.SHA1(OPMNamespace, "<module.uuid>:demo:default")`
- **AND** `Instance.Source` is non-nil

#### Scenario: SynthesizeInstance rejects unconcrete result

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with `in.Values == cue.Value{}` against a module whose `#config` has required fields with no defaults
- **THEN** the returned error is non-nil, is framed `instance "<name>": …`, and wraps the concreteness diagnostic

#### Scenario: SynthesizeInstance surfaces synth errors before validation

- **WHEN** `k.SynthesizeInstance(ctx, synth.InstanceInput{Module: nil, Name: "x", Namespace: "y"})` is called
- **THEN** the returned error is non-nil and originates from `synth.Instance` (not from the concreteness check)

#### Scenario: SynthesizeInstance uses the Kernel's cue.Context

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called
- **AND** a developer inspects the cue.Context underlying the returned `Instance.Package`
- **THEN** that context is the same instance returned by `k.CueContext()`

### Requirement: SynthesizeInstance is documented as the recommended in-memory entry point

The package documentation and the `Kernel.SynthesizeInstance` godoc SHALL state that `SynthesizeInstance` is the recommended entry point for building an instance from typed inputs, mirroring how `Kernel.AcquireInstanceFromDir` (over `Kernel.LoadInstancePackage`) is the entry point for building an instance from a directory-based CUE package. The documentation SHALL NOT present a helper-level composition that reaches instance processing directly, since instance processing is not exported.

#### Scenario: Documentation directs callers to the kernel method

- **WHEN** a developer reads the godoc on `opm/helper/synth/`
- **THEN** the documentation states that `Kernel.SynthesizeInstance` is the recommended entry point
- **AND** notes that direct use of `synth.Instance` yields a built value and staged source only, without instance processing

#### Scenario: SynthesizeInstance godoc points to LoadInstancePackage

- **WHEN** a developer reads the `Kernel.SynthesizeInstance` godoc
- **THEN** the file-driven mirror it names is `Kernel.AcquireInstanceFromDir`, the validated entry over `Kernel.LoadInstancePackage`
- **AND** no reference to `Kernel.ProcessModuleInstance` or the removed `Kernel.LoadInstanceFile` remains

### Requirement: Tier-2 validation runs where values are applied

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema at the point they are applied to the instance, regardless of whether a Tier-1 helper validated them upstream. Values are applied inside a CUE build: `AcquireInstanceFromDir` with extra values renders the sources into the package overlay and additionally checks the sources against `#config` so a violation is reported at the sources' own positions; `SynthesizeInstance` renders the values into the synthesized package. Both then assert concreteness on the whole built spec. `Kernel.Render` SHALL NOT perform a second validation pass: the render build imports the instance as processed, which is already concrete.

#### Scenario: Kernel re-validates after Detailed

- **WHEN** a frontend validates sources with `k.ValidateConfigDetailed` and then supplies the same sources via `WithValues` to `k.AcquireInstanceFromDir`
- **THEN** the kernel validates them again where they are applied: the build unifies them with the package's `values`, and the sources are checked against `#config` at their own positions
- **AND** any schema violation produces a CUE-native error walkable via `cueerrors.Errors`, wrapped with the instance name, whose positions name the originating source's `Origin` rather than the rendered overlay file

#### Scenario: Kernel validates without Detailed

- **WHEN** a frontend skips `ValidateConfigDetailed` and supplies raw values via `WithValues` or `synth.InstanceInput.Values` directly
- **THEN** the kernel still produces correct schema-validation errors from the build
- **AND** for `SynthesizeInstance` the only loss is per-source attribution in error positions (`pos.Filename()` names the rendered values source)

#### Scenario: Render does not re-validate

- **WHEN** a caller invokes `k.Render` with an instance returned by `AcquireInstanceFromDir` or `SynthesizeInstance`
- **THEN** no `#config` validation runs inside `Render`; the staged instance package is imported by the build as it was processed

### Requirement: Kernel.LoadSourceFromFile auto-unwraps the values field

The `*Kernel.LoadSourceFromFile(path string)` method SHALL load the file at `path` as a CUE instance via `load.Instances`, evaluate it against the kernel's `*cue.Context`, and:

- If the evaluated value contains a top-level `values:` field whose `Exists()` is true and `Err()` is nil, the returned `Source.Value` SHALL be that field.
- Otherwise the returned `Source.Value` SHALL be the whole evaluated value.

The method SHALL set `Source.Origin` to the absolute path of the loaded file; `Source` carries no other label. The method SHALL NOT depend on `loaderfile.LoadValuesFile` (which is removed).

#### Scenario: Values file is auto-unwrapped

- **WHEN** a caller invokes `k.LoadSourceFromFile("./values.cue")` against a file containing `values: { foo: "bar" }`
- **THEN** the returned `Source.Value` is the inner `{ foo: "bar" }` value
- **AND** `Source.Origin` is the absolute path of `values.cue`

#### Scenario: File without values field passes through

- **WHEN** a caller invokes `k.LoadSourceFromFile("./flat.cue")` against a file with no top-level `values:` field
- **THEN** the returned `Source.Value` is the whole evaluated file value
- **AND** `Source.Origin` is populated as above
