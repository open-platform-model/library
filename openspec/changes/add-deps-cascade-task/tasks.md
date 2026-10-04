Gate for every section: `task check` (fmt, vet, lint, test, docs:bundle:check) green on the whole tree, plus `shellcheck` on every script under `.tasks/cascade/` (the stub copy included). This change edits no Go file, so `task check` guards against accidental edits.

Merge gate (not a task): `.github` `add-cascade-resolver` is merged before this change's PR merges (`RELEASING.md` "Rollout and changes"). Sections 1-4 build and test against the stub; task 4.5 is the only step that needs the real resolver.

## 1. Spike and inputs: class map, pin report, stub

- [x] 1.1 Spike: in a scratch copy of the tree (never this worktree), confirm the design's open assumptions and write any correction into design.md:
  - the S2 round trip from `older.tsv` back to the current pins gives the committed bytes. Use a hand-run `cue mod get opmodel.dev/core@<D> opmodel.dev/catalogs/opm@<K>` plus `cue mod tidy` in the four catalog modules, from core `v2.0.0-beta.1` and catalog `v4.5.0`;
  - the `awk` core-block rewrite leaves every other byte of a render tree alone.
- [x] 1.2 Add `.tasks/cascade/classes` with the three lines from design.md D1, verbatim from contract §5.3.
- [x] 1.3 Add `.tasks/cascade/pins.sh` (mode 0755) per design.md D2. Check by hand that `pins.sh WORKTREE` and `pins.sh HEAD` print the same two rows, and that `pins.sh 3c3b8c1~1` prints core `v2.0.0-beta.1`.
- [x] 1.4 Copy the contract §7 stub to `.tasks/cascade/testdata/stub-resolve.sh` (mode 0755). Check that `sha256sum` prints `970130f7d55c07f5b86d4f5b6f392330427ff923eb34f93553656bcd4b893d9c`.
- [x] 1.5 Add `.tasks/cascade/testdata/older.tsv` (design.md D7). Re-check its three rows against GHCR: the manifest exists, and the `pin-of` row matches the modulefile layer.
- [x] 1.6 Gate green, then commit `ci(cascade): add the cascade pin report and class map`.

## 2. The deps:cascade task

- [x] 2.1 Add `.tasks/cascade/cascade.sh` (mode 0755), phases 0, A, B and C per design.md D3. Use the exit-handling shape shown there, with no `|| true` and no `set +e` around resolver or `cue` calls.
- [x] 2.2 Implement the loader edit. It is an anchored rewrite of the `DefaultSchemaModule` literal, with a check that exactly one line changed and a `need-human-review` warning under key `opmodel.dev/core@v2`.
- [x] 2.3 Implement the text re-pin of `testdata/cue.mod/module.cue` and of every `find testdata/render -path '*/cue.mod/module.cue'` result, with an `is-frozen` check per file.
- [x] 2.4 Implement the module loop over `CASCADE_MODULE_GLOBS` (design.md D4). It covers scope, explicit-version `get` and `tidy`, frozen keys left out and verified after tidy, and third-party raises warned.
- [x] 2.5 Implement the warnings:
  - catalog needs a newer core;
  - `language.version` against `cue.yml`'s setup-cue version;
  - `docs/getting-started.md`;
  - tidy raises.
- [x] 2.6 Add the tasks to `Taskfile.yml` beside `deps:release-check`:
  - `deps:cascade`, `deps:cascade:title` and `deps:cascade:body`, sharing one YAML-anchored `CASCADE_RESOLVER_PATH` task var;
  - the contract §3 precondition and message;
  - env `CASCADE_MODULE_GLOBS: '{{.CUE_MODULE_GLOBS}}'`;
  - preconditions for `git`, `yq` (mikefarah v4) and `jq`.
- [x] 2.7 Check by hand in a scratch copy, with `CASCADE_RESOLVER=$PWD/.tasks/cascade/testdata/stub-resolve.sh` and a table of the tree's own versions, that `task -x deps:cascade` exits 3 and leaves `git status --porcelain` empty.
- [x] 2.8 Gate green, then commit `ci(cascade): add the deps:cascade task`.

## 3. Offline tests in the required job

