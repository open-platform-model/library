# platform-materialization Specification

## Purpose
TBD - created by archiving change add-platform-materialize. Update Purpose after archive.
## Requirements
### Requirement: Subscription Resolution

The Materialize operation SHALL walk the platform `#registry` (path-keyed `[#ModulePathType]: #Subscription`) and, for each subscription with `enable: true`, resolve the catalog builds selected by its filter against the configured OCI registry. A subscription with `enable: false` SHALL be skipped and contribute no transformers.

An enabled subscription with no `filter` SHALL select the highest published **stable** (non-pre-release) SemVer for that path. Pre-release versions (those carrying a SemVer pre-release identifier, e.g. `0.6.0-dev.*`) SHALL NOT be selected by the no-filter default. When a path has published *only* pre-release versions, the no-filter default SHALL fall back to the highest published version so the path still materializes.

#### Scenario: Enabled subscription with no filter

- **WHEN** a platform subscribes to a catalog path with `enable: true` and no `filter`
- **THEN** Materialize selects the highest published stable SemVer for that path and pulls it
- **AND** the catalog's transformers are indexed

#### Scenario: Pre-release excluded from the no-filter default

- **WHEN** a path has published `0.5.0`, `0.5.1`, and a pre-release `0.6.0-dev.1` and the subscription has no `filter`
- **THEN** Materialize selects `0.5.1`
- **AND** does not select `0.6.0-dev.1`

#### Scenario: Pre-release-only path falls back

- **WHEN** every published version for a no-filter subscription is a pre-release
- **THEN** Materialize selects the highest published pre-release so the path still materializes

#### Scenario: Disabled subscription skipped

- **WHEN** a subscription sets `enable: false`
- **THEN** Materialize pulls no builds from that path
- **AND** no transformers from that path appear in the materialized index

### Requirement: Version Enumeration and Filtering

For each subscription path, the Materialize operation SHALL enumerate the published versions via the registry's version listing, then narrow the candidate set Go-side in this order (D10): `filter.range` (a SemVer constraint expression) restricts the set, `filter.allow` force-includes specific versions, `filter.deny` force-excludes specific versions. Range expressions SHALL be parsed with a SemVer constraint library because CUE cannot evaluate range syntax. The `v`-prefixed module version form returned by the registry SHALL be normalized against the bare-SemVer catalog FQN form.

Pre-release versions are selectable only by explicit opt-in: a `filter.allow` entry naming the exact pre-release version, or a `filter.range` whose constraint itself contains a pre-release identifier (standard SemVer-constraint semantics, under which an ordinary constraint does not admit pre-releases). The no-filter default never admits a pre-release except via the pre-release-only fallback above.

#### Scenario: Range restricts the candidate set

- **WHEN** a subscription filter is `range: ">=0.1.0 <0.2.0"` and the path has published `0.1.0`, `0.1.1`, `0.2.0`
- **THEN** Materialize selects `0.1.0` and `0.1.1`
- **AND** excludes `0.2.0`

#### Scenario: Deny excludes an in-range version

- **WHEN** `filter.deny` lists a version that `filter.range` would otherwise admit
- **THEN** the denied version is absent from the survivor set

#### Scenario: Allow includes an out-of-range version

- **WHEN** `filter.allow` lists a version outside `filter.range`
- **THEN** the allowed version is present in the survivor set

#### Scenario: Allow opts a pre-release in

- **WHEN** `filter.allow` names an exact published pre-release version
- **THEN** that pre-release is present in the survivor set even though the no-filter default would exclude it

### Requirement: Transformer Indexing

For each selected catalog build, the Materialize operation SHALL read the build's `#Catalog.#transformers` map and index every entry by its stamped FQN into a composed transformer map, and SHALL build a `#matchers.{resources,traits}` reverse index mapping each required/optional primitive FQN to the list of transformers that reference it.

#### Scenario: Reverse index populated

- **WHEN** a selected catalog exposes a transformer requiring resource FQN `<r>`
- **THEN** the materialized `#matchers.resources[<r>]` list contains that transformer

