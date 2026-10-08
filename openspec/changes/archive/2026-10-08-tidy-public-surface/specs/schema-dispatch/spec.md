## MODIFIED Requirements

### Requirement: Path inventory exposed as package-level vars

The library SHALL keep every CUE path the kernel's Go code reads on an OPM artifact as one package-level `cue.Path` variable, declared once, and the documentation of its package SHALL name each path's readers. `opm/schema` SHALL export exactly the three paths a frontend reads on an artifact `Package` for which no accessor exists: `Metadata`, `Module` and `CatalogProvides`. Every other path SHALL live in the internal package `opm/internal/corepath`, where no consumer can import or reassign it; a frontend reaches those fields through the artifact accessors (`Instance.Components`, `Instance.Values`, `Instance.ConfigSchema`, `Module.ConfigSchema`, `Module.DebugValues`, `Platform.Contracts`, `Catalog.Provides`). The inventory SHALL be exactly these paths.

Exported from `opm/schema`:

- `Metadata`: metadata decoding of every artifact kind (including the module metadata `Instance.ModuleMetadata` decodes), instance processing and the registry loader's identity read.
- `Module`: the instance's reference to its `#Module`, read by `Instance.ConfigSchema`, `Instance.ModuleMetadata` and values checking at acquire and synthesis.
- `CatalogProvides`: `Catalog.Provides()`.

In `opm/internal/corepath`:

- `Components`: `Instance.Components`.
- `Values`: `Instance.Values`, the values check and conflict attribution of an instance build, and the top-level `values:` unwrap of a file-backed values source.
- `Config`: `Module.ConfigSchema`, `Instance.ConfigSchema`, and values checking against `#config` at acquire and synthesis.
- `DebugValues`: `Module.DebugValues`, the documented frontend read of a module's debug overlay.
- `Contracts`: the contract-inventory decode `NewPlatformFromValue` records at construction, which `Platform.Contracts()` returns (on a `Platform` the constructor did not build, the first `Contracts()` or `CoreFloor()` call runs the same decode).
- `ContractsProvidedBy`: the core-floor presence test `NewPlatformFromValue` records at construction, which `Platform.CoreFloor()` reports and `Kernel.Render` checks before staging. The inventory decode reads the same field relative to `Contracts`, and the render glue reads it in CUE.
- `ContractsCollisions` and `ContractsCollidingEntries`: they name fields that the inventory decode reads relative to `Contracts` and the render glue reads in CUE. In Go only tests read the variables, which document the collision report.
- `Transformers`: `Catalog.Provides()` reads it on both of its paths, to refuse an unevaluated `#transformers`, and the deprecated provider-set fold, for a catalog built against a core older than `schema.ProvidesSince`, reads every transformer through it.
- `RequiredResources`, `RequiredTraits` and `Fulfilment`: the deprecated fold reads them relative to a transformer and to its demand entries, not from an artifact root.

`opm/schema` SHALL export the core-release constants `ProvidedBySince` (the render floor) and `ProvidesSince` (which path `Catalog.Provides` takes), and SHALL NOT export a constant for the first core release that reports contract collisions: no code reads one.

The paths the retired Go matcher, executor and context builder read (`Registry`, `Transform`, `TransformerRequiredLabels`, `TransformerRequiredResources`, `TransformerRequiredTraits`, `TransformerOptionalTraits`, `ModuleInstance`, `Component`, `Context`, `Output`, `MatchLabels`, `MetadataLabels`, `MetadataAnnotations`, `MetadataFQN`, `ComponentResources`, `ComponentTraits`) and `ModuleMetadataPath` SHALL stay removed: the render build reads the instance's components and the platform's `#composedTransformers` and `#contracts` in CUE, inside the generated glue. A path that no kernel code reads, by variable or relative to an inventory path, is removed, not retained for a possible consumer.

#### Scenario: Consumer references a path directly

- **WHEN** a kernel consumer needs the module metadata of an instance for which it holds only the `Package`
- **THEN** it imports `opm/schema` and references `schema.Module`, then `schema.Metadata`
- **AND** does not call any `Paths()` method or look up a binding

#### Scenario: Paths with an accessor are not exported

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** the only `cue.Path` variables are `Metadata`, `Module` and `CatalogProvides`
- **AND** none of `Components`, `Values`, `Config`, `DebugValues`, `Contracts`, `ContractsProvidedBy`, `ContractsCollisions`, `ContractsCollidingEntries`, `Transformers`, `RequiredResources`, `RequiredTraits`, `Fulfilment` or `CollisionsSince` exists

#### Scenario: Matcher and transformer paths are gone

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `Registry`, `Transform`, `TransformerRequiredLabels`, `TransformerRequiredResources`, `TransformerRequiredTraits`, `TransformerOptionalTraits`, `ModuleInstance`, `Component`, `Context`, `Output`, `MatchLabels`, `MetadataLabels`, `MetadataAnnotations`, `MetadataFQN`, `ComponentResources`, `ComponentTraits`, `ModuleMetadataPath` exists

