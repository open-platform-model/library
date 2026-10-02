# Tasks: derive-fixture-versions

Worktree `library/.claude/worktrees/derive-fixture-versions`, branch
`test/derive-fixture-versions` (from `origin/main`). `.cue-cache/mod` was seeded by copying the
main checkout's. Every command runs inside the worktree with the registry env exported on two
lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

Every section lands as a release-hidden type (`chore`, `test` or `build`). The PR title is
`test(fixtures): derive test versions from the pins`. Commit bodies never start a line with
`word(` and carry no bare at-sign. The only trailer is
`Co-Authored-By: Claude <noreply@anthropic.com>`.

## 1. Spike: confirm the derive set with a default-move dry run (design DF3, DF6)

- [x] 1.1 Baseline: `go test ./opm/... -count=1 -short` is green on the untouched tree. If
      `TestGenerate_BuildsThroughTheKernel` fails only in the full run, rerun
      `go test ./opm/helper/platformmodule -count=1` alone and record it as the known flake, not a
      finding. Verify: green, or green on the rerun.
- [x] 1.2 Dry run on the untouched tree: set `DefaultSchemaModule` (`opm/schema/loader.go`) and
      `registrytest.DefaultCoreVersion` (`opm/internal/registrytest/registrytest.go`) to
      `v2.0.0-alpha.13`, text-replace `v: "v2.0.0-beta.1"` with `v: "v2.0.0-alpha.13"` in every
      `cue.mod/module.cue` under `testdata/` and `modules/`, and run
      `go test ./opm/... -count=1 -short`. Record every failing test and `_test.go` line. Verify:
      the failures match the design DF6 spike table (the six `render_test.go` `ResolvedVersions`
      rows and `loader_test.go:167,176`), or the table and DF3 are updated to what the run shows.
- [x] 1.3 `git checkout -- .` (every edit reverted; `git status --short` shows only
      `openspec/`). If 1.2 changed the derive set, update design.md DF3 and DF6 and the
      `.cascade-frozen` list in task 3.7. Verify: the design table and the run agree.
- [x] 1.4 `task check` green. If 1.2 or 1.3 changed design.md, commit
      `chore(openspec): confirm which test version literals a bump must move`; otherwise there is
      nothing to commit and the section is checked off with the next commit.

## 2. Parity harness derives the catalog build (opm/kernel; design DF1)

- [x] 2.1 `opm/kernel/parity_harness_test.go`: add `shippedCatalogVersion` (reads the
      `opmodel.dev/catalogs/opm@v4` pin from `testdata/parity/cue.mod/module.cue` with
      `cuelang.org/go/mod/modfile`) and `shippedTransformer(version, name)`. The seven
      `shippedCases` rows carry the bare transformer name. `TestParity_ShippedCatalog` resolves
      the rows once before `assertRowsCoverPairs`. Verify:
      `grep -nE '@4\.[0-9]+\.[0-9]+' opm/kernel/parity_harness_test.go` prints nothing.
- [x] 2.2 Add the consistency check: after reading `catalogVersion` off the oracle,
      `require.Equal(t, pinned, oracleCatalog, ...)` with the message "the oracle's catalog
      reports %s, the parity module pins %s". It runs before the `ResolvedVersions` check and
      before any case. Update the file's header comment to say where the version comes from.
      Verify, both local and reverted before commit: (a) negative: make `shippedCatalogVersion`
      temporarily return `"0.0.0"`; the test fails with that message and nothing else;
      (b) positive: text-set the catalog pin to `v4.4.1` in both `testdata/parity/cue.mod` and
      `testdata/parity/opm_platform/cue.mod` (no Go edit, no tidy) and run
      `OPM_FLOW_TEST_FORCE=1 go test ./opm/kernel -run 'TestParity_ShippedCatalog$' -count=1`; it
      is green, which proves the case rows follow the pin.
- [x] 2.3 `OPM_FLOW_TEST_FORCE=1 go test ./opm/kernel -run 'TestParity' -count=1` green against
      GHCR on the committed pins. Verify: `TestParity_ShippedCatalog` and its seven subtests
      pass, `TestParity_ShippedCatalogDiscriminated` passes, and the probe group passes.
- [x] 2.4 `task check` green, then commit
      `test(kernel): derive parity transformer ids from the pinned catalog`.

## 3. Current-core literals derive from the default; frozen literals declared (registrytest, kernel, renderstage, schema; design DF2, DF3, DF4)

- [x] 3.1 `opm/internal/registrytest/registrytest.go`: `var DefaultCoreVersion =
      schema.DefaultSchemaVersion()`. The doc comment says it is derived. Remove the first
      (now tautological) assertion of `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` and
      keep its `testdata/render` walk. Verify: `go test ./opm/internal/registrytest -count=1`
      green.
