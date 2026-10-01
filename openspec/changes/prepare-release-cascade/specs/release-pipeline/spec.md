## Purpose

Defines what the library's release automation guarantees to downstream repos: a release never
ships a dev, pseudo-version or replaced pin, each release run publishes whether it released and
under which tag, and only changes a consumer can observe cut a release.

## ADDED Requirements

### Requirement: Release PRs refuse dev and replaced pins

The library's CI SHALL run a release-pin check on every release PR, identified by a branch name
(`github.head_ref`, falling back to `github.ref_name`) that starts with `release-please--`. The
check SHALL fail the PR's `Go tests` check when the PR head carries any of:

- a `replace` directive in `go.mod`;
- a requirement on a `github.com/open-platform-model/*` Go module at a Go pseudo-version;
- a CUE dependency pin whose version contains `-0.dev.` in any tracked `cue.mod/module.cue`;
- a quoted version literal containing `-0.dev.` in a non-test Go source file under `opm/`
  (the shipped default core pin lives there);
- a tracked `cue.mod/local-module.cue` anywhere in the repository.

On failure the check SHALL name every offending file and line or module. The check SHALL run as
a step of the existing `Go tests` job, not as a job of its own, so it can never be reported as
skipped while the job passes. The same check SHALL be runnable locally through the Taskfile.

Source: workspace RELEASING.md, section "Gates" (G1).

#### Scenario: Clean release PR passes

- **WHEN** a release PR's head has no replace directive, no OPM Go pseudo-version, no `-0.dev.`
  pin and no tracked `cue.mod/local-module.cue`
- **THEN** the release-pin step SHALL succeed and the `Go tests` check SHALL depend only on the
  test results

#### Scenario: Replace directive on a release PR

- **WHEN** a release PR's `go.mod` carries a `replace` directive
- **THEN** the release-pin step SHALL fail and its output SHALL print the replace directive

#### Scenario: OPM Go module at a pseudo-version

- **WHEN** a release PR's `go.mod` requires a `github.com/open-platform-model/*` module at a
  version such as `v1.0.1-0.20261001120000-abcdef123456`
- **THEN** the release-pin step SHALL fail and SHALL name that module and version

#### Scenario: Third-party pseudo-versions are allowed

- **WHEN** a release PR's `go.mod` requires a module outside `github.com/open-platform-model/`
  at a pseudo-version (as `cuelabs.dev/go/oci/ociregistry` is today)
- **THEN** the release-pin step SHALL NOT fail on that requirement

#### Scenario: Dev CUE pin in a cue.mod

- **WHEN** a tracked `cue.mod/module.cue` pins a dependency at a version containing `-0.dev.`
- **THEN** the release-pin step SHALL fail and SHALL print the file and line

#### Scenario: Dev core pin in the shipped default

- **WHEN** a non-test Go file under `opm/` carries a quoted version literal containing
  `-0.dev.`, such as a default core module `"opmodel.dev/core@v2.0.1-0.dev.3"`
- **THEN** the release-pin step SHALL fail and SHALL print the file and line

#### Scenario: Tracked CUE local replacement

- **WHEN** a file named `cue.mod/local-module.cue` is tracked anywhere in the repository
- **THEN** the release-pin step SHALL fail and SHALL print its path

#### Scenario: Ordinary PR is not gated

- **WHEN** a PR whose branch does not start with `release-please--` carries a dev pin
- **THEN** the release-pin step SHALL be skipped and SHALL NOT fail the `Go tests` check

### Requirement: Release job publishes its outcome

The release workflow's release-please job SHALL expose the job outputs `releases_created` and
`tag_name`, taken from the release-please step, so a later job in the same workflow can act only
when a release was really cut and can name its tag.

Source: workspace RELEASING.md, section "The cascade".

#### Scenario: A release is cut

- **WHEN** a merged release PR makes release-please tag a new library version
- **THEN** the job output `releases_created` SHALL be `true` and `tag_name` SHALL be the new tag,
  for example `v1.0.0-beta.2`

#### Scenario: No release is cut

- **WHEN** a push to `main` only opens or updates the release PR
- **THEN** the job output `releases_created` SHALL NOT be `true`

### Requirement: Only consumer-visible commit types cut a release

The release configuration SHALL treat `feat`, `fix`, `perf`, `revert`, `deps` and `refactor` as
releasing types, and SHALL hide `docs`, `test`, `ci`, `build` and `chore`, so a range of commits
that holds only hidden types opens no release PR. `refactor` SHALL stay visible so library
rewrites reach downstream repos early.

Source: owner decision D15, recorded in workspace RELEASING.md, section "Bump rule".

#### Scenario: Docs-only commits

- **WHEN** every commit on `main` since the last release has type `docs`
- **THEN** release-please SHALL NOT open or update a release PR for them

#### Scenario: Refactor still releases

- **WHEN** a `refactor` commit lands on `main`
- **THEN** release-please SHALL open or update the release PR and list the commit under
  "Code Refactoring"
