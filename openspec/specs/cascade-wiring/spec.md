# cascade-wiring Specification

## Purpose
How the library's workflows join the org release cascade: the `deps-cascade.yml` receiver (the reusable `cascade-receive.yml` for compute and gates, and a caller-owned `publish` job that alone holds the App key), its dry-run switch `CASCADE_DRY_RUN`, the `cascade-gates.yml` PR gate caller, one pinned `.github` `main` SHA on every cascade reference, and the wiring check in the required `Go tests` job that refuses any other shape.

## Requirements

### Requirement: The receiver workflow calls cascade-receive.yml and runs on dispatch, on a schedule and by hand
The library SHALL have a workflow `.github/workflows/deps-cascade.yml` whose job `cascade` calls the org's reusable `cascade-receive.yml` (the `compute` and `gates` jobs, which hold no secret). It SHALL run on `repository_dispatch` of type `upstream-released`, on a daily schedule at `17 5 * * *` UTC, and on `workflow_dispatch` with the boolean inputs `dry_run` and `gates_only`.
**Source**: workspace `RELEASING.md`, sections "The receiver" and "Concurrency"; Phase 3 wiring contract (version 3.1) §5, §5.1 and §5.2.

#### Scenario: An upstream release arrives

- **WHEN** core or catalog_opm sends `upstream-released` to the library
- **THEN** a `Deps cascade` run starts on `main` in the group `deps-cascade` and calls `cascade-receive.yml`

#### Scenario: The daily sweep

- **WHEN** no dispatch arrived and the schedule fires
- **THEN** a `Deps cascade` run starts with no payload and resolves the newest published upstreams itself

### Requirement: The receiver workflow has fixed top-level keys and its cascade job has fixed grants
`.github/workflows/deps-cascade.yml`: its top-level keys SHALL be exactly `name`, `on`, `permissions`, `concurrency` and `jobs`, with `permissions: {}`. The `cascade` job SHALL be granted only `contents: read`, `pull-requests: read` and `statuses: write`, and SHALL pass `setup-go: false` and no `labels-managed` or `cue-version`, so CUE is the reusable workflow's default.
**Source**: workspace `RELEASING.md`, sections "The receiver" and "Concurrency"; Phase 3 wiring contract (version 3.1) §5, §5.1 and §5.2.

#### Scenario: The workflow keys and the cascade job grants are fixed

- **WHEN** `.github/workflows/deps-cascade.yml` is read
- **THEN** its top-level keys are exactly `name`, `on`, `permissions`, `concurrency` and `jobs`, with `permissions: {}`, and the `cascade` job is granted only `contents: read`, `pull-requests: read` and `statuses: write`

### Requirement: The receiver workflow uses a concurrency group by run kind
`.github/workflows/deps-cascade.yml`: it SHALL use the concurrency group `deps-cascade` without cancelling for real runs on `main`, `deps-cascade-gates` for gates-only runs on `main`, and `deps-cascade-<ref>` for runs from any other ref.
**Source**: workspace `RELEASING.md`, sections "The receiver" and "Concurrency"; Phase 3 wiring contract (version 3.1) §5, §5.1 and §5.2.

#### Scenario: A gates-only run does not displace a pending real run

- **WHEN** a real run is active, another real run is pending, and a gates-only run is dispatched
- **THEN** the gates-only run runs in group `deps-cascade-gates`, and the pending run in `deps-cascade` is neither replaced nor cancelled

### Requirement: The receiver has a publish job of a fixed shape with one step
`deps-cascade.yml` SHALL have a job `publish` that needs `cascade`, runs on `ubuntu-latest`, declares `environment: cascade`, has a 15-minute timeout, is granted only `contents: read` and `pull-requests: read`, and has exactly one step, which runs the `cascade-publish` action pinned to the library's `.github` SHA with the inputs `dry-run`, `gates-only: ${{ inputs.gates_only == true }}`, `labels-managed: false`, `client-id` and `private-key`, and nothing else.
**Source**: Phase 3 wiring contract (version 3.1) §2.2, §5 and §10.1 item 9; workspace `RELEASING.md`, section "Two-job split"; `.github` README at `7b9ad1b`, "Receiver caller".