- [x] 3.2 `opm/kernel/render_test.go`: the six `ResolvedVersions` core rows (157, 206, 491, 526,
      992, 1059) and the line 906 module file read `registrytest.DefaultCoreVersion`. The line 32
      comment names the constant, not the release. `opm/internal/renderstage/stage_test.go`:
      lines 345 and 569 likewise; lines 475 and 533 stay (frozen, coupled to `platformModFile`).
      Verify: `grep -n 'beta\.1' opm/kernel/render_test.go` prints nothing, and
      `grep -n 'beta\.1' opm/internal/renderstage/stage_test.go` prints only line 475.
- [x] 3.3 `opm/schema/loader_test.go`: `TestDefaultSchemaModule_PinsVerifiedRelease` asserts
      shape: prefix `opmodel.dev/core@v2.`, not the bare major, and
      `schema.OCILoader{}.PinnedVersion()` returns `(DefaultSchemaVersion(), true)` (no new
      `go.mod` requirement; `PinnedVersion` already uses `module.ParseVersion`).
      `TestDefaultSchemaVersion_IsTheDefaultModulesVersion` asserts
      `DefaultSchemaModule == "opmodel.dev/core@" + DefaultSchemaVersion()` and a `v2.` prefix.
      Update both doc comments to the schema-dispatch delta (review of the bump change carries
      the deliberateness). The `PinnedVersion` parse table (lines 194-195) stays. Verify:
      `go test ./opm/schema -count=1` green, and lines 160-180 hold no `beta.1`.
- [x] 3.4 Doc comments (design DF4): `opm/schema/loader.go:21-23` says the constant names the
      verified release instead of naming `2.0.0-beta.1` (the `2.0.0-alpha.13` collisions sentence
      stays); `loader.go:48` describes the `DefaultSchemaVersion` form without a release;
      `opm/schema/cache.go:63` ties its example to `[DefaultSchemaModule]`, not to a release.
      Verify: `grep -n 'beta\.1' opm/schema/loader.go opm/schema/cache.go` prints only the
      constant (line 44) and the `PinnedVersion` format example (line 109); `go vet ./opm/schema`
      clean.
- [x] 3.5 Header comments: in the 11 `testdata/render/platform*/platform.cue` files and in
      `modules/opm_platform/platform.cue`, "core 2.0.0-beta.1" becomes "the core its cue.mod
      pins" (wrapped to the existing width). Update the `AGENTS.md` `testdata/` layout line per
      design DF4. Verify: `grep -rn 'beta\.1' --include=platform.cue testdata modules` prints
      nothing; `git add -- '*.cue' && task cue:fmt && git diff --exit-code -- '*.cue'` (stage the
      comment edits first, so the diff shows only what `cue:fmt` changed) exits 0;
      `task cue:vet` is green.
- [x] 3.6 `TestGenerate_ExplicitCorePin` (`opm/helper/platformmodule/generate_test.go`) stays as
      it is (design DF3: frozen; it becomes a non-default pin on the next core bump with no edit).
      Verify: `git diff origin/main -- opm/helper/platformmodule` is empty.
- [x] 3.7 New root `.cascade-frozen` in the workspace format (`frozen:` list of `path`, `pins`,
      `reason`). One entry per file the DF3 table marks *freeze*:
      `opm/kernel/render_collision_test.go`, `opm/kernel/render_core_floor_test.go`,
      `opm/platform/contracts_test.go`, `opm/errors/coretooold_test.go` (core);
      `opm/helper/platformmodule/closure_test.go` and `generate_test.go` (core,
      `opmodel.dev/catalogs/opm@v4`, `opmodel.dev/catalogs/k8s@v1`);
      `opm/internal/renderstage/modfile_test.go` and `stage_test.go` (core,
      `opmodel.dev/catalogs/opm@v4`); `opm/catalog/requires_test.go` (core);
      `opm/helper/objectset/objectset_test.go` (`opmodel.dev/catalogs/opm@v4`);
      `opm/internal/registrytest/registrytest_test.go` and `opm/schema/loader_test.go` (core).
      Each reason is one sentence; for `stage_test.go` and `loader_test.go` the reason says which
      literals are frozen, because the same file also reads the derived version; for
      `closure_test.go` it says the literals are nodes of an in-memory fake module graph, not
      pins (design DF3). Start the file
      with a comment that names its reader (release-cascade tooling, workspace RELEASING.md,
      section "Cascade files"). Verify: `yq '.frozen[].path' .cascade-frozen | xargs ls` lists
      every path, and every entry has a non-empty `pins` and `reason`.
- [x] 3.8 Core-bump reverse proof (not committed): the DF6 dry run on this tree, moving only
      `DefaultSchemaModule` to `opmodel.dev/core@v2.0.0-alpha.13` and text-replacing the core
      `v:` in every `cue.mod/module.cue` under `testdata/` and `modules/`. Run
      `go test ./opm/... -count=1 -short`, then `git checkout -- .`. Verify: green with no
      `*_test.go` edited (spec scenario "A default move edits no test file"); any failure names a
      literal that 3.1 to 3.4 missed.
- [x] 3.9 `task check` green, then commit
      `test(fixtures): derive current-core test literals from the schema default`.

## 4. `task cue:deps:update` in one pass, core held, loud (Taskfile.yml; design DF5)

