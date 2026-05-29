## ADDED Requirements

### Requirement: Dedicated catalog publish task with version stamping

The library SHALL provide a `cue:publish:catalog` task that publishes the catalog at a concrete SemVer. The task SHALL copy the catalog source to a transient build directory (excluding `cue.mod/{pkg,gen,usr}`), write `identity/version_override.cue` pinning a concrete `Version`, run `cue vet` from the build directory, and run `cue mod publish v<MAJOR.MINOR.PATCH>` from the build directory. The committed source tree SHALL NOT carry `identity/version_override.cue`.

#### Scenario: Publish stamps the concrete version

- **WHEN** `cue:publish:catalog` runs with version `0.1.0`
- **THEN** the build directory's `identity.Version` resolves to `"0.1.0"`, `cue vet` passes, and the module is published to the registry as the OCI tag `v0.1.0`

#### Scenario: Source tree stays at the dev default

- **WHEN** the committed `library/modules/opm/identity/` directory is inspected
- **THEN** no `version_override.cue` is present and `identity.Version` resolves to `"0.0.0-dev"`

### Requirement: Publish guard rejects the dev version

The publish path SHALL refuse to publish when the resolved `identity.Version` is `"0.0.0-dev"`. An attempt to publish the unstamped catalog SHALL fail with a non-zero exit and a diagnostic identifying the dev version as the cause.

#### Scenario: Unstamped publish is blocked

- **WHEN** a publish is attempted without a concrete `version_override.cue` stamp (resolved `Version == "0.0.0-dev"`)
- **THEN** the publish exits non-zero and no `0.0.0-dev` tag is pushed to the registry

### Requirement: Version layering separates OCI tag from FQN

The catalog OCI tag SHALL be `v`-prefixed (`v0.1.0`) as required by CUE, while the SemVer recorded in `metadata.version`/FQN SHALL be bare (`0.1.0`) as required by `core`'s `#FQNType`. The committed `cue-versions.yml` SHALL carry the catalog version `v`-prefixed; `identity.Version` SHALL carry it bare.

#### Scenario: Tag and FQN use matching numbers, different prefixes

- **WHEN** the catalog is published at `cue-versions.yml` version `v0.1.0`
- **THEN** the registry tag is `v0.1.0` and the catalog's `metadata.fqn` resolves to `opmodel.dev/catalogs/opm@0.1.0` (no `v`)

### Requirement: Registry-presence publish workflow

A `publish-catalog.yml` workflow SHALL publish the catalog on push to `main` using a registry-presence (version-gated) trigger: it SHALL read the catalog version from `cue-versions.yml`, check whether that tag is already present in the registry, and invoke `cue:publish:catalog` only when the tag is absent. The trigger SHALL be stateless — the workflow SHALL NOT write any checksum, version, or other state back to the repository. The workflow SHALL be catalog-only and SHALL NOT use release-please; the Go-module release-please flow SHALL be unaffected.

#### Scenario: Skips when version already published

- **WHEN** the workflow runs on `main` and the `cue-versions.yml` catalog version is already present in the registry
- **THEN** no publish occurs and no commit is pushed back to the repository

#### Scenario: Publishes when version is absent

- **WHEN** the `cue-versions.yml` catalog version is manually bumped in a merged PR and the workflow runs on `main`
- **THEN** the new tag is found absent from the registry, the catalog is validated, and it is published at that version

### Requirement: Legacy publish paths retired

Publishing of `opmodel.dev/modules/opm` and `opmodel.dev/modules/opm-platform` SHALL cease. The replaced `publish-cue.yml` and the dead `apis/core` entry in `cue-versions.yml` SHALL be removed; `cue-versions.yml` SHALL key the catalog under its new identity restarting at `v0.1.0`.

#### Scenario: opm_platform is no longer published

- **WHEN** the publish configuration is inspected after this change
- **THEN** no publish path targets `opmodel.dev/modules/opm-platform`, and `opm_platform` exists only as an unpublished in-repo fixture

#### Scenario: cue-versions.yml is cleaned and retargeted

- **WHEN** `cue-versions.yml` is inspected after this change
- **THEN** there is no `apis/core` entry, no `modules/opm` legacy entry, and the catalog is keyed under its `catalogs/opm` identity at `v0.1.0`
