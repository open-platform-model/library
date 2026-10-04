## Why

Phase 2 gave the library `task -x deps:cascade` (archived as `2026-10-04-add-deps-cascade-task`), but nothing runs it and nothing tells the library's downstream repos when a library release is out. Phase 3 of the rollout (workspace `RELEASING.md`, section "Rollout and changes", the row `3 | core and the four | join-release-cascade`, `RELEASING.md:569`) joins each repo to the cascade: the upstream side notifies, the downstream side receives, and release PRs carry the G2 and G3 gate statuses.

The library sits in the middle of the graph (`RELEASING.md:169-175`). It receives from core and catalog_opm, and dispatches to opm-operator and cli. Today:

- `release.yml` already exposes `releases_created` and `tag_name` (`.github/workflows/release.yml:18-20`, Phase 1 `prepare-release-cascade`), but no job reads them.
- There is no `deps-cascade.yml`, so a core or catalog release reaches the library only when someone runs the task by hand.
- No release PR carries `cascade/freshness` or `cascade/settled` (`RELEASING.md:332-333`).
- The library has none of the cascade labels (`gh api repos/open-platform-model/library/labels` lists only GitHub's defaults, `autorelease: *`, `dependencies`, `go` and `github_actions`), so the owner's `need-human-review` checkpoint on a core bump (owner selection 9) has no label to land on yet.

The shared notify, receive and gates workflows are built once in the org `.github` repo by its change `add-release-cascade-workflows`. The interface every repo implements is the Phase 3 wiring contract (supervisor scratchpad `p3-wiring-contract.md`, version 2; cited as "wiring §N"). Where it and `RELEASING.md` disagree, the branch stops and asks; it does not pick a side (wiring, preamble).

## What Changes

- **Notify.** `release.yml` gains a job `notify-downstream` that calls `open-platform-model/.github/.github/workflows/cascade-notify.yml@main` with the released tag (wiring §4.3, §4.5 library row). It runs only when release-please cut a release and the repo variable `CASCADE_NOTIFY` is not `off`. It depends on `release-please` only: the library ships no artifact besides the tag, so `publish-docs` never gates it. The reusable workflow dispatches to opm-operator and cli and waits for the Go proxy first, best effort, for up to 10 minutes (wiring §4.1 step 4).
- **Receiver.** A new `.github/workflows/deps-cascade.yml` (wiring §5) calls the shared `cascade-receive.yml@main` on `repository_dispatch` (`upstream-released`), on a daily schedule at `17 5 * * *`, and on `workflow_dispatch` (`dry_run`, `gates_only`). Library values (wiring §5.1): `setup-go: false`, `labels-managed: false`; CUE `v0.17.1` by the reusable workflow's default.
  - It is a dry run until the repo variable `CASCADE_DRY_RUN` is exactly `false` (wiring §9.1). The supervisor sets it to `true` before this PR merges; Phase 4 sets `false`.
  - The receiver accepts payloads from core and catalog_opm only (wiring §3.2). That allowlist lives in `.github`, not here.
- **Gates caller.** A new `.github/workflows/cascade-gates.yml` (wiring §8.3) posts `cascade/freshness` and `cascade/settled` on every PR through the shared `cascade-gates.yml@main`: `n/a` on ordinary PRs, and a gates-only receiver run for each `release-please--*` PR. Both stay warnings (`CASCADE_G2_MODE`, `CASCADE_G3_MODE` unset means `warn`; owner selections 11 and 12). Neither becomes a required check here.
- **`need-human-review` on a core bump.** No new code in the library: `pins.sh` already labels the core row `need-human-review` (`.tasks/cascade/pins.sh:41`), the resolver turns that into the body's `cascade-labels` marker, and the shared receiver adds every marker label to the PR and, with `labels-managed: false`, creates the five bot labels in this repo with the `RELEASING.md` colours (wiring §6.4 step 4, §7.4). This change makes that path a stated requirement of the library and checks it in a dry-run summary before Phase 4.
- `AGENTS.md` "Release cascade task" gains two sentences: what the receiver and notify job do, and the three repo variables.

Not in this change:

- The shared workflows, their scripts and the sandbox cycle (`.github` `add-release-cascade-workflows`).
- Setting `CASCADE_DRY_RUN`, `CASCADE_NOTIFY`, `CASCADE_G2_MODE` or `CASCADE_G3_MODE`. Those are repo variables the supervisor or owner sets.
- Going live (Phase 4) and making G2 or G3 required (Phase 5, `require-pin-freshness-gate`).
- Any edit to `.tasks/cascade/`, the Taskfile or `opm/`.
- The docs example pins in `docs/getting-started.md` and `AGENTS.md` (a separate docs follow-up).

Depends on (merge gates, not tasks):

- `.github` `add-release-cascade-workflows` is merged first. All three callers name `@main` (owner selection 13), so before that merge they would call workflows that do not exist (wiring §1).
- The repo variable `CASCADE_DRY_RUN=true` is set in the library before this PR merges (wiring §1, §10).
- Already merged: the library's `prepare-release-cascade` (release job outputs) and `add-deps-cascade-task` (the task, `RELEASING.md:569`), and the Phase 0 settings: the `cascade` Environment with `CASCADE_APP_PRIVATE_KEY` and `CASCADE_APP_CLIENT_ID` (`gh api repos/open-platform-model/library/environments` lists `cascade`).

## Capabilities

### New Capabilities

- `cascade-wiring`: how the library takes part in the cascade at run time: the receiver's triggers and dry-run rule, the per-PR gate statuses, and how a core bump reaches the PR as `need-human-review`.

### Modified Capabilities

- `release-pipeline`: one added requirement, "A release notifies the library's downstream repos". No existing requirement or scenario changes.

## Impact

**SemVer: no release.** Every commit is `ci(...)` or `docs(...)`, which release-please hides. No file under `opm/` changes, the public surface is unchanged, and so are cli and opm-operator. The PR title is `ci: join the release cascade` (wiring §1).

**Files:** `.github/workflows/release.yml` (one job), `.github/workflows/deps-cascade.yml` (new), `.github/workflows/cascade-gates.yml` (new), `AGENTS.md` (two sentences).

**Required checks.** The library's `main` ruleset requires only `Go tests` (integration 15368) plus the org mention-guard. The new workflows add no required context, so they cannot block an unrelated PR. The cascade PR itself is gated by `Go tests` like any other PR.

**Complexity (Principle VII).** Three caller files of about 20 to 40 lines each with no logic of their own; every rule lives once in `.github`. The alternative, copying the receiver into each repo, is the fallback the wiring contract reserves for a failed E1 (wiring §13.1).
