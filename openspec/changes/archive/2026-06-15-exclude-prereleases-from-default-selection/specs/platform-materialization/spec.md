## MODIFIED Requirements

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