#### Scenario: Identical builds collapse

- **WHEN** two selected builds expose byte-identical transformer bodies at the same FQN
- **THEN** they collapse to a single composed-map entry via CUE unification

#### Scenario: Divergent builds conflict

- **WHEN** two selected builds expose divergent transformer bodies at the same FQN
- **THEN** Materialize returns a `MaterializeError` wrapping the CUE conflict

### Requirement: MaterializedPlatform Output Shape

The Materialize operation SHALL return a `*MaterializedPlatform` whose `Package` answers `LookupPath` for `#composedTransformers`, `#matchers.resources`, and `#matchers.traits`, and which records the resolved version per subscription path for diagnostics. Inputs SHALL NOT be mutated. The `Package` SHALL be consumed by the compile pipeline as **read-only input** — the `FillPath` argument / cross-read source — and SHALL NOT be used as the owner of the compile build context (which is sourced from the caller Kernel; see the `kernel-runtime` capability).

Under the v0.17 CUE toolchain, the returned `*MaterializedPlatform` SHALL be safe for concurrent **read-only** consumption: a platform materialized once MAY be rendered against simultaneously by multiple Kernels' compile pipelines (one Kernel per goroutine) without a mutex and without re-materialization. This is the basis of the materialize-once-reuse-many model the Platform-CR design depends on. Concurrent consumers SHALL NOT mutate the shared `Package`; the pipeline only looks up and fills *from* it, building results in each caller Kernel's own context.

#### Scenario: Composed transformers reachable

- **WHEN** Materialize succeeds
- **THEN** `mp.Package.LookupPath("#composedTransformers")` exists and contains the indexed transformers
- **AND** the resolved version for each subscription path is retrievable for diagnostics

#### Scenario: Idempotent and non-mutating

- **WHEN** Materialize is called twice with identical inputs
- **THEN** the returned platforms are semantically identical
- **AND** the input `*platform.Platform` is unchanged after each call

#### Scenario: Consumed as read-only compile input

- **WHEN** `Kernel.Compile` renders a release against a materialized platform
- **THEN** the platform's `Package` is read (looked up and filled-from) but the build context for the rendered output comes from the caller Kernel, not from `mp.Package.Context()`

#### Scenario: Safe for concurrent read-only sharing across Kernels

- **WHEN** a single `*MaterializedPlatform` is materialized once and then rendered against concurrently by multiple per-goroutine Kernels under the v0.17 toolchain
- **THEN** the shared `Package` is consumed read-only by every concurrent compile without mutation or re-materialization
- **AND** no data race occurs and each render produces the output correct for its own release

### Requirement: MaterializeError Diagnostic

A pull, decode, or indexing failure SHALL surface as a `MaterializeError` carrying a `Kind` discriminator (`"catalog"` for subscription failures; `"core-schema"` reserved for schema-load failures per D24), the subscription path, the attempted version, and the wrapped cause.

#### Scenario: Unresolvable subscription path

- **WHEN** a subscribed path cannot be resolved against the registry
- **THEN** Materialize returns a `MaterializeError` with `Kind == "catalog"` naming the subscription path

#### Scenario: Cause is unwrappable

- **WHEN** a `MaterializeError` is returned
- **THEN** the wrapped cause is reachable via `errors.Unwrap`

### Requirement: Opt-In Materialize Cache

The library SHALL provide a `opm/materialize/cache` package exposing a `MaterializeCache` interface (`Get(key string) (*MaterializedPlatform, bool)` and `Put(key string, mp *MaterializedPlatform)`), a reference implementation, and a key-derivation helper over the platform `#registry` subtree. The `Kernel` SHALL NOT hold a materialize cache; consumers wire their own.

#### Scenario: Reference cache round-trips

- **WHEN** a consumer constructs the reference cache and `Put`s a materialized platform under a derived key
- **THEN** a subsequent `Get` with the same key returns it

#### Scenario: Kernel holds no cache

- **WHEN** a developer inspects the `Kernel` struct
- **THEN** it has no materialize-cache field

