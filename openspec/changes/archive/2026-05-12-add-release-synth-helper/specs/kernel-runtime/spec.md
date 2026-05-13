## ADDED Requirements

### Requirement: Kernel.SynthesizeRelease method

The `*Kernel` type SHALL expose a method `SynthesizeRelease(ctx context.Context, in synth.ReleaseInput) (*module.Release, error)` that combines `synth.Release` and `Kernel.ProcessModuleRelease` into a single call: it builds the release spec by unifying inputs against the embedded schema, then validates the supplied values against the module's `#config`, fills the values into the spec, enforces concreteness, decodes release metadata, and returns the constructed `*module.Release`.

The method SHALL use the Kernel's owned `*cue.Context` when calling `synth.Release`. The method SHALL NOT consult any additional values source — `in.Values` is passed through to `Kernel.ProcessModuleRelease` unchanged.

#### Scenario: SynthesizeRelease produces an end-to-end release

- **WHEN** `k.SynthesizeRelease(ctx, synth.ReleaseInput{Module: mod, Name: "demo", Namespace: "default", Values: concreteValues})` is called against a registered v1alpha2 module
- **THEN** the returned `*module.Release` is non-nil
- **AND** `Release.APIVersion` equals the module's API version
- **AND** `Release.Metadata.Name` equals `"demo"`, `Release.Metadata.Namespace` equals `"default"`
- **AND** `Release.Metadata.UUID` equals `uuid.SHA1(OPMNamespace, "<module.uuid>:demo:default")`

#### Scenario: SynthesizeRelease rejects unconcrete result

- **WHEN** `k.SynthesizeRelease(ctx, in)` is called with `in.Values == cue.Value{}` against a module whose `#config` has required fields with no defaults
- **THEN** the returned error is non-nil and wraps the `Kernel.ProcessModuleRelease` concreteness diagnostic

#### Scenario: SynthesizeRelease surfaces synth errors before validation

- **WHEN** `k.SynthesizeRelease(ctx, synth.ReleaseInput{Module: nil, Name: "x", Namespace: "y"})` is called
- **THEN** the returned error is non-nil and originates from `synth.Release` (not from `Kernel.ProcessModuleRelease`)

#### Scenario: SynthesizeRelease uses the Kernel's cue.Context

- **WHEN** `k.SynthesizeRelease(ctx, in)` is called
- **AND** a developer inspects the cue.Context underlying the returned `Release.Package`
- **THEN** that context is the same instance returned by `k.CueContext()`

### Requirement: SynthesizeRelease is documented as the recommended in-memory entry point

The package documentation and the `Kernel.SynthesizeRelease` godoc SHALL state that `SynthesizeRelease` is the recommended entry point for building a release from typed inputs, mirroring how `LoadReleaseFile` is the recommended entry point for building a release from a file. Callers that explicitly want the helper-level primitive MAY call `synth.Release` followed by `Kernel.ProcessModuleRelease` directly.

#### Scenario: Documentation directs callers to the kernel method

- **WHEN** a developer reads the godoc on `opm/helper/synth/`
- **THEN** the documentation states that `Kernel.SynthesizeRelease` is the recommended entry point
- **AND** notes that direct use of `synth.Release` is appropriate when the caller does not hold a `*Kernel`
