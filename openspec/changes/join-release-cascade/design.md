## Context

The cascade design is the workspace `RELEASING.md` ("The cascade", `RELEASING.md:162-325`; "Gates", `:327-358`; "Stop switches", `:454-462`; "The opm-cascade App", `:516-527`; "Phases" and "Changes", `:529-571`). The binding interface for Phase 3 is the wiring contract (supervisor scratchpad `p3-wiring-contract.md`, version 2; "wiring §N"). The resolver and task interface is the Phase 2 contract (`.github` `openspec/changes/archive/2026-10-04-add-cascade-resolver/contract.md`; "P2 §N").

What the library already has on `main` (`8241250`):

- `release.yml` job `release-please` with outputs `releases_created` and `tag_name` from step `release` (`.github/workflows/release.yml:18-20`, `:36-42`). Workflow-level permissions are `contents: write` and `pull-requests: write` (`:8-10`). `publish-docs` (`:57-69`) needs `release-please` and calls docs-kit's `publish.yml@v0.5.0`.
- `task -x deps:cascade`, `:title`, `:body` (`Taskfile.yml:862-928`). The resolver path comes from `CASCADE_RESOLVER` or the sibling `.github` checkout (`Taskfile.yml:887-892`).
- `pins.sh` labels the core row `need-human-review` (`.tasks/cascade/pins.sh:41`); `cascade.sh` warns on every core move (`.tasks/cascade/cascade.sh:232`). The S5 test asserts the body's `<!-- cascade-labels: need-human-review -->` marker (`.tasks/cascade/test.sh:403`) and its absence on a catalog-only move (`:419`).
- The task reads `CASCADE_EXPECT` as whitespace-split `<key>=<value>` pairs (`.tasks/cascade/cascade.sh:73-76`) with the keys `opmodel.dev/core@v2` and `opmodel.dev/catalogs/opm@v4` (`.tasks/cascade/lib.sh:12-13`), the same keys as wiring §3.4.
- The task keeps its state under `$(git rev-parse --absolute-git-dir)/cascade` (`.tasks/cascade/cascade.sh:43-46`), outside the working tree, and sets its own `CUE_REGISTRY` (`:58`).
- `cascade-task.yml` already uses the receiver's layout: the library at `repo/`, `.github` at `org-github/` with `persist-credentials: false` (`.github/workflows/cascade-task.yml:34-46`; P2 §11 C4).

Live settings (read 2026-10-04): Environment `cascade` exists; no repo variables are set; the `main` ruleset requires only `Go tests` (integration 15368) plus the org mention-guard workflow; none of the cascade labels exist; `sha_pinning_required` is false (wiring research finding 3, opm-operator only).

## Goals / Non-Goals

**Goals:**

- The library notifies opm-operator and cli after a release, through the shared notify workflow.
- The library receives core and catalog_opm releases, daily and on dispatch, as a dry run until Phase 4.
- Every library PR carries `cascade/freshness` and `cascade/settled`, as warnings.
- A core bump reaches the cascade PR as `need-human-review` (owner selection 9).

**Non-Goals:**

- Any logic in the library's workflows beyond triggers, permissions, concurrency and inputs. Every rule is in `.github` (wiring §2).
- Required checks, `enforce` modes, or going live.
- Changes to the task, `opm/` or the Taskfile.

## Decisions

### D1. Notify depends on `release-please` only

The library's release is the git tag (`release.yml:32-35`). Notify uses the `library` row of wiring §4.5:

```yaml
  notify-downstream:
    name: Notify downstream
    needs: release-please
    if: needs.release-please.outputs.releases_created == 'true' && vars.CASCADE_NOTIFY != 'off'
    permissions:
      contents: read
    uses: open-platform-model/.github/.github/workflows/cascade-notify.yml@main
    with:
      tag: ${{ needs.release-please.outputs.tag_name }}
```

- `publish-docs` is a sibling, never a `need` (wiring §4.5 "library"). A failed docs bundle must not stop the cascade, and docs.yml already recovers a missed bundle (`release.yml:50-51`).
- "Published" for a Go module means `proxy.golang.org` serves it (`RELEASING.md:212-213`). The proxy lags the tag, so the shared notify waits for it, best effort, for up to 10 minutes, and the receivers' `--expect` wait and the daily sweep cover the rest (wiring §4.1 step 4). The library adds nothing for this.
- The job-level `permissions` override the workflow-level `contents: write` (`release.yml:8-10`), so the called workflow gets a read-only `GITHUB_TOKEN`. The App token minted inside the reusable workflow does the dispatch (wiring §2.3).
- No `secrets:` and no `secrets: inherit` (wiring §2.2). The key is read only by the reusable job that declares `environment: cascade`.
- The job sits after `publish-docs` in the file, so the existing job comments stay where they are.