#### Scenario: A live run with a change to push

- **WHEN** `CASCADE_DRY_RUN` is `false`, the run is on `main`, and compute plans the action `push`
- **THEN** the `publish` job starts in the `cascade` Environment, and the `cascade-publish` action mints the App token and pushes `deps/cascade`

#### Scenario: Compute plans nothing to publish

- **WHEN** compute plans the action `noop` or `skip`
- **THEN** the `publish` job is skipped and no `cascade` deployment is recorded

### Requirement: The publish job never starts for a gates-only run
The `publish` job's `if:` SHALL require `inputs.gates_only != true`, read from the caller's own input and never from an output of the reusable job, which ran repo code.
**Source**: Phase 3 wiring contract (version 3.1) §2.2, §5 and §10.1 item 9; workspace `RELEASING.md`, section "Two-job split"; `.github` README at `7b9ad1b`, "Receiver caller".

#### Scenario: A gates-only run never reaches publish

- **WHEN** `Cascade gates` dispatches `deps-cascade.yml` with `gates_only: true` while `CASCADE_DRY_RUN` is `false`
- **THEN** the `publish` job is skipped, and were it started, the `cascade-publish` action would fail before minting because its `gates-only` input is not `false`

### Requirement: Only the caller-owned notify-downstream or publish job reads the App key and no reusable call passes secrets
`secrets.CASCADE_APP_PRIVATE_KEY`: the key SHALL be read only in the caller-owned `notify-downstream` or `publish` job, which declares `environment: cascade` and passes `secrets.CASCADE_APP_PRIVATE_KEY` only as the `private-key` input of the SHA-pinned cascade action; that job SHALL have no checkout or `run:` of its own, and no `env:`, `container:` or `services:` (the action checks out the repo but never runs it); no reusable call SHALL pass `secrets:` or `secrets: inherit`.
**Source**: Phase 3 wiring contract (version 3.1) §2.2, §5 and §10.1 item 9; workspace `RELEASING.md`, section "Two-job split"; `.github` README at `7b9ad1b`, "Receiver caller".

#### Scenario: A reusable call is given the secrets

- **WHEN** a PR adds `secrets: inherit` to the `cascade` job, or reads `secrets.CASCADE_APP_PRIVATE_KEY` in any other job
- **THEN** the `Go tests` job fails at the step "Verify the cascade wiring"

### Requirement: No other job in any library workflow declares the cascade Environment
No other job in any library workflow SHALL declare the `cascade` Environment.
**Source**: Phase 3 wiring contract (version 3.1) §2.2, §5 and §10.1 item 9; workspace `RELEASING.md`, section "Two-job split"; `.github` README at `7b9ad1b`, "Receiver caller".

#### Scenario: Another job reaches the key through a variant spelling

- **WHEN** a PR gives a job other than `notify-downstream` and `publish` an `environment` that names `cascade` in any letter case or is an expression, or reads the key as `secrets.cascade_app_private_key`, `secrets['CASCADE_APP_PRIVATE_KEY']` or `toJSON(secrets)`
- **THEN** the `Go tests` job fails at the step "Verify the cascade wiring"

### Requirement: The receiver treats an unset or other CASCADE_DRY_RUN value as a dry run
`deps-cascade.yml` SHALL pass `${{ inputs.dry_run == true || vars.CASCADE_DRY_RUN != 'false' }}` as `dry-run` to both the reusable workflow and the `cascade-publish` step, and the `publish` job's `if:` SHALL also require `inputs.dry_run != true`, `vars.CASCADE_DRY_RUN == 'false'` and `github.ref == 'refs/heads/main'`. An unset, deleted or any other value SHALL mean a dry run.
**Source**: workspace `RELEASING.md`, section "Stop switches"; Phase 3 wiring contract (version 3.1) §5 and §9.1.

