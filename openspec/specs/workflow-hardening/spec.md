# workflow-hardening Specification

## Purpose
Defines what the library's GitHub Actions workflows guarantee about the credentials they hold:
explicit least-privilege token grants, no write token or persisted credential for code a PR
runs, CI tools pinned to a committed checksum, and code-owner review of the release and cascade
files.

## Requirements

### Requirement: Every workflow states least-privilege token grants

Every workflow file under `.github/workflows/` SHALL declare `permissions:` at the workflow level, either `{}` or read-only grants, and a job that needs more SHALL declare its own grants, naming only what its steps use. No job that runs on `pull_request` SHALL hold a write grant. Every checkout step in this repository's workflow files SHALL set `persist-credentials: false`, since none of their later steps fetches or pushes through the checkout's remote. (A reusable workflow called from another repository, such as docs-kit's `publish.yml`, owns its own checkouts.) Source: owner selection 30; security pass findings N4 and GOV-3.

#### Scenario: The repo default token becomes read-only

- **WHEN** the repository's default workflow permissions are set to read-only
- **THEN** every workflow still runs as before, because none relies on the default grant

#### Scenario: A same-repo PR runs the tests

- **WHEN** a pull request from a branch in this repository, such as `deps/cascade`, runs `Go tests`, `golangci-lint` or `Validate CUE modules`
- **THEN** the job holds a `contents: read` token and `.git/config` carries no credential

### Requirement: Third-party CI tools are pinned and checksum-verified

A CI step that downloads a tool binary SHALL name an exact version and SHALL verify the download against a sha256 committed in this repository before running anything from it; no step SHALL pipe a script fetched from a mutable ref into a shell. Source: security pass, governance review missed item 1.

#### Scenario: The lint job installs golangci-lint

- **WHEN** the lint job installs golangci-lint
- **THEN** it downloads the v2.8.0 linux-amd64 release archive over HTTPS, checks it with `sha256sum -c` against the committed digest, and only then extracts and runs it

#### Scenario: The download does not match

- **WHEN** the downloaded archive's sha256 differs from the committed digest
- **THEN** the step fails before extraction and `task lint` does not run

### Requirement: Release and cascade files need a code owner's review

The repository SHALL carry `.github/CODEOWNERS` naming the repository owners for `/.github/`, `/.tasks/`, `/Taskfile*.yml`, `/release-please-config.json`, `/.release-please-manifest.json` and `/.cascade-frozen`, so the `main` ruleset's code-owner review applies to every change of the workflows, the cascade task, the Taskfiles, the release configuration and the cascade freeze list. Source: owner selection 28.

#### Scenario: A PR edits a workflow

- **WHEN** a pull request changes a file under `.github/workflows/`
- **THEN** GitHub requests a review from the code owners of `/.github/`

#### Scenario: A PR edits only library code

- **WHEN** a pull request changes only files under `opm/`
- **THEN** no code-owner review is required by this file