### D2. Receiver caller values

`deps-cascade.yml` is the wiring §5 file with the §5.1 library row: cron `17 5 * * *`, `setup-go: false`, `labels-managed: false`. No `setup-cue` or `cue-version` input: the default `v0.17.1` equals the library's `cue.yml` and `cascade-task.yml` setup-cue version.

- **No Go.** `cascade.sh` runs no `go` command, and the task's preconditions are `git`, mikefarah `yq` and `jq` (`Taskfile.yml:896-906`). G2 runs the same task at the release head (wiring §8.2), so it needs no Go either.
- **Labels.** The library has no `labels.yml` and no label sync, so the receiver creates the five bot labels itself (wiring §6.4 step 4).
- **Cron.** `17 5 * * *` is the same minute as `cascade-task.yml`'s weekly `17 5 * * 1` (`cascade-task.yml:19-20`). They are separate workflows in separate concurrency groups, and both only read GHCR, so the Monday overlap is harmless. Kept as wiring §5.1 states it rather than diverging per repo.
- `dry-run` is `inputs.dry_run == true || vars.CASCADE_DRY_RUN != 'false'` (wiring §5). That fails closed: only the string `false` is live. GitHub compares expression strings without regard to case, so `False` and `FALSE` are live too; "exactly `false`" in wiring §9.1 is true up to case. The expression is kept as the contract gives it, so the four receivers stay identical, and the point is raised to the supervisor.

### D3. `need-human-review` needs no library code

The chain, end to end:

1. `cascade.sh` moves `DefaultSchemaModule` (`opm/schema/loader.go:43`) and warns (`cascade.sh:232`).
2. `pins.sh` reports the core row with label `need-human-review` (`pins.sh:41`). The resolver's `body` puts the labels of every moved pin into `<!-- cascade-labels: ... -->` (P2 §4.4). S5 asserts this both ways (`test.sh:403`, `:419`).
3. The shared receiver's `compute` takes `plan.labels` as `deps-cascade` plus that marker (wiring §7.4), and `publish` verifies the set, creates the labels (`labels-managed: false`) and adds them (wiring §6.4 steps 2, 4, 5).
4. The bot never removes `need-human-review` (wiring §6.4, last paragraph).

So the library's job is to pass `labels-managed: false` and to state the requirement (spec `cascade-wiring`).

What the Phase 3 dry run shows depends on the upstreams at that time; it matches a local `task -x deps:cascade` on `main`. On 2026-10-04 catalog_opm had published `opm-v4.5.2` while the library pinned `v4.5.1`, so the dry run is a real `test(fixtures)` push with an empty `cascade-labels` marker, not a `noop`. That non-noop summary can serve as the library's Phase 4 evidence (wiring §1, §15 item 5). The core path, and with it the `need-human-review` marker, is shown with the real resolver by the spike (task 1.2), which commits an older core in a scratch clone. The first live core-move PR carrying `need-human-review` is checked by the supervisor after Phase 4; it is a check, not a Phase 4 gate.

### D4. Gates caller

`cascade-gates.yml` is the wiring §8.3 file verbatim. It runs on `pull_request_target` and checks out nothing, so no PR code runs with the write token. The library already runs `pr-title.yml` on `pull_request_target` the same way (`.github/workflows/pr-title.yml:11-12`). `actions: write` is needed only for the gates-only `workflow_dispatch` of `deps-cascade.yml`; `statuses: write` for the `n/a` and `pending` posts.

### D5. Merge order and the `@main` references

- The three callers name `@main` (owner selection 13). Merged before `.github` `add-release-cascade-workflows`, the next release run would fail its `notify-downstream` job on a missing workflow, and every PR would show a failed `Cascade gates` run. So that change merges first (wiring §1). This is a merge gate, not a task.
- The supervisor sets `CASCADE_DRY_RUN=true` before the merge (wiring §1). An unset variable is already a dry run (D2), so the explicit `true` only makes the state visible.
- A dispatch that reaches the library before this change merges returns 204 and starts nothing (wiring §1).

## Research & Decisions

### Where notify hooks in

