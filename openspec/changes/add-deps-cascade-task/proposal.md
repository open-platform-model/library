## Why

The release cascade (workspace `RELEASING.md`, section "The cascade") moves each repo's upstream pins with one repo-owned task, `task deps:cascade`, that a shared receiver workflow runs on every upstream release and once a day. Phase 2 of the rollout (`RELEASING.md`, section "Rollout and changes", the `add-deps-cascade-task` row) gives each of the four consuming repos that task, its title and body tasks, and its tests, before any workflow calls them in Phase 3.

The library has no such task. Today a core bump is hand-made: the last one, `3c3b8c1` (`fix(deps): pin core v2.0.0-beta.2`, #173), edited `DefaultSchemaModule` in `opm/schema/loader.go:43`, five `cue.mod` files under `modules/` and `testdata/`, and all 30 `testdata/render/**/cue.mod/module.cue` trees (36 files). `task cue:deps:update` (`Taskfile.yml:303-451`) covers only the `CUE_MODULE_GLOBS` set (`Taskfile.yml:17-21`), resolves the catalog at its bare major, and ignores `.cascade-hold`. It never reaches `testdata/cue.mod` or the render trees, which `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` (`opm/internal/registrytest/registrytest_test.go:43-70`) requires to equal the default. A missed tree turns `task test` red.

The owner decided how the cascade treats the library's core pin (owner selection 9, quoted in the supervisor's `owner-selections-verbatim.md`): "Bot proposes, glue review". The bot bumps the constant and the fixtures in the cascade PR, and the PR carries the label `need-human-review`. A human reviews the glue before merging. `RELEASING.md`, section "Labels", makes that label mandatory on every library core bump.

The interface every repo implements is fixed by the Phase 2 cascade contract (supervisor scratchpad `p2-cascade-contract.md`, version 1; cited below as "contract §N"). Where it and `RELEASING.md` disagree, `RELEASING.md` wins.

## What Changes

- **`task deps:cascade`** runs `.tasks/cascade/cascade.sh`. It moves the library's two upstream pins in the working tree and nothing else: no commit, no branch, no push. It exits 0 when the tree changed, 3 when there was nothing to do, and any other code on error (`RELEASING.md`, section "The receiver"; contract §5.1, §5.2).
  - **Core (shipped).** It moves `DefaultSchemaModule` to the newest published `opmodel.dev/core@v2` in the shipped major. `DefaultSchemaVersion()` and `registrytest.DefaultCoreVersion` follow, with no second literal. The move always carries `need-human-review` (contract §6.2).
  - **Core in the test trees (test).** It re-pins core as text in `testdata/cue.mod/module.cue` and every `testdata/render/**/cue.mod/module.cue` found by a walk, never from a list. It skips `.cascade-frozen` entries and never runs `cue mod tidy` there.
  - **The opm catalog (test).** It moves `opmodel.dev/catalogs/opm@v4` to the newest published v4. The catalog stays when that build pins a core newer than `DefaultSchemaModule`, with the warning "advance core first".
  - **Module loop.** One explicit-version `cue mod get` plus `cue mod tidy` runs per `CUE_MODULE_GLOBS` module where a pin moved. Third-party pins are never named.
- **`task deps:cascade:title` and `task deps:cascade:body`** are thin wrappers over the shared resolver's `title` and `body`. They take the repo's `.tasks/cascade/classes` path-class map and `.tasks/cascade/pins.sh` pin report. The body's `cascade-labels` marker names `need-human-review` whenever core moved.
- **`task deps:cascade:test`** runs `.tasks/cascade/test.sh` against the canonical resolver stub (contract §7), copied byte for byte. It runs the offline set (S1 no-op, S3 resolver error, S6 dirty tree, S7 lagging render tree, S8 catalog that needs a newer core) and the network set (S2 older pins plus idempotence, S4 frozen, S5 title and body against the real resolver when one is given).
- **CI.**
  - The offline set becomes one step in the required `Go tests` job (`.github/workflows/test.yml`).
  - A new, non-required workflow, `.github/workflows/cascade-task.yml`, runs the full set on relevant PRs, on `workflow_dispatch` and weekly.
- `AGENTS.md` gains a short paragraph on the cascade task.

Not in this change:

- The receive workflow (`join-release-cascade`, Phase 3).
- The catch-up PR that brings the library's `main` current (contract §8).
- The rewire of the workspace `task deps:update` (Phase 5).
- Any edit to `task cue:deps:update`, whose behavior stays as `fixture-pin-maintenance` specifies.

Depends on:

- `.github` `add-cascade-resolver`, which must be merged before this change's PR merges (`RELEASING.md`, section "Rollout and changes"). The tasks are built and tested against the stub, so implementation does not wait. S5 and the merge do wait.
- The library's `prepare-release-cascade` and `derive-fixture-versions`. Both are merged and archived (`openspec/changes/archive/2026-10-02-*`).

## Capabilities

### New Capabilities

- `deps-cascade`: what `task deps:cascade` moves in the library, in which order, what it never touches, how it reports the result through exit codes, warnings, title and body, and how it is tested.

### Modified Capabilities

None. `fixture-pin-maintenance` keeps specifying `task cue:deps:update`, and its `.cascade-frozen` requirement already binds "release-cascade tooling". This change consumes that requirement and does not alter it.

## Impact

**SemVer: no release.** Every commit is `ci(cascade): ...`, which release-please hides. No file under `opm/` changes. `DefaultSchemaModule` moves only when the task runs in a later cascade PR, never in this change. The public surface is unchanged, and so are the downstream consumers (cli, opm-operator).

**Files added:**

- `.tasks/cascade/`: `cascade.sh`, `pins.sh`, `lib.sh`, `classes`, `test.sh`, and `testdata/stub-resolve.sh`, `testdata/older.tsv`, `testdata/s1-calls.txt`;
- four tasks in `Taskfile.yml`;
- one step in `.github/workflows/test.yml`;
- `.github/workflows/cascade-task.yml`.

**New local tool requirements.** The task needs these only when a pin moves: `yq` v4 (mikefarah; also needed by the stub's frozen reader), `cue`, `jq` and `curl` (through the resolver). The resolver is found beside the workspace checkout or through `CASCADE_RESOLVER` (contract §3).

**Complexity (Principle VII).** It replaces a 36-file hand edit that a reviewer must check line by line. The logic lives in one shell script beside the Taskfile, outside `opm/`, and adds no Go.
