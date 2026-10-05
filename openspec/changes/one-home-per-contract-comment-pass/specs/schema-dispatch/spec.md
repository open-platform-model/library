## REMOVED Requirements

### Requirement: Single OPM schema, externally resolved, with no apiVersion field

**Reason**: Two of its scenarios give the default as the bare major `"opmodel.dev/core@v2"`, and one ("Default resolves within the v2 major") is false by name: the default loader pins an exact release (`schema.DefaultSchemaModule`) and the bare major is opt-in. OpenSpec refuses a MODIFIED that drops a scenario, so the requirement is re-added under a new name.

**Migration**: Replaced by "Single OPM schema, externally resolved and pinned by default", which keeps the requirement body and three scenarios verbatim, renames "Schema resolved via module identifier" with the default corrected, and replaces "Default resolves within the v2 major" with "Default resolves the pinned release" and "Bare major resolves within the v2 major (opt-in)".

## ADDED Requirements

### Requirement: Single OPM schema, externally resolved and pinned by default

The library SHALL consume exactly one OPM CUE schema package: `opmodel.dev/core@v2` (or a caller-pinned exact version, in any major, via `OCILoader.Module`), resolved through CUE's module system against `CUE_REGISTRY`. The default loader SHALL resolve the exact release `schema.DefaultSchemaModule` names; the bare major `opmodel.dev/core@v2` SHALL be opt-in, through `OCILoader.Module`. The library MUST NOT vendor or embed the schema source under `library/apis/core/` or any other in-tree location. The schema package MUST NOT define a top-level `#ApiVersion` constant. Artifact roots (`#Module`, `#ModuleInstance`, `#Component`, `#ComponentTransformer`, `#Platform`, `#Resource`, `#Trait`) MUST NOT carry an `apiVersion` field.

#### Scenario: No in-tree schema source

- **WHEN** the library tree is inspected after this change
- **THEN** no directory `library/apis/` exists
- **AND** no Go file embeds `opmodel.dev/core` source via `//go:embed`

#### Scenario: Schema resolved via the default module identifier

- **WHEN** the kernel's `*schema.Cache` is populated for the first time
- **THEN** the underlying load goes through `cue/load.Instances` against the configured module identifier (default `schema.DefaultSchemaModule`, an exact release)
- **AND** the resolved value's `LookupPath(cue.ParsePath("#ModuleInstance"))` exists

#### Scenario: Evaluated module has no apiVersion field

- **WHEN** an artifact authored against the library schema is loaded and evaluated
- **THEN** `apiVersion` on the artifact root does not exist

#### Scenario: Default resolves the pinned release

- **WHEN** `(schema.OCILoader{}).Load(ctx)` runs with no `Module` override
- **THEN** the loader loads exactly the release `schema.DefaultSchemaModule` names, with no bare-major expansion

#### Scenario: Bare major resolves within the v2 major (opt-in)

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v2"}).Load(ctx)` is called
- **THEN** the bare major is expanded through the loader's bare-major mechanism and resolves to the highest published version within the v2 major
- **AND** SemVer prerelease ordering applies, so a `v2.0.0-0.dev.*` snapshot tag never outranks a `v2.0.0-alpha.N` tag

#### Scenario: Caller-pinned earlier major still loads

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1"}).Load(ctx)` is called
- **THEN** the loader resolves exactly that version and returns its schema value
- **AND** no code path upgrades or rewrites the caller's pin

## MODIFIED Requirements

### Requirement: A pinned schema release is known without a load

`OCILoader` SHALL report, without any I/O, the exact core release its module identifier names: `PinnedVersion()` returns that release and `true` when `Module` (or `DefaultSchemaModule` when `Module` is empty) names a full release, and `("", false)` when it names a bare major. A kernel whose configured loader pins an exact release SHALL derive the core release its synthesized instances import from that pin and SHALL NOT load the schema to do so; a kernel whose loader pins no exact release (a bare-major `OCILoader`, or any `Loader` that is not an `OCILoader`) SHALL resolve the release through its schema cache.

#### Scenario: Default loader is pinned

- **WHEN** `(schema.OCILoader{}).PinnedVersion()` is called
- **THEN** it returns `DefaultSchemaVersion()` and `true`, with no registry or cache access

#### Scenario: Bare major is not pinned

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v2"}).PinnedVersion()` is called
- **THEN** it returns `""` and `false`

#### Scenario: Synthesis on a pinned kernel loads no schema

- **WHEN** a kernel constructed with the default loader synthesizes an instance
- **THEN** the synthesized package imports core at the pinned release's major, and `k.SchemaCache().ResolvedVersion()` is still `""` afterwards because no load ran

#### Scenario: Synthesis on a bare-major kernel resolves through the cache

- **WHEN** a kernel constructed with `WithSchemaLoader(schema.OCILoader{Module: "opmodel.dev/core@v2"})` synthesizes an instance
- **THEN** the schema cache is loaded once and the synthesized package imports core at the resolved release's major

### Requirement: Path inventory exposed as package-level vars

The library SHALL expose every CUE path the kernel's Go code reads on an OPM artifact as an exported package-level `cue.Path` variable in `opm/schema`, and the package documentation SHALL name each path's readers. The inventory SHALL be exactly these paths:

- `Metadata`: metadata decoding of every artifact kind (including the module metadata `Instance.ModuleMetadata` decodes), instance processing and the registry loader's identity read.
- `Components`: `Instance.Components`.
- `Values`: `Instance.Values`, the values check and conflict attribution of an instance build, and the top-level `values:` unwrap of a file-backed values source.
- `Config`: `Module.ConfigSchema`, `Instance.ConfigSchema`, and values checking against `#config` at acquire and synthesis.
- `Module`: the instance's reference to its `#Module`, read by `Instance.ConfigSchema`, `Instance.ModuleMetadata` and values checking at acquire and synthesis.
- `DebugValues`: `Module.DebugValues`, the documented frontend read of a module's debug overlay.
- `Contracts`: `Platform.Contracts()`.
- `ContractsProvidedBy`: the core-floor presence check `Kernel.Render` runs before staging. `Platform.Contracts()` decodes the same field relative to `Contracts`, and the render glue reads it in CUE.
- `ContractsCollisions` and `ContractsCollidingEntries`: they name fields that `Platform.Contracts()` reads relative to `Contracts` and the render glue reads in CUE. In Go only tests read the variables, which document the collision report.
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
