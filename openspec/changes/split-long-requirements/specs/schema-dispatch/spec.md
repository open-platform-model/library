## MODIFIED Requirements

### Requirement: Path inventory exposed as package-level vars
The library SHALL keep every CUE path the kernel's Go code reads on an OPM artifact as one package-level `cue.Path` variable, declared once, and the documentation of its package SHALL name each path's readers. `opm/schema` SHALL export exactly the three paths a frontend reads on an artifact `Package` for which no accessor exists: `Metadata`, `Module` and `CatalogProvides`. The paths and their readers are listed in scenario "The exported paths and their readers".

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

#### Scenario: The exported paths and their readers

- **WHEN** a developer reads the documentation of `opm/schema`
- **THEN** `Metadata`: metadata decoding of every artifact kind (including the module metadata `Instance.ModuleMetadata` decodes), instance processing and the registry loader's identity read.
- **AND** `Module`: the instance's reference to its `#Module`, read by `Instance.ConfigSchema`, `Instance.ModuleMetadata` and values checking at acquire and synthesis.
- **AND** `CatalogProvides`: `Catalog.Provides()`.

## REMOVED Requirements

### Requirement: DefaultSchemaModule constant
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "DefaultSchemaModule names an exact core release", "2.0.0-alpha.13 is the first release that reports contract collisions", "The default release reports the provider count, the comparable report and the contract inventory", "The default release reports the registry shape and the context projection", "The default is not the render floor", "The default advances only through a deliberate change that re-verifies the glue and fixtures", "Doc comments cite the default by the constant's name", "The constant is the only Go source that a default move edits", "No test spells the default release where it means the current core", "Synthetic, floor and skew literals stay literal and are declared frozen", "The schema tests assert the shape of the constant" and "Every served test fixture declares the default core version".

### Requirement: Schema Loader interface
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The Loader interface has a single method Load" and "The library exports no Loader implementation other than OCILoader".

### Requirement: OCILoader is the only public Loader
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "OCILoader is the sole public implementation of Loader", "OCILoader.Load resolves its empty fields", "OCILoader.Load builds the instance and wraps its errors", "OCILoader.Load treats a built value that carries an error as a load failure" and "OCILoader.Load uses no custom OCI client and mutates no process-global state".

### Requirement: A pinned schema release is known without a load
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "OCILoader reports its pinned core release without any I/O" and "A kernel derives the synthesis release from the loader's pin or from its schema cache".

### Requirement: Schema Cache memoizes a single Load per instance
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Cache.Get invokes the Loader exactly once per Cache", "The Cache treats a loaded value that carries an error as a load failure", "The Cache does not retry a failed Load" and "No Loader result is cached at package scope".

### Requirement: Metadata decoders are free functions
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Each artifact's metadata decodes through one unexported free function in the package of its caller", "Each decoder reads a raw cue.Value and returns the canonical metadata struct or an error" and "Consumers reach decoded metadata through constructors and a constructor adds no error prefix".

### Requirement: Front-door docs describe the default core as an exact pin
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Front-door docs state an exact pin for the default core" and "Front-door docs say how an additive schema change reaches a render".

### Requirement: Single OPM schema, externally resolved and pinned by default
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The library consumes one OPM CUE schema resolved through CUE's module system", "The default loader resolves the exact release and the bare major is opt-in", "The library vendors or embeds no schema source" and "The schema defines no top-level ApiVersion and artifact roots carry no apiVersion field".

## ADDED Requirements

### Requirement: DefaultSchemaModule names an exact core release
`schema.DefaultSchemaModule` SHALL name an exact core release, not the floating `opmodel.dev/core@v2` major. It is the release the kernel's render glue, fixtures and parity oracle were verified against. The constant is the only record of which release that is: this specification, the doc comments and the tests name no release as the current default. `OCILoader.Load` with an empty `Module` field SHALL resolve this identifier.

#### Scenario: Empty Module resolves the v2 default

