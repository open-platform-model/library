## ADDED Requirements

### Requirement: The receiver runs on dispatch, daily and by hand

The library SHALL have a workflow `.github/workflows/deps-cascade.yml` that calls the org's shared `cascade-receive.yml` reusable workflow at `@main`. It SHALL run on `repository_dispatch` of type `upstream-released`, on a daily schedule at `17 5 * * *` UTC, and on `workflow_dispatch` with the boolean inputs `dry_run` and `gates_only`. It SHALL declare `permissions: {}` at the top and grant the calling job only `contents: read`, `pull-requests: read` and `statuses: write`. It SHALL pass no secrets. It SHALL pass `setup-go: false` and `labels-managed: false`, and leave the CUE version at the reusable workflow's default. It SHALL use the concurrency group `deps-cascade` without cancelling for real runs on `main`, `deps-cascade-gates` for gates-only runs on `main`, and `deps-cascade-<ref>` for runs from any other ref. Source: workspace `RELEASING.md`, sections "The receiver" and "Concurrency"; Phase 3 wiring contract §5 and §5.1.

#### Scenario: An upstream release arrives

- **WHEN** core or catalog_opm sends `upstream-released` to the library
- **THEN** a `Deps cascade` run starts on `main` in the group `deps-cascade` and calls `cascade-receive.yml`

#### Scenario: The daily sweep

- **WHEN** no dispatch arrived and the schedule fires
- **THEN** a `Deps cascade` run starts with no payload and resolves the newest published upstreams itself

#### Scenario: A gates-only run does not displace a pending real run

- **WHEN** a real run is active, another real run is pending, and a gates-only run is dispatched
- **THEN** the gates-only run runs in group `deps-cascade-gates`, and the pending run in `deps-cascade` is neither replaced nor cancelled

### Requirement: The receiver pushes nothing until CASCADE_DRY_RUN is exactly false

`deps-cascade.yml` SHALL pass `dry-run: true` to the reusable workflow unless the repo variable `CASCADE_DRY_RUN` is exactly `false` and the run was not started with `dry_run: true`. An unset, deleted or any other value SHALL mean a dry run. In a dry run the receiver computes the diff, title, body and labels and writes them to the job summary, and pushes nothing, opens or edits no PR, and adds no label or comment. The gate statuses SHALL still be posted in a dry run. Source: workspace `RELEASING.md`, section "Stop switches"; Phase 3 wiring contract §9.1.

#### Scenario: The variable is unset

- **WHEN** a dispatch run starts and `CASCADE_DRY_RUN` is not set
- **THEN** the run is a dry run and `deps/cascade` is not pushed

#### Scenario: Live, but a manual dry run

- **WHEN** `CASCADE_DRY_RUN` is `false` and a human starts the workflow with `dry_run: true`
- **THEN** that run is a dry run and pushes nothing

#### Scenario: Nothing to move on main

- **WHEN** a dry run starts while every library pin is current on `main`
- **THEN** the job summary reports mode `fresh` and action `noop`, matching the exit 3 of `task -x deps:cascade` on a clean checkout of `main`

### Requirement: A core bump reaches the cascade PR as need-human-review

When a cascade run moves `DefaultSchemaModule`, the resulting PR SHALL carry the label `need-human-review`, and the label SHALL exist in the library with colour `e99695` and the description "Glue edits a human must review before merging". The library SHALL get this from its pin report's core row label and from the shared receiver adding every label the body's `cascade-labels` marker names; `deps-cascade.yml` SHALL pass `labels-managed: false`, so the receiver creates the five bot labels in this repo. A catalog-only move SHALL NOT add the label. The bot SHALL NOT remove the label once set. Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RELEASING.md`, section "Labels".

#### Scenario: Core moved

- **WHEN** a live run moves `DefaultSchemaModule` from `v2.0.0-beta.2` to `v2.0.0-beta.3`
- **THEN** the cascade PR carries `deps-cascade` and `need-human-review`, and its title starts `fix(deps):`

#### Scenario: Only the opm catalog moved

- **WHEN** a live run moves only the opm catalog
- **THEN** the cascade PR carries `deps-cascade` but not `need-human-review`, and its title starts `test(fixtures):`

#### Scenario: Dry run of a core move

- **WHEN** a dry run would move `DefaultSchemaModule`
- **THEN** the job summary lists `need-human-review` among the labels, and no label is created or added

### Requirement: Every PR carries the cascade gate statuses

The library SHALL have a workflow `.github/workflows/cascade-gates.yml` that runs on `pull_request_target` (`opened`, `reopened`, `synchronize`), declares `permissions: {}` at the top, grants the calling job only `statuses: write` and `actions: write`, and calls the org's shared `cascade-gates.yml` at `@main` with the modes from the repo variables `CASCADE_G2_MODE` and `CASCADE_G3_MODE`, each `warn` when unset. It SHALL check out no PR code. The concurrency group SHALL be per PR number, cancelling an older run. Neither `cascade/freshness` nor `cascade/settled` SHALL become a required check in this change. For a PR opened by Dependabot, whether the statuses can be posted is subject to the `.github` sandbox result E7 (a `pull_request_target` run for Dependabot may get a read-only token); until E7 is recorded this requirement does not promise the statuses on Dependabot PRs. Source: workspace `RELEASING.md`, section "Gates" (G2, G3); owner selections 11 and 12; Phase 3 wiring contract §8.3.

#### Scenario: An ordinary PR

- **WHEN** a PR whose head ref does not start with `release-please--` is opened
- **THEN** both `cascade/freshness` and `cascade/settled` are posted as `success` with the description `n/a: not a release PR`

#### Scenario: A release PR in warn mode

- **WHEN** release-please pushes to its `release-please--branches--main` PR and both mode variables are unset
- **THEN** a gates-only `Deps cascade` run is dispatched on `main`, and it posts both statuses on the new head as `success`, prefixed `WARN:` when a gate finds a problem

#### Scenario: A shipped pin is behind

- **WHEN** core `v2.0.0-beta.3` is published and the release PR head still pins `v2.0.0-beta.2` in `DefaultSchemaModule`
- **THEN** `cascade/freshness` on that head reads `WARN: behind: core v2.0.0-beta.2→v2.0.0-beta.3`
