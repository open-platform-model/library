## Context

- **Design sources.**
  - Workspace `RELEASING.md`, sections "The cascade" (especially "The receiver", "Title from diff class", "Labels", "What each repo's task moves"), "Gates", "Cascade files" and "Rollout and changes".
  - The Phase 2 cascade contract, version 1: the supervisor's scratchpad file `p2-cascade-contract.md`, cited as "contract §N". It fixes the resolver interface, the stub, the four tasks, the shared rules (§5.2), the library's moves (§6.2) and the test shape (§8).
  - Where the contract and `RELEASING.md` disagree, `RELEASING.md` wins, and the conflict goes to the supervisor.
- **Pins in the library today** (all current against what is published on 2026-10-04):

| Pin | Where | Value |
| --- | --- | --- |
| core, shipped | `opm/schema/loader.go:43`, `const DefaultSchemaModule = "opmodel.dev/core@v2.0.0-beta.2"` | beta.2 |
| core, derived | `opm/schema/loader.go:51-54` `DefaultSchemaVersion()`; `opm/internal/registrytest/registrytest.go:114` `var DefaultCoreVersion = schema.DefaultSchemaVersion()` | follows |
| core, test | `modules/opm_platform`, `testdata/modules/web_app`, `testdata/parity`, `testdata/parity/opm_platform` and `testdata/cue.mod` (`cue.mod/module.cue`) | beta.2 |
| core, test | 30 `testdata/render/**/cue.mod/module.cue` trees | beta.2 |
| opm catalog, test | the four `CUE_MODULE_GLOBS` modules above, all except `testdata/cue.mod` | `v4.5.1` |
| third-party | `cue.dev/x/k8s.io@v0` `v0.12.0` in `modules/opm_platform`, `testdata/parity`, `testdata/parity/opm_platform` | never moved |

- **Guard on the render trees.** `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` (`opm/internal/registrytest/registrytest_test.go:43-70`) walks `testdata/render`. It requires every core pin there to equal `schema.DefaultSchemaVersion()`, with at least 13 files.
- **Parity.** The parity harness reads its catalog version from `testdata/parity/cue.mod/module.cue` (`derive-fixture-versions`), so a catalog move edits only `cue.mod` files.
- **Frozen files.** `.cascade-frozen` lists 12 Go test files whose literals stay old on purpose (spec `fixture-pin-maintenance`, "Intentionally old version literals are declared").
- **What the existing task cannot do.** `task cue:deps:update` (`Taskfile.yml:303-451`) holds core at the loader value, but:
  - it resolves every other `opmodel.dev/*` dep at its bare major, so a `.cascade-hold` cannot apply;
  - it skips `testdata/cue.mod` and the render trees (`CUE_MODULE_GLOBS`, `Taskfile.yml:17-21`);
  - it prints "(unchanged)" and exits 0 either way, so it cannot report "nothing to do".

## Goals / Non-Goals

**Goals:**

- One command that brings both upstream pins to the newest published in-major version, honouring frozen pins and holds, and that exits 0, 3 or other exactly as `RELEASING.md` "The receiver" requires.
- Its output is a diff the shared title and body classify correctly: `fix(deps)` plus `need-human-review` when core moved; `test(fixtures)` when only the catalog moved.
- It is idempotent: a second run exits 3 and leaves the tree byte-identical.
- It is tested offline in the required `Go tests` job and over the network in a non-required job.

**Non-Goals:**

- The receive workflow, the PR, labels on GitHub, and `deps-cascade:breaking` (Phase 3; contract §9.8).
- Changing `task cue:deps:update` or its spec.
- Moving `docs/getting-started.md:51,56`. It is prose and already stale at `v2.0.0-beta.1`. It is only warned about.
- Bringing `main` current. That is the supervisor's catch-up PR (contract §8).

## Decisions

### D1. Files and tasks

Everything lives under `.tasks/cascade/` (contract §5.1):

| File | Role |
| --- | --- |
| `cascade.sh` | moves the pins (D3) |
| `pins.sh <ref>` | prints the TSV pin report (D2) |
| `classes` | the path-class map, verbatim from contract §5.3 |
| `test.sh` | the scenarios (D7) |
| `testdata/stub-resolve.sh` | the contract §7 stub, byte for byte, mode 0755, sha256 `970130f7d55c07f5b86d4f5b6f392330427ff923eb34f93553656bcd4b893d9c` |
| `testdata/older.tsv` | the static older published versions (D7) |
| `testdata/s1-calls.txt` | the expected S1 call list (D7) |

