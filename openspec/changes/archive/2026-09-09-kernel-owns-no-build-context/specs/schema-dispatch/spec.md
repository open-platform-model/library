## MODIFIED Requirements

### Requirement: Schema Cache memoizes a single Load per instance

The library SHALL expose `opm/schema.Cache` as a struct with at minimum a `Loader Loader` field. `(*Cache).Get() (cue.Value, error)` SHALL invoke `Loader.Load(ctx)` exactly once per `Cache` instance via `sync.Once`-equivalent synchronization, passing a `cue.Context` the Cache creates on first use, owns for its lifetime and never exposes. Subsequent calls — including the call that loses the race — SHALL return the cached `cue.Value` (or the cached error) without re-invoking the Loader. A caller that must compile a value against the schema obtains the schema's context from the returned value (`Value.Context()`).

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

#### Scenario: Two Cache instances do not share state

- **WHEN** two distinct `*Cache` values built from logically-equivalent Loaders are each called with `Get`
- **THEN** each Cache runs its own Load invocation in its own context; populating one does not populate the other

#### Scenario: The schema context is private

- **WHEN** a consumer inspects the exported methods of `Cache`
- **THEN** none returns or accepts a `*cue.Context`, and the schema value's context is reachable only through `Value.Context()`

### Requirement: Cache exposes the resolved schema version

`(*Cache).ResolvedVersion() string` SHALL return the schema module version that the underlying Loader resolved during the first successful Load (e.g., `"v2.0.0-alpha.4"` when the default `opmodel.dev/core@v2` resolved to `v2.0.0-alpha.4`). Before the first successful Load, `ResolvedVersion()` SHALL return the empty string.

#### Scenario: ResolvedVersion is empty before Get

- **WHEN** `cache.ResolvedVersion()` is called before any `cache.Get`
- **THEN** it returns `""`

#### Scenario: ResolvedVersion returns the resolved tag after Get

- **WHEN** `cache.Get()` succeeds against `opmodel.dev/core@v2` resolving to `v2.0.0-alpha.4`
- **THEN** `cache.ResolvedVersion()` returns `"v2.0.0-alpha.4"`

#### Scenario: ResolvedVersion stays empty after failed Load

- **WHEN** `cache.Get()` returns an error on first call
- **THEN** subsequent `cache.ResolvedVersion()` calls return `""`
