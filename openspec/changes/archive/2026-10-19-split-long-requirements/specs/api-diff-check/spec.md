## MODIFIED Requirements

### Requirement: The base tag decides between warning and failing
The check SHALL derive its outcome from the base release tag alone: when the tag has a prerelease suffix it SHALL warn and pass, and when the tag is a release it SHALL fail on any incompatible change of the pull request. In both modes it SHALL name the base tag and each change. No repository variable or input SHALL change the mode in CI. Source: ADR-013, decision j4; ADR-010; ADR-004; CONSTITUTION VI.

#### Scenario: A breaking change on the beta line

- **WHEN** the base tag is `v1.0.0-beta.4` and the pull request makes an incompatible change
- **THEN** the job passes, shows each change as a warning annotation and writes a job summary naming the `feat!` commit and `BREAKING CHANGE:` footer the change needs

#### Scenario: A breaking change after GA

- **WHEN** the base tag is a release such as `v1.0.0` and the pull request makes an incompatible change
- **THEN** the job fails, shows each change as an error annotation and names the migration fragment the change needs

#### Scenario: A compatible pull request

- **WHEN** the pull request makes no incompatible change
- **THEN** the job passes and says the change adds no incompatible change since the base tag

#### Scenario: No release tag is reachable

- **WHEN** no `v[0-9]*` tag is reachable from the base commit, as in a shallow clone
- **THEN** the job fails and says to fetch the tags, in warn mode too

### Requirement: The check runs read-only with a committed tool checksum
The check's workflow SHALL declare `permissions: contents: read`, SHALL check out with `persist-credentials: false`, and SHALL pin every action by full commit SHA. The diff tool SHALL be built from a tools module in this repository whose committed `go.sum` the Go toolchain verifies before building, at one exact version. Source: library#181 (the workflow hardening pass); workflow-hardening.

#### Scenario: A same-repo pull request runs the check

- **WHEN** a pull request from a branch in this repository runs the check
- **THEN** the job holds a `contents: read` token and `.git/config` carries no credential

#### Scenario: A tampered tool module

- **WHEN** a module the tool build needs does not match the hash in the committed `go.sum`
- **THEN** the build fails and no comparison runs

#### Scenario: A maintainer checks a release-tag base locally

- **WHEN** a maintainer runs `task api:diff BASE=v0.7.0`
- **THEN** the check runs in fail mode, because `v0.7.0` is a release tag

## REMOVED Requirements

### Requirement: Pull requests show breaking changes to the public API
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Pull requests compare the exported API against the base release tag", "The base release tag is the nearest reachable v tag", "Inherited incompatible changes do not count as a change of the pull request", "A move of the core pin constant is allowed".

## ADDED Requirements

### Requirement: Pull requests compare the exported API against the base release tag
A pull request that changes Go code or the module files SHALL run a check that compares the exported API of the module's non-internal packages under `opm/` at the base release tag with the API at the pull request head, and SHALL list every incompatible change the comparison reports. Source: ADR-013, decision j4.

#### Scenario: A pull request removes an exported function

- **WHEN** a pull request removes an exported function from a package under `opm/` that the base release tag exported
- **THEN** the check lists the removal as an incompatible change of the pull request

#### Scenario: A pull request changes only internal packages

- **WHEN** a pull request changes only packages under `opm/internal/`
- **THEN** the check lists no incompatible change of the pull request

### Requirement: The base release tag is the nearest reachable v tag
The base release tag SHALL be the nearest tag matching `v[0-9]*` reachable from the pull request's base commit, not the highest version tag in the repository. Source: ADR-013, decision j4.

#### Scenario: An older release tag exists

- **WHEN** the repository carries the release tag `v0.7.0` and the nearest tag reachable from the base commit is `v1.0.0-beta.4`
- **THEN** the check compares against `v1.0.0-beta.4`

### Requirement: Inherited incompatible changes do not count as a change of the pull request
An incompatible change that the base commit already carries relative to the base release tag SHALL be listed as inherited and SHALL NOT count as a change of the pull request. Source: ADR-013, decision j4.

#### Scenario: A break merged earlier since the tag

- **WHEN** the base branch already carries an incompatible change since the base tag and the pull request does not touch it
- **THEN** the check lists that change as inherited from the base branch, does not warn or fail on it, and names no remedy for it

### Requirement: A move of the core pin constant is allowed
A value change of `schema.DefaultSchemaModule`, which the release cascade makes on every core move, SHALL be listed as allowed through one fixed line prefix and SHALL NOT count as a change of the pull request. Source: ADR-013, decision j4.

#### Scenario: A release-cascade pull request moves core

- **WHEN** a pull request changes only the value of `schema.DefaultSchemaModule`
- **THEN** the check lists the change as allowed and lists no incompatible change of the pull request

#### Scenario: A pull request removes the core pin constant

- **WHEN** a pull request removes `schema.DefaultSchemaModule`
- **THEN** the check lists the removal as an incompatible change of the pull request

### Requirement: The check states the remedy of its mode
In warn mode the check SHALL state that a breaking change needs a `feat!` commit with a `BREAKING CHANGE:` footer (ADR-010); in fail mode it SHALL state that a breaking change is MAJOR and needs a migration fragment per `migrations/README.md` (ADR-004). Source: ADR-013, decision j4; ADR-010; ADR-004; CONSTITUTION VI.

#### Scenario: Each mode states its remedy

- **WHEN** the check reports a breaking change in warn mode, and again in fail mode
- **THEN** warn mode states that the change needs a `feat!` commit with a `BREAKING CHANGE:` footer, and fail mode states that the change is MAJOR and needs a migration fragment per `migrations/README.md`

### Requirement: A failure of the check itself fails the job in both modes
A failure of the check itself (a tool build error, no reachable tag) SHALL fail the job in both modes. Source: ADR-013, decision j4; ADR-010; ADR-004; CONSTITUTION VI.

#### Scenario: A tool build error fails the job

- **WHEN** the tool build fails
- **THEN** the job fails, in warn mode too

### Requirement: The library's go.mod does not require the diff tool
The library's own `go.mod` SHALL NOT require the diff tool. Source: library#181 (the workflow hardening pass); workflow-hardening.

#### Scenario: The library module does not require the tool

- **WHEN** the requirements of the library's own `go.mod` are listed
- **THEN** the diff tool is not among them

### Requirement: The check is not a required status check by this change
The check SHALL NOT be a required status check by this change. Source: library#181 (the workflow hardening pass); workflow-hardening.

#### Scenario: The check is not a required status check

- **WHEN** the required status checks of the repository are listed after this change
- **THEN** the check is not among them

### Requirement: The check runs locally through task api:diff
The same check SHALL run locally through `task api:diff`, with an optional `BASE` tag override that CI ignores. Source: library#181 (the workflow hardening pass); workflow-hardening.

#### Scenario: CI ignores the BASE override

- **WHEN** the check runs in CI
- **THEN** the optional `BASE` tag override of `task api:diff` is ignored
