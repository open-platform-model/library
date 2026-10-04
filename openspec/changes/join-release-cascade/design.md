## Context

The cascade design is the workspace `RELEASING.md` ("The cascade", with "Notify after publish", "Pinning the cascade code", "The receiver" and "Two-job split"; "Gates"; "Stop switches"; "Moving the cascade pin"; "Rollout and changes"). The binding interface for Phase 3 is the wiring contract, version 3.1 with changelogs 3.1.1 and 3.1.2, archived with the `.github` change at `openspec/changes/archive/2026-10-04-add-release-cascade-workflows/contract.md` ("wiring §N"), plus the supervisor's addendum for the five join changes (the `release.yml` env allow-list, `runs-on: ubuntu-latest` on the key-holding jobs, the canary rule for later bumps, the dropped sandbox). The resolver and task interface is the Phase 2 contract (`.github` `openspec/changes/archive/2026-10-04-add-cascade-resolver/contract.md`; "P2 §N"). The `.github` README "Cascade workflows" and "Pinning and bumps" say the same as wiring §2.4, §4.6, §5 and §8.3.

The shared code is on `.github` `main` at the squash commit of PR 9, `2376ffae4bfc665f327d51581350dea694c01504` (`compare …main` prints `identical`, read 2026-10-04). It holds the composite actions `cascade-notify` and `cascade-publish`, the reusable workflows `cascade-receive.yml` (jobs `compute` and `gates`, no secret) and `cascade-gates.yml`, and the resolver. The reusable `cascade-notify.yml` that version 2 of the contract called no longer exists: the sandbox showed (E1) that a reusable-workflow job does not see the caller's Environment secrets without `secrets: inherit`, so every job that mints the App token is the caller's own.

What the library has on `main` (`3d7ce15`):

- `release.yml` job `release-please` with outputs `releases_created` and `tag_name` from step `release` (`.github/workflows/release.yml:18-20`). Workflow-level permissions are `contents: write` and `pull-requests: write` (`:8-10`); there is no workflow-level `env`. `publish-docs` needs `release-please` and calls docs-kit's `publish.yml`.
- `task -x deps:cascade`, `:title`, `:body`. The resolver path comes from `CASCADE_RESOLVER` or the sibling `.github` checkout.
- `pins.sh` labels the core row `need-human-review` (`.tasks/cascade/pins.sh:41`). The S5 test asserts the body's `<!-- cascade-labels: need-human-review -->` marker and its absence on a catalog-only move (`.tasks/cascade/test.sh`).
- The task reads `CASCADE_EXPECT` as whitespace-split `<key>=<value>` pairs with the keys `opmodel.dev/core@v2` and `opmodel.dev/catalogs/opm@v4` (`.tasks/cascade/lib.sh`), the same keys as wiring §3.4. It keeps its state under `$(git rev-parse --absolute-git-dir)/cascade`, outside the working tree.
- `cascade-task.yml` uses the receiver's layout (the library at `repo/`, `.github` at `org-github/` with `persist-credentials: false`), checks `.github` out at `ref: main`, and skips S5 when the resolver is missing there.
- The required `Go tests` job (`test.yml` job `test`) runs on every PR with no path filter and installs Task; `.github/dependabot.yml` configures `github-actions` and already ignores `open-platform-model/docs-kit*`.

Live settings (read 2026-10-04): Environment `cascade` (branch policy `main` only) with `CASCADE_APP_PRIVATE_KEY` and `CASCADE_APP_CLIENT_ID`; repo variable `CASCADE_DRY_RUN=true`; the `main` ruleset requires `Go tests` (integration 15368) plus the org mention-guard; `sha_pinning_required` is false; none of the cascade labels exist.

## Goals / Non-Goals

**Goals:**

- The library notifies opm-operator and cli after a release, through the pinned `cascade-notify` action in its own `cascade` Environment job.
- The library receives core and catalog_opm releases, daily and on dispatch, as a dry run until Phase 4, with the publish step in its own `cascade` Environment job.
- Every library PR carries `cascade/freshness` and `cascade/settled`, as warnings.
- A core bump reaches the cascade PR as `need-human-review` (owner selection 9).
- Every cascade reference carries one `.github` `main` SHA, and CI on every PR keeps the key-holding jobs in their contract shape.