**Context**: notify must run only when the artifact is public (`RELEASING.md:166-167`).
**Explored**: the library's `release.yml` on `main` (`:12-69`), the wiring research report section 1 (supervisor scratchpad, research `r-wiring`), and wiring §4.5.
**Decision**: `needs: release-please`, gated on `releases_created` and `CASCADE_NOTIFY` (D1).
**Rationale**: the tag is the release; the Go proxy lag is handled once in the shared notify workflow rather than per repo.

### Whether the receiver needs Go

**Context**: wiring §5.1 sets `setup-go: false` for the library; a missing toolchain would fail `compute` only at the first real move.
**Explored**: `cascade.sh` and `lib.sh` for `go` commands (none), and the task's preconditions (`Taskfile.yml:896-906`).
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

### Dependabot and the `@main` references

**Context**: `.github/dependabot.yml` updates `github-actions` weekly and already ignores `open-platform-model/docs-kit*` because that pin moves with `.opm-docs-version`.
**Explored**: the file's ignore list. Not verified: whether Dependabot proposes a SHA for a reusable-workflow reference pinned to a branch name. The plan review answered (moderately confident, untested) that Dependabot's github-actions updater proposes bumps only for version-tag or SHA refs, not branch refs such as `@main`.
**Decision**: no ignore entry in this change or in any of the five joins. If a Dependabot PR for an `open-platform-model/.github` reference does appear, close it and add `open-platform-model/.github*` to all five in one follow-up.
**Rationale**: a per-repo answer would make the five joins differ; a Dependabot PR is visible and closable, so the cost of waiting is small.

## Risks / Trade-offs

- **[E1 fails: the Environment secret is not visible to a reusable-workflow job]** → the shared notify and publish become composite actions and each caller grows a local `cascade` job (wiring §13.1). The library's three files would change shape. Mitigation: A's sandbox cycle proves E1 before A merges, and this change merges after A.
- **[A gates-only run per release-PR push]** → each one runs the task at the release head, with network and a cold CUE cache. Mitigation: release PRs move rarely; the run is not a required check.
- **[Schedule disabled after 60 days without repo activity]** → the library is active; `gh workflow enable` restores it (wiring research section 2).
- **[Workflow edits on `main` under `WF_GUARD_RULE=strict`]** → this PR itself changes workflows, so an open bot-only cascade PR would be recreated after it merges (wiring §7.6). Mitigation: in dry run there is no cascade PR.

## Migration Plan

1. Merge `.github` `add-release-cascade-workflows`.
2. The supervisor sets `CASCADE_DRY_RUN=true` in the library.
3. Merge this change (`ci: join the release cascade`).
4. Phase 3 gate (the supervisor): `gh workflow run deps-cascade.yml -R open-platform-model/library -f dry_run=true`; the summary matches a local `task -x deps:cascade` on `main` at that time. On 2026-10-04 that is exit 0, mode `fresh`, action `push` shown as a dry run, title `test(fixtures): bump opm catalog to v4.5.2` and an empty `cascade-labels` marker; with every pin current it is exit 3 and action `noop`. The next PR shows both cascade contexts.
5. Rollback: set `CASCADE_NOTIFY=off` and disable `deps-cascade.yml` and `cascade-gates.yml` (stop switches, wiring §9.2), or revert the PR.

## Open Questions

- None blocking. The E-tests (E1, E1b, E2 to E7) belong to `.github` `add-release-cascade-workflows`. E7 decides whether a Dependabot PR gets the gate statuses (spec `cascade-wiring`, "Every PR carries the cascade gate statuses").
- For the supervisor: the sandbox-down `pins.sh` row (wiring §11.3) carries no label, so E2 never shows a marker label being created and added. Giving that row a label (for example `need-human-review`) and adding "label created with `e99695` and the `RELEASING.md` description, and added to the PR" to E2 would cover the marker-to-label leg of owner selection 9 before Phase 4.

## Plan review

The plan review of commit `22ea3b5` (1 blocker, 2 majors, 2 minors, 3 nits) is applied in full: the expected spike and dry-run results now match `main` at the time rather than a fixed `noop` (1); the spike forces the core path with no Go and runs a realistic catalog dispatch (2); the core-move check after merge is no longer a Phase 4 gate, and the sandbox label point is raised above (3); the gate-status requirement is scoped by E7 (4); the gates-only scenario is corrected (5); section 1 commits as `docs(openspec)` (6); the Taskfile citation is `896-906` (7); and the section gate names the known cache-race flake (8). No finding was rejected.
