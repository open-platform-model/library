## MODIFIED Requirements

### Requirement: OCILoader is the only public Loader

The library SHALL expose `opm/schema.OCILoader` as the sole public implementation of `Loader`. Its struct fields SHALL be exactly `Module string`, `Registry string`, `CacheDir string`. The zero value of `OCILoader` SHALL be a valid Loader.

`OCILoader.Load(ctx)` SHALL:

- Resolve `Module` to `DefaultSchemaModule` when the field is empty.
- Resolve `Registry` to the value derived from `os.Environ`'s `CUE_REGISTRY` when the field is empty.
- Resolve `CacheDir` to the value derived from `os.Environ`'s `CUE_CACHE_DIR` (or CUE's default when that is also empty) when the field is empty.
- Invoke `cuelang.org/go/cue/load.Instances([]string{module}, &load.Config{Env: derivedEnv})` with the resolved values plumbed into `Env`.
- Call `ctx.BuildInstance` on the returned instance and return the resulting `cue.Value` and any error wrapped with context.
- Treat a built value that carries an error (`Value.Err()` non-nil, e.g. an unresolved reference or a conflict in the loaded module) as a load failure: return the zero `cue.Value` and a non-nil error wrapping the build error with a message that identifies the module identifier being loaded, never the errored value with a nil error.

`OCILoader.Load` MUST NOT use any custom OCI client (e.g., `oras-go`), MUST NOT bypass CUE's module cache, and MUST NOT mutate process-global state (no `os.Setenv`).

#### Scenario: Zero-value OCILoader resolves defaults

- **WHEN** `(schema.OCILoader{}).Load(ctx)` is called in an environment with `CUE_REGISTRY` and `CUE_CACHE_DIR` set
- **THEN** the loader resolves `Module` to `schema.DefaultSchemaModule`, threads the env into `load.Config.Env`, and returns a non-zero `cue.Value` containing `#ModuleInstance`

#### Scenario: Explicit overrides take precedence over env

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1", Registry: "opmodel.dev=ghcr.io/open-platform-model", CacheDir: "/tmp/cache"}).Load(ctx)` is called
- **THEN** the registry mapping and cache directory used by `load.Instances` reflect the explicit values regardless of the process environment

#### Scenario: Load failures are wrapped

- **WHEN** `load.Instances` or `BuildInstance` returns an error (e.g., registry unreachable on cache miss, malformed cached module, unknown module path)
- **THEN** `OCILoader.Load` returns the zero `cue.Value` and a non-nil error wrapping the underlying error with a message that identifies the module identifier being loaded

#### Scenario: A module that loads but does not build is a load failure

- **WHEN** `OCILoader.Load` loads a core module whose files load cleanly but whose built value carries an error (an unresolved reference)
- **THEN** it returns the zero `cue.Value` and a non-nil error naming the module identifier and carrying the build error, and the value is never returned with a nil error

### Requirement: Schema Cache memoizes a single Load per instance

The library SHALL expose `opm/schema.Cache` as a struct with at minimum a `Loader Loader` field. `(*Cache).Get() (cue.Value, error)` SHALL invoke `Loader.Load(ctx)` exactly once per `Cache` instance via `sync.Once`-equivalent synchronization, passing a `cue.Context` the Cache creates on first use, owns for its lifetime and never exposes. Subsequent calls — including the call that loses the race — SHALL return the cached `cue.Value` (or the cached error) without re-invoking the Loader. The Cache SHALL NOT retry a failed Load: an error, including one for a schema module whose build failed, stays memoised for the Cache's lifetime, and a caller that must re-fetch constructs a fresh Cache. A caller that must compile a value against the schema obtains the schema's context from the returned value (`Value.Context()`).

The library MUST NOT cache the `Loader`'s result at package scope. There SHALL be no package-level singleton schema value. Each `Cache` owns its memoization and its context.

#### Scenario: Repeated Get returns the cached value

- **WHEN** `cache.Get()` is called twice on the same `*Cache`
- **THEN** both calls return the same `cue.Value` and the underlying `Loader.Load` runs exactly once

#### Scenario: Concurrent first Get is safe

- **WHEN** two goroutines call `cache.Get()` on the same `*Cache` before the cache is warmed
- **THEN** exactly one `Loader.Load` invocation runs and both goroutines receive the same result

#### Scenario: Loader errors are cached

- **WHEN** the first `cache.Get()` returns a non-nil error
- **THEN** subsequent `cache.Get()` calls return the same wrapped error without re-invoking the Loader

#### Scenario: An errored schema build is memoised as an error

- **WHEN** a `Cache` whose `OCILoader` loads a core module that does not build is called with `Get` twice
- **THEN** both calls return the zero `cue.Value` and the same non-nil error, never an errored value with a nil error, and `ResolvedVersion()` stays `""`

#### Scenario: Two Cache instances do not share state

- **WHEN** two distinct `*Cache` values built from logically-equivalent Loaders are each called with `Get`
- **THEN** each Cache runs its own Load invocation in its own context; populating one does not populate the other

#### Scenario: The schema context is private

- **WHEN** a consumer inspects the exported methods of `Cache`
- **THEN** none returns or accepts a `*cue.Context`, and the schema value's context is reachable only through `Value.Context()`