- [x] 3.1 Add `.tasks/cascade/test.sh` (mode 0755). It holds the sandbox, the table builder, the checksum and pins checks, and the S1 (with the new-major warning pass-through), S3, S6, S7, S8, S9 and S10 scenarios (design.md D7; contract §8). It prints `PASS`/`FAIL` lines and exits 0 or 1.
- [x] 3.2 Add `.tasks/cascade/testdata/s1-calls.txt` with the four normalized lines from design.md D7.
- [x] 3.3 Add the `deps:cascade:test` task, with `CASCADE_TEST_SET` passed through and a precondition on `yq`. It needs no resolver var or precondition (contract v1.1 clarification C7).
- [x] 3.4 Add a step to `.github/workflows/test.yml` job `Go tests`, after "Install Task". It runs `task -x deps:cascade:test` with env `CASCADE_TEST_SET: offline`, after a `yq --version | grep -q mikefarah` check. Run `actionlint` on the file.
- [x] 3.5 Run `CASCADE_TEST_SET=offline task -x deps:cascade:test` locally, with no `CASCADE_RESOLVER` set: every scenario PASS, and the worktree unchanged afterwards.
- [x] 3.6 Gate green, then commit `ci(cascade): run the offline cascade task tests in Go tests`.

## 4. Network tests, workflow and docs

- [x] 4.1 Extend `test.sh` with S2 (older pins, an empty golden list of advance paths, and a second run that exits 3), S4 (`testdata/modules/web_app/cue.mod/module.cue` frozen for both keys, appended to the real `.cascade-frozen` in the sandbox), and S5 with its catalog-only variant, gated on `CASCADE_RESOLVER_REAL`, otherwise `SKIP S5`. S2, S4 and S5 copy the main checkout's `.cue-cache/mod` beside the sandbox and export `CUE_CACHE_DIR` to it.
- [x] 4.2 Add `.github/workflows/cascade-task.yml` per design.md D6:
  - job `Cascade task (network)`, `timeout-minutes: 20`, `permissions: contents: read`;
  - triggers: PR paths (`.tasks/cascade/**`, `.tasks/*.yaml`, `Taskfile.yml`, `.cascade-frozen`, `.cascade-hold`, the workflow file), `workflow_dispatch` and a weekly schedule;
  - SHA-pinned actions, setup-cue `v0.17.1`, no setup-go;
  - the library checked out at `path: repo`, the task run with `working-directory: repo` and no `CASCADE_RESOLVER`;
  - the `org-github` checkout of `open-platform-model/.github` at `main` with `persist-credentials: false`, then `CASCADE_RESOLVER_REAL` exported only when the resolver file exists.

  Run `actionlint`.
- [x] 4.3 Add a short "Release cascade task" paragraph to `AGENTS.md` (Build/test section). It names the four tasks, `task -x`, the exit codes, `need-human-review` on a core move, and that `cue:deps:update` stays the hand-run task.
- [x] 4.4 Run `task -x deps:cascade:test` (full set) locally with network: S1-S4 PASS, S5 SKIP or PASS.
- [ ] 4.5 Once `add-cascade-resolver` is merged in `.github`, run the full set again with `CASCADE_RESOLVER_REAL` pointing at the workspace checkout's real resolver, and check that S5 passes. Run `task -x deps:cascade` with the real resolver on a clean scratch copy of `main`, and record its exit code and diff in design.md D8 (the Phase 2 gate evidence).
- [x] 4.6 Run `openspec validate add-deps-cascade-task --strict`.
- [x] 4.7 Gate green, then commit `ci(cascade): test the cascade task over the network`.

## 5. Verify and archive

- [ ] 5.1 Run `openspec verify` (the repo's `openspec-verify-change` skill) and fix every CRITICAL finding.
- [ ] 5.2 Run `openspec archive add-deps-cascade-task`, then write the `## Purpose` line of the new main spec `openspec/specs/deps-cascade/spec.md` by hand.
- [ ] 5.3 Run `openspec validate --all --strict`.
- [ ] 5.4 Commit `chore(openspec): archive add-deps-cascade-task`. The archive rides the implementing PR (owner decision 4; contract §10).