#### Scenario: The variable is unset

- **WHEN** a dispatch run starts and `CASCADE_DRY_RUN` is not set
- **THEN** the run is a dry run, the `publish` job is skipped, and `deps/cascade` is not pushed

#### Scenario: The dry-run expression reaches both consumers

- **WHEN** `deps-cascade.yml` runs
- **THEN** it passes `${{ inputs.dry_run == true || vars.CASCADE_DRY_RUN != 'false' }}` as `dry-run` to both the reusable workflow and the `cascade-publish` step

### Requirement: The receiver publishes only when CASCADE_DRY_RUN is false and the run is not a manual dry run
(Requirement: The receiver treats an unset or other CASCADE_DRY_RUN value as a dry run) So the receiver SHALL publish only when the repo variable `CASCADE_DRY_RUN` is `false` (GitHub compares expression strings without regard to case, so `False` or `FALSE` also counts) and the run was not started with `dry_run: true`, and the `cascade-publish` action SHALL mint nothing unless its `dry-run` input is `false`, so a mistyped `if:` cannot make a dry run publish.
**Source**: workspace `RELEASING.md`, section "Stop switches"; Phase 3 wiring contract (version 3.1) §5 and §9.1.

#### Scenario: Live, but a manual dry run

- **WHEN** `CASCADE_DRY_RUN` is `false` and a human starts the workflow with `dry_run: true`
- **THEN** that run is a dry run and pushes nothing

### Requirement: A dry run writes its plan to the job summary and changes nothing
In a dry run the receiver SHALL compute the diff, title, body and labels and write them to the job summary, and push nothing, open or edit no PR, and add no label or comment. The gate statuses SHALL still be posted in a dry run.
**Source**: workspace `RELEASING.md`, section "Stop switches"; Phase 3 wiring contract (version 3.1) §5 and §9.1.

#### Scenario: Nothing to move on main

- **WHEN** a dry run starts while every library pin is current on `main`
- **THEN** the job summary reports mode `fresh` and action `noop`, matching the exit 3 of `task -x deps:cascade` on a clean checkout of `main`

#### Scenario: The gate statuses are still posted in a dry run

- **WHEN** a dry run starts
- **THEN** the gate statuses are posted

### Requirement: A core move labels the cascade PR need-human-review, a catalog-only move does not, and the bot never removes the label
When a cascade run moves `DefaultSchemaModule`, the resulting PR SHALL carry the label `need-human-review`, and the label SHALL exist in the library with colour `e99695` and the description "Glue edits a human must review before merging". A catalog-only move SHALL NOT add the label. The bot SHALL NOT remove the label once set.
**Source**: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RELEASING.md`, section "Labels".

#### Scenario: Core moved

- **WHEN** a live run moves `DefaultSchemaModule` from `v2.0.0-beta.2` to `v2.0.0-beta.3`
- **THEN** the cascade PR carries `deps-cascade` and `need-human-review`, and its title starts `fix(deps):`

#### Scenario: Only the opm catalog moved

- **WHEN** a live run moves only the opm catalog
- **THEN** the cascade PR carries `deps-cascade` but not `need-human-review`, and its title starts `test(fixtures):`

#### Scenario: Dry run of a core move

- **WHEN** a dry run would move `DefaultSchemaModule`
- **THEN** the job summary lists `need-human-review` among the labels, and no label is created or added

### Requirement: The need-human-review label comes from the pin report's core row and from the cascade-publish action
(Requirement: A core move labels the cascade PR need-human-review, a catalog-only move does not, and the bot never removes the label) The library SHALL get this from its pin report's core row label and from the `cascade-publish` action adding every label the body's `cascade-labels` marker names; the `publish` step SHALL pass `labels-managed: false`, so the action creates the five bot labels in this repo.
**Source**: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RELEASING.md`, section "Labels".

#### Scenario: The publish step leaves label creation to the action

