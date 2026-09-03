## MODIFIED Requirements

### Requirement: DefaultSchemaModule constant

`schema.DefaultSchemaModule` SHALL name an exact core release, not the floating `opmodel.dev/core@v2` major: the release the kernel's render glue, fixtures and parity oracle were verified against. At this change that is `opmodel.dev/core@v2.0.0-alpha.7`, the first release carrying the D5 registry shape and the D12 context projection. `OCILoader.Load` with an empty `Module` field SHALL resolve this identifier. The constant advances only by a deliberate change that re-verifies the glue and fixtures against the new release; a default that floats ahead of the glue breaks every synthesized artifact on a cold cache. Doc comments citing the default module identifier (`opm/kernel`, `opm/schema`) SHALL cite the pinned identifier, and the pin assertion in `opm/schema/loader_test.go` SHALL move with the constant.

#### Scenario: Empty Module resolves the v2 default

- **WHEN** `(schema.OCILoader{Registry: "opmodel.dev=ghcr.io/open-platform-model"}).Load(ctx)` is called with `Module` unset
- **THEN** the loader resolves `Module` to `"opmodel.dev/core@v2.0.0-alpha.7"`, threads the env into `load.Config.Env`, and returns a non-zero `cue.Value` containing `#ModuleInstance`

#### Scenario: ResolvedVersion reports the v2 resolution

- **WHEN** `cache.Get(ctx)` succeeds against the default
- **THEN** `cache.ResolvedVersion()` returns `"v2.0.0-alpha.7"`

#### Scenario: No doc comment cites a deleted package or the floating major

- **WHEN** a developer searches `opm/` for the default module identifier
- **THEN** every citation names the pinned release and none lives in `opm/materialize`, which no longer exists

### Requirement: Path inventory exposed as package-level vars

The library SHALL expose every CUE path some production code path reads as exported package-level `cue.Path` variables in `opm/schema`. After the render cutover the readers are metadata decoding, instance processing and module accessors, so the inventory is exactly `Metadata`, `Components`, `Values`, `Config`, `Module`, `ModuleMetadataPath` and `DebugValues` (the last kept for the documented frontend read of a module's debug overlay). The paths the Go matcher, executor and context builder read (`Transformers`, `Registry`, `Transform`, `TransformerRequiredLabels`, `TransformerRequiredResources`, `TransformerRequiredTraits`, `TransformerOptionalTraits`, `ModuleInstance`, `Component`, `Context`, `Output`, `MatchLabels`, `MetadataLabels`, `MetadataAnnotations`, `MetadataFQN`, `ComponentResources`, `ComponentTraits`) SHALL be removed with those readers: the render build reads the instance's components and the platform's `#composedTransformers` in CUE, inside the generated glue. A path with no reader is removed, not retained for a possible consumer.

#### Scenario: Consumer references a path directly

- **WHEN** a kernel consumer needs the path to an instance's `components` field
- **THEN** it imports `opm/schema` and references `schema.Components`
- **AND** does not call any `Paths()` method or look up a binding

#### Scenario: Matcher and transformer paths are gone

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `Transformers`, `Registry`, `Transform`, `TransformerRequiredLabels`, `TransformerRequiredResources`, `TransformerRequiredTraits`, `TransformerOptionalTraits`, `ModuleInstance`, `Component`, `Context`, `Output`, `MatchLabels`, `MetadataLabels`, `MetadataAnnotations`, `MetadataFQN`, `ComponentResources`, `ComponentTraits` exists

#### Scenario: Platform view and context sub-paths are not exported

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `KnownResources`, `KnownTraits`, `ComposedTransformers`, `Matchers`, `MatchersResources`, `MatchersTraits`, `ContextModuleInstanceMetadata`, `ContextComponentMetadata`, `ContextRuntimeName` exists
- **AND** the render glue reads `#composedTransformers` inside the build; no Go code path navigates a platform value by path

## REMOVED Requirements

### Requirement: Transformer-context builder

**Reason**: `schema.BuildTransformerContext`, `InstanceView`, `ModuleInstanceContextData` and `ComponentContextData` are deleted with `opm/schema/context.go`. Core `v2.0.0-alpha.7` projects every `#TransformerContext` field except `#runtimeName` from `#moduleInstance` and `#component` (0019 D12); the kernel supplies `#runtimeName` as a literal in the generated glue (`transform-input-fill`, "`#context` is projected by core").

**Migration**: none; no frontend called the builder.

### Requirement: Match drops the binding parameter

**Reason**: `opm/compile.Match`, `(*compile.Module).Execute`, `kernel.Match` and `kernel.Compile` are deleted with the compile pipeline (0019 D9/D10). The absence of a cross-artifact `apiVersion` check is already carried by "Module, Instance, Platform structs do not carry APIVersion".

**Migration**: `Render`.
