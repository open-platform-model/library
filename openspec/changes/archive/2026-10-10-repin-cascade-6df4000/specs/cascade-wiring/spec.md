## MODIFIED Requirements

### Requirement: The cascade references carry one .github SHA, checked on every PR

Every cascade reference in the library's workflows SHALL name the same full 40-character SHA of a commit on `open-platform-model/.github` `main`, followed by the comment `# .github main`: the `cascade-notify` step in `release.yml`, the `cascade-receive.yml` call and the `cascade-publish` step in `deps-cascade.yml`, the `cascade-gates.yml` call in `cascade-gates.yml`, and the `ref:` of the `open-platform-model/.github` checkout in `cascade-task.yml`. No reference SHALL name a branch or a tag. `cascade-task.yml` SHALL fail, not skip S5, when the resolver is missing at that commit. `.github/dependabot.yml` SHALL ignore `open-platform-model/.github*` under `github-actions`. `.github/dependabot.yml` SHALL give the `github-actions` entry a cooldown of 7 days. The script `.tasks/cascade/wiring-check.sh` SHALL be a byte-identical copy of `.github/scripts/cascade/wiring-check.sh` at the pinned SHA, with the library's values in `.tasks/cascade/wiring-check.yaml` (receiver, an empty `env-allow`, `publish-workflows` `release.yml` and `docs.yml`, CI workflow `test.yml` job `test`, the notify `needs`, `if:` and `tag` of `release.yml`, `labels-managed: false`). It SHALL check these references and the shapes of this capability, SHALL refuse any `release.yml` workflow-level `env` key and any `runs-on` other than `ubuntu-latest` on `notify-downstream` and `publish`, and SHALL run offline through `task cascade:wiring:check` (also in the aggregate `task check`) and, in the required `Go tests` job on every PR, as the step "Verify the cascade wiring" with `GH_TOKEN: ${{ github.token }}` and `run: bash .tasks/cascade/wiring-check.sh --pin-on-main`, which also confirms the SHA is on `.github`'s `main` and that the running copy is byte for byte the file at that SHA. In `test.yml`, the workflow's and the `test` job's `env` SHALL name only `CUE_*`, `OPM_*`, `REGISTRY` or `IMAGE_NAME`, the job SHALL have no `container` or `services`, and every step before "Verify the cascade wiring" SHALL be an action from another repo pinned to a full SHA with only `id`, `name`, `uses` and `with`. The copy moves only with the pin. Source: owner decision 24, extended to the two reusable workflows and the resolver checkout by the change join-release-cascade (`openspec/changes/archive/2026-10-04-join-release-cascade`); `.github` README at `6df4000`, "Pinning and bumps", "The wiring check" and "Keeping the copy in sync"; Phase 3 wiring contract (version 3.1, changelogs 3.1.1 and 3.1.2) §2.4 and §10.1 items 2, 3, 6 and 7, with the `release.yml` env allow-list and `runs-on` rules the join-release-cascade change added (its `design.md`).

#### Scenario: The wiring is as the contract gives it

- **WHEN** `Go tests` runs on a PR that leaves the cascade workflows unchanged
- **THEN** the step "Verify the cascade wiring" prints `cascade wiring: ok, .github <SHA> (.github main)`

#### Scenario: One reference moves alone

- **WHEN** a PR changes the `cascade-gates.yml` call to another SHA, or to `@main`, and leaves the other four references
- **THEN** the step "Verify the cascade wiring" fails and names the mismatch

#### Scenario: A key-holding job gains an env

- **WHEN** a PR adds `env: {BASH_ENV: repo/x.sh}` to the `publish` job, or any key to `release.yml`'s workflow-level `env`
- **THEN** the step "Verify the cascade wiring" fails

#### Scenario: A .github cascade change lands

- **WHEN** a new commit merges to `.github` `main`
- **THEN** the library's cascade keeps running its pinned commit until a `ci(deps): pin the cascade to .github <sha7>` PR moves all five references together

#### Scenario: A pin that is not on .github main

- **WHEN** a PR moves every cascade reference to a SHA that GitHub resolves under `open-platform-model/.github` but that is not on its `main` (a fork commit)
- **THEN** the step "Verify the cascade wiring" fails on the compare check, after every shape matched

#### Scenario: The copy drifts from .github

- **WHEN** a reviewer pipes `.github/scripts/cascade/wiring-check.sh` at the pinned SHA to `cmp - .tasks/cascade/wiring-check.sh`
- **THEN** `cmp` reports no difference

#### Scenario: CI catches a copy that differs from the pin

- **WHEN** a PR edits `.tasks/cascade/wiring-check.sh`, or moves the references to a new SHA without replacing the copy
- **THEN** the step "Verify the cascade wiring" fails with `differs from`, after every shape matched and the compare check passed

#### Scenario: A run step before the wiring step

- **WHEN** a PR adds a `run:` step to the `test` job before "Verify the cascade wiring", or a `BASH_ENV` key to the workflow's `env`
- **THEN** the step "Verify the cascade wiring" fails