- **WHEN** the `publish` step runs
- **THEN** it passes `labels-managed: false`, so the action creates the five bot labels in this repo

### Requirement: The cascade-gates.yml workflow runs on pull_request_target with fixed grants and checks out no PR code
The library SHALL have a workflow `.github/workflows/cascade-gates.yml` that runs on `pull_request_target` (`opened`, `reopened`, `synchronize`), declares `permissions: {}` at the top, grants the calling job only `statuses: write` and `actions: write`, and calls the org's reusable `cascade-gates.yml`, pinned to the library's `.github` SHA, with the modes from the repo variables `CASCADE_G2_MODE` and `CASCADE_G3_MODE`, each `warn` when unset. It SHALL check out no PR code.
**Source**: workspace `RELEASING.md`, section "Gates" (G2, G3); owner selections 11 and 12; Phase 3 wiring contract (version 3.1) §8.3.

#### Scenario: An ordinary PR

- **WHEN** a PR whose head ref does not start with `release-please--` is opened
- **THEN** both `cascade/freshness` and `cascade/settled` are posted as `success` with the description `n/a: not a release PR`

#### Scenario: A release PR in warn mode

- **WHEN** release-please pushes to its `release-please--branches--main` PR and both mode variables are unset
- **THEN** a gates-only `Deps cascade` run is dispatched on `main`, and it posts both statuses on the new head as `success`, prefixed `WARN:` when a gate finds a problem

#### Scenario: A shipped pin is behind

- **WHEN** core `v2.0.0-beta.3` is published and the release PR head still pins `v2.0.0-beta.2` in `DefaultSchemaModule`
- **THEN** `cascade/freshness` on that head reads `WARN: behind: core v2.0.0-beta.2→v2.0.0-beta.3`

### Requirement: The cascade gates cancel an older run per PR and are no required check
`.github/workflows/cascade-gates.yml`: The concurrency group SHALL be per PR number, cancelling an older run. Neither `cascade/freshness` nor `cascade/settled` SHALL become a required check in this change.
**Source**: workspace `RELEASING.md`, section "Gates" (G2, G3); owner selections 11 and 12; Phase 3 wiring contract (version 3.1) §8.3.

#### Scenario: A newer run cancels an older run of the same PR

- **WHEN** two runs of `cascade-gates.yml` start for one PR number
- **THEN** the older run is cancelled

#### Scenario: The gate statuses are not required checks

- **WHEN** this change lands
- **THEN** neither `cascade/freshness` nor `cascade/settled` is a required check

### Requirement: Every cascade reference names the same full .github main SHA and no branch or tag
Every cascade reference in the library's workflows SHALL name the same full 40-character SHA of a commit on `open-platform-model/.github` `main`, followed by the comment `# .github main`: the `cascade-notify` step in `release.yml`, the `cascade-receive.yml` call and the `cascade-publish` step in `deps-cascade.yml`, the `cascade-gates.yml` call in `cascade-gates.yml`, and the `ref:` of the `open-platform-model/.github` checkout in `cascade-task.yml`. No reference SHALL name a branch or a tag.
**Source**: owner decision 24, extended to the two reusable workflows and the resolver checkout by the change join-release-cascade (`openspec/changes/archive/2026-10-04-join-release-cascade`); `.github` README at `6df4000`, "Pinning and bumps", "The wiring check" and "Keeping the copy in sync"; Phase 3 wiring contract (version 3.1, changelogs 3.1.1 and 3.1.2) §2.4 and §10.1 items 2, 3, 6 and 7, with the `release.yml` env allow-list and `runs-on` rules the join-release-cascade change added (its `design.md`).

#### Scenario: One reference moves alone

- **WHEN** a PR changes the `cascade-gates.yml` call to another SHA, or to `@main`, and leaves the other four references
- **THEN** the step "Verify the cascade wiring" fails and names the mismatch

#### Scenario: A .github cascade change lands

- **WHEN** a new commit merges to `.github` `main`
- **THEN** the library's cascade keeps running its pinned commit until a `ci(deps): pin the cascade to .github <sha7>` PR moves all five references together