**Non-Goals:**

- Any logic in the library's workflows beyond triggers, permissions, concurrency, inputs and the two one-step key-holding jobs. Every rule is in `.github` (wiring §2).
- Required checks, `enforce` modes, or going live.
- Changes to the cascade task or `opm/`.

## Decisions

### D1. Notify is a caller-owned job that depends on `release-please` only

The library's release is the git tag. Notify is the library block of wiring §4.6, appended as the last job:

```yaml
  notify-downstream:
    name: Notify downstream
    needs: release-please
    if: needs.release-please.outputs.releases_created == 'true' && vars.CASCADE_NOTIFY != 'off'
    runs-on: ubuntu-latest
    environment: cascade
    timeout-minutes: 20
    permissions:
      contents: read
    steps:
      - name: Notify downstream
        uses: open-platform-model/.github/.github/actions/cascade-notify@<SHA> # .github main
        with:
          tag: ${{ needs.release-please.outputs.tag_name }}
          client-id: ${{ vars.CASCADE_APP_CLIENT_ID }}
          private-key: ${{ secrets.CASCADE_APP_PRIVATE_KEY }}
```

- `publish-docs` is a sibling, never a `need` (wiring §4.5 "library"). A failed docs bundle must not stop the cascade, and `docs.yml` already recovers a missed bundle.
- "Published" for a Go module means `proxy.golang.org` serves it. The proxy lags the tag, so the action polls it for up to 10 minutes when the repo is `library`, best effort, and the receivers' `--expect` wait and the daily sweep cover the rest (wiring §4.1 step 4). The library adds nothing for this.
- The key is read only here: the job declares `environment: cascade` and passes `secrets.CASCADE_APP_PRIVATE_KEY` only as the `private-key` input of the SHA-pinned action, which mints a token scoped to opm-operator and cli. The job has no checkout or `run:` of its own and no `env:`, `container:` or `services:` (wiring §2.2, §10.1 item 9). The job-level `permissions` narrow the workflow-level `contents: write` to `contents: read`.
- The `cascade` Environment allows `main` only, and `release.yml` runs only on pushes to `main`, so no other ref can start this job (E1b).

### D2. Receiver caller values

`deps-cascade.yml` is the wiring §5 file with the library `jobs:` map of §5.2: cron `17 5 * * *`, `setup-go: false` on the `cascade` job, `labels-managed: false` on the `publish` step, no `cue-version`.

- **Two jobs.** `cascade` calls `cascade-receive.yml` (compute runs the task with read-only permissions and no secret; gates posts statuses with `GITHUB_TOKEN`). `publish` is the library's own job: `environment: cascade`, one step running the pinned `cascade-publish` action, which checks the plan, mints the token right before use, and pushes `deps/cascade` and edits the PR. It checks the repo out but never runs it.
- **No Go.** `cascade.sh` runs no `go` command, and the task's preconditions are `git`, mikefarah `yq` and `jq`. G2 runs the same task at the release head (wiring §8.2), so it needs no Go either. The spike confirms it (below).
- **CUE version.** The reusable default `v0.17.1` equals the library's `cue.yml`, `cascade-task.yml` and `go.mod` CUE version today. The library §5.2 block passes no `cue-version`, and every line but comments is byte for byte, so the branch keeps it so. The consequence (implementation review finding 2): after a library CUE bump, the receiver still runs `cue mod get` and `tidy` with `v0.17.1` until the reusable default moves, and `cascade.sh` checks `language.version` against `cue.yml`, not against the CUE that runs. A comment above `setup-go` says so, so a CUE bump touches this file. Whether the library should pass `cue-version` like catalog_opm (wiring §5.1 allows it; §5.2 does not show it) is raised to the supervisor.
- **Labels.** The library has no `labels.yml` and no label sync, so `cascade-publish` creates the five bot labels itself (`labels-managed: false`, wiring §6.4).
- **Cron.** `17 5 * * *` is the same minute as `cascade-task.yml`'s weekly `17 5 * * 1`. They are separate workflows in separate concurrency groups, and both only read GHCR, so the Monday overlap is harmless.
- **Dry run.** Both `dry-run` values are `${{ inputs.dry_run == true || vars.CASCADE_DRY_RUN != 'false' }}` (wiring §5), and the `publish` `if:` repeats `inputs.dry_run != true && vars.CASCADE_DRY_RUN == 'false'` so a dry run starts no Environment job. That fails closed: an unset, deleted or other value is a dry run. GitHub compares expression strings without regard to case, so `False` and `FALSE` are live too; the action receives the expression's result (`true` or `false`), so its "exactly `false`" check cannot tell them apart. The branch says "is `false` (compared case-insensitively)" wherever it describes the switch; the contract, the `.github` README and `RELEASING.md` "Stop switches" say "exactly `false`", which is raised to the supervisor.

