## MODIFIED Requirements

### Requirement: Path inventory exposed as package-level vars

The library SHALL expose every CUE path the kernel's Go code reads on an OPM artifact as an exported package-level `cue.Path` variable in `opm/schema`, and the package documentation SHALL name each path's readers. The inventory SHALL be exactly these paths:

- `Metadata`: metadata decoding of every artifact kind (including the module metadata `Instance.ModuleMetadata` decodes), instance processing and the registry loader's identity read.
- `Components`: `Instance.Components`.
- `Values`: `Instance.Values`, the values check and conflict attribution of an instance build, and the top-level `values:` unwrap of a file-backed values source.
- `Config`: `Module.ConfigSchema`, `Instance.ConfigSchema`, and values checking against `#config` at acquire and synthesis.
- `Module`: the instance's reference to its `#Module`, read by `Instance.ConfigSchema`, `Instance.ModuleMetadata` and values checking at acquire and synthesis.
- `DebugValues`: `Module.DebugValues`, the documented frontend read of a module's debug overlay.
- `Contracts`: the contract-inventory decode `NewPlatformFromValue` records at construction, which `Platform.Contracts()` returns (on a `Platform` the constructor did not build, the first `Contracts()` or `CoreFloor()` call runs the same decode).
- `ContractsProvidedBy`: the core-floor presence test `NewPlatformFromValue` records at construction, which `Platform.CoreFloor()` reports and `Kernel.Render` checks before staging. The inventory decode reads the same field relative to `Contracts`, and the render glue reads it in CUE.
- `ContractsCollisions` and `ContractsCollidingEntries`: they name fields that the inventory decode reads relative to `Contracts` and the render glue reads in CUE. In Go only tests read the variables, which document the collision report.
- `CatalogProvides`: `Catalog.Provides()`.
- `Transformers`: `Catalog.Provides()` reads it on both of its paths, to refuse an unevaluated `#transformers`, and the deprecated provider-set fold, for a catalog built against a core older than `schema.ProvidesSince`, reads every transformer through it.
- `RequiredResources`, `RequiredTraits` and `Fulfilment`: the deprecated fold reads them relative to a transformer and to its demand entries, not from an artifact root.

The paths the retired Go matcher, executor and context builder read (`Registry`, `Transform`, `TransformerRequiredLabels`, `TransformerRequiredResources`, `TransformerRequiredTraits`, `TransformerOptionalTraits`, `ModuleInstance`, `Component`, `Context`, `Output`, `MatchLabels`, `MetadataLabels`, `MetadataAnnotations`, `MetadataFQN`, `ComponentResources`, `ComponentTraits`) and `ModuleMetadataPath` SHALL stay removed: the render build reads the instance's components and the platform's `#composedTransformers` and `#contracts` in CUE, inside the generated glue. A path that no kernel code reads, by variable or relative to an inventory path, is removed, not retained for a possible consumer.

#### Scenario: Consumer references a path directly

- **WHEN** a kernel consumer needs the path to an instance's `components` field
- **THEN** it imports `opm/schema` and references `schema.Components`
- **AND** does not call any `Paths()` method or look up a binding

#### Scenario: Matcher and transformer paths are gone

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `Registry`, `Transform`, `TransformerRequiredLabels`, `TransformerRequiredResources`, `TransformerRequiredTraits`, `TransformerOptionalTraits`, `ModuleInstance`, `Component`, `Context`, `Output`, `MatchLabels`, `MetadataLabels`, `MetadataAnnotations`, `MetadataFQN`, `ComponentResources`, `ComponentTraits`, `ModuleMetadataPath` exists

#### Scenario: Platform view and context sub-paths are not exported

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `KnownResources`, `KnownTraits`, `ComposedTransformers`, `Matchers`, `MatchersResources`, `MatchersTraits`, `ContextModuleInstanceMetadata`, `ContextComponentMetadata`, `ContextRuntimeName` exists
- **AND** the render glue reads `#composedTransformers` inside the build; no Go code path navigates a platform value by path

#### Scenario: Instance and module reads go through the inventory

- **WHEN** a frontend calls `Instance.Values()`, `Instance.ModuleMetadata()` or `Module.DebugValues()`
- **THEN** each reads its field through the matching inventory path: `schema.Values`; `schema.Module`, then `schema.Metadata`; `schema.DebugValues`