- [x] 4.1 Rewrite the `cue:deps:update` script per design DF5. Read the default core from
      `opm/schema/loader.go` unless the `DEFAULT_CORE` task var is set, and fail with a message
      when the read comes back empty. Run one `cue mod get` per module over every direct
      `opmodel.dev/*` dependency (core at the default, others at `<path>@vN`; third-party
      dependencies are not named), then `cue mod tidy`. On failure,
      print the module directory and CUE's output, then rerun the get without core in a
      throwaway copy and name each moved dependency whose own module file
      (`mod/download/<path>/@v/<version>.mod` under
      `${CUE_CACHE_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/cue}`, falling back to the extracted
      `mod/extract/<path>@<version>/cue.mod/module.cue`) requires a
      core newer than the default; exit 1. Do not use `|| true`, `cue env`, or `/dev/null` for
      the main get and tidy. Keep the old-to-new report. Update `desc` and `summary` (mention
      `DEFAULT_CORE`). Verify: `grep -n '|| true'` in the task body prints nothing.
- [x] 4.2 Fix the stale "tracks the latest" claims: the comment on the `cue:catalog:drift` step
      in `.github/workflows/cue.yml` and the drift sentence in the `cue-versions.yml` header
      (existence of the pinned build, not currency). Verify: both match the `cue:catalog:drift`
      summary, and `task cue:check` still passes (the header is a comment; the checksums are
      unchanged).
- [x] 4.3 Exercise without committing pins: (a) on the current tree, `task cue:deps:update`
      exits 0 and moves only what is newer on GHCR (expected: the catalog `v4.4.2` to `v4.4.4`,
      core stays `v2.0.0-beta.1`), then `git checkout -- '*.cue'`; (b) with one module's catalog
      pin text-edited to an unpublished `v4.99.0`, the task exits non-zero and prints that module
      and CUE's error, then revert; (c) `DEFAULT_CORE=v2.0.0-alpha.13 task cue:deps:update` exits
      non-zero, prints CUE's "other requirements prevent changing module opmodel.dev/core@v2"
      error with the module directory, and names `opmodel.dev/catalogs/opm@v4.4.4` as requiring
      core `v2.0.0-beta.1`, also with `CUE_CACHE_DIR` pointing at an empty directory (a cold
      cache holds only `mod/download/`). Verify: the three outcomes as stated, and `git status --short` is
      clean afterwards apart from `Taskfile.yml`, `cue.yml` and `cue-versions.yml`.
- [x] 4.4 `task check` green, then commit
      `build: update test cue deps in one pass and fail loudly`.

## 5. Proof: bump the catalog pins to opm 4.4.4 (fixtures)

- [x] 5.1 `task cue:deps:update`. Verify:
      `git diff --name-only | grep -v '/cue.mod/module.cue$'` prints nothing; the four modules
      (`modules/opm_platform`, `testdata/modules/web_app`, `testdata/parity`,
      `testdata/parity/opm_platform`) read `opmodel.dev/catalogs/opm@v4` `v4.4.4`; core still
      reads `v2.0.0-beta.1`. `cue.dev/x/k8s.io` moves only if the new catalog requires a newer
      build; record it if so (design Risks).
- [x] 5.2 Cross-cutting checks against GHCR: `task cue:check`, `task cue:catalog:drift`,
      `OPM_FLOW_TEST_FORCE=1 task cue:test:flow`,
      `OPM_FLOW_TEST_FORCE=1 go test ./opm/kernel -run 'TestParity|TestFlow' -count=1`. Verify:
      all green with no Go file edited. If a parity case diverges on `4.4.4`, stop: revert 5.1,
      record the divergence in design.md Risks, leave section 5 unchecked, and report. Sections 1
      to 4 stand alone.
- [x] 5.3 `task check` green, then commit `test(fixtures): bump catalog pins to opm 4.4.4`.

## 6. Verify and archive

- [x] 6.1 Whole-tree gates on the final tree: `task check`, `task cue:check`,
      `task cue:catalog:drift`, `OPM_FLOW_TEST_FORCE=1 task cue:test:flow`, the parity run of 5.2,
      and `go test -race ./opm/kernel ./opm/internal/renderstage -count=1`. Verify: all green.
- [x] 6.2 `openspec validate derive-fixture-versions --strict` passes. Verify: the command
      prints that the change is valid.
- [x] 6.3 Archive the change on this branch (openspec archive), so the archive rides the
      implementing PR; never push to main (owner decision 2026-10-01, workspace RELEASING.md,
      section "Owner settings"). Run `openspec archive derive-fixture-versions --yes`. Verify:
      `render-parity` gains the new requirement; `schema-dispatch` carries the MODIFIED text with
      all five original scenarios plus the two new ones;
      `openspec/specs/fixture-pin-maintenance/spec.md` exists with its Purpose;
      `openspec validate --all --strict` passes. There is no `enhancement.yaml`, so no delivery
      log runs.
- [x] 6.4 Commit `chore(openspec): archive derive-fixture-versions`.
