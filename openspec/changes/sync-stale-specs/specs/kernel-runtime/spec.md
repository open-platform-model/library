## REMOVED Requirements

### Requirement: SynthesizeInstance is documented as the recommended in-memory entry point

**Reason**: Its scenario "SynthesizeInstance godoc points to LoadInstancePackage" is titled with the opposite of what it asserts: the body requires that no reference to `Kernel.LoadInstancePackage` remains. OpenSpec cannot rename a scenario under MODIFIED, so the requirement is restated under "SynthesizeInstance is documented as the typed-input entry point", with that scenario renamed.
**Migration**: None. The other scenarios are unchanged. The sentence on the imported core release is corrected to the code: the synthesized package imports `core` at the major of the kernel's schema release, and the release that import resolves to comes from the module's own `cue.mod/module.cue`.

## ADDED Requirements

### Requirement: SynthesizeInstance is documented as the typed-input entry point

The package documentation and the `Kernel.SynthesizeInstance` godoc SHALL state that `SynthesizeInstance` is the entry point for building an instance from typed inputs, mirroring `Kernel.AcquireInstanceFromDir` for a directory-based CUE package, and that the module it takes comes from `AcquireModuleFromRegistry` or `AcquireModuleFromDir`. The documentation SHALL state that the synthesized package imports `core` at the major of the kernel's schema release (read from the configured loader without a schema load when that loader pins an exact release, and resolved through the kernel's schema cache otherwise), and that the release that import resolves to comes from the module's own `cue.mod/module.cue`. The documentation SHALL NOT present a helper-level composition that reaches synthesis or instance processing directly, since neither is exported.

#### Scenario: Documentation directs callers to the kernel method

- **WHEN** a developer reads the godoc on `opm/kernel`
- **THEN** the documentation states that `Kernel.SynthesizeInstance` is the entry point for typed-input synthesis, names the two acquire verbs that produce its module, and states that the synthesized package imports `core` at the major of the kernel's schema release while the release that import resolves to comes from the module's own `cue.mod/module.cue`
- **AND** no reference to `synth.Instance`, `opm/helper/synth` or `Kernel.LoadInstancePackage` remains

#### Scenario: SynthesizeInstance godoc names its directory mirror and module sources

- **WHEN** a developer reads the `Kernel.SynthesizeInstance` godoc
- **THEN** the directory-driven mirror it names is `Kernel.AcquireInstanceFromDir`, and the module sources it names are `AcquireModuleFromRegistry` and `AcquireModuleFromDir`
- **AND** no reference to `Kernel.LoadInstancePackage`, `Kernel.ProcessModuleInstance` or `synth.Instance` remains

#### Scenario: Pinned kernel synthesizes without touching the schema cache

- **WHEN** a frontend constructs a kernel with the default loader and synthesizes an instance
- **THEN** no schema load runs for the synthesis; the module's own dependency list resolves `core` inside the build
