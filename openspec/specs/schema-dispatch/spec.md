# schema-dispatch Specification

## Purpose
Defines the single-schema dispatch surface that replaces the retired multi-`apiVersion` binding registry. The library consumes exactly one OPM CUE schema (`opmodel.dev/core@v2`) resolved at runtime via CUE's module system and exposes its paths, metadata decoders, transformer-context builder, and a caller-configurable `Loader` plus per-Kernel `*schema.Cache` in `opm/schema`. Callers no longer detect schema versions, look up bindings, or carry `APIVersion` on artifact structs.
## Requirements

### Requirement: Single OPM schema, externally resolved, with no apiVersion field

The library SHALL consume exactly one OPM CUE schema package: `opmodel.dev/core@v2` (or a caller-pinned exact version, in any major, via `OCILoader.Module`), resolved through CUE's module system against `CUE_REGISTRY`. The library MUST NOT vendor or embed the schema source under `library/apis/core/` or any other in-tree location. The schema package MUST NOT define a top-level `#ApiVersion` constant. Artifact roots (`#Module`, `#ModuleInstance`, `#Component`, `#ComponentTransformer`, `#Platform`, `#Resource`, `#Trait`) MUST NOT carry an `apiVersion` field.

#### Scenario: No in-tree schema source

- **WHEN** the library tree is inspected after this change
- **THEN** no directory `library/apis/` exists
- **AND** no Go file embeds `opmodel.dev/core` source via `//go:embed`

#### Scenario: Schema resolved via module identifier

- **WHEN** the kernel's `*schema.Cache` is populated for the first time
- **THEN** the underlying load goes through `cue/load.Instances` against the configured module identifier (default `"opmodel.dev/core@v2"`)
- **AND** the resolved value's `LookupPath(cue.ParsePath("#ModuleInstance"))` exists

#### Scenario: Evaluated module has no apiVersion field

- **WHEN** an artifact authored against the library schema is loaded and evaluated
- **THEN** `apiVersion` on the artifact root does not exist

#### Scenario: Default resolves within the v2 major

- **WHEN** `(schema.OCILoader{}).Load(ctx)` runs with no `Module` override
- **THEN** the bare major `"opmodel.dev/core@v2"` is expanded through the loader's existing bare-major mechanism and resolves to the highest published version within the v2 major
- **AND** SemVer prerelease ordering applies, so a `v2.0.0-0.dev.*` snapshot tag never outranks a `v2.0.0-alpha.N` tag

#### Scenario: Caller-pinned earlier major still loads

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1"}).Load(ctx)` is called
- **THEN** the loader resolves exactly that version and returns its schema value
- **AND** no code path upgrades or rewrites the caller's pin

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

### Requirement: Schema Loader interface

The library SHALL define `opm/schema.Loader` as an interface with a single method `Load(ctx *cue.Context) (cue.Value, error)`. Implementations SHALL return a built `cue.Value` representing the OPM core schema package (`#Module`, `#ModuleInstance`, `#Platform`, `#Resource`, `#Trait`, `#ComponentTransformer`, etc. reachable via `LookupPath`).

The library SHALL NOT export any `Loader` implementation other than `OCILoader` (see "OCILoader is the only public Loader" requirement). Internal-only or test-only `Loader` implementations MUST NOT appear in the public API surface.

#### Scenario: Loader is the contract not the implementation

- **WHEN** Go code declares a variable of type `schema.Loader`
- **THEN** any value implementing `Load(*cue.Context) (cue.Value, error)` satisfies the type
- **AND** the public package documents `OCILoader` as the canonical implementation

#### Scenario: No second public Loader

- **WHEN** a consumer enumerates `opm/schema` package symbols
- **THEN** `OCILoader` is the only exported type satisfying `Loader`

### Requirement: OCILoader is the only public Loader

The library SHALL expose `opm/schema.OCILoader` as the sole public implementation of `Loader`. Its struct fields SHALL be exactly `Module string`, `Registry string`, `CacheDir string`. The zero value of `OCILoader` SHALL be a valid Loader.

`OCILoader.Load(ctx)` SHALL:

- Resolve `Module` to `"opmodel.dev/core@v2"` (`DefaultSchemaModule`) when the field is empty.
- Resolve `Registry` to the value derived from `os.Environ`'s `CUE_REGISTRY` when the field is empty.
- Resolve `CacheDir` to the value derived from `os.Environ`'s `CUE_CACHE_DIR` (or CUE's default when that is also empty) when the field is empty.
- Invoke `cuelang.org/go/cue/load.Instances([]string{module}, &load.Config{Env: derivedEnv})` with the resolved values plumbed into `Env`.
- Call `ctx.BuildInstance` on the returned instance and return the resulting `cue.Value` and any error wrapped with context.

`OCILoader.Load` MUST NOT use any custom OCI client (e.g., `oras-go`), MUST NOT bypass CUE's module cache, and MUST NOT mutate process-global state (no `os.Setenv`).

#### Scenario: Zero-value OCILoader resolves defaults