- **WHEN** `(schema.OCILoader{Registry: "opmodel.dev=ghcr.io/open-platform-model"}).Load(ctx)` is called with `Module` unset
- **THEN** the loader resolves `Module` to `schema.DefaultSchemaModule`, threads the env into `load.Config.Env`, and returns a non-zero `cue.Value` containing `#ModuleInstance`

#### Scenario: ResolvedVersion reports the v2 resolution

- **WHEN** `cache.Get()` succeeds against the default
- **THEN** `cache.ResolvedVersion()` returns `schema.DefaultSchemaVersion()`

### Requirement: 2.0.0-alpha.13 is the first release that reports contract collisions
`2.0.0-alpha.13` remains the first release that reports contract collisions on the derived `#Platform.#contracts` inventory (`collisions` and `collidingEntries`, with `routable` false while any exist, and with `defined` and `definedBy` folding only single-definer keys; core change `fold-colliding-contract-keys`). The default's documentation SHALL keep naming `2.0.0-alpha.13` as that release and SHALL NOT attribute those reports to a beta release.

#### Scenario: The documentation names the release of the collision report

- **WHEN** the default's documentation is read
- **THEN** The default's documentation SHALL keep naming `2.0.0-alpha.13` as that release and SHALL NOT attribute those reports to a beta release.

### Requirement: The default release reports the provider count, the comparable report and the contract inventory
`2.0.0-alpha.13` SHALL report contract collisions on top of the per-registry-entry provider count (`providedBy`), the comparable-predicate report (`comparable` and `discriminated`; enhancement 0015), the contract inventory (enhancement 0015: contracts are published as members beside transformers, one provider per contract stands, an unfulfilled contract is reported and only over-subscription refuses platform-package generation).

#### Scenario: The default reports the earlier inventories

- **WHEN** the default release is read
- **THEN** the per-registry-entry provider count (`providedBy`), the comparable-predicate report (`comparable` and `discriminated`; enhancement 0015)

### Requirement: The default release reports the registry shape and the context projection
`2.0.0-alpha.13` SHALL report contract collisions on top of the registry shape (enhancement 0019: a platform imports its catalog and embeds it whole into the registry entry) and the context projection (enhancement 0019: the kernel fills `#runtimeName` alone).

#### Scenario: The default reports the registry shape

- **WHEN** the default release is read
- **THEN** the registry shape (enhancement 0019: a platform imports its catalog and embeds it whole into the registry entry)

### Requirement: The default is not the render floor
The default is not the render floor: `Kernel.Render` and `Platform.Contracts()` SHALL accept every core from `schema.ProvidedBySince` (`2.0.0-alpha.12`) on, and a platform pinning a release between the floor and `2.0.0-alpha.13` decodes an absent collision report as no collision.

#### Scenario: A platform between the floor and the default renders

- **WHEN** a platform pins a release between the floor and `2.0.0-alpha.13`
- **THEN** decodes an absent collision report as no collision

### Requirement: The default advances only through a deliberate change that re-verifies the glue and fixtures
The constant SHALL advance only through a deliberate change that re-verifies the glue and fixtures against the new release; a default that floats ahead of the glue breaks every synthesized artifact on a cold cache. The deliberate re-verification is the review of the change that advances it.

#### Scenario: The default advances through a reviewed change

- **WHEN** the default advances
- **THEN** The constant advances only through a deliberate change that re-verifies the glue and fixtures against the new release

### Requirement: Doc comments cite the default by the constant's name
Doc comments that cite the default module identifier (`opm/kernel`, `opm/schema`) SHALL cite it by the constant's name (`DefaultSchemaModule`), never by the floating major, and SHALL NOT present a literal release as the current default; a release literal in a doc comment is a format example only, so a default move edits no doc comment.

#### Scenario: No doc comment cites a deleted package or the floating major

