## Why

Phase 2 gave the library `task -x deps:cascade` (archived as `2026-10-04-add-deps-cascade-task`), but nothing runs it and nothing tells the library's downstream repos when a library release is out. Phase 3 of the rollout (workspace `RELEASING.md`, section "Rollout and changes", the row `3 | core and the four | join-release-cascade`) joins each repo to the cascade: the upstream side notifies, the downstream side receives, and release PRs carry the G2 and G3 gate statuses.

The library sits in the middle of the graph (`RELEASING.md`, "Notify after publish"). It receives from core and catalog_opm, and dispatches to opm-operator and cli. Today:

- `release.yml` already exposes `releases_created` and `tag_name` (`.github/workflows/release.yml:18-20`, Phase 1 `prepare-release-cascade`), but no job reads them.
- There is no `deps-cascade.yml`, so a core or catalog release reaches the library only when someone runs the task by hand.
- No release PR carries `cascade/freshness` or `cascade/settled` (`RELEASING.md`, "Gates").
- The library has none of the cascade labels, so the owner's `need-human-review` checkpoint on a core bump (owner selection 9) has no label to land on yet.
- `cascade-task.yml` checks the shared resolver out at `ref: main` and skips S5 when the resolver is missing, a fallback from before the resolver reached `.github` `main`.

The shared cascade code is built once in the org `.github` repo by its change `add-release-cascade-workflows` (`.github` PR 9, merged as `2376ffae4bfc665f327d51581350dea694c01504`): two composite actions, `cascade-notify` and `cascade-publish`, and two reusable workflows, `cascade-receive.yml` and `cascade-gates.yml`. The interface every repo implements is the Phase 3 wiring contract (version 3.1, with changelogs 3.1.1 and 3.1.2), archived with that change at `.github` `openspec/changes/archive/2026-10-04-add-release-cascade-workflows/contract.md` and cited as "wiring §N", together with the supervisor's addendum for the five join changes (the `release.yml` env allow-list, `runs-on: ubuntu-latest` on the key-holding jobs, the canary rule, the dropped sandbox). Where those and `RELEASING.md` disagree, the branch stops and asks; it does not pick a side (wiring, preamble).

## What Changes

- **Notify.** `release.yml` gains a job `notify-downstream`, the library block of wiring §4.6 byte for byte: it depends on `release-please` only, runs when release-please cut a release and the repo variable `CASCADE_NOTIFY` is not `off`, declares `environment: cascade`, grants `contents: read`, and has one step that runs the `cascade-notify` action pinned to the `.github` `main` SHA with the released tag, `vars.CASCADE_APP_CLIENT_ID` and `secrets.CASCADE_APP_PRIVATE_KEY` as inputs. The action waits for the Go proxy (best effort, up to 10 minutes) because the repo is `library`, mints the App token scoped to opm-operator and cli, and dispatches (wiring §4.1). `publish-docs` never gates it.
- **Receiver.** A new `.github/workflows/deps-cascade.yml`, the wiring §5 file with the library `jobs:` map of §5.2: the job `cascade` calls the reusable `cascade-receive.yml` (compute and gates, no secret in reach) on `repository_dispatch` (`upstream-released`), daily at `17 5 * * *`, and on `workflow_dispatch` (`dry_run`, `gates_only`), with `setup-go: false` and CUE at the reusable default `v0.17.1`; the caller-owned job `publish` declares `environment: cascade` and runs the pinned `cascade-publish` action with the §5 `dry-run` expression, `labels-managed: false` and the key as an input.
  - It is a dry run until the repo variable `CASCADE_DRY_RUN` is `false` (wiring §9.1). GitHub compares expression strings without regard to case, so `False` counts as `false` too. The supervisor set it to `true` before this PR merges; Phase 4 sets `false`. The switch is read in three places: the reusable `dry-run` input, the `publish` `if:`, and the action's required `dry-run` input, which mints nothing unless it is `false`.
  - The receiver accepts payloads from core and catalog_opm only (wiring §3.2). That allowlist lives in `.github`, not here.
