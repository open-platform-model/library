## MODIFIED Requirements

### Requirement: MaterializeError Diagnostic

A pull, decode, identity, or indexing failure SHALL surface as a `MaterializeError` carrying a `Kind` discriminator, the subscription path, the attempted version, and the wrapped cause. Exactly one kind exists, `"catalog"`; the library SHALL NOT reserve kinds for failures `Materialize` does not emit.

#### Scenario: Unresolvable subscription path

- **WHEN** a subscribed path cannot be resolved against the registry
- **THEN** Materialize returns a `MaterializeError` with `Kind == "catalog"` naming the subscription path

#### Scenario: Cause is unwrappable

- **WHEN** a `MaterializeError` is returned
- **THEN** the wrapped cause is reachable via `errors.Unwrap`

#### Scenario: One kind constant

- **WHEN** a developer inspects the exported identifiers of `opm/errors`
- **THEN** `MaterializeKindCatalog` exists and `MaterializeKindCoreSchema` does not

## REMOVED Requirements

### Requirement: Opt-In Materialize Cache
**Reason**: `opm/materialize/cache` (the `MaterializeCache` interface, the `LRU` reference implementation and the `Key` derivation) had no consumer. The operator holds one generation-keyed slot, and the CLI relies on CUE's on-disk module cache; neither wanted a content-keyed LRU. The kernel-holds-no-cache posture (Principle I) is unchanged and stays stated on the Materialize method.
**Migration**: A consumer that wants memoization keys on whatever invalidation signal it owns (a CR generation, a file hash) and stores the `*MaterializedPlatform` itself; the library ships no reference cache.