`classes` holds exactly:

```
test testdata/
test modules/
test *_test.go
```

`opm/schema/loader.go`, `go.mod` and `go.sum` fall through to `shipped`.

**Taskfile.** The four tasks go in `Taskfile.yml` beside `deps:release-check` (`Taskfile.yml:857`), under the exact names `deps:cascade`, `deps:cascade:title`, `deps:cascade:body` and `deps:cascade:test`.

- An included `.tasks/cascade.yaml` with namespace `deps` was considered and dropped. It would collide with the existing top-level `deps:release-check`.
- The `CASCADE_RESOLVER_PATH` var is declared at task level through one YAML anchor shared by the four tasks, never as a global `vars:` entry (contract §3).
- Each task exports `CASCADE_RESOLVER` and has the precondition `test -x '{{.CASCADE_RESOLVER_PATH}}'`, with the contract §3 message.
- `deps:cascade` also has the preconditions `git`, `yq` (mikefarah v4) and `jq`. `cue` is checked inside `cascade.sh`, and only when a pin will move, so the offline set needs no `cue` on the runner.

### D2. `pins.sh`

`pins.sh WORKTREE|<ref>` prints two rows in this order:

```
opmodel.dev/core@v2	core	shipped	<v>	need-human-review
opmodel.dev/catalogs/opm@v4	opm catalog	test	<v>	
```

- **Core** is read from `opm/schema/loader.go` with the existing idiom `grep -oP 'DefaultSchemaModule = "opmodel\.dev/core@\K[^"]+'` (`Taskfile.yml:351`).
- **Catalog** is read from the representative file `testdata/parity/cue.mod/module.cue`, as the `v:` inside the `"opmodel.dev/catalogs/opm@v4": {` block.
- **At a ref,** the same files are read with `git show <ref>:<path>`. A file or block missing at that ref omits the row.
- Exit 1 on any other failure.
- `cue` is not used, so the report works offline and on a runner without `cue`.

### D3. `cascade.sh` order

`cascade.sh` starts with `set -euo pipefail` and follows contract §5.2 rules 1-15. The library-specific order is contract §6.2.

**Phase 0, setup.**

1. Clean-tree check (or the `CASCADE_ALLOW_DIRTY=1` snapshot).
2. Create `STATE=$(git rev-parse --git-dir)/cascade` and truncate `$STATE/warnings`. Export `CASCADE_WARNINGS`.
3. Run `"$CASCADE_RESOLVER" check-files --repo-root .`.
4. Set `CUE_REGISTRY=OPM_REGISTRY=opmodel.dev=ghcr.io/open-platform-model,registry.cue.works`. This is the library's `CUE_GHCR_REGISTRY`, `Taskfile.yml:44`. The library resolves no `testing.opmodel.dev` fixture from a registry.

**Phase A, resolve.** No file is written until every target is known.

1. `D0` = the loader value. Call `newest cue opmodel.dev/core@v2 --current D0 --repo-root .`, plus `--expect` when `CASCADE_EXPECT` names `opmodel.dev/core@v2`.
   - Exit 0: the target `D` is the printed version.
   - Exit 3: `D = D0`.
   - Any other exit: the task exits non-zero.
2. `P0` = the parity file's catalog. Call `newest cue opmodel.dev/catalogs/opm@v4 --current P0 --repo-root .`.
   - On 0, with target `t`: run `pin-of opmodel.dev/catalogs/opm@v4 t opmodel.dev/core@v2`.
     - If that core is newer than `D` (by `semver-cmp`), the catalog stays (`K = P0`). The task warns "catalog `t` needs core `c`, newer than `DefaultSchemaModule` `D`; advance core first".
     - Otherwise `K = t`.
     - `pin-of` exit 3 (no core dep) also gives `K = t`.
   - On 3, `K = P0`, and no `pin-of` call is made.
   - This is the library's reading of contract §6.2 step 3. It differs from the generic S1 call list in contract §8, which mentions a `pin-of` for core. In the library `pin-of` only judges a catalog move, so it is not called when nothing moves.
3. **Language check.** For each CUE upstream that moved, call `language-of <module> <target>`. Compare the result with the pinned CUE version: the `version:` of the `cue-lang/setup-cue` step in `.github/workflows/cue.yml`, today `v0.17.1` (contract §5.2 rule 10).
   - Read it with `yq '.jobs.cue.steps[] | select(.uses | test("setup-cue")) | .with.version'`.
   - If the upstream's version is newer, warn. If the pinned version cannot be read, warn with key `-`.
   - For the library, the CUE that actually evaluates core at render time is the SDK, `cuelang.org/go` in `go.mod` (`AGENTS.md`, "CUE toolchain pin"), not the `cue` CLI. Both are `v0.17.1` today. The task follows the contract and reads `cue.yml`. Whether it should also compare against the `go.mod` SDK is an open point for the supervisor.