- **WHEN** a developer searches `opm/` for the default module identifier
- **THEN** every citation names the constant `DefaultSchemaModule` (or, on the constant's own line, the exact release it holds), none names the floating major as the default, and none lives in `opm/materialize`, which no longer exists

### Requirement: The constant is the only Go source that a default move edits
The constant SHALL be the only Go source that the default move edits. The fixture harness's declared core version SHALL be derived from it, never kept as a second literal. Advancing the default therefore edits the constant and the fixture `cue.mod` pins, and no test file and no doc comment.

#### Scenario: A default move edits no test file

- **WHEN** `DefaultSchemaModule` advances to a newer published core and the fixture `cue.mod` pins are re-pinned with it
- **THEN** the suite passes with no `*_test.go` edited, and the only Go change is the constant

### Requirement: No test spells the default release where it means the current core
No test SHALL spell the default release where it means the current core: a test that expects the current core (a resolved-version row, a module file it authors beside the served fixtures, a staged module file built beside them) SHALL read the harness's declared core version.

#### Scenario: A test reads the current core from the harness

- **WHEN** a test expects the current core
- **THEN** a test that expects the current core (a resolved-version row, a module file it authors beside the served fixtures, a staged module file built beside them) SHALL read the harness's declared core version

### Requirement: Synthetic, floor and skew literals stay literal and are declared frozen
Synthetic, floor and skew literals that happen to equal or predate the default are not "the current core"; they SHALL stay literal and are declared in `.cascade-frozen` (capability `fixture-pin-maintenance`). Test files that keep an older or synthetic core literal on purpose are declared as frozen (capability `fixture-pin-maintenance`).

#### Scenario: A synthetic literal is declared frozen

- **WHEN** a test file keeps an older or synthetic core literal on purpose
- **THEN** Test files that keep an older or synthetic core literal on purpose are declared as frozen (capability `fixture-pin-maintenance`).

### Requirement: The schema tests assert the shape of the constant
The schema tests SHALL assert the constant's shape: an exact release on the `v2` major whose version suffix is `DefaultSchemaVersion()`. They SHALL NOT repeat the release as a literal.

#### Scenario: The pin assertion checks shape, not the release

- **WHEN** the schema tests check the default
- **THEN** they assert that the constant names an exact release on the `v2` major and that `DefaultSchemaVersion()` is its version suffix, and no assertion spells that release as a literal

### Requirement: Every served test fixture declares the default core version
The core version every served test fixture declares SHALL be the same release, so a fixture never pins a core the default kernel does not render against. A test that exercises an older core copies a fixture and re-pins the copy. It never commits a fixture pinning another release, and it reads the fixture's current pin from the fixture harness's declared core version rather than from a literal.

#### Scenario: Served fixtures pin the default release

- **WHEN** a test authors a module beside the served fixtures with the fixture harness's declared core version
- **THEN** that version equals the release `DefaultSchemaModule` pins, and the render fixtures under `testdata/render` declare the same release in every `cue.mod`

#### Scenario: An older-core test survives a default move

- **WHEN** the default core release advances and every served fixture is re-pinned with it
- **THEN** a test that copies a served fixture and re-pins the copy to an older core still finds the fixture's pin through the harness's declared core version and needs no edit of its own

### Requirement: The Loader interface has a single method Load
The library SHALL define `opm/schema.Loader` as an interface with a single method `Load(ctx *cue.Context) (cue.Value, error)`. Implementations SHALL return a built `cue.Value` representing the OPM core schema package (`#Module`, `#ModuleInstance`, `#Platform`, `#Resource`, `#Trait`, `#ComponentTransformer`, etc. reachable via `LookupPath`).

#### Scenario: Loader is the contract not the implementation

- **WHEN** Go code declares a variable of type `schema.Loader`
- **THEN** any value implementing `Load(*cue.Context) (cue.Value, error)` satisfies the type
- **AND** the public package documents `OCILoader` as the canonical implementation

### Requirement: The library exports no Loader implementation other than OCILoader
The library SHALL NOT export any `Loader` implementation other than `OCILoader` (see "OCILoader is the sole public implementation of Loader" requirement). Internal-only or test-only `Loader` implementations MUST NOT appear in the public API surface.

#### Scenario: No second public Loader

- **WHEN** a consumer enumerates `opm/schema` package symbols
- **THEN** `OCILoader` is the only exported type satisfying `Loader`

### Requirement: OCILoader is the sole public implementation of Loader
The library SHALL expose `opm/schema.OCILoader` as the sole public implementation of `Loader`. Its struct fields SHALL be exactly `Module string`, `Registry string`, `CacheDir string`. The zero value of `OCILoader` SHALL be a valid Loader.

#### Scenario: Zero-value OCILoader resolves defaults

- **WHEN** `(schema.OCILoader{}).Load(ctx)` is called in an environment with `CUE_REGISTRY` and `CUE_CACHE_DIR` set
- **THEN** the loader resolves `Module` to `schema.DefaultSchemaModule`, threads the env into `load.Config.Env`, and returns a non-zero `cue.Value` containing `#ModuleInstance`

### Requirement: OCILoader.Load resolves its empty fields
`OCILoader.Load(ctx)` SHALL:
- Resolve `Module` to `DefaultSchemaModule` when the field is empty.
- Resolve `Registry` to the value derived from `os.Environ`'s `CUE_REGISTRY` when the field is empty.
- Resolve `CacheDir` to the value derived from `os.Environ`'s `CUE_CACHE_DIR` (or CUE's default when that is also empty) when the field is empty.

