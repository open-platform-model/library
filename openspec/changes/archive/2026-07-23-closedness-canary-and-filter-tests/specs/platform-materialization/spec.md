# Delta: platform-materialization (closedness-canary-and-filter-tests)

No behavior change. The Version Enumeration and Filtering requirement gains scenarios for the pre-release-range opt-in and mixed-family ordering it already specifies in prose (the enhancement 0006 OQ18 mechanism), so the semantics are test-pinned before OQ18 is decided.

## MODIFIED Requirements

### Requirement: Version Enumeration and Filtering

For each subscription path, the Materialize operation SHALL enumerate the published versions via the registry's version listing, then narrow the candidate set Go-side in this order (D10): `filter.range` (a SemVer constraint expression) restricts the set, `filter.allow` force-includes specific versions, `filter.deny` force-excludes specific versions. Range expressions SHALL be parsed with a SemVer constraint library because CUE cannot evaluate range syntax. The `v`-prefixed module version form returned by the registry SHALL be normalized against the bare-SemVer catalog FQN form.

Pre-release versions are selectable only by explicit opt-in: a `filter.allow` entry naming the exact pre-release version, or a `filter.range` whose constraint itself contains a pre-release identifier (standard SemVer-constraint semantics, under which an ordinary constraint does not admit pre-releases). The no-filter default never admits a pre-release except via the pre-release-only fallback above. Within an admitting range, pre-release families order by standard SemVer identifier comparison — in particular `dev.*` identifiers sort above `alpha.*` identifiers of the same base version.

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

#### Scenario: Range carrying a pre-release identifier opts pre-releases in

- **WHEN** a subscription filter is `range: ">=0.6.0-dev.0 <0.7.0"` and the path has published `0.5.0`, `0.6.0-dev.1`, `0.6.0`
- **THEN** Materialize selects `0.6.0-dev.1` and `0.6.0`
- **AND** excludes `0.5.0`

#### Scenario: Dev pre-releases out-sort alpha pre-releases within an admitting range

- **WHEN** a subscription filter is `range: ">=1.0.0-alpha"` and the path has published `1.0.0-alpha`, `1.0.0-alpha.1`, and a CI tag `1.0.0-dev.1784212239.g0c11c12`
- **THEN** all three versions are in the survivor set
- **AND** the resolved (highest) version is the `1.0.0-dev.*` tag
