## ADDED Requirements

### Requirement: Schema Loader interface

The library SHALL define `opm/schema.Loader` as an interface with a single method `Load(ctx *cue.Context) (cue.Value, error)`. Implementations SHALL return a built `cue.Value` representing the OPM core schema package (`#Module`, `#ModuleRelease`, `#Platform`, `#Resource`, `#Trait`, `#ComponentTransformer`, etc. reachable via `LookupPath`).

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

- Resolve `Module` to `"opmodel.dev/core@v0"` when the field is empty.
- Resolve `Registry` to the value derived from `os.Environ`'s `CUE_REGISTRY` when the field is empty.
- Resolve `CacheDir` to the value derived from `os.Environ`'s `CUE_CACHE_DIR` (or CUE's default when that is also empty) when the field is empty.
- Invoke `cuelang.org/go/cue/load.Instances([]string{module}, &load.Config{Env: derivedEnv})` with the resolved values plumbed into `Env`.
- Call `ctx.BuildInstance` on the returned instance and return the resulting `cue.Value` and any error wrapped with context.

`OCILoader.Load` MUST NOT use any custom OCI client (e.g., `oras-go`), MUST NOT bypass CUE's module cache, and MUST NOT mutate process-global state (no `os.Setenv`).

#### Scenario: Zero-value OCILoader resolves defaults

- **WHEN** `(schema.OCILoader{}).Load(ctx)` is called in an environment with `CUE_REGISTRY` and `CUE_CACHE_DIR` set
- **THEN** the loader resolves `Module` to `"opmodel.dev/core@v0"`, threads the env into `load.Config.Env`, and returns a non-zero `cue.Value` containing `#ModuleRelease`

#### Scenario: Explicit overrides take precedence over env

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v0.3.0", Registry: "opmodel.dev=ghcr.io/open-platform-model", CacheDir: "/tmp/cache"}).Load(ctx)` is called
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

`(*Cache).ResolvedVersion() string` SHALL return the schema module version that the underlying Loader resolved during the first successful Load (e.g., `"v0.3.0"` when the default `opmodel.dev/core@v0` resolved to `v0.3.0`). Before the first successful Load, `ResolvedVersion()` SHALL return the empty string.

#### Scenario: ResolvedVersion is empty before Get

- **WHEN** `cache.ResolvedVersion()` is called before any `cache.Get`
- **THEN** it returns `""`

#### Scenario: ResolvedVersion returns the resolved tag after Get

- **WHEN** `cache.Get(ctx)` succeeds against `opmodel.dev/core@v0` resolving to `v0.3.0`
- **THEN** `cache.ResolvedVersion()` returns `"v0.3.0"`

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

- **WHEN** `kernel.New(ctx, kernel.WithSchemaLoader(schema.OCILoader{Module: "opmodel.dev/core@v0.3.0"}))` is called
- **THEN** `k.SchemaCache().Loader` equals the supplied OCILoader value

## MODIFIED Requirements

### Requirement: Single OPM schema, externally resolved, with no apiVersion field

The library SHALL consume exactly one OPM CUE schema package: `opmodel.dev/core@v0` (or a caller-pinned exact version within the v0 major), resolved through CUE's module system against `CUE_REGISTRY`. The library MUST NOT vendor or embed the schema source under `library/apis/core/` or any other in-tree location. The schema package MUST NOT define a top-level `#ApiVersion` constant. Artifact roots (`#Module`, `#ModuleRelease`, `#Component`, `#ComponentTransformer`, `#Platform`, `#Resource`, `#Trait`) MUST NOT carry an `apiVersion` field.

#### Scenario: No in-tree schema source

- **WHEN** the library tree is inspected after this change
- **THEN** no directory `library/apis/` exists
- **AND** no Go file embeds `opmodel.dev/core` source via `//go:embed`

#### Scenario: Schema resolved via module identifier

- **WHEN** the kernel's `*schema.Cache` is populated for the first time
- **THEN** the underlying load goes through `cue/load.Instances` against the configured module identifier (default `"opmodel.dev/core@v0"`)
- **AND** the resolved value's `LookupPath(cue.ParsePath("#ModuleRelease"))` exists

#### Scenario: Evaluated module has no apiVersion field

- **WHEN** an artifact authored against the library schema is loaded and evaluated
- **THEN** `apiVersion` on the artifact root does not exist

## REMOVED Requirements

### Requirement: Cached embedded-schema loader

**Reason:** Schema acquisition no longer goes through an embedded filesystem. The `schema.SchemaValue(ctx)` package function and `schema.EmbeddedSchema()` accessor are removed. Schema loading is now performed by a caller-configurable `Loader` (default `OCILoader`) memoized in a per-Kernel `*schema.Cache`; see the new requirements "Schema Loader interface", "OCILoader is the only public Loader", "Schema Cache memoizes a single Load per instance", and "Cache exposes the resolved schema version" in this delta.

**Migration:** Callers replace `schema.SchemaValue(ctx)` with `k.SchemaCache().Get(ctx)` where `k` is the kernel instance. Callers that previously relied on the package-level singleton MUST hold the Kernel (and its Cache) alive across operations to preserve memoization; in-process cache lifetime is now per-Kernel rather than per-package. Tests that previously called `schema.EmbeddedSchema()` to walk the embed FS MUST switch to the pre-seeded `testdata/cue-cache/` pattern described in the `schema-testing` capability delta. The "No registry round-trip required" scenario no longer applies — the loader now contacts `CUE_REGISTRY` on a cold disk cache.
