# platform-materialization Specification

## Purpose
TBD - created by archiving change add-platform-materialize. Update Purpose after archive.
## Requirements
### Requirement: Subscription Resolution

The Materialize operation SHALL walk the platform `#registry` (path-keyed `[#ModulePathType]: #Subscription`) and, for each subscription with `enable: true`, resolve the catalog builds selected by its filter against the configured OCI registry. A subscription with `enable: false` SHALL be skipped and contribute no transformers.

#### Scenario: Enabled subscription with no filter

- **WHEN** a platform subscribes to a catalog path with `enable: true` and no `filter`
- **THEN** Materialize selects the highest published SemVer for that path and pulls it
- **AND** the catalog's transformers are indexed

#### Scenario: Disabled subscription skipped

- **WHEN** a subscription sets `enable: false`
- **THEN** Materialize pulls no builds from that path
- **AND** no transformers from that path appear in the materialized index

### Requirement: Version Enumeration and Filtering

For each subscription path, the Materialize operation SHALL enumerate the published versions via the registry's version listing, then narrow the candidate set Go-side in this order (D10): `filter.range` (a SemVer constraint expression) restricts the set, `filter.allow` force-includes specific versions, `filter.deny` force-excludes specific versions. Range expressions SHALL be parsed with a SemVer constraint library because CUE cannot evaluate range syntax. The `v`-prefixed module version form returned by the registry SHALL be normalized against the bare-SemVer catalog FQN form.

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

The Materialize operation SHALL return a `*MaterializedPlatform` whose `Package` answers `LookupPath` for `#composedTransformers`, `#matchers.resources`, and `#matchers.traits`, and which records the resolved version per subscription path for diagnostics. Inputs SHALL NOT be mutated.

#### Scenario: Composed transformers reachable

- **WHEN** Materialize succeeds
- **THEN** `mp.Package.LookupPath("#composedTransformers")` exists and contains the indexed transformers
- **AND** the resolved version for each subscription path is retrievable for diagnostics

#### Scenario: Idempotent and non-mutating

- **WHEN** Materialize is called twice with identical inputs
- **THEN** the returned platforms are semantically identical
- **AND** the input `*platform.Platform` is unchanged after each call

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

