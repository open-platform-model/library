## REMOVED Requirements

### Requirement: Every workflow states least-privilege token grants
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Every workflow declares permissions at the workflow level and no pull_request job holds a write grant" and "Every checkout step sets persist-credentials false".

## ADDED Requirements

### Requirement: Every workflow declares permissions at the workflow level and no pull_request job holds a write grant
Every workflow file under `.github/workflows/` SHALL declare `permissions:` at the workflow level, either `{}` or read-only grants, and a job that needs more SHALL declare its own grants, naming only what its steps use. No job that runs on `pull_request` SHALL hold a write grant.
**Source**: owner selection 30; security pass findings N4 and GOV-3.

#### Scenario: The repo default token becomes read-only

- **WHEN** the repository's default workflow permissions are set to read-only
- **THEN** every workflow still runs as before, because none relies on the default grant

#### Scenario: A same-repo PR runs the tests

- **WHEN** a pull request from a branch in this repository, such as `deps/cascade`, runs `Go tests`, `golangci-lint` or `Validate CUE modules`
- **THEN** the job holds a `contents: read` token and `.git/config` carries no credential

### Requirement: Every checkout step sets persist-credentials false
Every checkout step in this repository's workflow files SHALL set `persist-credentials: false`, since none of their later steps fetches or pushes through the checkout's remote. (A reusable workflow called from another repository, such as docs-kit's `publish.yml`, owns its own checkouts.)
**Source**: owner selection 30; security pass findings N4 and GOV-3.

#### Scenario: A checkout does not persist credentials

- **WHEN** a workflow of this repository runs a checkout step
- **THEN** the step sets `persist-credentials: false`, since none of the later steps fetches or pushes through the checkout's remote
