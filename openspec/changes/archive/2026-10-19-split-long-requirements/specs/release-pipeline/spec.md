## REMOVED Requirements

### Requirement: Release PRs refuse dev and replaced pins
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The library's CI runs a release-pin check on every release PR", "The release-pin check fails the Go tests check on a replace directive, an OPM Go pseudo-version or a dev CUE pin", "The release-pin check fails the Go tests check on a dev core literal in Go source or a tracked cue.mod/local-module.cue", "The release-pin check names every offending file and line or module", "The release-pin check runs as a step of the Go tests job" and "The release-pin check is runnable locally through the Taskfile".

### Requirement: A release notifies the library's downstream repos
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The release workflow has a notify-downstream job that runs cascade-notify with the release tag", "The notify-downstream job depends on the release-please job only and runs only for a release with CASCADE_NOTIFY not off", "Only the caller-owned notify-downstream job reads the cascade App key", "The documentation-bundle job does not gate the notify-downstream job" and "Which repos notify-downstream dispatches to and the Go proxy wait are the action's behaviour".

### Requirement: The release App key is read only in the release Environment
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Every job that reads RELEASE_APP_PRIVATE_KEY declares the release Environment" and "The release workflow declares no workflow-level permissions and the release-please job mints a narrow App token".

## ADDED Requirements

### Requirement: The library's CI runs a release-pin check on every release PR
The library's CI SHALL run a release-pin check on every release PR, identified by a branch name (`github.head_ref`, falling back to `github.ref_name`) that starts with `release-please--`.
**Source**: workspace RELEASING.md, section "Gates" (G1).

#### Scenario: Ordinary PR is not gated

- **WHEN** a PR whose branch does not start with `release-please--` carries a dev pin
- **THEN** the release-pin step SHALL be skipped and SHALL NOT fail the `Go tests` check

### Requirement: The release-pin check fails the Go tests check on a replace directive, an OPM Go pseudo-version or a dev CUE pin
The release-pin check SHALL fail the PR's `Go tests` check when the PR head carries any of:
- a `replace` directive in `go.mod`;
- a requirement on a `github.com/open-platform-model/*` Go module at a Go pseudo-version;
- a CUE dependency pin whose version contains `-0.dev.` in any tracked `cue.mod/module.cue`.
**Source**: workspace RELEASING.md, section "Gates" (G1).

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

### Requirement: The release-pin check fails the Go tests check on a dev core literal in Go source or a tracked cue.mod/local-module.cue
The release-pin check SHALL fail the PR's `Go tests` check when the PR head carries any of:
- a quoted version literal containing `-0.dev.` in a non-test Go source file under `opm/` (the shipped default core pin lives there);
- a tracked `cue.mod/local-module.cue` anywhere in the repository.
**Source**: workspace RELEASING.md, section "Gates" (G1).

#### Scenario: Dev core pin in the shipped default

- **WHEN** a non-test Go file under `opm/` carries a quoted version literal containing
  `-0.dev.`, such as a default core module `"opmodel.dev/core@v2.0.1-0.dev.3"`
- **THEN** the release-pin step SHALL fail and SHALL print the file and line

#### Scenario: Tracked CUE local replacement

- **WHEN** a file named `cue.mod/local-module.cue` is tracked anywhere in the repository
- **THEN** the release-pin step SHALL fail and SHALL print its path

### Requirement: The release-pin check names every offending file and line or module
On failure the release-pin check SHALL name every offending file and line or module.
**Source**: workspace RELEASING.md, section "Gates" (G1).

#### Scenario: The failing check names each offender

- **WHEN** the release-pin check fails
- **THEN** it names every offending file and line or module

### Requirement: The release-pin check runs as a step of the Go tests job
The release-pin check SHALL run as a step of the existing `Go tests` job, not as a job of its own, so it can never be reported as skipped while the job passes.
**Source**: workspace RELEASING.md, section "Gates" (G1).

#### Scenario: Clean release PR passes

- **WHEN** a release PR's head has no replace directive, no OPM Go pseudo-version, no `-0.dev.`
  pin and no tracked `cue.mod/local-module.cue`
- **THEN** the release-pin step SHALL succeed and the `Go tests` check SHALL depend only on the
  test results