### Requirement: cascade-task.yml fails, not skips S5, when the resolver is missing
(Requirement: Every cascade reference names the same full .github main SHA and no branch or tag) `cascade-task.yml` SHALL fail, not skip S5, when the resolver is missing at that commit.
**Source**: owner decision 24, extended to the two reusable workflows and the resolver checkout by the change join-release-cascade (`openspec/changes/archive/2026-10-04-join-release-cascade`); `.github` README at `6df4000`, "Pinning and bumps", "The wiring check" and "Keeping the copy in sync"; Phase 3 wiring contract (version 3.1, changelogs 3.1.1 and 3.1.2) §2.4 and §10.1 items 2, 3, 6 and 7, with the `release.yml` env allow-list and `runs-on` rules the join-release-cascade change added (its `design.md`).

#### Scenario: A missing resolver fails the task

- **WHEN** the resolver is missing at the pinned commit
- **THEN** `cascade-task.yml` fails and does not skip S5

### Requirement: Dependabot ignores the .github repository and waits 7 days on github-actions
`.github/dependabot.yml` SHALL ignore `open-platform-model/.github*` under `github-actions`. `.github/dependabot.yml` SHALL give the `github-actions` entry a cooldown of 7 days.
**Source**: owner decision 24, extended to the two reusable workflows and the resolver checkout by the change join-release-cascade (`openspec/changes/archive/2026-10-04-join-release-cascade`); `.github` README at `6df4000`, "Pinning and bumps", "The wiring check" and "Keeping the copy in sync"; Phase 3 wiring contract (version 3.1, changelogs 3.1.1 and 3.1.2) §2.4 and §10.1 items 2, 3, 6 and 7, with the `release.yml` env allow-list and `runs-on` rules the join-release-cascade change added (its `design.md`).

#### Scenario: Dependabot leaves the .github pin alone

- **WHEN** Dependabot checks the `github-actions` entry
- **THEN** it ignores `open-platform-model/.github*` and applies a cooldown of 7 days

### Requirement: The wiring-check script is a byte-identical copy of the .github script and moves only with the pin
The script `.tasks/cascade/wiring-check.sh` SHALL be a byte-identical copy of `.github/scripts/cascade/wiring-check.sh` at the pinned SHA, with the library's values in `.tasks/cascade/wiring-check.yaml` (receiver, an empty `env-allow`, `publish-workflows` `release.yml` and `docs.yml`, CI workflow `test.yml` job `test`, the notify `needs`, `if:` and `tag` of `release.yml`, `labels-managed: false`). The copy moves only with the pin.
**Source**: owner decision 24, extended to the two reusable workflows and the resolver checkout by the change join-release-cascade (`openspec/changes/archive/2026-10-04-join-release-cascade`); `.github` README at `6df4000`, "Pinning and bumps", "The wiring check" and "Keeping the copy in sync"; Phase 3 wiring contract (version 3.1, changelogs 3.1.1 and 3.1.2) §2.4 and §10.1 items 2, 3, 6 and 7, with the `release.yml` env allow-list and `runs-on` rules the join-release-cascade change added (its `design.md`).

#### Scenario: The copy drifts from .github

- **WHEN** a reviewer pipes `.github/scripts/cascade/wiring-check.sh` at the pinned SHA to `cmp - .tasks/cascade/wiring-check.sh`
- **THEN** `cmp` reports no difference