- **Gates caller.** A new `.github/workflows/cascade-gates.yml` (wiring §8.3, byte for byte but the header comment) posts `cascade/freshness` and `cascade/settled` on every PR through the pinned reusable `cascade-gates.yml`: `n/a` on ordinary PRs, and a gates-only receiver run for each `release-please--*` PR. Both stay warnings (`CASCADE_G2_MODE`, `CASCADE_G3_MODE` unset means `warn`; owner selections 11 and 12). Neither becomes a required check here.
- **One pin.** Every cascade reference names the same full `.github` `main` SHA with the comment `# .github main` (owner decision 24 for the actions; the supervisor's extension for the two reusable workflows and the resolver checkout): the notify and publish steps, the `cascade-receive.yml` and `cascade-gates.yml` calls, and the `ref:` of the resolver checkout in `cascade-task.yml`, which also loses its S5 skip fallback (wiring §10.1 item 3). A `.github` change reaches the library only through a `ci(deps): pin the cascade to .github <sha7>` PR (wiring §2.4).
- **Wiring check.** A new `.tasks/cascade/wiring-check.sh`, the wiring §10.1 item 6 script with the addendum's two additions (library: no `release.yml` workflow `env` key allowed; `runs-on: ubuntu-latest` on `notify-downstream` and `publish`), run by a new `task cascade:wiring:check`, by the aggregate `task check`, and as the step "Verify the cascade wiring" in the required `Go tests` job. It keeps the two key-holding jobs in the shape the contract gives them after merge. It guards against mistakes; review plus the `main` ruleset guard against a deliberate edit.
- **Dependabot.** `.github/dependabot.yml` ignores `open-platform-model/.github*` under `github-actions` (wiring §10.1 item 7), so Dependabot never moves one cascade reference alone.
- **`need-human-review` on a core bump.** No new code in the library: `pins.sh` already labels the core row `need-human-review` (`.tasks/cascade/pins.sh:41`), the resolver turns that into the body's `cascade-labels` marker, and `cascade-publish` adds every marker label to the PR and, with `labels-managed: false`, creates the five bot labels in this repo with the `RELEASING.md` colours (wiring §6.4, §7.4). The real resolver's marker on a core move is shown by the spike (task 1.2); the first live core-move PR is checked after Phase 4 (task 8.6).
- `AGENTS.md` "Release cascade task" describes the receiver, the notify job, the pin, the wiring check and the repo variables.

Not in this change:

- The shared actions, workflows and scripts (`.github` `add-release-cascade-workflows`, merged).
- Setting `CASCADE_DRY_RUN`, `CASCADE_NOTIFY`, `CASCADE_G2_MODE` or `CASCADE_G3_MODE`. Those are repo variables the supervisor or owner sets.
- Going live (Phase 4) and making G2 or G3 required (Phase 5, `require-pin-freshness-gate`).
- Any edit to the cascade task itself (`cascade.sh`, `lib.sh`, `pins.sh`, `test.sh`) or `opm/`.

Depends on (merge gates, not tasks):

- `.github` `add-release-cascade-workflows` is merged (done: squash `2376ffae4bfc665f327d51581350dea694c01504`), and `gh api repos/open-platform-model/.github/compare/<SHA>...main --jq .status` prints `identical` or `ahead` for the pinned SHA (wiring §10.1 "Pre-merge check").
- The repo variable `CASCADE_DRY_RUN=true` is set in the library (done by the supervisor, wiring "Supervisor records").
- Already merged: the library's `prepare-release-cascade` (release job outputs) and `add-deps-cascade-task` (the task), and the Phase 0 settings: the `cascade` Environment (`main` only) with `CASCADE_APP_PRIVATE_KEY` and `CASCADE_APP_CLIENT_ID`.

## Capabilities

### New Capabilities

- `cascade-wiring`: how the library takes part in the cascade at run time in CI: the receiver's triggers, its caller-owned publish job and dry-run rule, the per-PR gate statuses, the one `.github` pin and the wiring check that holds it, and how a core bump reaches the PR as `need-human-review`. It is a capability of its own, not an addition to `deps-cascade`: `deps-cascade` specifies the repo-local task (`task -x deps:cascade`, its exit codes, its edits and its tests), which runs the same on a laptop as in CI, while `cascade-wiring` specifies the GitHub workflows that call it and the shared `.github` code, which the task neither knows nor needs.

### Modified Capabilities

- `release-pipeline`: one added requirement, "A release notifies the library's downstream repos". No existing requirement or scenario changes.

## Impact

**SemVer: no release.** Every commit is `ci(...)`, `docs(...)` or `chore(...)`, which release-please hides. No file under `opm/` changes, the public surface is unchanged, and so are cli and opm-operator. The PR title is `ci: join the release cascade` (wiring §1).

**Files:** `.github/workflows/release.yml` (one job), `.github/workflows/deps-cascade.yml` (new), `.github/workflows/cascade-gates.yml` (new), `.github/workflows/cascade-task.yml` (resolver `ref:` and the S5 fallback), `.github/workflows/test.yml` (one step), `.github/dependabot.yml` (one ignore entry), `.tasks/cascade/wiring-check.sh` (new), `Taskfile.yml` (one task, one line in `check`), `AGENTS.md`.

**Required checks.** The library's `main` ruleset requires only `Go tests` (integration 15368) plus the org mention-guard. The wiring check is a step of `Go tests`, not a new context; the new workflows add no required context, so they cannot block an unrelated PR. The cascade PR itself is gated by `Go tests` like any other PR.

**Complexity (Principle VII).** Three caller files with no logic of their own (every rule lives once in `.github`), plus one check script of about 130 lines that holds the callers to the contract's shape. The alternative, copying the receiver into each repo, gives five copies of the key-handling code instead of one.