#### Scenario: Explicit overrides take precedence over env

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1", Registry: "opmodel.dev=ghcr.io/open-platform-model", CacheDir: "/tmp/cache"}).Load(ctx)` is called
- **THEN** the registry mapping and cache directory used by `load.Instances` reflect the explicit values regardless of the process environment

### Requirement: OCILoader.Load builds the instance and wraps its errors
`OCILoader.Load(ctx)` SHALL:
- Invoke `cuelang.org/go/cue/load.Instances([]string{module}, &load.Config{Env: derivedEnv})` with the resolved values plumbed into `Env`.
- Call `ctx.BuildInstance` on the returned instance and return the resulting `cue.Value` and any error wrapped with context.

#### Scenario: Load failures are wrapped

- **WHEN** `load.Instances` or `BuildInstance` returns an error (e.g., registry unreachable on cache miss, malformed cached module, unknown module path)
- **THEN** `OCILoader.Load` returns the zero `cue.Value` and a non-nil error wrapping the underlying error with a message that identifies the module identifier being loaded

### Requirement: OCILoader.Load treats a built value that carries an error as a load failure
`OCILoader.Load(ctx)` SHALL:
- Treat a built value that carries an error (`Value.Err()` non-nil, e.g. an unresolved reference or a conflict in the loaded module) as a load failure: return the zero `cue.Value` and a non-nil error wrapping the build error with a message that identifies the module identifier being loaded, never the errored value with a nil error.

#### Scenario: A module that loads but does not build is a load failure

- **WHEN** `OCILoader.Load` loads a core module whose files load cleanly but whose built value carries an error (an unresolved reference)
- **THEN** it returns the zero `cue.Value` and a non-nil error naming the module identifier and carrying the build error, and the value is never returned with a nil error

### Requirement: OCILoader.Load uses no custom OCI client and mutates no process-global state
`OCILoader.Load` MUST NOT use any custom OCI client (e.g., `oras-go`), MUST NOT bypass CUE's module cache, and MUST NOT mutate process-global state (no `os.Setenv`).

#### Scenario: Load mutates no process-global state

- **WHEN** `OCILoader.Load` runs
- **THEN** MUST NOT mutate process-global state (no `os.Setenv`)

### Requirement: OCILoader reports its pinned core release without any I/O
`OCILoader` SHALL report, without any I/O, the exact core release its module identifier names: `PinnedVersion()` returns that release and `true` when `Module` (or `DefaultSchemaModule` when `Module` is empty) names a full release, and `("", false)` when it names a bare major.

#### Scenario: Default loader is pinned

- **WHEN** `(schema.OCILoader{}).PinnedVersion()` is called
- **THEN** it returns `DefaultSchemaVersion()` and `true`, with no registry or cache access

#### Scenario: Bare major is not pinned

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v2"}).PinnedVersion()` is called
- **THEN** it returns `""` and `false`