**Phase B, tools.** There is nothing to build. If any pin will move, check that `cue` is on `PATH`. Exit 1 if it is missing.

**Phase C, edit.** Only where something moved, in this order:

1. **Loader** (shipped). When `D != D0`, rewrite line 43's literal to `opmodel.dev/core@D`. It is a single anchored `sed` on `const DefaultSchemaModule = "opmodel.dev/core@...`.
   - The task checks that exactly one line changed. Otherwise it exits 1.
   - It appends the warning "`DefaultSchemaModule` moved from `D0` to `D`; `need-human-review`: re-verify the glue (`opm/schema/loader.go`) before merging", under key `opmodel.dev/core@v2`.
   - This signal accompanies the `labels` column of `pins.sh`, which puts `need-human-review` in the body's `cascade-labels` marker.
2. **Text re-pin** (test). The files are `testdata/cue.mod/module.cue` plus every path from `find testdata/render -path '*/cue.mod/module.cue' | sort`. For each file whose core `v:` differs from `D`:
   - skip it if `is-frozen <file> opmodel.dev/core@v2` answers 0;
   - otherwise rewrite only the `v:` inside the `"opmodel.dev/core@v2": {` block. An `awk` state machine does this and leaves every other byte alone. The synthetic `testing.opmodel.dev/library-render/*` deps are never touched.
   - No `cue` command runs on these trees.
   - This step runs on every run, not only when `D` moved. A tree that lags the loader therefore catches up, and a run on a consistent tree touches nothing.
3. **Module loop** (test). For each directory from `CUE_MODULE_GLOBS` that has a `cue.mod/module.cue` (the list is in D4):
   - **Scope.** A module is in scope when its core differs from `D`, or when it has the catalog and the catalog is below `K`.
   - **Get arguments.** For each in-scope module, build the list `opmodel.dev/core@D`, plus `opmodel.dev/catalogs/opm@K` when the module has the catalog. Drop any key for which `is-frozen <dir>/cue.mod/module.cue <key>` answers 0.
   - **Run.** If the list is non-empty, run `cue mod get <list>` then `cue mod tidy` in the module, with the output captured and printed on failure.
   - **Frozen check.** After tidy, check that every frozen key's `v:` is byte-unchanged. If not, exit 1 naming the file and key (contract §5.2 rule 8).
   - **Third-party check.** Compare every non-`opmodel.dev` dep's `v:` before and after. A raised one becomes the warning "tidy raised `<dep>` from `<a>` to `<b>`" under key `-`. It is never reverted.
4. **Docs warning.** If `docs/getting-started.md` names a core version other than `D`, warn under key `-`: "`docs/getting-started.md` still names `<v>`".

**Result.** Exit 0 if `git status --porcelain --untracked-files=all` is non-empty (or the snapshot differs under `CASCADE_ALLOW_DIRTY=1`). Otherwise exit 3.

Pseudo-shape of phase A's exit handling (contract §5.2 rule 5; no `|| true`, no `set +e` around it):

```bash
resolve() { # resolve KIND COORD CURRENT -> prints target; returns 0 moved, 3 stay
  local out rc=0
  out=$("$CASCADE_RESOLVER" newest "$1" "$2" --current "$3" --repo-root . $(expect_for "$2")) || rc=$?
  case "$rc" in
    0) printf '%s\n' "$out" ;;
    3) printf '%s\n' "$3" ;;
    *) printf 'cascade: newest %s %s failed (exit %s)\n' "$1" "$2" "$rc" >&2; exit "$rc" ;;
  esac
}
```

### D4. Module set

The loop uses the same four globs as `CUE_MODULE_GLOBS` (`Taskfile.yml:17-21`): `modules/*`, `testdata/modules/*`, `testdata/parity` and `testdata/parity/opm_platform`.

- The Taskfile passes them to the script as the env `CASCADE_MODULE_GLOBS: '{{.CUE_MODULE_GLOBS}}'`, so there is one list.
- `testdata/cue.mod` is not in the loop. It pins core only, and it is the module root of `testdata/`, whose subtrees include the render fixtures and do not resolve. It is re-pinned as text in step C2, as the render trees are.
- The task never calls `task cue:deps:update`. That task resolves the catalog at its bare major and cannot honour a hold. The contract allows refactoring it to take explicit versions and share the loop, but that would change a specified behavior (`fixture-pin-maintenance`) for no gain in this change.

