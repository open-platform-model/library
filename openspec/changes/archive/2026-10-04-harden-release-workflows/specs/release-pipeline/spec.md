## ADDED Requirements

### Requirement: The release App key is read only in the release Environment

Every job that reads `RELEASE_APP_PRIVATE_KEY` SHALL declare `environment: release`, the Environment whose deployment branch policy admits `main` only, and no other job SHALL read it. Today that is the release workflow's release-please job, which SHALL be granted no `GITHUB_TOKEN` permission (`permissions: {}`), since it acts with the App token alone. The release workflow SHALL declare `permissions: {}` at the workflow level, so each of its jobs carries only its own grants. Source: owner selection 29; security pass finding GOV-2.

#### Scenario: A release run on main

- **WHEN** a push to `main` runs the release workflow
- **THEN** the release-please job runs in the `release` Environment, mints the App token and opens, updates or releases the release PR as before

#### Scenario: A workflow on another branch reaches for the key

- **WHEN** a workflow run on a branch other than `main` runs a job that declares `environment: release`
- **THEN** the Environment's branch policy refuses the job before any step reads the key

#### Scenario: The secret is not yet in the Environment

- **WHEN** the owner has not yet stored `RELEASE_APP_PRIVATE_KEY` in the `release` Environment
- **THEN** the job reads the organization secret of the same name and the release still runs
