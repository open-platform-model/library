# Tasks: move-to-core-beta4

Worktree: `library/.claude/worktrees/move-to-core-beta4`. Branch: `fix/move-to-core-beta4`, from `origin/main` `ca7c56b`.

Setup:

- Seed `.cue-cache` by copying the main checkout's: `cp -a /var/home/emil/dev/open-platform-model/library/.cue-cache <worktree>/`. Never symlink it.
- Run every command inside the worktree, with a private absolute `TMPDIR` (`mktemp -d` under the session scratchpad).
- Export the registry env on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

- Before starting, check the open library PRs (`gh pr list -R open-platform-model/library --state open`). If a `deps/cascade` PR moves core to beta.4, stop and report: that PR carries the move.

Known flake: `TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run because of a cross-process cache race. Rerun `go test ./opm/helper/platformmodule -count=1` on its own before treating it as a finding.

Commit rules:

- No body line starts with `word(`, and no bare at-sign appears.
- The only trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`.
- Decisions are cited as `NNNN:Dn` or as an owner decision by its walkthrough id (`owner decision j3`). Never cite a bare `Dn` or an identifier the repo cannot resolve.

## 1. Pin core v2.0.0-beta.4 (schema, testdata, docs examples; design "Which tool makes the move", "Whether any library fixture is mis-keyed")

- [x] 1.1 Run `task -x deps:cascade` in the worktree.
      - It finds `../.github` through the common git dir. If it does not, pass `CASCADE_RESOLVER=/var/home/emil/dev/open-platform-model/.github/.github/scripts/cascade/cascade-resolve.sh`.
      - Expect exit 0, one `need-human-review` warning for the loader in `$(git rev-parse --absolute-git-dir)/cascade/warnings` (in a worktree `.git` is a file), and these edits and no others:
        - `opm/schema/loader.go:43` reads `"opmodel.dev/core@v2.0.0-beta.4"`;
        - 35 `cue.mod/module.cue` core pins move to `v2.0.0-beta.4`: `testdata/cue.mod`, every `testdata/render` tree including the 14 registry modules, and the four `CUE_MODULE_GLOBS` modules;
        - the `OCILoader` examples in `docs/getting-started.md` and `AGENTS.md`.
      - If `opmodel.dev/catalogs/opm@v4` moves too (a catalog release published since planning), revert that move: keep `v4.6.0`, record it, and leave it to the cascade.
      - Fallback, only if the resolver cannot run: edit the constant by hand, run `DEFAULT_CORE=v2.0.0-beta.4 task cue:deps:update`, then edit the `testdata/cue.mod` and `testdata/render` pins and the two docs examples as text.
      - Verify: `git grep -n 'v2.0.0-beta.3' -- '*module.cue' opm/schema/loader.go docs/getting-started.md AGENTS.md` prints nothing. `git diff --stat` touches no `*_test.go`, no `.cascade-frozen` file, no `.tasks/` file, no spec and no doc comment.
- [x] 1.2 Check the derived pins: `go test ./opm/internal/registrytest ./opm/schema ./opm/internal/loader ./opm/helper/platformmodule -count=1` is green. `registrytest.DefaultCoreVersion` and `schema.DefaultSchemaVersion()` now read `v2.0.0-beta.4`.
- [x] 1.3 Re-vet every library fixture against beta.4 (owner decision j3):
      - `task test` is green, including the shipped parity group and the kernel render suite over every `testdata/render` scenario, platform and registry module. Confirm with `go test -v ./opm/kernel -run 'TestParity_Shipped' -count=1` that the shipped group ran rather than skipped.
      - `task cue:vet` is green over the `CUE_MODULE_GLOBS` modules.
      - `grep -rnE '#(resources|traits|blueprints):' --include=*.cue testdata modules`, then read each map. Every entry is keyed `(X.metadata.fqn): X`, or the map is empty.
      - If anything is mis-keyed, key it by `(X.metadata.fqn)` in this section and list it in the commit body.
- [x] 1.4 Run the gates on the moved tree:
      - `task check` is green: fmt, vet, lint, test, docs bundle check, cascade wiring check, api-diff test and consumer-build test.
      - Run `task api:diff`. Its Allowed section shows `./opm/schema.DefaultSchemaModule: value changed from ... to "opmodel.dev/core@v2.0.0-beta.4"`, and it charges no incompatible change to this branch.
      - Clone fresh `cli` and `opm-operator` at `main` into the scratch dir. From the worktree root, run `GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> . <work-dir>` for each. Both are green, and the commits are recorded for the PR body.
- [x] 1.5 Commit `fix(deps): bump core to v2.0.0-beta.4`, the title `task -x deps:cascade:title` prints. The body says, in two or three lines:
      - every core pin and the docs examples move to the first core that binds attachment keys to `metadata.fqn` (owner decision j3);
      - every library fixture passes it with no edit;
      - the opm catalog stays at `v4.6.0`, and no parity literal exists to move (`opm/kernel/parity_harness_test.go:157` reads the catalog build from `testdata/parity/cue.mod`).