### Requirement: The wiring check refuses shapes of this capability and runs offline through task cascade:wiring:check
`.tasks/cascade/wiring-check.sh`: it SHALL check these references (Requirement: Every cascade reference names the same full .github main SHA and no branch or tag) and the shapes of this capability, SHALL refuse any `release.yml` workflow-level `env` key and any `runs-on` other than `ubuntu-latest` on `notify-downstream` and `publish`, and SHALL run offline through `task cascade:wiring:check` (also in the aggregate `task check`).
**Source**: owner decision 24, extended to the two reusable workflows and the resolver checkout by the change join-release-cascade (`openspec/changes/archive/2026-10-04-join-release-cascade`); `.github` README at `6df4000`, "Pinning and bumps", "The wiring check" and "Keeping the copy in sync"; Phase 3 wiring contract (version 3.1, changelogs 3.1.1 and 3.1.2) §2.4 and §10.1 items 2, 3, 6 and 7, with the `release.yml` env allow-list and `runs-on` rules the join-release-cascade change added (its `design.md`).

#### Scenario: A key-holding job gains an env

- **WHEN** a PR adds `env: {BASH_ENV: repo/x.sh}` to the `publish` job, or any key to `release.yml`'s workflow-level `env`
- **THEN** the step "Verify the cascade wiring" fails

### Requirement: The wiring check runs in the required Go tests job on every PR and confirms the pin and the copy
`.tasks/cascade/wiring-check.sh`: it SHALL run, in the required `Go tests` job on every PR, as the step "Verify the cascade wiring" with `GH_TOKEN: ${{ github.token }}` and `run: bash .tasks/cascade/wiring-check.sh --pin-on-main`, which also confirms the SHA (Requirement: Every cascade reference names the same full .github main SHA and no branch or tag) is on `.github`'s `main` and that the running copy is byte for byte the file at that SHA.
**Source**: owner decision 24, extended to the two reusable workflows and the resolver checkout by the change join-release-cascade (`openspec/changes/archive/2026-10-04-join-release-cascade`); `.github` README at `6df4000`, "Pinning and bumps", "The wiring check" and "Keeping the copy in sync"; Phase 3 wiring contract (version 3.1, changelogs 3.1.1 and 3.1.2) §2.4 and §10.1 items 2, 3, 6 and 7, with the `release.yml` env allow-list and `runs-on` rules the join-release-cascade change added (its `design.md`).

#### Scenario: The wiring is as the contract gives it

- **WHEN** `Go tests` runs on a PR that leaves the cascade workflows unchanged
- **THEN** the step "Verify the cascade wiring" prints `cascade wiring: ok, .github <SHA> (.github main)`

#### Scenario: A pin that is not on .github main

- **WHEN** a PR moves every cascade reference to a SHA that GitHub resolves under `open-platform-model/.github` but that is not on its `main` (a fork commit)
- **THEN** the step "Verify the cascade wiring" fails on the compare check, after every shape matched

#### Scenario: CI catches a copy that differs from the pin

- **WHEN** a PR edits `.tasks/cascade/wiring-check.sh`, or moves the references to a new SHA without replacing the copy
- **THEN** the step "Verify the cascade wiring" fails with `differs from`, after every shape matched and the compare check passed

### Requirement: The test.yml workflow keeps a narrow env and only pinned actions before the wiring step
In `test.yml`, the workflow's and the `test` job's `env` SHALL name only `CUE_*`, `OPM_*`, `REGISTRY` or `IMAGE_NAME`, the job SHALL have no `container` or `services`, and every step before "Verify the cascade wiring" SHALL be an action from another repo pinned to a full SHA with only `id`, `name`, `uses` and `with`.
**Source**: owner decision 24, extended to the two reusable workflows and the resolver checkout by the change join-release-cascade (`openspec/changes/archive/2026-10-04-join-release-cascade`); `.github` README at `6df4000`, "Pinning and bumps", "The wiring check" and "Keeping the copy in sync"; Phase 3 wiring contract (version 3.1, changelogs 3.1.1 and 3.1.2) §2.4 and §10.1 items 2, 3, 6 and 7, with the `release.yml` env allow-list and `runs-on` rules the join-release-cascade change added (its `design.md`).

#### Scenario: A run step before the wiring step

- **WHEN** a PR adds a `run:` step to the `test` job before "Verify the cascade wiring", or a `BASH_ENV` key to the workflow's `env`
- **THEN** the step "Verify the cascade wiring" fails
