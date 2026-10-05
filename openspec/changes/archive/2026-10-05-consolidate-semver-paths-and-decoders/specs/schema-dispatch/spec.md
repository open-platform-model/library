## MODIFIED Requirements

### Requirement: Metadata decoders are free functions

The library SHALL decode each artifact's metadata through one unexported free function per artifact, living in the package of its single caller (`opm/module` for the module, `opm/platform` for the platform, `opm/catalog` for the catalog, `opm/kernel` for the instance); `opm/schema` SHALL NOT export a decoder. Each decoder MUST accept a raw `cue.Value` at the artifact root, read it through the `opm/schema` metadata path, and return the canonical decoded metadata struct (`ModuleMetadata`, `InstanceMetadata`, `PlatformMetadata`, `CatalogMetadata`, which stay exported from `opm/schema`) or a non-nil error. Consumers reach decoded metadata only through the artifact constructors and the kernel's acquisition paths (`Module.Metadata`, `Instance.Metadata`, `Platform.Metadata`, `Catalog.Metadata`). A constructor (`NewModuleFromValue`, `NewPlatformFromValue`, `NewCatalogFromValue`) SHALL return its decoder's error without adding a prefix of its own, so a decode failure names its artifact exactly once (`decoding module metadata: ...`, never `decoding module metadata: decoding module metadata: ...`).

#### Scenario: Decoding a module artifact

- **WHEN** `module.NewModuleFromValue(v)` is called with the root of a valid `#Module` value
- **THEN** `Module.Metadata` is a `*schema.ModuleMetadata` with `Name`, `ModulePath`, `Version`, `FQN`, `UUID`, `Labels`, `Annotations` populated

#### Scenario: Decoding a catalog artifact

- **WHEN** `catalog.NewCatalogFromValue(v)` is called with the root of a valid `#Catalog` value
- **THEN** `Catalog.Metadata` is a `*schema.CatalogMetadata` with `ModulePath`, `Version`, `FQN`, `Description`, `Labels`, `Annotations` populated
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

#### Scenario: Decoders are not exported

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `DecodeModuleMetadata`, `DecodeInstanceMetadata`, `DecodePlatformMetadata`, `DecodeCatalogMetadata` exists

#### Scenario: Provider metadata falls back to caller-supplied name

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** neither `DecodeProviderMetadata` nor `ProviderMetadata` exists; the provider artifact was retired with the platform construct and is not among the kinds the kernel accepts

#### Scenario: A decode failure names its artifact once

- **WHEN** `module.NewModuleFromValue(v)`, `platform.NewPlatformFromValue(v)` or `catalog.NewCatalogFromValue(v)` is given a value whose `metadata` field exists but does not decode into the metadata struct
- **THEN** it returns a nil artifact and an error whose text contains `decoding <kind> metadata:` exactly once

### Requirement: Schema Cache memoizes a single Load per instance

The library SHALL expose `opm/schema.Cache` as a struct with at minimum a `Loader Loader` field. `(*Cache).Get() (cue.Value, error)` SHALL invoke `Loader.Load(ctx)` exactly once per `Cache` instance via `sync.Once`-equivalent synchronization, passing a `cue.Context` the Cache creates on first use, owns for its lifetime and never exposes. Subsequent calls — including the call that loses the race — SHALL return the cached `cue.Value` (or the cached error) without re-invoking the Loader. Whatever `Loader` it wraps, the Cache SHALL treat a loaded value that carries an error (`Value.Err()` non-nil) and arrives with a nil error as a load failure: `Get` returns the zero `cue.Value` and a non-nil error that wraps the value's error and calls the loaded schema unusable. That covers both a value whose build failed and the zero `cue.Value` a Loader returns with a nil error; the error SHALL NOT describe the zero value as a build error. The Cache SHALL NOT retry a failed Load: an error, including one for a schema whose build failed, stays memoised for the Cache's lifetime, and a caller that must re-fetch constructs a fresh Cache. A caller that must compile a value against the schema obtains the schema's context from the returned value (`Value.Context()`).

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

- **WHEN** a `Cache` over any `Loader` (an `OCILoader` loading a core module that does not build, or another Loader returning an errored value with a nil error) is called with `Get` twice
- **THEN** both calls return the zero `cue.Value` and the same non-nil error, never an errored value with a nil error, the Loader runs once, and `ResolvedVersion()` stays `""`

#### Scenario: Two Cache instances do not share state

- **WHEN** two distinct `*Cache` values built from logically-equivalent Loaders are each called with `Get`
- **THEN** each Cache runs its own Load invocation in its own context; populating one does not populate the other

#### Scenario: The schema context is private

- **WHEN** a consumer inspects the exported methods of `Cache`
- **THEN** none returns or accepts a `*cue.Context`, and the schema value's context is reachable only through `Value.Context()`

#### Scenario: A zero value from the Loader is an unusable schema

- **WHEN** a `Cache` over a Loader that returns the zero `cue.Value` and a nil error is called with `Get` twice
- **THEN** both calls return the zero `cue.Value` and the same non-nil error saying the loaded schema is unusable, not that it carries a build error, the Loader runs once, and `ResolvedVersion()` stays `""`