### Requirement: A kernel derives the synthesis release from the loader's pin or from its schema cache
A kernel whose configured loader pins an exact release SHALL derive the core release its synthesized instances import from that pin and SHALL NOT load the schema to do so; a kernel whose loader pins no exact release (a bare-major `OCILoader`, or any `Loader` that is not an `OCILoader`) SHALL resolve the release through its schema cache.

#### Scenario: Synthesis on a pinned kernel loads no schema

- **WHEN** a kernel constructed with the default loader synthesizes an instance
- **THEN** the synthesized package imports core at the pinned release's major, and `k.SchemaCache().ResolvedVersion()` is still `""` afterwards because no load ran

#### Scenario: Synthesis on a bare-major kernel resolves through the cache

- **WHEN** a kernel constructed with `WithSchemaLoader(schema.OCILoader{Module: "opmodel.dev/core@v2"})` synthesizes an instance
- **THEN** the schema cache is loaded once and the synthesized package imports core at the resolved release's major

### Requirement: Cache.Get invokes the Loader exactly once per Cache
The library SHALL expose `opm/schema.Cache` as a struct with at minimum a `Loader Loader` field. `(*Cache).Get() (cue.Value, error)` SHALL invoke `Loader.Load(ctx)` exactly once per `Cache` instance via `sync.Once`-equivalent synchronization, passing a `cue.Context` the Cache creates on first use, owns for its lifetime and never exposes. Subsequent calls — including the call that loses the race — SHALL return the cached `cue.Value` (or the cached error) without re-invoking the Loader.

#### Scenario: Repeated Get returns the cached value

- **WHEN** `cache.Get()` is called twice on the same `*Cache`
- **THEN** both calls return the same `cue.Value` and the underlying `Loader.Load` runs exactly once

#### Scenario: Concurrent first Get is safe

- **WHEN** two goroutines call `cache.Get()` on the same `*Cache` before the cache is warmed
- **THEN** exactly one `Loader.Load` invocation runs and both goroutines receive the same result

### Requirement: The Cache treats a loaded value that carries an error as a load failure
Whatever `Loader` it wraps, the Cache SHALL treat a loaded value that carries an error (`Value.Err()` non-nil) and arrives with a nil error as a load failure: `Get` returns the zero `cue.Value` and a non-nil error that wraps the value's error and calls the loaded schema unusable. That covers both a value whose build failed and the zero `cue.Value` a Loader returns with a nil error; the error SHALL NOT describe the zero value as a build error.

#### Scenario: A zero value from the Loader is an unusable schema

- **WHEN** a `Cache` over a Loader that returns the zero `cue.Value` and a nil error is called with `Get` twice
- **THEN** both calls return the zero `cue.Value` and the same non-nil error saying the loaded schema is unusable, not that it carries a build error, the Loader runs once, and `ResolvedVersion()` stays `""`

### Requirement: The Cache does not retry a failed Load
The Cache SHALL NOT retry a failed Load: an error, including one for a schema whose build failed, stays memoised for the Cache's lifetime, and a caller that must re-fetch constructs a fresh Cache. A caller that must compile a value against the schema obtains the schema's context from the returned value (`Value.Context()`).

#### Scenario: Loader errors are cached

- **WHEN** the first `cache.Get()` returns a non-nil error
- **THEN** subsequent `cache.Get()` calls return the same wrapped error without re-invoking the Loader

#### Scenario: An errored schema build is memoised as an error

- **WHEN** a `Cache` over any `Loader` (an `OCILoader` loading a core module that does not build, or another Loader returning an errored value with a nil error) is called with `Get` twice
- **THEN** both calls return the zero `cue.Value` and the same non-nil error, never an errored value with a nil error, the Loader runs once, and `ResolvedVersion()` stays `""`

#### Scenario: The schema context is private

- **WHEN** a consumer inspects the exported methods of `Cache`
- **THEN** none returns or accepts a `*cue.Context`, and the schema value's context is reachable only through `Value.Context()`

### Requirement: No Loader result is cached at package scope
The library MUST NOT cache the `Loader`'s result at package scope. There SHALL be no package-level singleton schema value. Each `Cache` owns its memoization and its context.

