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
| `lib.sh` | sourced helpers shared by the three scripts: the dependency-block reader `dep_v`, the block-scoped rewrite `set_dep_v`, the loader reader and the pin keys. Not in the contract §5.1 list; it is a sourced library, not a task entry point, and keeps one copy of the `awk` that reads and edits `cue.mod` blocks |
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
- The `CASCADE_RESOLVER_PATH` var is declared at task level through one YAML anchor shared by `deps:cascade`, `deps:cascade:title` and `deps:cascade:body`, never as a global `vars:` entry (contract §3).
- Those three tasks export `CASCADE_RESOLVER` and have the precondition `test -x '{{.CASCADE_RESOLVER_PATH}}'`, with the contract §3 message.
- `deps:cascade:test` has neither (contract v1.1 clarification C7): `test.sh` runs every scenario against the stub, and S5 reads `CASCADE_RESOLVER_REAL`. Its only precondition is `yq`.
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
4. Set the registries as two separate exports, `export CUE_REGISTRY=opmodel.dev=ghcr.io/open-platform-model,registry.cue.works` and `export OPM_REGISTRY=` the same value. This is the library's `CUE_GHCR_REGISTRY`, `Taskfile.yml:50`. The library resolves no `testing.opmodel.dev` fixture from a registry.

**Phase A, resolve.** No file is written until every target is known.

1. `D0` = the loader value. First call `is-frozen opm/schema/loader.go opmodel.dev/core@v2 --repo-root .` (read-only; contract v1.1 clarification C3 allows it before `newest`). On 0, warn "`opm/schema/loader.go` is frozen for `opmodel.dev/core@v2`; `DefaultSchemaModule` stays at `D0`", set `D = D0` and skip `newest`, so steps 2 and 3 judge the core the run really has (implementation review finding 1). Otherwise call `newest cue opmodel.dev/core@v2 --current D0 --repo-root .`, plus `--expect` when `CASCADE_EXPECT` names `opmodel.dev/core@v2`. `CASCADE_EXPECT` is split with `read -ra`, never by an unquoted expansion, so an untrusted payload cannot glob (contract §2.9).
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
   - For the library, the CUE that actually evaluates core at render time is the SDK, `cuelang.org/go` in `go.mod` (`AGENTS.md`, "CUE toolchain pin"), not the `cue` CLI. Both are `v0.17.1` today. The task follows the contract (rule 10, "one file") and reads `cue.yml` only. The plan review recommends warning against the lower of `cue.yml` and the `cuelang.org/go` version in `go.mod`; contract v1.1 clarification C9 keeps the single pinned file for now and makes the lower-of comparison a follow-up.

**Phase B, tools.** There is nothing to build. Phase B builds each loop module's `get` arguments (step C3), frozen keys already left out, from `D`, `K` and the files as they are, which is read-only apart from `is-frozen` calls. A module is in scope when its list moves something. If any loop module is in scope, check that `cue` is on `PATH`, and exit 1 if it is missing. A lagging loop module runs `get` and `tidy` even when no upstream pin moves, so the check keys on scope, not on a moved pin; a module whose only lag is a frozen key is out of scope, so a no-op run never needs `cue` (implementation review finding 4).

**Phase C, edit.** Only where something moved, in this order:

1. **Loader** (shipped). When `D != D0` (never for a frozen loader, phase A step 1), rewrite line 43's literal to `opmodel.dev/core@D`. It is a single anchored `sed` on `const DefaultSchemaModule = "opmodel.dev/core@...`.
   - The task checks that exactly one line changed. Otherwise it exits 1.
   - It appends the warning "`DefaultSchemaModule` moved from `D0` to `D`; `need-human-review`: re-verify the glue (`opm/schema/loader.go`) before merging", under key `opmodel.dev/core@v2`.
   - This signal accompanies the `labels` column of `pins.sh`, which puts `need-human-review` in the body's `cascade-labels` marker.
2. **Text re-pin** (test). The files are `testdata/cue.mod/module.cue` plus every path from `find testdata/render -path '*/cue.mod/module.cue' | sort`. For each file whose core `v:` differs from `D`:
   - skip it if `is-frozen <file> opmodel.dev/core@v2` answers 0;
   - otherwise rewrite only the `v:` inside the `"opmodel.dev/core@v2": {` block. An `awk` state machine does this and leaves every other byte alone. The synthetic `testing.opmodel.dev/library-render/*` deps are never touched.
   - No `cue` command runs on these trees.
   - This step runs on every run, not only when `D` moved. A tree that lags the loader therefore catches up, and a run on a consistent tree touches nothing.
