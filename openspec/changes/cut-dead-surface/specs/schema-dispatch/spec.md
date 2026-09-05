## MODIFIED Requirements

### Requirement: Metadata decoders are free functions

The library SHALL decode each artifact's metadata through one unexported free function per artifact, living in the package of its single caller (`opm/module` for the module, `opm/platform` for the platform, `opm/kernel` for the instance); `opm/schema` SHALL NOT export a decoder. Each decoder MUST accept a raw `cue.Value` at the artifact root, read it through the `opm/schema` metadata path, and return the canonical decoded metadata struct (`ModuleMetadata`, `InstanceMetadata`, `PlatformMetadata`, which stay exported from `opm/schema`) or a non-nil error. Consumers reach decoded metadata only through the artifact constructors and the kernel's acquisition paths (`Module.Metadata`, `Instance.Metadata`, `Platform.Metadata`).

#### Scenario: Decoding a module artifact

- **WHEN** `module.NewModuleFromValue(v)` is called with the root of a valid `#Module` value
- **THEN** `Module.Metadata` is a `*schema.ModuleMetadata` with `Name`, `ModulePath`, `Version`, `FQN`, `UUID`, `Labels`, `Annotations` populated

#### Scenario: Missing metadata is fatal for module/instance/platform

- **WHEN** a module or platform constructor, or the kernel's instance processing, is given a value whose `metadata` field is absent
- **THEN** it returns an error stating "metadata field is required" and no partial artifact

#### Scenario: Platform metadata hoists top-level type

- **WHEN** `platform.NewPlatformFromValue(v)` is called on a `#Platform` whose root has `type: "kubernetes"` alongside its `metadata` block
- **THEN** the returned `Platform.Metadata.Type` is `"kubernetes"`

#### Scenario: Decoders are not exported

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `DecodeModuleMetadata`, `DecodeInstanceMetadata`, `DecodePlatformMetadata` exists

#### Scenario: Provider metadata falls back to caller-supplied name

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** neither `DecodeProviderMetadata` nor `ProviderMetadata` exists; the provider artifact was retired with the platform construct and the kernel accepts exactly three artifacts

### Requirement: Module, Instance, Platform structs do not carry APIVersion

`opm/module.Module`, `opm/module.Instance`, and `opm/platform.Platform` MUST NOT have an `APIVersion` field. The constructors `module.NewModuleFromValue(v)` and `platform.NewPlatformFromValue(v)`, and the kernel's internal instance processing, MUST decode metadata through their package-internal decoders directly without consulting any binding registry. No `NewInstanceFromValue` constructor SHALL be exported.

#### Scenario: Module struct has no APIVersion field

- **WHEN** Go code references `module.Module{}`
- **THEN** the literal compiles without an `APIVersion` field
- **AND** there is no exported `apiversion.Version`-typed accessor

#### Scenario: NewModuleFromValue decodes metadata directly

- **WHEN** `module.NewModuleFromValue(v)` is called with a valid `#Module` value
- **THEN** the returned `*module.Module` has `Metadata` populated, no version-dispatch lookup occurs, and `Package == v`

#### Scenario: Instances are decoded by the kernel only

- **WHEN** a consumer inspects the exported identifiers of `opm/module`
- **THEN** `NewInstanceFromValue` does not exist
- **AND** an instance returned by `Kernel.AcquireInstanceFromDir` or `Kernel.SynthesizeInstance` has `Metadata` populated from its `metadata` field