### D5. What the task never touches

Contract §5.2 rule 14, as it applies to the library:

- `.cascade-frozen` and `.cascade-hold`;
- `.release-please-manifest.json`, `release-please-config.json` and `CHANGELOG.md`;
- anything under `.github/`;
- `.opm-docs-version` and `docs-kit.cue`;
- `cue-versions.yml`;
- every `language.version`;
- every Go file except `opm/schema/loader.go`.

`cue:publish:smart` and every `cue:publish*` task stay out of the cascade (rule 15).

### D6. CI placement

- **Required, offline.** One step in `test.yml` job `Go tests`, after "Install Task":
  - `run: task -x deps:cascade:test`, with env `CASCADE_TEST_SET: offline` and `CASCADE_RESOLVER: ${{ github.workspace }}/.tasks/cascade/testdata/stub-resolve.sh`.
  - The env satisfies the contract §3 precondition without a checkout of `.github`. `test.sh` uses the stub for every scenario anyway.
  - `ubuntu-latest` ships mikefarah `yq` v4. The step checks `yq --version` first and fails clearly if it is not there.
- **Non-required, network.** A new `.github/workflows/cascade-task.yml`:
  - job `Cascade task (network)`, with `timeout-minutes: 20` and `permissions: contents: read`;
  - triggers: `pull_request` with paths `.tasks/cascade/**`, `Taskfile.yml` and the workflow file; `workflow_dispatch`; and a weekly `schedule`;
  - steps: checkout, setup-go, setup-cue `v0.17.1`, setup-task, a checkout of `open-platform-model/.github` at `main` into `org-github` with `persist-credentials: false`, then `task -x deps:cascade:test` with `CASCADE_RESOLVER_REAL` set to the real resolver.
  - Every action is SHA-pinned as `test.yml` pins it.
  - The `.github` checkout step is present from the start. Until `add-cascade-resolver` merges, the real resolver path is absent and S5 skips with a printed line (D7).

### D7. Tests (`test.sh`)

The shape follows contract §8 exactly. Library specifics:

- **Sandbox.** Each sandbox also gets `cp -a` of the main checkout's `.cue-cache/mod` when it exists, never a symlink, and `chmod -R u+w` before cleanup. The main checkout is the parent of `git rev-parse --git-common-dir`.
- **Kind and coordinate map** used to build the stub table from `pins.sh WORKTREE`:
  - `opmodel.dev/core@v2` maps to `cue opmodel.dev/core@v2`;
  - `opmodel.dev/catalogs/opm@v4` maps to `cue opmodel.dev/catalogs/opm@v4`.
- **Current rows.** Two `newest` rows, one `pin-of opmodel.dev/catalogs/opm@v4 <tree catalog> opmodel.dev/core@v2 <tree core>`, and `language-of` rows for both modules at the tree versions with `v0.17.0`. The library has no version-advance modules, so it has no `published` rows.
- **`older.tsv`** (checked on 2026-10-04 by reading GHCR modulefiles):

  ```
  opmodel.dev/core@v2	v2.0.0-beta.1
  opmodel.dev/catalogs/opm@v4	v4.5.0
  pin-of	opmodel.dev/catalogs/opm@v4	v4.5.0	opmodel.dev/core@v2	v2.0.0-beta.1
  ```

  `v4.5.0` is picked over `v4.4.x` because it is after the k8s-catalog retirement, so it has the same dependency shape as `v4.5.1`.
- **`s1-calls.txt`** (normalized: each version replaced by `V`, lines sorted):

  ```
  check-files --repo-root .
  newest cue opmodel.dev/catalogs/opm@v4 --current V --repo-root .
  newest cue opmodel.dev/core@v2 --current V --repo-root .
  ```

- **S3.** Sets the core `newest` row to `ERROR`. Core is the first pin resolved.
- **S2 setup.** Writes `older.tsv`'s core into the loader, into `testdata/cue.mod` and every render tree, and into the five module files. Writes `older.tsv`'s catalog into the four catalog files.
  - **Expected.** Exit 0, and the copy is byte-identical to the original tree. The library has no version-advance paths, so the golden list is empty.
  - **Rerun.** A second run, with `CASCADE_BASE` still at the setup SHA, exits 3.