- **WHEN** `(schema.OCILoader{}).Load(ctx)` is called in an environment with `CUE_REGISTRY` and `CUE_CACHE_DIR` set
- **THEN** the loader resolves `Module` to `"opmodel.dev/core@v2"`, threads the env into `load.Config.Env`, and returns a non-zero `cue.Value` containing `#ModuleInstance`

#### Scenario: Explicit overrides take precedence over env

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1", Registry: "opmodel.dev=ghcr.io/open-platform-model", CacheDir: "/tmp/cache"}).Load(ctx)` is called
- **THEN** the registry mapping and cache directory used by `load.Instances` reflect the explicit values regardless of the process environment

#### Scenario: Load failures are wrapped

- **WHEN** `load.Instances` or `BuildInstance` returns an error (e.g., registry unreachable on cache miss, malformed cached module, unknown module path)
- **THEN** `OCILoader.Load` returns the zero `cue.Value` and a non-nil error wrapping the underlying error with a message that identifies the module identifier being loaded

### Requirement: Schema Cache memoizes a single Load per instance

The library SHALL expose `opm/schema.Cache` as a struct with at minimum a `Loader Loader` field. `(*Cache).Get(ctx *cue.Context) (cue.Value, error)` SHALL invoke `Loader.Load(ctx)` exactly once per `Cache` instance via `sync.Once`-equivalent synchronization. Subsequent calls — including the call that loses the race — SHALL return the cached `cue.Value` (or the cached error) without re-invoking the Loader.

The library MUST NOT cache the `Loader`'s result at package scope. There SHALL be no package-level singleton schema value. Each `Cache` owns its memoization.

#### Scenario: Repeated Get returns the cached value

- **WHEN** `cache.Get(ctx)` is called twice on the same `*Cache`
- **THEN** both calls return the same `cue.Value` and the underlying `Loader.Load` runs exactly once

#### Scenario: Concurrent first Get is safe

- **WHEN** two goroutines call `cache.Get(ctx)` on the same `*Cache` before the cache is warmed
- **THEN** exactly one `Loader.Load` invocation runs and both goroutines receive the same result

#### Scenario: Loader errors are cached

- **WHEN** the first `cache.Get(ctx)` returns a non-nil error
- **THEN** subsequent `cache.Get(ctx)` calls return the same wrapped error without re-invoking the Loader

#### Scenario: Two Cache instances do not share state

- **WHEN** two distinct `*Cache` values built from logically-equivalent Loaders are each called with `Get`
- **THEN** each Cache runs its own Load invocation; populating one does not populate the other

### Requirement: Cache exposes the resolved schema version

`(*Cache).ResolvedVersion() string` SHALL return the schema module version that the underlying Loader resolved during the first successful Load (e.g., `"v2.0.0-alpha.4"` when the default `opmodel.dev/core@v2` resolved to `v2.0.0-alpha.4`). Before the first successful Load, `ResolvedVersion()` SHALL return the empty string.

#### Scenario: ResolvedVersion is empty before Get

- **WHEN** `cache.ResolvedVersion()` is called before any `cache.Get`
- **THEN** it returns `""`

#### Scenario: ResolvedVersion returns the resolved tag after Get

- **WHEN** `cache.Get(ctx)` succeeds against `opmodel.dev/core@v2` resolving to `v2.0.0-alpha.4`
- **THEN** `cache.ResolvedVersion()` returns `"v2.0.0-alpha.4"`

#### Scenario: ResolvedVersion stays empty after failed Load

- **WHEN** `cache.Get(ctx)` returns an error on first call
- **THEN** subsequent `cache.ResolvedVersion()` calls return `""`

### Requirement: PublicRegistry const documents the canonical mapping

The library SHALL expose `opm/schema.PublicRegistry` as an exported string constant whose value is `"opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"`. The library MUST NOT auto-apply this value as a default; callers opt in by setting `CUE_REGISTRY` to `schema.PublicRegistry` (or by passing it via `OCILoader.Registry`).

#### Scenario: Constant value

- **WHEN** Go code references `schema.PublicRegistry`
- **THEN** the constant resolves to `"opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"`

#### Scenario: Library does not auto-set CUE_REGISTRY

- **WHEN** `(schema.OCILoader{}).Load(ctx)` is called in an environment with `CUE_REGISTRY` unset
- **THEN** the library does not mutate the process environment
- **AND** the load result depends on whatever default registry CUE resolves to (typically returns a "module not found" error for `opmodel.dev/core`)

### Requirement: Kernel exposes a single Cache via SchemaCache accessor

`(*opm/kernel.Kernel).SchemaCache() *schema.Cache` SHALL return the `*schema.Cache` owned by the Kernel instance. Repeated calls SHALL return the same pointer for the lifetime of the Kernel. The accessor MUST NOT trigger a schema load by itself; only `(*Cache).Get` triggers a load.

#### Scenario: Accessor is identity-stable

- **WHEN** `k.SchemaCache()` is called twice on the same `*Kernel`
- **THEN** both calls return the same `*schema.Cache` pointer

