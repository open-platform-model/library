## ADDED Requirements

### Requirement: A registry entry stamps its version bare

An entry's version MAY be written as bare SemVer (`4.0.1`) or with the `v` prefix (`v4.0.1`).
`Generate` SHALL stamp the entry's expected `version` in `platform.cue` without the `v` prefix, so
both spellings render the same `platform.cue` and unify with the catalog's bare
`metadata.version`. A platform generated from a subscription in either spelling SHALL build
through `AcquirePlatformFromDir` against that catalog.

#### Scenario: A v-prefixed entry stamps the bare version

- **WHEN** an entry for a catalog is generated with version `v4.0.1`
- **THEN** its `#registry` entry in `platform.cue` carries `version: "4.0.1"`, byte-identical to the output for `4.0.1`

#### Scenario: A platform generated from either spelling builds

- **WHEN** a platform module is generated from a subscription to a catalog published at `v0.1.0`, once with version `0.1.0` and once with `v0.1.0`
- **THEN** both generated platforms build through the kernel, and the registry entry's `version` reads `0.1.0` in both