- **S4 frozen choice.** `testdata/modules/web_app/cue.mod/module.cue`, frozen for `opmodel.dev/core@v2` and `opmodel.dev/catalogs/opm@v4`. It is a loop module with both OPM keys and no third-party dep.
  - **Expected.** That file is byte-unchanged; everything else is as in S2; exit 0.
  - The repo's real 12 entries stay in the file during the scenario.
- **S5.** Runs only when `CASCADE_RESOLVER_REAL` is set and executable; otherwise it prints `SKIP S5`. After S2 it checks:
  - `title` prints `fix(deps): bump core to <tree core> and opm catalog to <tree catalog>`;
  - `body` holds both markers, two table rows, `need-human-review` in `cascade-labels`, and `## Notes` last.
- **Exit-code discipline.** Every invocation is `task -x ...` (contract §3).

### D8. Phase 2 gate for the library

- `RELEASING.md` "Phases" asks that "a run on `main` exits 3".
- **Today.** The library's `main` pins core `v2.0.0-beta.2` (the newest core) and catalog `v4.5.1` (the newest v4). `v4.5.1` pins core `v2.0.0-beta.1` (read from its GHCR modulefile), which is not newer than the loader. So a run on today's `main` is expected to exit 3 without a catch-up.
- **After catalog_opm's catch-up.** That catch-up releases a catalog on core beta.2, and the library's catalog pin then moves through the supervisor's catch-up PR (contract §8, tier 1).

## Research & Decisions

### Loader edit as text, not through Go tooling

**Context**: `DefaultSchemaModule` is a Go `const`. It could be rewritten with `gofmt -r`, a small Go program, or `sed`.
**Explored**: commit `3c3b8c1` changed only the string literal. The registrytest guard reads the derived version. `deps:release-check` (`Taskfile.yml:916`) already scans that file for dev literals.
**Decision**: an anchored `sed` on the one `const DefaultSchemaModule = "opmodel.dev/core@...` line, with a check that exactly one line changed.
**Rationale**: a literal-only edit needs no build. It also keeps `cascade.sh` working when the new core breaks compilation, which is what the CI of the cascade PR is there to show.

### Text re-pin of render trees

**Context**: the 30 render trees pin synthetic `testing.opmodel.dev/library-render/*` deps that no registry serves (`AGENTS.md` notes they are "served in-process").
**Explored**: `cue mod get` and `tidy` would fail there. `registrytest_test.go:34-42` says a default move "re-pins them by hand or by the cascade".
**Decision**: an `awk` rewrite scoped to the `"opmodel.dev/core@v2": {` block, run on a sorted `find`, never a list.
**Rationale**: it is the only edit those trees can take, and the walk makes new trees covered automatically.

### Explicit versions instead of `cue:deps:update`

**Context**: contract §5.2 rule 6 requires `<module>@<exact version>` and holds. `cue:deps:update` resolves the catalog at its bare major.
**Explored**: refactoring `cue:deps:update` to take explicit versions (contract §6.2 step 4 allows it), against a separate loop in `cascade.sh`.
**Decision**: a separate loop. `cue:deps:update` and its spec stay unchanged.
**Rationale**: it keeps the change to tooling the cascade owns, and leaves the hand-run task's culprit diagnostics as they are. The duplication is about 30 lines of module discovery.

### `pin-of` only when the catalog moves

**Context**: contract §8's S1 description mentions a `pin-of` for core, written for the consistent-set repos. Library core follows the loader, not the catalog (`RELEASING.md`, "The receiver", "Consistent set": "library differs").
**Decision**: the library calls `pin-of` only to judge a catalog move. `s1-calls.txt` has no `pin-of` line.
**Rationale**: no catalog move means no question to ask. This is reported to the supervisor as an open point, not a conflict.

## Risks / Trade-offs

- **A new core breaks the kernel.** This is intended. The task still produces the diff, and the cascade PR's CI and the `need-human-review` reviewer judge it. The loader comment (`opm/schema/loader.go:38-42`) is the human's checklist.
- **The flaky platformmodule cache race** (`TestGenerate_BuildsThroughTheKernel`). It can turn the cascade PR's `task test` red. `test.sh` does not run `go test`, so the task's own tests are unaffected.
- **`tidy` reorders or reformats a `module.cue`** in a way the original did not have. S2's byte-identity check would show it. The spike (task 1.1) confirms a round trip from the older pins gives the committed bytes.
- **The stub's `semver-cmp`** is correct only for `(alpha|beta|rc)(.N)*` prereleases. Every version the library compares has that form.
- **Proxy and GHCR lag** is the resolver's concern (`--expect`). The task only forwards `CASCADE_EXPECT`.