### D3. `need-human-review` needs no library code

The chain, end to end:

1. `cascade.sh` moves `DefaultSchemaModule` (`opm/schema/loader.go`) and warns.
2. `pins.sh` reports the core row with label `need-human-review` (`pins.sh:41`). The resolver's `body` puts the labels of every moved pin into `<!-- cascade-labels: ... -->` (P2 §4.4). S5 asserts this both ways.
3. `compute` takes `plan.labels` as `deps-cascade` plus that marker (wiring §7.4), and `cascade-publish` verifies the set, creates the labels (`labels-managed: false`) and adds them (wiring §6.4).
4. The bot never removes `need-human-review`.

So the library's job is to pass `labels-managed: false` and to state the requirement (spec `cascade-wiring`).

What the Phase 3 dry run shows depends on the upstreams at that time; it matches a local `task -x deps:cascade` on `main`. On 2026-10-04 catalog_opm had published `opm-v4.5.2` while the library pinned `v4.5.1`, so the dry run was a real `test(fixtures)` push with an empty `cascade-labels` marker, not a `noop`. Such a non-noop summary can serve as the library's Phase 4 evidence (wiring §1, §15 item 5). The core path, and with it the `need-human-review` marker, was shown with the real resolver only by the spike (task 1.2), which committed an older core in a scratch clone. The first live core-move PR carrying `need-human-review` is checked by the supervisor after Phase 4 (task 8.6); it is a check, not a Phase 4 gate.

### D4. Gates caller

`cascade-gates.yml` is the wiring §8.3 file, every line but the header comment byte for byte. It runs on `pull_request_target` and checks out nothing, so no PR code runs with the write token. The library already runs `pr-title.yml` on `pull_request_target` the same way. `actions: write` is needed only for the gates-only `workflow_dispatch` of `deps-cascade.yml`; `statuses: write` for the `n/a` and `pending` posts. The sandbox (E7, on a private repo, so a proxy for the public library) showed a Dependabot `pull_request_target` run's token had `statuses: write` and posted `n/a`.

### D5. The wiring check

`.tasks/cascade/wiring-check.sh` is the wiring §10.1 item 6 script with `RECEIVER=true` and `PIN_COMMENT='.github main'`, plus the addendum:

- **`release.yml` env allow-list.** The workflow-level `env` reaches every step of `notify-downstream`, the action's bash steps included. The contract's deny-list (`BASH_ENV`, `ENV`, `NODE_OPTIONS`) misses other variables that run code in bash, such as `SHELLOPTS` or `PS4`. So the script reads `release.yml`'s `env` and refuses any key not in `ENV_ALLOW`, which is empty for the library (it has no workflow-level `env`). It also requires the `env` to be absent or a plain map, so an expression cannot hide its keys from the check.
- **`runs-on: ubuntu-latest`** on `notify-downstream` and `publish`, so a key-holding job cannot move to a self-hosted or other runner by mistake.

It runs as `task cascade:wiring:check`, in the aggregate `task check`, and as the step "Verify the cascade wiring" of the required `Go tests` job, right after "Install Task", using the runner's mikefarah `yq`. It is not a new job or context. The script lives in the PR's own tree, so a PR can edit it together with the workflows: it guards against mistakes, and review plus the `main` ruleset guard against a deliberate edit. actionlint does not check a composite action's inputs, so this check is the one for `cascade-notify`'s and `cascade-publish`'s `with:` keys.

