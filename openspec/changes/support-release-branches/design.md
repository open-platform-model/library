## Context

See proposal.md for the motivation. State on `main` at planning time (2026-10-01, `02344e5`):

- `.github/workflows/release.yml` has one job, `release-please`, triggered by pushes to
  `main` only: mint the release App token, then run `googleapis/release-please-action`
  v5.0.0 (pinned by SHA `45996ed`) without a `target-branch` input.
- `release-please-config.json` has one package, `"."` (release-type `go`,
  `versioning: prerelease`, `prerelease: true`, `prerelease-type: beta`); the manifest holds
  `1.0.0-beta.1`. Tags are `vX.Y.Z[-beta.N]` (no component, `v` prefix).
- The CI workflows (`test.yml`, `lint.yml`, `cue.yml`) trigger on `pull_request` with no
  branch filter, so a backport PR into `release/v1.0` already gets the full gate set;
  `pr-title.yml` runs on `pull_request_target` for every base branch too.
- The org (owner-configured, outside this repo): ruleset `tags-immutable` refuses tag update
  and deletion with no bypass; ruleset `tags-create-app-only` restricts tag creation to the
  `opm-release-please` App; ruleset `release-branches` covers `refs/heads/release/*` with
  deletion, non-fast-forward and pull_request (squash only), no bypass.

## Goals / Non-Goals

**Goals:**

- A push to a `release/vX.Y` branch runs release-please against that branch, so the branch
  gets its own release PR and its own patch tags.
- Cutting a release branch is one dispatch that delegates to the shared org workflow; the
  library carries only a thin caller.
- Agents and humans reading `AGENTS.md` find the immutable-tag rule, the Go recovery and the
  branch model.

**Non-Goals:**

- Cutting a branch now. Beta fixes roll forward on `main`.
- Changing `main`'s release-please configuration. The branch-local settings arrive with the
  PR the shared workflow opens into the new branch.
- A per-repo tag check. Tag creation is App-only and tags cannot move, so a tag naming the
  wrong commit cannot arise; the check was dropped by the owner on 2026-10-01.
- Tag or release mutation of any kind, and auto-repair.

## Research & Decisions

### Release-please must be told the branch

**Context**: On a push to `release/v1.0` the action must open its release PR against, and
tag from, that branch.
**Explored**: `action.yml` at the pinned SHA `45996ed`: input `target-branch`, "branch to
open pull release PR against (detected by default)", default `''`. `src/index.ts` passes it
as `defaultBranch` to the release-please `Manifest`; empty means the repository's default
branch, `main`. So without the input a release-branch push would release `main`'s state.
**Decision**: The release step sets `target-branch: ${{ github.ref_name }}`. For a push to
`main` that is `main`, identical to today's detected default; for `release/v1.0` it is
`release/v1.0`.
**Rationale**: One job serves every branch with no conditional; `main` keeps its exact
behaviour. The value is a branch name GitHub supplies for a push event matching the
trigger, so it is never user-controlled text beyond the branch name the ruleset governs,
and it is passed as an action input, not interpolated into a shell body.

### Trigger pattern

**Context**: Which pushes run the release job.
**Explored**: `release/**` (as the sibling repos use) versus `release/v*`.
**Decision**: `branches: [main, 'release/**']`, matching the shared workflow's assumption and
the other four repos.
**Rationale**: One pattern across repos keeps the shared workflow's "ensure the release
trigger covers the branch" step a no-op here. A stray non-version branch under `release/`
would only get a release PR nobody merges; the AGENTS line says branches are cut by the
dispatch only.

### Thin caller for the shared cut workflow

**Context**: The cut (branch from the highest `vX.Y.*` tag, then a PR setting
`versioning: always-bump-patch` and `prerelease: false` for package `.`) is identical in
five repos up to the tag prefix and package path.
**Explored**: an inline job per repo (five copies to keep in sync, the drift problem the
workspace already has with `hack/fixtures.sh`) versus a reusable workflow in
`open-platform-model/.github` (public, so callable from every org repo).
**Decision**: `.github/workflows/cut-release-branch.yml`, `workflow_dispatch` with one
required string input `minor` (`X.Y`), one job that `uses:` the org workflow pinned by
full commit SHA (version comment beside it), `with: tag_prefix: v`, `minor`, package path
`.`, `secrets: inherit` so the called workflow can mint the release App token, and job
permissions `contents: write`, `pull-requests: write` (the ceiling the called workflow
gets).

```yaml
on:
  workflow_dispatch:
    inputs:
      minor:
        description: Released minor to branch, X.Y (for example 1.0)
        required: true
        type: string
permissions:
  contents: write
  pull-requests: write
jobs:
  cut:
    uses: open-platform-model/.github/.github/workflows/cut-release-branch.yml@<sha> # <ref>
    with:
      tag_prefix: v
      minor: ${{ inputs.minor }}
      package_path: "."   # input name as the shared workflow finally declares it
    secrets: inherit
```

**Rationale**: The library keeps no cut logic, so a fix to the cut lands once. Pinning by
SHA matches every other action in this repo; Dependabot keeps it current. The exact input
names are confirmed against the merged shared workflow at apply time (task 2.1).

### Commit types

**Decision**: `ci(release)` for both workflow commits, `chore(agents)` for the `AGENTS.md`
line, `chore(openspec)` for the archive. PR title
`ci(release): support release/vX.Y maintenance branches`.
**Rationale**: `docs` releases in this repo; none of these changes alter what a consumer
gets.

## Risks / Trade-offs

- [The release-branch path never runs for real until the first branch is cut, at GA] →
  `main` is unaffected by construction (`github.ref_name` equals the detected default), and
  the first cut is a deliberate owner action whose release PR is reviewed before anything is
  tagged. The cut and the branch release are exercised in `release-flow-sandbox` before the
  first real cut.
- [The shared workflow's interface changes before apply] → section 2 starts by reading the
  merged `cut-release-branch.yml` and records its SHA and input names; it does not start
  until that workflow is on `.github` `main`.
- [`secrets: inherit` hands every library secret to the called workflow] → the called
  workflow lives in an org repo with the same owners and is pinned by SHA, so what runs is
  reviewed code. If the shared workflow declares its secrets explicitly, task 2.2 passes
  only those instead.

## Migration Plan

None. Rollback is reverting the commits; no branch exists yet and nothing holds state.