3. **Module loop** (test). For each directory from `CUE_MODULE_GLOBS` that has a `cue.mod/module.cue` (the list is in D4):
   - **Candidates.** A module is a candidate when its core differs from `D`, or when it has the catalog and the catalog is below `K`.
   - **Get arguments** (built in phase B). For each candidate, build the list `opmodel.dev/core@D`, plus `opmodel.dev/catalogs/opm@K` only when the module has the catalog and `semver-cmp <module catalog> K` prints `-1`. A module whose catalog is at or above `K` gets core only, because `cue mod get` can downgrade a version it is given (`cue help mod get`), and a catalog above the target is never lowered. Drop any key for which `is-frozen <dir>/cue.mod/module.cue <key>` answers 0. The module is in scope when the list is non-empty and is not just `opmodel.dev/core@D` on a module already at `D` (its only lag was a frozen catalog).
   - **Run.** If the list is non-empty, run `cue mod get <list>` then `cue mod tidy` in the module, with the output captured and printed on failure.
   - **Frozen check.** After tidy, check that every frozen key's `v:` is byte-unchanged. If not, exit 1 naming the file and key (contract §5.2 rule 8).
   - **Third-party check.** Compare every non-`opmodel.dev` dep's `v:` before and after. A raised one becomes the warning "tidy raised `<dep>` from `<a>` to `<b>`" under key `-`. It is never reverted.
4. **Docs warning.** For `docs/getting-started.md` and `AGENTS.md`, each `opmodel.dev/core@v<full semver>` literal other than `opmodel.dev/core@D` gives a warning under key `-`: "`<file>` still names `<v>`". Both files carry the same reproducible-pin example (`docs/getting-started.md:51`, `AGENTS.md:346`), so both are checked; contract §6.2 step 5 names only the first, and checking the second is a strict addition (a warning, never an edit).

**Result.** Exit 0 if `git status --porcelain --untracked-files=all` is non-empty (or the snapshot differs under `CASCADE_ALLOW_DIRTY=1`). Otherwise exit 3.

Pseudo-shape of phase A's exit handling (contract §5.2 rule 5; no `|| true`, no `set +e` around it):

