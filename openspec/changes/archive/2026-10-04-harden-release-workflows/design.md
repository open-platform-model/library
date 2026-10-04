## Context

The library's workflows run on `pull_request`, `pull_request_target`, `push` to `main`, `schedule`, `workflow_dispatch` and `repository_dispatch`. Only `release.yml` reads `RELEASE_APP_PRIVATE_KEY`; only `release.yml` and `deps-cascade.yml` read `CASCADE_APP_PRIVATE_KEY`, both already in the `cascade` Environment and held in shape by `task cascade:wiring:check`.

## Goals / Non-Goals

**Goals:** every workflow states its token grants, so the repo default can become read-only without a failing run; the release key is readable only from `main`; no PR-run code gets a write token or a persisted credential; the linter binary is the one whose checksum is committed; the release and cascade files need a code owner's review.

**Non-Goals:** the `.github` pin bump and the canonical wiring check (wave 2), repo settings, rulesets and secrets (supervisor and owner).

## Decisions

- The `release-please` job MUST declare `environment: release` and SHALL have `permissions: {}`. It mints the App token from `vars.RELEASE_APP_CLIENT_ID` and the key, and release-please acts with that token only, so the job needs nothing from `GITHUB_TOKEN`. The mint step names `owner` and `repositories` (this repo) and asks for `permission-contents: write` and `permission-pull-requests: write` only: the release branch, tags and GitHub releases, and the release PR with its comments and labels (GitHub's label and comment endpoints accept a pull-requests grant). Without them the token carries every permission of the App's installation, including `issues: write`. The workflow-level `permissions:` becomes `{}`; the two other jobs already declare their own grants (`publish-docs`: `contents: read`, `packages: write`, `id-token: write`; `notify-downstream`: `contents: read`, which the wiring check holds).
- `cue.yml`, `lint.yml` and `test.yml` SHALL declare `permissions: contents: read` at the workflow level. Their steps read the repository and pull public CUE modules and Go modules anonymously; no step calls the GitHub API, pushes or publishes. Every repo checkout in a workflow that runs repository or dependency code SHALL set `persist-credentials: false`; no step there fetches or pushes through the checkout's remote.
- `pr-title.yml` (workflow `pull-requests: read`), `cascade-gates.yml`, `deps-cascade.yml` and `docs.yml` (each `permissions: {}` with per-job grants) already state their grants and stay as they are.
- golangci-lint is installed from `https://github.com/golangci/golangci-lint/releases/download/v2.8.0/golangci-lint-2.8.0-linux-amd64.tar.gz`, fetched with `--proto '=https'`, and checked with `sha256sum -c` against a digest in the step's `env` before extraction. A version bump moves `GOLANGCI_LINT_VERSION` and `GOLANGCI_LINT_SHA256` together by hand; Dependabot does not see a `run:` step.

## Research & Decisions

### How to pin golangci-lint
**Context**: The install script came from the mutable `HEAD` branch; its own checksum check runs inside the untrusted script.
**Explored**: (a) the install script from the `v2.8.0` tag; (b) `golangci/golangci-lint-action` pinned by SHA with `install-only`; (c) the release archive by URL plus a committed sha256.
**Decision**: (c).
**Rationale**: (a) still runs a remote script and trusts the checksum file it downloads next to the archive. (b) adds an action with its own download logic and an Actions cache by default. (c) is the only option whose trust root is a value in this repository. The digest `7048bc6b25c9515ed092c83f9fa8709ca97937ead52d9ff317a143299ee97a50` was read from the release's `golangci-lint-2.8.0-checksums.txt`, matched the GitHub asset digest, and matched a local `sha256sum` of the downloaded archive on 2026-10-04.

### Whether the release-please job needs any GITHUB_TOKEN grant
**Context**: Moving the workflow's `contents: write` and `pull-requests: write` off the job must not break the release PR or the tag.
**Explored**: The job's two steps: `actions/create-github-app-token` (signs a JWT with the key, calls the App API with it) and `googleapis/release-please-action` given `token: ${{ steps.app-token.outputs.token }}`.
**Decision**: `permissions: {}`.
**Rationale**: Neither step reads `github.token` when the token input is set. The deployment record the `release` Environment creates is written by Actions itself and needs no job grant, as `notify-downstream` (grant `contents: read`, Environment `cascade`) already shows.

### The in-tree wiring check
**Context**: The plan allows the smallest edit to `.tasks/cascade/wiring-check.sh` if it refuses this change.
**Explored**: Its Environment rule matches environments that mention `cascade` or are expressions; its permission rules cover only the two cascade key jobs.
**Decision**: No edit. `environment: release` does not match, and the notify job's grants do not change.

## Risks / Trade-offs

- [The App token lacks a scope the job used through `GITHUB_TOKEN`] → none known; release-please already ran with the App token alone for every API call. The first release after merge is the proof, and a failure there reruns cleanly.
- [The scoped App token lacks a permission release-please uses] → release-please's calls (commits, branches, tags, releases, PRs, PR labels and comments) fall under `contents` and `pull-requests`. If the first release run after merge is refused, adding `permission-issues: write` to the mint step restores what the token held before.
- [A later workflow is added without `permissions:`] → once the repo default is read-only it fails closed (read token), not open.
- [The CODEOWNERS file is itself under `/.github/`] → intended: changing who owns the release files needs an owner's review.