### D6. One pin, the bump procedure, and merge order

- **The pin.** Five references carry `2376ffae4bfc665f327d51581350dea694c01504` and ` # .github main`: the notify and publish `uses:` steps, the `cascade-receive.yml` and `cascade-gates.yml` calls, and the `ref:` of the resolver checkout in `cascade-task.yml` (wiring §2.4). Owner decision 24 pins the two actions; the supervisor extended the pin to the two workflows (compute runs its own commit's scripts, so a mixed pin would split compute and publish across `plan.json`) and to the resolver checkout (so CI tests the resolver the receiver runs). This branch was written while A was open but pins the `main` squash SHA directly, because A had merged when it was brought to version 3.1.
- **No S5 fallback.** With the resolver checkout pinned to a commit that has the resolver, the old "skip S5 when the resolver is missing" branch would only turn a bad pin into a silent skip, so `cascade-task.yml` now fails with `::error::no cascade resolver at the pinned .github commit` (wiring §10.1 item 3).
- **Bumps.** A later `.github` cascade change reaches the library only through a `ci(deps): pin the cascade to .github <sha7>` PR that replaces the SHA in all five references (found with `grep -rn -A1 'open-platform-model/.github' .github/workflows`), passes `compare` and `task cascade:wiring:check`, and merges after its CI printed `cascade wiring: ok`. One receiver goes first as the canary in dry run; when the `.github` diff touches `cascade-publish` or `cascade-notify`, the others wait for the canary's first live run of that action (wiring §2.4, addendum item 4). There is no sandbox step (owner decision 26).
- **Dependabot.** The `github-actions` ecosystem ignores `open-platform-model/.github*`, so Dependabot never moves one reference alone and breaks "one SHA per repo" (wiring §10.1 item 7).
- **Merge order.** A merged first (done). Before this PR merges, the supervisor checks `compare <SHA>...main` prints `identical` or `ahead` and that `CASCADE_DRY_RUN` reads back `true` (done 2026-10-04; an unset variable would also be a dry run, the explicit `true` makes the state visible). A dispatch that reaches the library before this change merges returns 204 and starts nothing (wiring §1).

## Research & Decisions

### Where notify hooks in

**Context**: notify must run only when the artifact is public (`RELEASING.md`, "Notify after publish").
**Explored**: the library's `release.yml` on `main`, the wiring research report section 1 (supervisor scratchpad, research `r-wiring`), and wiring §4.5.
**Decision**: `needs: release-please`, gated on `releases_created` and `CASCADE_NOTIFY` (D1).
**Rationale**: the tag is the release; the Go proxy lag is handled once in the shared notify action rather than per repo.

### Whether the receiver needs Go

**Context**: wiring §5.1 sets `setup-go: false` for the library; a missing toolchain would fail `compute` only at the first real move.
**Explored**: `cascade.sh` and `lib.sh` for `go` commands (none), and the task's preconditions in `Taskfile.yml`.
**Decision**: no Go.
**Rationale**: the task edits a Go string literal with `sed` and never builds. The spike (below, run b) confirms it: with no `go` on `PATH`, a run that moves core and the catalog reaches `cue mod get` and `tidy` in all three `CUE_MODULE_GLOBS` modules and exits 0, and leaves no `.task/` directory.

### Receiver spike

**Context**: the receiver's `compute` runs the task in a fresh checkout at `repo/`, with `.github` at `org-github/` and `CASCADE_RESOLVER` pointing into it (wiring §6.2 steps 2, 3, 10), and G2 runs it again in a detached worktree of a release head (wiring §8.2). Run 2026-10-04 against library `origin/main` `8241250` and `.github` `main` `6a18e7d` (the real resolver), with `PATH` holding only `git`, `task`, `cue` v0.17.1, mikefarah `yq` v4.53.3, `jq`, `curl` and `/usr/bin`, no `go`, and an empty `CUE_CACHE_DIR` per run. Scripts: supervisor scratchpad `p3-library-join-run.sh`, `p3-library-join-older.sh`, `p3-library-join-g2.sh`.
**Explored**:

- **(a) `main` as it is** (`CASCADE_BASE=origin/main`): `task -x deps:cascade` exit 0. It rewrote four `cue.mod/module.cue` files (`modules/opm_platform`, `testdata/modules/web_app`, `testdata/parity`, `testdata/parity/opm_platform`) to `opmodel.dev/catalogs/opm@v4.5.2`, because catalog_opm published `opm-v4.5.2` on 2026-10-04 and the library pins `v4.5.1`. `task -x deps:cascade:title` gave `test(fixtures): bump opm catalog to v4.5.2`; the body carried `<!-- cascade-labels:  -->` (empty). The two doc-pin warnings (`docs/getting-started.md`, `AGENTS.md`) are the separate docs follow-up.
- **(b) The forced core path**: a second clone with core set to `v2.0.0-beta.1` in `opm/schema/loader.go` and the catalog to `v4.5.0` in every `cue.mod/module.cue` (the pins of `.tasks/cascade/testdata/older.tsv`, applied as `test.sh` `set_older` does, 36 files), committed, and `CASCADE_BASE` set to that commit. Exit 0; `DefaultSchemaModule` moved to `v2.0.0-beta.2`; title `fix(deps): bump core to v2.0.0-beta.2 and opm catalog to v4.5.2`; body marker `<!-- cascade-labels: need-human-review -->`.
- **(c) A realistic dispatch** on a third clone of `main`: `CASCADE_SOURCE=catalog_opm`, `CASCADE_TAGS=opm-v4.5.2`, `CASCADE_EXPECT='opmodel.dev/catalogs/opm@v4=v4.5.2'`. Exit 0, the same diff and title as (a); the body differs from (a) only in the triggering-releases line (`catalog_opm` `opm-v4.5.2` instead of "None recorded").
- **G2** (task 1.3) on clone (a), reset to `main`: a detached worktree of `origin/main` outside `repo/`, the task under `env -u CASCADE_EXPECT -u CASCADE_SOURCE -u CASCADE_TAGS -u CASCADE_NOTES_FILE` with `CASCADE_BASE=<sha>`: exit 0, and the resolver's `classify` put all four changed paths in class `test`, so G2 reports `ok: only test/release-tool pins behind`. The warnings file landed in `repo/.git/worktrees/<name>/cascade/warnings`, none in the worktree; after `worktree remove` the clone's `HEAD` and status were unchanged.

**Decision**: the receiver's call sequence works on the library as wiring §5.1 configures it (`setup-go: false`, CUE v0.17.1), and the `need-human-review` marker comes out of the real resolver on a core move.
**Rationale**: every run matched what `main` gives at the time, which is the expected result (tasks 1.2, 1.3). Run (a) is also the result the Phase 3 dry run should show (Migration Plan step 4).

### Dependabot and the cascade pin

**Context**: `.github/dependabot.yml` updates `github-actions` weekly and already ignores `open-platform-model/docs-kit*` because that pin moves with `.opm-docs-version`. Version 2 of this change called `@main` and added no ignore, on the untested view that Dependabot proposes no bump for a branch ref.
**Explored**: wiring §2.4 and §10.1 item 7; the five references are now SHAs, which Dependabot's `github-actions` updater does propose bumps for.
**Decision**: ignore `open-platform-model/.github*`, with the wiring §10.1 item 7 comment, after the docs-kit entry.
**Rationale**: a Dependabot bump would move one reference alone and break "one SHA per repo"; the wiring check would then fail the PR, but ignoring it avoids the noise.

### Wiring check tests

Filled in by task 5.4.

### Re-grep

Filled in by task 7.2.

## Risks / Trade-offs

- **[A gates-only run per release-PR update]** → each push release-please makes to its PR (about one per releasable merge to `main`) starts one gates-only `Deps cascade` run, which runs the task at the release head with network and a cold CUE cache. Mitigation: it is not a required check, it runs in its own concurrency group (`deps-cascade-gates`), and a failure posts `WARN:` in `warn` mode.
- **[The wiring check is in the PR's own tree]** → a PR can weaken the check together with the workflows. Mitigation: it is meant to catch mistakes; review and the `main` ruleset (PR required, owner bypass pull-request-only) catch a deliberate edit (D5).
- **[Receiver CUE version is not tied to the library's]** → after a library CUE bump, the receiver still runs the reusable default. Mitigation: the comment in `deps-cascade.yml` (D2); the question is with the supervisor.
- **[`False` counts as `false`]** → a mistyped `False` makes the receiver live. Mitigation: Phase 4 sets the value deliberately; the question of saying so in the contract is with the supervisor (D2).
- **[A `.github` change to `cascade-publish` first runs live in one repo]** → the canary rule (D6) confines it; the offline suites in `.github` are the only evidence before that.
- **[Schedule disabled after 60 days without repo activity]** → the library is active; `gh workflow enable` restores it.
- **[Workflow edits on `main` under `WF_GUARD_RULE=tree`]** → a workflow change on `main` while a bot-only cascade PR is open is handled by `cascade-publish` (wiring §7.6). In dry run there is no cascade PR.

## Migration Plan

1. `.github` `add-release-cascade-workflows` merged (squash `2376ffae4bfc665f327d51581350dea694c01504`). Done.
2. The supervisor set `CASCADE_DRY_RUN=true` in the library. Done.
3. Pre-merge check (wiring §10.1): `compare` prints `identical` or `ahead`; `PIN_COMMENT='.github main'` and `RECEIVER=true`; the `Go tests` job printed `cascade wiring: ok, .github <SHA> (.github main)`; `grep -rn -A1 'open-platform-model/.github' .github/workflows` shows only `<SHA>`; the re-grep leaves only allowed hits; the Dependabot ignore is present; `CASCADE_DRY_RUN` reads `true`. Then merge this change (`ci: join the release cascade`).
4. Phase 3 gate (the supervisor): `gh workflow run deps-cascade.yml -R open-platform-model/library -f dry_run=true`; the run succeeds, `Publish` is skipped, the `Compute` log shows `scripts from open-platform-model/.github <SHA>`, and the summary matches a local `task -x deps:cascade` on `main` at that time (exit 3 and action `noop` when every pin is current). The next PR shows both cascade contexts, and the next `cascade-task.yml` run checks the resolver out at `<SHA>`.
5. Rollback: set `CASCADE_NOTIFY=off` and disable `deps-cascade.yml` and `cascade-gates.yml` (stop switches, wiring §9.2), or revert the PR.

## Open Questions

For the supervisor (none blocks this branch; each keeps the contract text as it is):

- **CUE version of the receiver** (D2; implementation review finding 2): pass `cue-version: v0.17.1` in the library's `deps-cascade.yml` so it moves with `cue.yml`, which wiring §5.1 allows but the §5.2 block does not show, or keep the reusable default and add the receiver to the CUE-bump checklist.
- **"Exactly `false`"** (D2; implementation review finding 1): wiring §5, §9.1 and §14, the `.github` README and `RELEASING.md` "Stop switches" say the receiver is live only when `CASCADE_DRY_RUN` is exactly `false`, but GitHub compares the string case-insensitively, so `False` goes live as well. Either amend that text to "is `false` (compared case-insensitively)", or check the variable's case in the shared action.

## Plan review

The plan review of commit `22ea3b5` (1 blocker, 2 majors, 2 minors, 3 nits) was applied in full in `b9ec77d`.

## Version 3.1 rebuild

This branch was first built to wiring version 2 (reusable notify workflow, `@main`, no wiring check; heads `8d1e718`, `f9b045b`, `4ca8064`). It was rebuilt to version 3.1 after `.github` PR 9 merged, applying wiring §10.1 items 1 to 11 for the library and the supervisor's addendum. The implementation review of the version 2 branch (`p3-review-library-join.md`) is applied where it still holds: finding 1 ("exactly `false`" wording) in the branch's own text, with the contract-level part raised above; finding 2 (CUE version) as a comment and a question; finding 3 (the proposal over-claimed the dry run as proof of the core marker) in the proposal; finding 4 (the gates-only run cost) in Risks; finding 5 (a second cascade capability) by stating in the proposal why `cascade-wiring` is separate from `deps-cascade`. Its merge blockers M1 (`CASCADE_DRY_RUN` unset) and M2 (A unmerged) are resolved. Its agreed open question about the sandbox-down label row is dropped with the sandbox (owner decision 26).