### Requirement: The release-pin check is runnable locally through the Taskfile
The same check (Requirement: The library's CI runs a release-pin check on every release PR) SHALL be runnable locally through the Taskfile.
**Source**: workspace RELEASING.md, section "Gates" (G1).

#### Scenario: The check runs locally through the Taskfile

- **WHEN** a developer runs the release-pin check through the Taskfile
- **THEN** the check runs locally

### Requirement: The release workflow has a notify-downstream job that runs cascade-notify with the release tag
The release workflow SHALL have a job `notify-downstream`, the last job of the workflow, that runs the org's `cascade-notify` composite action, pinned to the library's `.github` `main` SHA, with the tag the release-please job reports in `tag_name`.
**Source**: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner decision 24; Phase 3 wiring contract (version 3.1) §4.6 and §10.1 items 1 and 9.

#### Scenario: A release is cut

- **WHEN** a merged release PR makes release-please tag `v1.0.0-beta.5` and `CASCADE_NOTIFY` is unset
- **THEN** `notify-downstream` runs after the release-please job in the `cascade` Environment and runs `cascade-notify` with `tag: v1.0.0-beta.5`

### Requirement: The notify-downstream job depends on the release-please job only and runs only for a release with CASCADE_NOTIFY not off
The `notify-downstream` job SHALL depend on the release-please job only, SHALL run only when that job's `releases_created` output is `true` and the repo variable `CASCADE_NOTIFY` is not `off`, SHALL run on `ubuntu-latest` with a 20-minute timeout, and SHALL be granted only `contents: read`.
**Source**: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner decision 24; Phase 3 wiring contract (version 3.1) §4.6 and §10.1 items 1 and 9.

#### Scenario: No release is cut

- **WHEN** a push to `main` only opens or updates the release PR
- **THEN** `notify-downstream` is skipped

#### Scenario: Notify is switched off

- **WHEN** a release is cut while the repo variable `CASCADE_NOTIFY` is `off`
- **THEN** `notify-downstream` is skipped and no dispatch is sent

### Requirement: Only the caller-owned notify-downstream job reads the cascade App key
`secrets.CASCADE_APP_PRIVATE_KEY`: the key SHALL be read only in the caller-owned `notify-downstream` job, which declares `environment: cascade` and passes `secrets.CASCADE_APP_PRIVATE_KEY` only as the `private-key` input of the SHA-pinned cascade action, with `vars.CASCADE_APP_CLIENT_ID` as `client-id`; that job SHALL have exactly that one step, no checkout or `run:` of its own, and no `env:`, `container:` or `services:`; no reusable call SHALL pass `secrets:` or `secrets: inherit`.
**Source**: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner decision 24; Phase 3 wiring contract (version 3.1) §4.6 and §10.1 items 1 and 9.

#### Scenario: The key is passed only to the cascade action

- **WHEN** the `notify-downstream` job reads `secrets.CASCADE_APP_PRIVATE_KEY`
- **THEN** it passes the key only as the `private-key` input of the SHA-pinned cascade action, the job has exactly that one step, and no reusable call passes `secrets:` or `secrets: inherit`

### Requirement: The documentation-bundle job does not gate the notify-downstream job
The documentation-bundle job SHALL NOT gate the `notify-downstream` job.
**Source**: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner decision 24; Phase 3 wiring contract (version 3.1) §4.6 and §10.1 items 1 and 9.

#### Scenario: The docs bundle fails

- **WHEN** a release is cut and the `publish-docs` job fails
- **THEN** `notify-downstream` still runs, because it does not depend on `publish-docs`

### Requirement: Which repos notify-downstream dispatches to and the Go proxy wait are the action's behaviour
Which repos `notify-downstream` dispatches to, and the Go proxy wait before it does, SHALL be the action's behaviour, not the library's.
**Source**: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner decision 24; Phase 3 wiring contract (version 3.1) §4.6 and §10.1 items 1 and 9.

#### Scenario: Dispatch targets and the Go proxy wait belong to the action

- **WHEN** `notify-downstream` dispatches after a release
- **THEN** which repos it dispatches to, and the Go proxy wait before it does, are the action's behaviour, not the library's

### Requirement: Every job that reads RELEASE_APP_PRIVATE_KEY declares the release Environment
Every job that reads `RELEASE_APP_PRIVATE_KEY` SHALL declare `environment: release`, the Environment whose deployment branch policy admits `main` only, and no other job SHALL read it. Today that is the release workflow's release-please job, which SHALL be granted no `GITHUB_TOKEN` permission (`permissions: {}`), since it acts with the App token alone.
**Source**: owner selection 29; security pass finding GOV-2.

#### Scenario: A workflow on another branch reaches for the key

- **WHEN** a workflow run on a branch other than `main` runs a job that declares `environment: release`
- **THEN** the Environment's branch policy refuses the job before any step reads the key

#### Scenario: The secret is not yet in the Environment

- **WHEN** the owner has not yet stored `RELEASE_APP_PRIVATE_KEY` in the `release` Environment
- **THEN** the job reads the organization secret of the same name and the release still runs

#### Scenario: The organization secret still reaches the library

- **WHEN** the organization secret `RELEASE_APP_PRIVATE_KEY` still lists this repository, whether or not the `release` Environment holds the key
- **THEN** a workflow on any branch that does not declare the Environment can still read the organization secret, so the Environment gate closes only once the owner removes this repository from the organization secret or deletes it

### Requirement: The release workflow declares no workflow-level permissions and the release-please job mints a narrow App token
The release workflow SHALL declare `permissions: {}` at the workflow level, so each of its jobs carries only its own grants. The release-please job SHALL mint the App token for this repository only (`owner`, `repositories`) with `contents: write` and `pull-requests: write` and no other permission, rather than every permission the App's installation holds.
**Source**: owner selection 29; security pass finding GOV-2.

#### Scenario: A release run on main

- **WHEN** a push to `main` runs the release workflow
- **THEN** the release-please job runs in the `release` Environment, mints the App token and opens, updates or releases the release PR as before
