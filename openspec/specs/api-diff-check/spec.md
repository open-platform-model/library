# api-diff-check Specification

## Purpose
Show every pull request the incompatible changes it makes to the library's exported `opm/` Go API since the last release tag, so a breaking change cannot land unnoticed: the check warns while the base tag is a prerelease and fails once it is a release (ADR-013, decision j4), never charges a pull request with breaks already on the base branch or with the release cascade's core-pin move, and builds its tool from a checksum committed in this repository.

## Requirements

### Requirement: Pull requests show breaking changes to the public API

A pull request that changes Go code or the module files SHALL run a check that compares the exported API of the module's non-internal packages under `opm/` at the base release tag with the API at the pull request head, and SHALL list every incompatible change the comparison reports. The base release tag SHALL be the nearest tag matching `v[0-9]*` reachable from the pull request's base commit, not the highest version tag in the repository. An incompatible change that the base commit already carries relative to that tag SHALL be listed as inherited and SHALL NOT count as a change of the pull request. A value change of `schema.DefaultSchemaModule`, which the release cascade makes on every core move, SHALL be listed as allowed through one fixed line prefix and SHALL NOT count either. Source: owner decision j4 (beta.1 walkthrough).

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

### Requirement: The base tag decides between warning and failing

The check SHALL derive its outcome from the base release tag alone: when the tag has a prerelease suffix it SHALL warn and pass, and when the tag is a release it SHALL fail on any incompatible change of the pull request. In both modes it SHALL name the base tag and each change. In warn mode it SHALL state that a breaking change needs a `feat!` commit with a `BREAKING CHANGE:` footer (ADR-010); in fail mode it SHALL state that a breaking change is MAJOR and needs a migration fragment per `migrations/README.md` (ADR-004). No repository variable or input SHALL change the mode in CI. A failure of the check itself (a tool build error, no reachable tag) SHALL fail the job in both modes. Source: owner decision j4 (beta.1 walkthrough); ADR-010; ADR-004; CONSTITUTION VI.

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
