## MODIFIED Requirements

### Requirement: SynthesizeInstance is documented as the recommended in-memory entry point

The package documentation and the `Kernel.SynthesizeInstance` godoc SHALL state that `SynthesizeInstance` is the entry point for building an instance from typed inputs, mirroring `Kernel.AcquireInstanceFromDir` for a directory-based CUE package, and that the module it takes comes from `AcquireModuleFromRegistry` or `AcquireModuleFromDir`. The documentation SHALL state that the core release the synthesized package imports is the kernel's pinned schema release, read from the configured loader without a schema load when that loader pins an exact release, and resolved through the kernel's schema cache otherwise. The documentation SHALL NOT present a helper-level composition that reaches synthesis or instance processing directly, since neither is exported.

#### Scenario: Documentation directs callers to the kernel method

- **WHEN** a developer reads the godoc on `opm/kernel`
- **THEN** the documentation states that `Kernel.SynthesizeInstance` is the entry point for typed-input synthesis, names the two acquire verbs that produce its module, and states where the imported core release comes from
- **AND** no reference to `synth.Instance`, `opm/helper/synth` or `Kernel.LoadInstancePackage` remains

#### Scenario: SynthesizeInstance godoc points to LoadInstancePackage

- **WHEN** a developer reads the `Kernel.SynthesizeInstance` godoc
- **THEN** the directory-driven mirror it names is `Kernel.AcquireInstanceFromDir`, and the module sources it names are `AcquireModuleFromRegistry` and `AcquireModuleFromDir`
- **AND** no reference to `Kernel.LoadInstancePackage`, `Kernel.ProcessModuleInstance` or `synth.Instance` remains

#### Scenario: Pinned kernel synthesizes without touching the schema cache

- **WHEN** a frontend constructs a kernel with the default loader and synthesizes an instance
- **THEN** no schema load runs for the synthesis; the module's own dependency list resolves `core` inside the build

### Requirement: Kernel.SynthesizeInstance method

The `*Kernel` type SHALL expose a method `SynthesizeInstance(ctx context.Context, in InstanceInput) (*module.Instance, error)`, where `InstanceInput` is declared in `opm/kernel` and carries `Module *module.Module` (required, source-carrying), `Name string` (required), `Namespace string` (required), `Values []Source` (optional; empty means "no values supplied"), `Labels map[string]string` and `Annotations map[string]string` (optional). It SHALL build the instance spec by single-build CUE evaluation inside the module's staged source with the unified values rendered into the package, check the values sources against the module's `#config` at their own positions after the build, assert concreteness on the built spec, decode instance metadata, and return the constructed `*module.Instance` with its `Source` populated. The core release the synthesized package imports is the kernel's: read from the configured loader's pin with no schema load when it names an exact release, and resolved through the kernel's schema cache when it names a bare major. The method SHALL NOT consult any additional values source.

A missing required input SHALL fail with an error wrapping the corresponding `opm/errors` sentinel (`ErrMissingModule`, `ErrMissingName`, `ErrMissingNamespace`); a module without staged source SHALL fail wrapping `ErrMissingSource`; a module whose `Source.Pkg` is non-empty SHALL fail with an error stating that a synthesizable module is its module's root package.

#### Scenario: SynthesizeInstance produces an end-to-end instance

- **WHEN** `k.SynthesizeInstance(ctx, kernel.InstanceInput{Module: mod, Name: "demo", Namespace: "default", Values: []kernel.Source{concrete}})` is called against an acquired module
- **THEN** the returned `*module.Instance` is non-nil
- **AND** `Instance.Metadata.Name` equals `"demo"`, `Instance.Metadata.Namespace` equals `"default"`
- **AND** `Instance.Metadata.UUID` equals `uuid.SHA1(OPMNamespace, "<module.uuid>:demo:default")`
- **AND** `Instance.Source` is non-nil

#### Scenario: SynthesizeInstance rejects unconcrete result

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with empty `in.Values` against a module whose `#config` has required fields with no defaults
- **THEN** the returned error is non-nil, is framed `instance "<name>": …`, and wraps the concreteness diagnostic

#### Scenario: SynthesizeInstance attributes a values violation to its source

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with a source whose value violates the module's `#config`
- **THEN** the returned error names the violating path and its positions report the source's `Origin`

#### Scenario: SynthesizeInstance surfaces synth errors before validation

- **WHEN** `k.SynthesizeInstance(ctx, kernel.InstanceInput{Module: nil, Name: "x", Namespace: "y"})` is called
- **THEN** the returned error wraps `oerrors.ErrMissingModule` and no build runs

#### Scenario: SynthesizeInstance refuses a subpackage module

- **WHEN** `k.SynthesizeInstance` is called with a module acquired from a subdirectory of its CUE module (`Source.Pkg` non-empty)
- **THEN** the returned error states that the module must be its module's root package and no build runs

#### Scenario: SynthesizeInstance uses the Kernel's cue.Context

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called
- **AND** a developer inspects the cue.Context underlying the returned `Instance.Package`
- **THEN** that context is the same instance returned by `k.CueContext()`

#### Scenario: Operator-shaped caller needs no cue.Context

- **WHEN** a frontend holds raw values bytes and their origin
- **THEN** it builds the input with `k.LoadSourceFromBytes(origin, raw)` and passes the result in `InstanceInput.Values`
- **AND** it does not call `k.CueContext()`
