## Purpose

How the library's release automation serves `release/vX.Y` maintenance branches alongside
`main`, how such a branch is cut, and that the automation never mutates a release tag.

## ADDED Requirements

### Requirement: Release runs against the pushed branch

The release workflow SHALL run release-please on every push to `main` and to a branch under
`release/`, and SHALL target the branch that was pushed, so that the release PR is opened
against that branch and its release is tagged from that branch's history. A push to `main`
SHALL release exactly as it did before release branches existed. Source: 0021:D10.

#### Scenario: Push to main

- **WHEN** a commit lands on `main`
- **THEN** release-please opens or updates the release PR against `main`, as before

#### Scenario: Push to a release branch

- **WHEN** a backport PR is merged into `release/v1.0`
- **THEN** release-please opens or updates a release PR against `release/v1.0`, and merging
  that PR tags the next `v1.0.Z` patch from `release/v1.0`

### Requirement: A release branch is cut only by the shared dispatch

The library SHALL provide a manually dispatched workflow that takes a released minor `X.Y`
and delegates to the organization's shared cut-release-branch workflow with tag prefix `v`
and package path `.`. The library SHALL NOT carry its own branch-cutting logic. Source:
0021:D10.

#### Scenario: Cutting release/v1.0

- **WHEN** a maintainer dispatches the cut workflow with minor `1.0`
- **THEN** the shared workflow is invoked with tag prefix `v`, minor `1.0` and package path
  `.`, and the library workflow itself creates no branch, tag or release

### Requirement: Release automation never mutates a tag

The library's release automation SHALL NOT move, delete or re-create any tag, and SHALL NOT
delete or re-point a published release. A wrong or broken release SHALL be recovered by
releasing the next version, whose `go.mod` carries a `retract` directive for the bad
version. Source: 0021:D10:R1/R2.

#### Scenario: Broken release is rolled forward

- **WHEN** release `v1.0.0-beta.2` turns out to be broken
- **THEN** the tag and the release stay as they are, and the next release carries a
  `retract` for `v1.0.0-beta.2` in `go.mod`