```bash
# resolve KIND COORD CURRENT: print the target (the newer version on resolver
# exit 0, CURRENT on exit 3) and return 0; on any other resolver exit, print a
# diagnostic and exit with that code. Callers use T=$(resolve ...) under
# set -e: the exit only leaves the command substitution's subshell, and set -e
# then stops the script on the assignment's non-zero status.
resolve() {
  local out rc=0 pairs p expect=()
  read -ra pairs <<<"${CASCADE_EXPECT:-}"
  for p in "${pairs[@]}"; do
    case "$p" in "$2"=*) expect=(--expect "${p#*=}") ;; esac
  done
  out=$("$CASCADE_RESOLVER" newest "$1" "$2" --current "$3" --repo-root . "${expect[@]}") || rc=$?
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

`deps:cascade:test` needs no resolver (contract v1.1 clarification C7): `test.sh` sets `CASCADE_RESOLVER` to the stub inside each sandbox, so neither CI step sets it. The network job uses the checkout layout of contract v1.1 clarification C4 (the repo at `repo`, `.github` at `org-github` beside it), and the offline step runs inside the required `Go tests` job (clarification C6).

- **Required, offline.** One step in `test.yml` job `Go tests`, after "Install Task":
  - `run: task -x deps:cascade:test`, with env `CASCADE_TEST_SET: offline`. No checkout of `.github` is needed.
  - `ubuntu-latest` ships mikefarah `yq` v4. The step checks `yq --version` first and fails clearly if it is not there.
- **Non-required, network.** A new `.github/workflows/cascade-task.yml`:
  - job `Cascade task (network)`, with `timeout-minutes: 20` and `permissions: contents: read`;
  - triggers: `pull_request` with paths `.tasks/cascade/**`, `.tasks/*.yaml`, `Taskfile.yml`, `.cascade-frozen`, `.cascade-hold` and the workflow file; `workflow_dispatch`; and a weekly `schedule`;
  - steps: checkout of the library with `path: repo`; setup-cue `v0.17.1`; setup-task; a checkout of `open-platform-model/.github` at `main` with `path: org-github` and `persist-credentials: false`, so it sits beside the repo and never inside its working tree (where `test.sh`'s `git ls-files --others` would copy it into every sandbox); then `task -x deps:cascade:test` with `working-directory: repo`.
  - `CASCADE_RESOLVER_REAL=$GITHUB_WORKSPACE/org-github/.github/scripts/cascade/cascade-resolve.sh` is exported by the run step only when that file exists.
  - No setup-go: nothing in the task or its tests runs Go.
  - Every action is SHA-pinned as `test.yml` pins it.
  - The `.github` checkout step is present from the start. Until `add-cascade-resolver` merges, the real resolver path is absent and S5 skips with a printed line (D7).

### D7. Tests (`test.sh`)

The shape follows contract §8 exactly. Library specifics:

- **Sandbox.** For the network scenarios (S2, S4, S5), each sandbox gets `cp -a` of the main checkout's `.cue-cache/mod` into `<tmp>/cue-cache/mod` when it exists, never a symlink, and `test.sh` exports `CUE_CACHE_DIR=<tmp>/cue-cache` for that run, so the `cue` CLI uses the copy instead of its default `~/.cache/cue`. The cache sits beside the sandbox repo, never inside it. `chmod -R u+w` runs before cleanup. The offline set skips the copy, since it runs no `cue`. The main checkout is the parent of `git rev-parse --git-common-dir`.
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
- **`s1-calls.txt`** (normalized: each full-semver match of `v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?` replaced by `V`, so `@v2` and `@v4` stay as written; lines sorted with `LC_ALL=C sort`):

  ```
  check-files --repo-root .
  is-frozen opm/schema/loader.go opmodel.dev/core@v2 --repo-root .
  newest cue opmodel.dev/catalogs/opm@v4 --current V --repo-root .
  newest cue opmodel.dev/core@v2 --current V --repo-root .
  ```

- **S1 new-major warning.** S1's table also carries a `warn cue opmodel.dev/core@v2` row, the stub's stand-in for the resolver's "new major available" warning. S1 asserts that the line reaches `.git/cascade/warnings` under key `opmodel.dev/core@v2` while the loader stays unchanged. Holds and the new-major probe themselves are resolver behaviour, tested in `.github` (contract §2.10); the library tests only that it obeys exit 3 and passes warnings through.
- **S3.** Sets the core `newest` row to `ERROR`. Core is the first pin resolved.
- **S7 lagging tree (offline).** One render tree's core `v:` is set to `older.tsv`'s core; the loader and the other files are unchanged; current rows only. Expected: exit 0, and the only path that differs from the setup commit is that file, now equal to the original bytes. No `cue` runs, since no loop module is in scope.
- **S8 catalog needs a newer core (offline).** Current rows, except that the catalog's `newest` row names a version above the tree (`v4.999.0`) and a `pin-of opmodel.dev/catalogs/opm@v4 v4.999.0 opmodel.dev/core@v2 v2.999.0` row names a core above the loader. Expected: exit 3, a clean tree, and "advance core first" in `.git/cascade/warnings`.
- **S9 frozen loader (offline).** The sandbox appends an entry freezing `opm/schema/loader.go` for `opmodel.dev/core@v2` to `.cascade-frozen`. The table offers core `v2.999.0`, catalog `v4.999.0` whose `pin-of` core is `v2.999.0`, and a `language-of` row for core `v2.999.0` at `v0.99.0`. Expected: exit 3, a clean tree, no `newest` call for core, the "is frozen" and "advance core first" warnings, and no `language.version` warning.
- **S10 frozen lag needs no cue (offline).** `testdata/modules/web_app/cue.mod/module.cue` gets `older.tsv`'s core and a `.cascade-frozen` entry for `opmodel.dev/core@v2`; current rows only. The run gets a `PATH` with no `cue` on it. Expected: exit 3 and a clean tree, since the module's only lag is frozen.
- **S2 setup.** Writes `older.tsv`'s core into the loader, into `testdata/cue.mod` and every render tree, and into the five module files. Writes `older.tsv`'s catalog into the four catalog files.
  - **Expected.** Exit 0, and the copy is byte-identical to the original tree. The library has no version-advance paths, so the golden list is empty.
  - **Rerun.** A second run, with `CASCADE_BASE` still at the setup SHA, exits 3.
- **S4 frozen choice.** `testdata/modules/web_app/cue.mod/module.cue`, frozen for `opmodel.dev/core@v2` and `opmodel.dev/catalogs/opm@v4`. It is a loop module with both OPM keys and no third-party dep.
  - **Expected.** That file is byte-unchanged; everything else is as in S2; exit 0.
  - The repo's real 12 entries stay in the file during the scenario.
- **S4b tidy raises a frozen key (network).** `testdata/modules/web_app/cue.mod/module.cue` gets core `v2.0.0-alpha.12`, frozen for `opmodel.dev/core@v2` only, and `older.tsv`'s catalog; current rows otherwise. The catalog moves to the tree's, whose core is newer than `v2.0.0-alpha.12`, so `tidy` raises the frozen core by MVS. Expected: an exit other than 0 and 3, the output names the frozen key and its old value, and no file other than that one differs from the setup commit (contract §5.2 rule 5 allows the partial edit; implementation review finding 2).
- **S5.** Runs only when `CASCADE_RESOLVER_REAL` is set and executable; otherwise it prints `SKIP S5`. After S2 it checks:
  - `title` prints `fix(deps): bump core to <tree core> and opm catalog to <tree catalog>`;
  - `body` holds both markers, two table rows, `need-human-review` in `cascade-labels`, and `## Notes` last.
  - **Catalog-only variant.** A second sandbox sets only the four catalog files to `older.tsv`'s catalog and runs `task -x deps:cascade` (exit 0). Then `title` prints `test(fixtures): bump opm catalog to <tree catalog>`, and the body's `cascade-labels` marker is empty.
- **Exit-code discipline.** Every invocation is `task -x ...` (contract §3).
- **Local runs.** Until `add-cascade-resolver` merges, no resolver sits at the workspace default path, so a local run of `deps:cascade`, `deps:cascade:title` or `deps:cascade:body` is prefixed `CASCADE_RESOLVER=$PWD/.tasks/cascade/testdata/stub-resolve.sh` (or points at the resolver worktree). `deps:cascade:test` needs no prefix (contract v1.1 clarification C7).

### D8. Phase 2 gate for the library

- `RELEASING.md` "Phases" asks that "a run on `main` exits 3".
- **Today.** The library's `main` pins core `v2.0.0-beta.2` (the newest core) and catalog `v4.5.1` (the newest v4). `v4.5.1` pins core `v2.0.0-beta.1` (read from its GHCR modulefile), which is not newer than the loader. So a run on today's `main` is expected to exit 3 without a catch-up.
- **Interim evidence (2026-10-04).** `task -x deps:cascade` on a clean copy of this branch at `f12233e` (the implementation review's re-run used `fe93069`, the same phase A), against the real resolver from the unmerged `add-cascade-resolver` branch (`a481a29`) and live GHCR, exited 3 with an empty `git status --porcelain`. The warnings file held only the two prose warnings (`docs/getting-started.md` and `AGENTS.md` still name `opmodel.dev/core@v2.0.0-beta.1`). The full test set passed with `CASCADE_RESOLVER_REAL` at the same resolver, S2, S4, S4b and S5 included. Task 4.5 repeats both once the resolver is on `.github` `main`.
- **After catalog_opm's catch-up.** That catch-up releases a catalog on core beta.2, and the library's catalog pin then moves through the supervisor's catch-up PR (contract §8, tier 1).

## Research & Decisions

### Spike results (task 1.1, 2026-10-04)

- **S2 round trip.** In a scratch copy, the four catalog modules were set to core `v2.0.0-beta.1` and catalog `v4.5.0` by text, then `cue mod get opmodel.dev/core@v2.0.0-beta.2 opmodel.dev/catalogs/opm@v4.5.1` and `cue mod tidy` ran in each (CUE v0.17.1, `CUE_CACHE_DIR` at a copy of `.cue-cache`). `git status --porcelain` was empty afterwards: the round trip gives the committed bytes.
- **`awk` block rewrite.** On `testdata/cue.mod/module.cue` and all 30 render trees, `set_dep_v` to `v2.0.0-beta.1` and back reproduced every file byte for byte, and the first rewrite differed only in the core `v:`. A missing block returns 1.
- **go-task.** Settled by the plan review (anchor shared by task-level `vars:`, `task -x` returns 3); not re-run.
- **`older.tsv`.** Read with the real resolver from the `add-cascade-resolver` branch: `published` answers 0 for core `v2.0.0-beta.1` and catalog `v4.5.0`; `pin-of` of catalog `v4.5.0` (and `v4.5.1`) names core `v2.0.0-beta.1`; `newest` answers 3 for both current pins.

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
**Rationale**: no catalog move means no question to ask. The plan review agreed: contract §6.2 step 3 calls `pin-of` only on a catalog target.

## Risks / Trade-offs

- **A new core breaks the kernel.** This is intended. The task still produces the diff, and the cascade PR's CI and the `need-human-review` reviewer judge it. The loader comment (`opm/schema/loader.go:38-42`) is the human's checklist.
- **The flaky platformmodule cache race** (`TestGenerate_BuildsThroughTheKernel`). It can turn the cascade PR's `task test` red. `test.sh` does not run `go test`, so the task's own tests are unaffected.
- **`tidy` reorders or reformats a `module.cue`** in a way the original did not have. S2's byte-identity check would show it. The spike (task 1.1) confirms a round trip from the older pins gives the committed bytes.
- **The stub's `semver-cmp`** is correct only for `(alpha|beta|rc)(.N)*` prereleases. Every version the library compares has that form.
- **Proxy and GHCR lag** is the resolver's concern (`--expect`). The task only forwards `CASCADE_EXPECT`.

## Plan review

The plan review of commit `0e97f9e` raised 14 findings. All were applied; none was rejected.

- 1 (network workflow red on every run): D6, library checked out at `path: repo`, stub as `CASCADE_RESOLVER`, `CASCADE_RESOLVER_REAL` only when the file exists.
- 2 (no archive section): `tasks.md` section 5. In this implementation run the archive task stays open; the archive rides the PR.
- 3 (module loop could lower a catalog): D3 step C3, catalog named only when below `K`.
- 4 (library branches untested): S7, S8 and the S5 catalog-only variant (D7).
- 5 (two scenarios untestable here): the spec scenarios are reworded as "the resolver answers ...", and S1 checks warning pass-through.
- 6 (local runs fail the precondition): D7 "Local runs", tasks 2.7, 3.5 and 4.4.
- 7 (cache copy did nothing): D7 "Sandbox", `CUE_CACHE_DIR`.
- 8 (trigger paths): D6.
- 9 (Phase 0 step 4 errors): D3.
- 10 (`resolve()` comment): D3.
- 11 (Phase B `cue` check too narrow): D3 Phase B keys on scope.
- 12 (S1 normalisation): D7.
- 13 (`setup-go` not needed): D6.
- 14 (`AGENTS.md:346` stale too): D3 step C4 checks both files.

Open questions answered by the review: Q1 and Q4 confirmed as written; Q5 removed the go-task item from the spike (task 1.1). Q2 (which CUE version the language check compares against) was settled by contract v1.1 clarification C9: one pinned file now, the lower-of comparison a follow-up. Q3 (the contract §8 tier-2 order waits on a library release that a `test(fixtures)` catch-up never cuts) is still open with the supervisor.

## Implementation review

The implementation review of `origin/main..9baf7e5` raised no blocker or major; all four findings were applied, none rejected.

- 1 (minor, frozen loader judged late): D3 phase A step 1 asks `is-frozen` before `newest`; offline S9.
- 2 (minor, no test for a frozen key that tidy raises): network S4b and a spec scenario.
- 3 (nit, `CASCADE_EXPECT` globbing): split with `read -ra` (D3 phase A step 1).
- 4 (nit, a frozen lag kept a module in scope): phase B builds the `get` lists and scope keys on them; offline S10 runs with no `cue` on `PATH`.

Contract v1.1 clarifications that bind this change: C4 (the network job's `repo` / `org-github` layout, D6), C5 (every stub call carries `CASCADE_STUB_TABLE`; the `older` check passes `/dev/null`, since `semver-cmp` reads no row), C6 (the offline step in the required `Go tests` job, D6) and C7 (`deps:cascade:test` needs no resolver, D1 and D6). The reproducible-pin examples in `docs/getting-started.md` and `AGENTS.md` that raise the two prose warnings are a separate docs PR, not this change.