#### Scenario: Platform view and context sub-paths are not exported

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `KnownResources`, `KnownTraits`, `ComposedTransformers`, `Matchers`, `MatchersResources`, `MatchersTraits`, `ContextModuleInstanceMetadata`, `ContextComponentMetadata`, `ContextRuntimeName` exists
- **AND** the render glue reads `#composedTransformers` inside the build; no Go code path navigates a platform value by path

#### Scenario: Instance and module reads go through the inventory

- **WHEN** a frontend calls `Instance.Values()`, `Instance.ModuleMetadata()` or `Module.DebugValues()`
- **THEN** each reads its field through the matching inventory path: `corepath.Values`; `schema.Module`, then `schema.Metadata`; `corepath.DebugValues`

### Requirement: Metadata decoders are free functions

The library SHALL decode each artifact's metadata through one unexported free function per artifact, living in the package of its single caller (`opm/module` for the module, `opm/platform` for the platform, `opm/catalog` for the catalog, `opm/kernel` for the instance); `opm/schema` SHALL NOT export a decoder. Each decoder MUST accept a raw `cue.Value` at the artifact root, read it through the `opm/schema` metadata path, and return the canonical decoded metadata struct (`module.ModuleMetadata`, `module.InstanceMetadata`, `platform.PlatformMetadata`, `catalog.CatalogMetadata`, each declared once, in the package of the artifact it describes; `opm/schema` SHALL NOT declare or re-export a metadata type) or a non-nil error. Consumers reach decoded metadata only through the artifact constructors and the kernel's acquisition paths (`Module.Metadata`, `Instance.Metadata`, `Platform.Metadata`, `Catalog.Metadata`). A constructor (`NewModuleFromValue`, `NewPlatformFromValue`, `NewCatalogFromValue`) SHALL return its decoder's error without adding a prefix of its own, so a decode failure names its artifact exactly once (`decoding module metadata: ...`, never `decoding module metadata: decoding module metadata: ...`).

#### Scenario: Decoding a module artifact

- **WHEN** `module.NewModuleFromValue(v)` is called with the root of a valid `#Module` value
- **THEN** `Module.Metadata` is a `*module.ModuleMetadata` with `Name`, `ModulePath`, `Version`, `FQN`, `UUID`, `Labels`, `Annotations` populated

#### Scenario: Decoding a catalog artifact

- **WHEN** `catalog.NewCatalogFromValue(v)` is called with the root of a valid `#Catalog` value
- **THEN** `Catalog.Metadata` is a `*catalog.CatalogMetadata` with `ModulePath`, `Version`, `FQN`, `Description`, `Labels`, `Annotations` populated
- **AND** it carries no `Name`, because a catalog's identity is its module path and the version stamped on every member it ships

#### Scenario: Missing metadata is fatal for module/instance/platform

- **WHEN** a module or platform constructor, or the kernel's instance processing, is given a value whose `metadata` field is absent
- **THEN** it returns an error stating "metadata field is required" and no partial artifact

#### Scenario: Missing metadata is fatal for a catalog

- **WHEN** `catalog.NewCatalogFromValue(v)` is given a value whose `metadata` field is absent
- **THEN** it returns an error stating "metadata field is required" and a nil catalog, on the same terms as the other three

#### Scenario: Platform metadata hoists top-level type

- **WHEN** `platform.NewPlatformFromValue(v)` is called on a `#Platform` whose root has `type: "kubernetes"` alongside its `metadata` block
- **THEN** the returned `Platform.Metadata.Type` is `"kubernetes"`

#### Scenario: Each metadata type has one name

- **WHEN** a developer inspects the exported identifiers of `opm/schema`, `opm/module`, `opm/platform` and `opm/catalog`
- **THEN** `ModuleMetadata` and `InstanceMetadata` exist only in `opm/module`, `PlatformMetadata` only in `opm/platform` and `CatalogMetadata` only in `opm/catalog`, and none of them is a type alias

#### Scenario: Decoders are not exported

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `DecodeModuleMetadata`, `DecodeInstanceMetadata`, `DecodePlatformMetadata`, `DecodeCatalogMetadata` exists

#### Scenario: Provider metadata falls back to caller-supplied name

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** neither `DecodeProviderMetadata` nor `ProviderMetadata` exists; the provider artifact was retired with the platform construct and is not among the kinds the kernel accepts

#### Scenario: A decode failure names its artifact once

- **WHEN** `module.NewModuleFromValue(v)`, `platform.NewPlatformFromValue(v)` or `catalog.NewCatalogFromValue(v)` is given a value whose `metadata` field exists but does not decode into the metadata struct
- **THEN** it returns a nil artifact and an error whose text contains `decoding <kind> metadata:` exactly once

