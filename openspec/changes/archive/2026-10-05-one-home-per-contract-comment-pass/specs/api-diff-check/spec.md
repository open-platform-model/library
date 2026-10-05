## MODIFIED Requirements

### Requirement: Pull requests show breaking changes to the public API

A pull request that changes Go code or the module files SHALL run a check that compares the exported API of the module's non-internal packages under `opm/` at the base release tag with the API at the pull request head, and SHALL list every incompatible change the comparison reports. The base release tag SHALL be the nearest tag matching `v[0-9]*` reachable from the pull request's base commit, not the highest version tag in the repository. An incompatible change that the base commit already carries relative to that tag SHALL be listed as inherited and SHALL NOT count as a change of the pull request. A value change of `schema.DefaultSchemaModule`, which the release cascade makes on every core move, SHALL be listed as allowed through one fixed line prefix and SHALL NOT count either. Source: ADR-013, decision j4.

#### Scenario: A pull request removes an exported function

- **WHEN** a pull request removes an exported function from a package under `opm/` that the base release tag exported
- **THEN** the check lists the removal as an incompatible change of the pull request

#### Scenario: A pull request changes only internal packages

- **WHEN** a pull request changes only packages under `opm/internal/`
- **THEN** the check lists no incompatible change of the pull request

#### Scenario: An older release tag exists

- **WHEN** the repository carries the release tag `v0.7.0` and the nearest tag reachable from the base commit is `v1.0.0-beta.4`
- **THEN** the check compares against `v1.0.0-beta.4`

#### Scenario: A release-cascade pull request moves core

- **WHEN** a pull request changes only the value of `schema.DefaultSchemaModule`
- **THEN** the check lists the change as allowed and lists no incompatible change of the pull request

#### Scenario: A pull request removes the core pin constant

- **WHEN** a pull request removes `schema.DefaultSchemaModule`
- **THEN** the check lists the removal as an incompatible change of the pull request

#### Scenario: A break merged earlier since the tag

- **WHEN** the base branch already carries an incompatible change since the base tag and the pull request does not touch it
- **THEN** the check lists that change as inherited from the base branch, does not warn or fail on it, and names no remedy for it


### Requirement: The check runs read-only with a committed tool checksum

The check's workflow SHALL declare `permissions: contents: read`, SHALL check out with `persist-credentials: false`, and SHALL pin every action by full commit SHA. The diff tool SHALL be built from a tools module in this repository whose committed `go.sum` the Go toolchain verifies before building, at one exact version, and the library's own `go.mod` SHALL NOT require the tool. The check SHALL NOT be a required status check by this change. The same check SHALL run locally through `task api:diff`, with an optional `BASE` tag override that CI ignores. Source: library#181 (the workflow hardening pass); workflow-hardening.

#### Scenario: A same-repo pull request runs the check

- **WHEN** a pull request from a branch in this repository runs the check
- **THEN** the job holds a `contents: read` token and `.git/config` carries no credential

#### Scenario: A tampered tool module

- **WHEN** a module the tool build needs does not match the hash in the committed `go.sum`
- **THEN** the build fails and no comparison runs

#### Scenario: A maintainer checks a release-tag base locally

- **WHEN** a maintainer runs `task api:diff BASE=v0.7.0`
- **THEN** the check runs in fail mode, because `v0.7.0` is a release tag