#### Scenario: Two Cache instances do not share state

- **WHEN** two distinct `*Cache` values built from logically-equivalent Loaders are each called with `Get`
- **THEN** each Cache runs its own Load invocation in its own context; populating one does not populate the other

### Requirement: Every other path lives in opm/internal/corepath
Every other path SHALL live in the internal package `opm/internal/corepath`, where no consumer can import or reassign it; a frontend reaches those fields through the artifact accessors (`Instance.Components`, `Instance.Values`, `Instance.ConfigSchema`, `Module.ConfigSchema`, `Module.DebugValues`, `Platform.Contracts`, `Catalog.Provides`). The paths and their readers are listed in scenario "The internal paths and their readers". The inventory SHALL be exactly these paths.

#### Scenario: The internal paths and their readers

- **WHEN** a developer reads the documentation of `opm/internal/corepath`
- **THEN** `Components`: `Instance.Components`.
- **AND** `Values`: `Instance.Values`, the values check and conflict attribution of an instance build, and the top-level `values:` unwrap of a file-backed values source.
- **AND** `Config`: `Module.ConfigSchema`, `Instance.ConfigSchema`, and values checking against `#config` at acquire and synthesis.
- **AND** `DebugValues`: `Module.DebugValues`, the documented frontend read of a module's debug overlay.
- **AND** `Contracts`: the contract-inventory decode `NewPlatformFromValue` records at construction, which `Platform.Contracts()` returns (on a `Platform` the constructor did not build, the first `Contracts()` or `CoreFloor()` call runs the same decode).
- **AND** `ContractsProvidedBy`: the core-floor presence test `NewPlatformFromValue` records at construction, which `Platform.CoreFloor()` reports and `Kernel.Render` checks before staging. The inventory decode reads the same field relative to `Contracts`, and the render glue reads it in CUE.
- **AND** `ContractsCollisions` and `ContractsCollidingEntries`: they name fields that the inventory decode reads relative to `Contracts` and the render glue reads in CUE. In Go only tests read the variables, which document the collision report.
- **AND** `Transformers`: `Catalog.Provides()` reads it on both of its paths, to refuse an unevaluated `#transformers`, and the deprecated provider-set fold, for a catalog built against a core older than `schema.ProvidesSince`, reads every transformer through it.
- **AND** `RequiredResources`, `RequiredTraits` and `Fulfilment`: the deprecated fold reads them relative to a transformer and to its demand entries, not from an artifact root.

### Requirement: opm/schema exports the core-release constants ProvidedBySince and ProvidesSince
`opm/schema` SHALL export the core-release constants `ProvidedBySince` (the render floor) and `ProvidesSince` (which path `Catalog.Provides` takes), and SHALL NOT export a constant for the first core release that reports contract collisions: no code reads one.

#### Scenario: The core-release constants are exported

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** `opm/schema` SHALL export the core-release constants `ProvidedBySince` (the render floor) and `ProvidesSince` (which path `Catalog.Provides` takes)

### Requirement: The paths of the retired matcher, executor and context builder stay removed
The paths the retired Go matcher, executor and context builder read (listed in scenario "Matcher and transformer paths are gone") and `ModuleMetadataPath` SHALL stay removed: the render build reads the instance's components and the platform's `#composedTransformers` and `#contracts` in CUE, inside the generated glue. A path that no kernel code reads, by variable or relative to an inventory path, is removed, not retained for a possible consumer.

#### Scenario: A path no kernel code reads is removed

- **WHEN** no kernel code reads a path, by variable or relative to an inventory path
- **THEN** A path that no kernel code reads, by variable or relative to an inventory path, is removed, not retained for a possible consumer.

### Requirement: Each artifact's metadata decodes through one unexported free function in the package of its caller
The library SHALL decode each artifact's metadata through one unexported free function per artifact, living in the package of its single caller (`opm/module` for the module, `opm/platform` for the platform, `opm/catalog` for the catalog, `opm/kernel` for the instance); `opm/schema` SHALL NOT export a decoder.