#### Scenario: Accessor does not load schema

- **WHEN** `k.SchemaCache()` is called and no other Kernel method has yet caused a schema load
- **THEN** the returned Cache's `ResolvedVersion()` returns `""` and no network or disk fetch has occurred

### Requirement: WithSchemaLoader configures the Kernel's Cache

`opm/kernel.WithSchemaLoader(l schema.Loader) Option` SHALL configure the Loader the Kernel's `*schema.Cache` uses. When `WithSchemaLoader` is not provided, the Kernel SHALL default to `&schema.Cache{Loader: schema.OCILoader{}}` (zero-value OCILoader resolving defaults from environment).

`Kernel` MUST NOT accept a pre-built `*schema.Cache` from the caller; only a `Loader` is configurable. This guarantees the Kernel owns its Cache identity (one Kernel = one Cache).

#### Scenario: Default loader applied when option omitted

- **WHEN** `kernel.New(ctx)` is called with no `WithSchemaLoader` option
- **THEN** `k.SchemaCache().Loader` equals `schema.OCILoader{}`

#### Scenario: Custom loader applied when option present

- **WHEN** `kernel.New(ctx, kernel.WithSchemaLoader(schema.OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1"}))` is called
- **THEN** `k.SchemaCache().Loader` equals the supplied OCILoader value

### Requirement: Path inventory exposed as package-level vars

The library SHALL expose every CUE path some production code path reads as exported package-level `cue.Path` variables in `opm/schema`. After the render cutover the readers are metadata decoding, instance processing, the loaders' identity reads and the instance's components and `#config` accessors, so the inventory is exactly `Metadata`, `Components`, `Values`, `Config`, `Module` and `DebugValues` (the last kept for the documented frontend read of a module's debug overlay). The paths the Go matcher, executor and context builder read (`Transformers`, `Registry`, `Transform`, `TransformerRequiredLabels`, `TransformerRequiredResources`, `TransformerRequiredTraits`, `TransformerOptionalTraits`, `ModuleInstance`, `Component`, `Context`, `Output`, `MatchLabels`, `MetadataLabels`, `MetadataAnnotations`, `MetadataFQN`, `ComponentResources`, `ComponentTraits`) and `ModuleMetadataPath` (read only by the `ModuleVersion` accessor the context builder needed) SHALL be removed with those readers: the render build reads the instance's components and the platform's `#composedTransformers` in CUE, inside the generated glue. A path with no reader is removed, not retained for a possible consumer.

#### Scenario: Consumer references a path directly

- **WHEN** a kernel consumer needs the path to an instance's `components` field
- **THEN** it imports `opm/schema` and references `schema.Components`
- **AND** does not call any `Paths()` method or look up a binding

#### Scenario: Matcher and transformer paths are gone

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `Transformers`, `Registry`, `Transform`, `TransformerRequiredLabels`, `TransformerRequiredResources`, `TransformerRequiredTraits`, `TransformerOptionalTraits`, `ModuleInstance`, `Component`, `Context`, `Output`, `MatchLabels`, `MetadataLabels`, `MetadataAnnotations`, `MetadataFQN`, `ComponentResources`, `ComponentTraits`, `ModuleMetadataPath` exists

#### Scenario: Platform view and context sub-paths are not exported

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `KnownResources`, `KnownTraits`, `ComposedTransformers`, `Matchers`, `MatchersResources`, `MatchersTraits`, `ContextModuleInstanceMetadata`, `ContextComponentMetadata`, `ContextRuntimeName` exists
- **AND** the render glue reads `#composedTransformers` inside the build; no Go code path navigates a platform value by path

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

### Requirement: Loader helpers return only the loaded value
`opm/helper/loader/file.LoadModulePackage`, `LoadInstancePackage`, and `LoadPlatformPackage` MUST have the signature `(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, error)`. The previous `apiversion.Version` return is removed. Their `(*Kernel)` wrappers MUST follow the same signature.

#### Scenario: LoadModulePackage signature
- **WHEN** a caller invokes `file.LoadModulePackage(ctx, dir, opts)`
- **THEN** it returns exactly two values: a `cue.Value` and an `error`

### Requirement: Module, Instance, Platform structs do not carry APIVersion
`opm/module.Module`, `opm/module.Instance`, and `opm/platform.Platform` MUST NOT have an `APIVersion` field. Their constructors (`NewModuleFromValue`, `NewInstanceFromValue`, `NewPlatformFromValue`) MUST call the appropriate `schema.Decode*Metadata` function directly without consulting any binding registry.

#### Scenario: Module struct has no APIVersion field
- **WHEN** Go code references `module.Module{}`
- **THEN** the literal compiles without an `APIVersion` field
- **AND** there is no exported `apiversion.Version`-typed accessor

#### Scenario: NewModuleFromValue decodes metadata directly
- **WHEN** `NewModuleFromValue(k, v)` is called with a valid `#Module` value
- **THEN** the returned `*module.Module` has `Metadata` populated, no version-dispatch lookup occurs, and `Package == v`
