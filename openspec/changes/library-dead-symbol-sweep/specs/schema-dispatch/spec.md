## MODIFIED Requirements

### Requirement: Path inventory exposed as package-level vars
The library SHALL expose every CUE path used by the kernel, matcher, and renderer as exported package-level `cue.Path` variables in `opm/schema`. The variable names MUST match the artifact-domain term (e.g. `Metadata`, `Components`, `Values`, `Config`, `Module`, `ModuleMetadataPath`, `DebugValues`, `Transformers`, `Registry`, `Transform`, `TransformerRequiredLabels`, `TransformerRequiredResources`, `TransformerRequiredTraits`, `TransformerOptionalTraits`, `ModuleInstance`, `Component`, `Context`, `Output`, `MatchLabels`, `MetadataLabels`, `MetadataAnnotations`, `MetadataFQN`, `ComponentResources`, `ComponentTraits`). The inventory SHALL contain only paths some production code path reads; a path with no reader is removed, not retained for a possible consumer.

#### Scenario: Consumer references a path directly
- **WHEN** a kernel consumer needs the path to an instance's `components` field
- **THEN** it imports `opm/schema` and references `schema.Components`
- **AND** does not call any `Paths()` method or look up a binding

#### Scenario: Platform view and context sub-paths are not exported
- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `KnownResources`, `KnownTraits`, `ComposedTransformers`, `Matchers`, `MatchersResources`, `MatchersTraits`, `ContextModuleInstanceMetadata`, `ContextComponentMetadata`, `ContextRuntimeName` exists
- **AND** the matcher and executor read the composed map and reverse index off `MaterializedPlatform.Transformers` / `.Matchers`, never through a path on the platform value

### Requirement: Metadata decoders are free functions
The library SHALL expose `DecodeModuleMetadata`, `DecodeInstanceMetadata`, and `DecodePlatformMetadata` as free functions in `opm/schema`, one per artifact the kernel accepts. Each function MUST accept a raw `cue.Value` at the artifact root and return the canonical decoded metadata struct or a non-nil error.

#### Scenario: Decoding a module artifact
- **WHEN** `schema.DecodeModuleMetadata(v)` is called with the root of a valid `#Module` value
- **THEN** it returns `*schema.ModuleMetadata` with `Name`, `ModulePath`, `Version`, `FQN`, `UUID`, `Labels`, `Annotations` populated and a nil error

#### Scenario: Missing metadata is fatal for module/instance/platform
- **WHEN** `DecodeModuleMetadata`/`DecodeInstanceMetadata`/`DecodePlatformMetadata` is called with a value whose `metadata` field is absent
- **THEN** it returns nil and an error stating "metadata field is required"

#### Scenario: Platform metadata hoists top-level type
- **WHEN** `DecodePlatformMetadata(v)` is called on a `#Platform` whose root has `type: "kubernetes"` alongside its `metadata` block
- **THEN** the returned `PlatformMetadata.Type` is `"kubernetes"`

#### Scenario: Provider metadata falls back to caller-supplied name
- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** neither `DecodeProviderMetadata` nor `ProviderMetadata` exists; the provider artifact was retired with the platform construct and the kernel accepts exactly three artifacts

## REMOVED Requirements

### Requirement: Default-namespace annotation key
**Reason**: `schema.AnnotationDefaultNamespace` had no reader in the library, `cli` or `opm-operator`. The annotation itself (`module.opmodel.dev/default-namespace`) remains declared on core's `#Module` and ADR-001 remains the decision; only the Go constant naming it is removed.
**Migration**: A consumer that reads the annotation writes the key string itself, as it does for every other core-declared label and annotation key.