#### Scenario: Decoders are not exported

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `DecodeModuleMetadata`, `DecodeInstanceMetadata`, `DecodePlatformMetadata`, `DecodeCatalogMetadata` exists

#### Scenario: Provider metadata falls back to caller-supplied name

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** neither `DecodeProviderMetadata` nor `ProviderMetadata` exists; the provider artifact was retired with the platform construct and is not among the kinds the kernel accepts

### Requirement: Each decoder reads a raw cue.Value and returns the canonical metadata struct or an error
Each decoder MUST accept a raw `cue.Value` at the artifact root, read it through the `opm/schema` metadata path, and return the canonical decoded metadata struct (`module.ModuleMetadata`, `module.InstanceMetadata`, `platform.PlatformMetadata`, `catalog.CatalogMetadata`, each declared once, in the package of the artifact it describes; `opm/schema` SHALL NOT declare or re-export a metadata type) or a non-nil error.

#### Scenario: Decoding a module artifact

- **WHEN** `module.NewModuleFromValue(v)` is called with the root of a valid `#Module` value
- **THEN** `Module.Metadata` is a `*module.ModuleMetadata` with `Name`, `ModulePath`, `Version`, `FQN`, `UUID`, `Labels`, `Annotations` populated

#### Scenario: Decoding a catalog artifact

- **WHEN** `catalog.NewCatalogFromValue(v)` is called with the root of a valid `#Catalog` value
- **THEN** `Catalog.Metadata` is a `*catalog.CatalogMetadata` with `ModulePath`, `Version`, `FQN`, `Description`, `Labels`, `Annotations` populated
- **AND** it carries no `Name`, because a catalog's identity is its module path and the version stamped on every member it ships

#### Scenario: Platform metadata hoists top-level type

- **WHEN** `platform.NewPlatformFromValue(v)` is called on a `#Platform` whose root has `type: "kubernetes"` alongside its `metadata` block
- **THEN** the returned `Platform.Metadata.Type` is `"kubernetes"`

#### Scenario: Each metadata type has one name

- **WHEN** a developer inspects the exported identifiers of `opm/schema`, `opm/module`, `opm/platform` and `opm/catalog`
- **THEN** `ModuleMetadata` and `InstanceMetadata` exist only in `opm/module`, `PlatformMetadata` only in `opm/platform` and `CatalogMetadata` only in `opm/catalog`, and none of them is a type alias

### Requirement: Consumers reach decoded metadata through constructors and a constructor adds no error prefix
Consumers reach decoded metadata only through the artifact constructors and the kernel's acquisition paths (`Module.Metadata`, `Instance.Metadata`, `Platform.Metadata`, `Catalog.Metadata`). A constructor (`NewModuleFromValue`, `NewPlatformFromValue`, `NewCatalogFromValue`) SHALL return its decoder's error without adding a prefix of its own, so a decode failure names its artifact exactly once (`decoding module metadata: ...`, never `decoding module metadata: decoding module metadata: ...`).

#### Scenario: Missing metadata is fatal for module/instance/platform

- **WHEN** a module or platform constructor, or the kernel's instance processing, is given a value whose `metadata` field is absent
- **THEN** it returns an error stating "metadata field is required" and no partial artifact

#### Scenario: Missing metadata is fatal for a catalog

- **WHEN** `catalog.NewCatalogFromValue(v)` is given a value whose `metadata` field is absent
- **THEN** it returns an error stating "metadata field is required" and a nil catalog, on the same terms as the other three

#### Scenario: A decode failure names its artifact once

- **WHEN** `module.NewModuleFromValue(v)`, `platform.NewPlatformFromValue(v)` or `catalog.NewCatalogFromValue(v)` is given a value whose `metadata` field exists but does not decode into the metadata struct
- **THEN** it returns a nil artifact and an error whose text contains `decoding <kind> metadata:` exactly once

### Requirement: Front-door docs state an exact pin for the default core
Where `README.md` or `AGENTS.md` states which core the default schema loader uses, it SHALL state an exact pin (`schema.DefaultSchemaModule`) with the bare major (`opmodel.dev/core@v2`) as opt-in, or link to this spec. Neither SHALL present floating-major resolution as the mechanism that absorbs additive schema changes.

