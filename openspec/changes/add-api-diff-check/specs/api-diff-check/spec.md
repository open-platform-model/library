## ADDED Requirements

### Requirement: Pull requests show breaking changes to the public API

A pull request that changes Go code or the module files SHALL run a check that compares the exported API of the module's non-internal packages under `opm/` at the base release tag with the API at the pull request head, and SHALL list every incompatible change the comparison reports that the committed allow list does not drop. The base release tag SHALL be the nearest tag matching `v*` reachable from the pull request's base commit, not the highest version tag in the repository. Source: owner decision j4 (beta.1 walkthrough).

#### Scenario: A pull request removes an exported function

- **WHEN** a pull request removes an exported function from a package under `opm/` that the base release tag exported
- **THEN** the check lists the removal as an incompatible change

#### Scenario: A pull request changes only internal packages

- **WHEN** a pull request changes only packages under `opm/internal/`
- **THEN** the check reports no incompatible change

#### Scenario: An older release tag exists

- **WHEN** the repository carries the release tag `v0.7.0` and the nearest tag reachable from the base commit is `v1.0.0-beta.4`
- **THEN** the check compares against `v1.0.0-beta.4`

#### Scenario: The release cascade moves the core pin

- **WHEN** the only API difference is a new value of the `schema.DefaultSchemaModule` string constant
- **THEN** the allow list drops it and the check reports no incompatible change

### Requirement: The base tag decides between warning and failing

The check SHALL derive its outcome from the base release tag alone: when the tag has a prerelease suffix it SHALL warn and pass, and when the tag is a release it SHALL fail on any listed incompatible change. In both modes it SHALL name the base tag and each change, and SHALL state that a breaking change needs a `feat!` commit with a `BREAKING CHANGE:` footer. No repository variable or input SHALL change the mode. A failure of the check itself (a tool build error, no reachable tag) SHALL fail the job in both modes. Source: owner decision j4 (beta.1 walkthrough); ADR-010.

#### Scenario: A breaking change on the beta line

- **WHEN** the base tag is `v1.0.0-beta.4` and the check lists an incompatible change
- **THEN** the job passes, shows each change as a warning annotation and writes a job summary naming the `feat!` commit and `BREAKING CHANGE:` footer the change needs

#### Scenario: A breaking change after GA

- **WHEN** the base tag is a release such as `v1.0.0` and the check lists an incompatible change
- **THEN** the job fails and shows each change as an error annotation

#### Scenario: A compatible pull request

- **WHEN** the check lists no incompatible change
- **THEN** the job passes and says the API is compatible with the base tag

### Requirement: The check runs read-only with a committed tool checksum

The check's workflow SHALL declare `permissions: contents: read`, SHALL check out with `persist-credentials: false`, and SHALL pin every action by full commit SHA. The diff tool SHALL be built from a tools module in this repository whose committed `go.sum` the Go toolchain verifies before building, at one exact version, and the library's own `go.mod` SHALL NOT require the tool. The check SHALL NOT be a required status check by this change. The same check SHALL run locally through `task api:diff`, with an optional `BASE` tag override. Source: supervisor note on library#181; workflow-hardening.

#### Scenario: A same-repo pull request runs the check

- **WHEN** a pull request from a branch in this repository runs the check
- **THEN** the job holds a `contents: read` token and `.git/config` carries no credential

#### Scenario: A tampered tool module

- **WHEN** a module the tool build needs does not match the hash in the committed `go.sum`
- **THEN** the build fails and no comparison runs

#### Scenario: A maintainer checks a release-tag base locally

- **WHEN** a maintainer runs `task api:diff BASE=v0.7.0`
- **THEN** the check runs in fail mode, because `v0.7.0` is a release tag
