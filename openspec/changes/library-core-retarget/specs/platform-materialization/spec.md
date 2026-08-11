# platform-materialization — Delta

## MODIFIED Requirements

### Requirement: Subscription Resolution

The Materialize operation SHALL walk the platform `#registry` (path-keyed `[#ModulePathType]: #Subscription`) and, for each subscription with `enable: true`, resolve the catalog builds selected by its filter against the configured OCI registry. A subscription with `enable: false` SHALL be skipped and contribute no transformers.

A subscription key MAY carry a major suffix (`path@vN`, the core-v2 `#ModulePathType` form). A major-suffixed key SHALL scope every step of resolution — version enumeration, filtering, and selection — to published versions within major `N`, and the suffix SHALL be split off before the key participates in load-instance composition (a `path@vN@vX.Y.Z` load ID is invalid). A major-free key SHALL resolve against the path's full published version list, exactly as before this change.

An enabled subscription with no `filter` SHALL select the highest published **stable** (non-pre-release) SemVer within its resolution scope. Pre-release versions (those carrying a SemVer pre-release identifier, e.g. `0.6.0-dev.*`) SHALL NOT be selected by the no-filter default. When the resolution scope contains *only* pre-release versions, the no-filter default SHALL fall back to the highest version in scope so the path still materializes.

#### Scenario: Enabled subscription with no filter

- **WHEN** a platform subscribes to a catalog path with `enable: true` and no `filter`
- **THEN** Materialize selects the highest published stable SemVer within the subscription's resolution scope and pulls it
- **AND** the catalog's transformers are indexed

#### Scenario: Major-suffixed key scopes resolution to its major

- **WHEN** a subscription key is `opmodel.dev/catalogs/opm@v2` and the path has published stable `0.6.0` plus pre-releases `1.0.0-alpha.9`, `2.0.0-alpha.1`, `2.0.0-alpha.2`
- **THEN** Materialize enumerates only the v2 versions
- **AND** the no-filter default selects `2.0.0-alpha.2` via the pre-release-only fallback
- **AND** stable `0.6.0` is never a candidate

#### Scenario: Major-free key keeps whole-path resolution

- **WHEN** a subscription key carries no major suffix and the path has published the same mixed-major list
- **THEN** Materialize enumerates every published version regardless of major
- **AND** the no-filter default selects stable `0.6.0`

#### Scenario: Major-suffixed key composes a valid load ID

- **WHEN** a major-suffixed subscription resolves version `2.0.0-alpha.2`
- **THEN** the catalog is pulled via a load ID composed from the bare path and the version
- **AND** no `…@v2@v2.0.0-alpha.2` form reaches the module loader

#### Scenario: Pre-release excluded from the no-filter default

- **WHEN** a path has published `0.5.0`, `0.5.1`, and a pre-release `0.6.0-dev.1` and the subscription has no `filter`
- **THEN** Materialize selects `0.5.1`
- **AND** does not select `0.6.0-dev.1`

#### Scenario: Pre-release-only scope falls back

- **WHEN** every published version in a no-filter subscription's resolution scope is a pre-release
- **THEN** Materialize selects the highest pre-release in scope so the path still materializes

#### Scenario: Disabled subscription skipped

- **WHEN** a subscription sets `enable: false`
- **THEN** Materialize pulls no builds from that path
- **AND** no transformers from that path appear in the materialized index