#### Scenario: The API stability section says pinned by default

- **WHEN** a developer reads `README.md` § API stability
- **THEN** where it says which core the default loader uses, it states an exact core release with a bare major opt-in, or links to this spec
- **AND** it does not say that additive shape changes are absorbed by floating-major resolution

#### Scenario: The schema versioning section says pinned by default

- **WHEN** a developer reads `AGENTS.md` § OPM schema versioning
- **THEN** where it says which core the default loader uses, it names `schema.DefaultSchemaModule` as an exact-release default and the bare major as opt-in, or links to this spec
- **AND** neither `README.md` nor `AGENTS.md` uses the word "floating"

### Requirement: Front-door docs say how an additive schema change reaches a render
Where `README.md` or `AGENTS.md` says that an additive schema change within a major needs no Go API change, it SHALL say how the change reaches a render: artifacts re-pin core in their own `cue.mod`, and `DefaultSchemaModule` moves separately, in the release cascade's `fix(deps)` PR, which re-verifies the render glue.

#### Scenario: An additive change is explained

- **WHEN** `README.md` or `AGENTS.md` says that an additive schema change within a major needs no Go API change
- **THEN** it SHALL say how the change reaches a render

### Requirement: The library consumes one OPM CUE schema resolved through CUE's module system
The library SHALL consume exactly one OPM CUE schema package: `opmodel.dev/core@v2` (or a caller-pinned exact version, in any major, via `OCILoader.Module`), resolved through CUE's module system against `CUE_REGISTRY`.

#### Scenario: Schema resolved via the default module identifier

- **WHEN** the kernel's `*schema.Cache` is populated for the first time
- **THEN** the underlying load goes through `cue/load.Instances` against the configured module identifier (default `schema.DefaultSchemaModule`, an exact release)
- **AND** the resolved value's `LookupPath(cue.ParsePath("#ModuleInstance"))` exists

#### Scenario: Caller-pinned earlier major still loads

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1"}).Load(ctx)` is called
- **THEN** the loader resolves exactly that version and returns its schema value
- **AND** no code path upgrades or rewrites the caller's pin

### Requirement: The default loader resolves the exact release and the bare major is opt-in
The default loader SHALL resolve the exact release `schema.DefaultSchemaModule` names; the bare major `opmodel.dev/core@v2` SHALL be opt-in, through `OCILoader.Module`.

#### Scenario: Default resolves the pinned release

- **WHEN** `(schema.OCILoader{}).Load(ctx)` runs with no `Module` override
- **THEN** the loader loads exactly the release `schema.DefaultSchemaModule` names, with no bare-major expansion

#### Scenario: Bare major resolves within the v2 major (opt-in)

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v2"}).Load(ctx)` is called
- **THEN** the bare major is expanded through the loader's bare-major mechanism and resolves to the highest published version within the v2 major
- **AND** SemVer prerelease ordering applies, so a `v2.0.0-0.dev.*` snapshot tag never outranks a `v2.0.0-alpha.N` tag

### Requirement: The library vendors or embeds no schema source
The library MUST NOT vendor or embed the schema source under `library/apis/core/` or any other in-tree location.

#### Scenario: No in-tree schema source

- **WHEN** the library tree is inspected after this change
- **THEN** no directory `library/apis/` exists
- **AND** no Go file embeds `opmodel.dev/core` source via `//go:embed`

### Requirement: The schema defines no top-level ApiVersion and artifact roots carry no apiVersion field
The schema package MUST NOT define a top-level `#ApiVersion` constant. Artifact roots (`#Module`, `#ModuleInstance`, `#Component`, `#ComponentTransformer`, `#Platform`, `#Resource`, `#Trait`) MUST NOT carry an `apiVersion` field.

#### Scenario: Evaluated module has no apiVersion field

- **WHEN** an artifact authored against the library schema is loaded and evaluated
- **THEN** `apiVersion` on the artifact root does not exist
