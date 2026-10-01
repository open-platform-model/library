## Purpose

How the library's release automation keeps a published release tag pinned to the commit
release-please released, never mutates a tag, and surfaces a mismatch for roll-forward
recovery.

## ADDED Requirements

### Requirement: A created release is checked against its tag

Whenever the release workflow creates a library release, it SHALL resolve the release tag on
the remote to the commit it names, peeling an annotated tag to its commit, and SHALL fail the
run unless that commit equals the release commit release-please reports. The check SHALL
match the tag by its exact ref name, so one tag name never resolves to another that shares
its prefix. A run that creates no release SHALL skip the check.

#### Scenario: Tag names the release commit

- **WHEN** release-please creates release `v1.0.0-beta.2` for commit `C` and the remote tag
  `v1.0.0-beta.2` names `C`
- **THEN** the release run succeeds and its log names the tag and `C`

#### Scenario: Tag names another commit

- **WHEN** release-please creates release `v1.0.0-beta.2` for commit `C` but the remote tag
  `v1.0.0-beta.2` already named commit `D` before the run
- **THEN** the release run fails with an error naming the tag, `D` and `C`, and the
  roll-forward recovery

#### Scenario: Tag missing or release commit unknown

- **WHEN** release-please reports a created release but the remote has no tag of that name,
  or release-please reports no release commit
- **THEN** the release run fails; it does not pass silently

#### Scenario: Annotated tag

- **WHEN** the release tag is an annotated tag object pointing at commit `C`
- **THEN** the check compares `C`, not the tag object's own id

#### Scenario: Push without a release

- **WHEN** a push to `main` only opens or updates the release PR
- **THEN** the check does not run and the run's outcome is unchanged

### Requirement: Release automation never mutates a tag

The library's release automation SHALL NOT move, delete or re-create any tag, and SHALL NOT
delete or re-point a published release, including when the tag check fails. A wrong or
broken release SHALL be recovered by releasing the next version, whose `go.mod` carries a
`retract` directive for the bad version.

#### Scenario: Mismatch recovery is roll-forward

- **WHEN** the tag check fails for `v1.0.0-beta.2`
- **THEN** the workflow leaves the tag and the release as they are, and the error tells the
  reader to release the next version with a `retract` for `v1.0.0-beta.2` in `go.mod`
