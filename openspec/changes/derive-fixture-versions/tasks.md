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

## 1. Spike: classify every test version literal by mutation (design D3, D6)

- [ ] 1.1 Baseline: `go test ./opm/... -count=1` is green on the untouched tree. If
      `TestGenerate_BuildsThroughTheKernel` fails only in the full run, rerun
      `go test ./opm/helper/platformmodule -count=1` alone and record it as the known flake, not a
      finding. Verify: green, or green on the rerun.
- [ ] 1.2 Freeze candidates: for each *freeze* row of the design D3 table, replace the literal
      with `v2.0.0-beta.99` (core) or `4.99.0` / `v4.99.0` (catalog) and run that package's tests
      with `-count=1`. Keep strings that must stay equal to each other equal (for example,
      `testModFile` and `want` in `requires_test.go`). Do not mutate the floor and skew literals
      (`alpha.9` to `alpha.12`), because their value is the point. Record each file as green
      (synthetic) or red (load-bearing). Verify: a recorded outcome for every row.
- [ ] 1.3 Derive candidates: revert 1.2, then mutate each *derive* row the same way
      (`render_test.go:157,206,491,526,906,992,1059`, `stage_test.go:345,475,569`). Verify: each
      one turns its package red, except possibly `stage_test.go:475`, whose outcome decides its
      row.
- [ ] 1.4 `git checkout -- opm` (every mutation reverted; `git status --short opm` prints
      nothing). Write the outcome table into design.md under D6 ("Spike result"), and move any
      row whose outcome contradicts D3 to the side it proved. Verify: the design table and the
      spike table agree.
- [ ] 1.5 `task check` green, then commit
      `chore(openspec): record which test version literals a bump must move`.

## 2. Parity harness derives the catalog build (opm/kernel; design D1)

- [ ] 2.1 `opm/kernel/parity_harness_test.go`: add `shippedCatalogVersion` (reads the
      `opmodel.dev/catalogs/opm@v4` pin from `testdata/parity/cue.mod/module.cue` with
      `cuelang.org/go/mod/modfile`) and `shippedTransformer(version, name)`. The seven
      `shippedCases` rows carry the bare transformer name. `TestParity_ShippedCatalog` resolves
      the rows once before `assertRowsCoverPairs`. Verify:
      `grep -nE '@4\.[0-9]+\.[0-9]+' opm/kernel/parity_harness_test.go` prints nothing.
- [ ] 2.2 Add the pin-equals-resolution assertion: after reading `catalogVersion` off the
      oracle, `require.Equal(t, pinned, oracleCatalog, ...)`. Its message names both versions and
      says resolution lifted the pin. It runs before the `ResolvedVersions` check and before any
      case. Update the file's header comment to say where the version comes from. Verify (a
      local negative, reverted before commit): changing the parity cue.mod pin text to `v4.4.1`
      without tidying makes the test fail with that message or with the oracle load error, never
      with "a failing oracle is a broken fixture".
- [ ] 2.3 `OPM_FLOW_TEST_FORCE=1 go test ./opm/kernel -run 'TestParity' -count=1` green against
      GHCR. Verify: `TestParity_ShippedCatalog` and its seven subtests pass,
      `TestParity_ShippedCatalogDiscriminated` passes, and the probe group passes.
- [ ] 2.4 `task check` green, then commit
      `test(kernel): derive parity transformer ids from the pinned catalog`.

## 3. Current-core literals derive from the default; frozen literals declared (registrytest, kernel, renderstage, schema; design D2, D3, D4)

- [ ] 3.1 `opm/internal/registrytest/registrytest.go`: `var DefaultCoreVersion =
      schema.DefaultSchemaVersion()`. The doc comment says it is derived. Remove the first
      (now tautological) assertion of `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` and
      keep its `testdata/render` walk. Verify: `go test ./opm/internal/registrytest -count=1`
      green.
- [ ] 3.2 `opm/kernel/render_test.go`: the six `ResolvedVersions` core rows and the line 906
      module file read `registrytest.DefaultCoreVersion`. The line 32 comment names the constant,
      not the release. `opm/internal/renderstage/stage_test.go`: lines 345 and 569 (and 475 if the
      spike marked it *derive*) likewise. Verify:
      `grep -n 'beta\.1' opm/kernel/render_test.go opm/internal/renderstage/stage_test.go` prints
      only rows the spike marked *freeze*.
- [ ] 3.3 `opm/schema/loader_test.go`: `TestDefaultSchemaModule_PinsVerifiedRelease` asserts
      shape (prefix `opmodel.dev/core@v2.`, a valid semver suffix, not the bare major), and
      `TestDefaultSchemaVersion_IsTheDefaultModulesVersion` asserts
      `DefaultSchemaModule == "opmodel.dev/core@" + DefaultSchemaVersion()` and a `v2.` prefix.
      Update both doc comments to the schema-dispatch delta (review of the bump change carries
      the deliberateness). The parse table (lines 194-195) stays. Verify:
      `go test ./opm/schema -count=1` green, and lines 160-180 hold no `beta.1`.
- [ ] 3.4 `opm/helper/platformmodule/generate_test.go` `TestGenerate_ExplicitCorePin`: the
      explicit pin becomes `v2.0.0-alpha.13` in both places (design D3, a pin that differs from
      the default). Verify: the test passes, and it also asserts
      `mf.Deps[CorePath].Version != schema.DefaultSchemaVersion()`.
- [ ] 3.5 Header comments: in the 12 `testdata/render/platform*/platform.cue` files and in
      `modules/opm_platform/platform.cue`, "core 2.0.0-beta.1" becomes "the core its cue.mod
      pins" (wrapped to the existing width). Update the `AGENTS.md` `testdata/` layout line per
      design D4. Verify: `grep -rn 'beta\.1' --include=platform.cue testdata modules` prints
      nothing; `task cue:fmt && git diff --exit-code -- '*.cue'` shows only the comment edits;
      `task cue:vet` is green.
- [ ] 3.6 New root `.cascade-frozen` in the workspace format (`frozen:` list of `path`, `pins`,
      `reason`). It has one entry per file the spike confirmed as *freeze*, at minimum
      `opm/kernel/render_collision_test.go`, `opm/kernel/render_core_floor_test.go`,
      `opm/platform/contracts_test.go`, `opm/errors/coretooold_test.go`,
      `opm/helper/platformmodule/closure_test.go` and `generate_test.go` (core and
      `opmodel.dev/catalogs/opm@v4`), `opm/internal/renderstage/modfile_test.go` (core and
      catalog), `opm/catalog/requires_test.go`, `opm/helper/objectset/objectset_test.go`
      (catalog), `opm/internal/registrytest/registrytest_test.go` and `opm/schema/loader_test.go`
      (core). Each reason is one sentence. Start the file with a comment that names its reader
      (release-cascade tooling, workspace RELEASING.md, section "Cascade files"). Verify:
      `yq '.frozen[].path' .cascade-frozen | xargs ls` lists every path, and every entry has a
      non-empty `pins` and `reason`.
- [ ] 3.7 Core-bump dry check (not committed): temporarily set `DefaultSchemaModule` to
      `opmodel.dev/core@v2.0.0-alpha.13` and text-replace `v: "v2.0.0-beta.1"` with
      `v: "v2.0.0-alpha.13"` in every `cue.mod/module.cue` under `testdata` and `modules`.
      alpha.13 is published and carries the identical schema. Run
      `go test ./opm/... -count=1 -short`, then `git checkout -- .`. Verify: green with no
      `*_test.go` edited (spec scenario "A default move edits no test file"); any failure names a
      literal that 3.2 to 3.4 missed.
- [ ] 3.8 `task check` green, then commit
      `test(fixtures): derive current-core test literals from the schema default`.

## 4. `task cue:deps:update` in one pass, core held, loud (Taskfile.yml; design D5)

- [ ] 4.1 Rewrite the `cue:deps:update` script per design D5. Read the default core from
      `opm/schema/loader.go`. Run one `cue mod get` per module over every direct dependency
      (core at the default, others at `<path>@vN`), then `cue mod tidy`. Capture output, print it
      with the module directory on failure, and exit 1. Do not use `|| true` and do not discard
      output to `/dev/null`. After tidy, fail if core is not the default, naming the moved
      dependencies whose own extracted `cue.mod/module.cue` requires a newer core. Keep the
      old-to-new report. Update `desc` and `summary`. Verify: `grep -n '|| true'` in the task
      body prints nothing.
- [ ] 4.2 Fix the stale comment on the `cue:catalog:drift` step in `.github/workflows/cue.yml`
      (existence of the pinned build, not currency). Verify: the comment matches the
      `cue:catalog:drift` summary.
- [ ] 4.3 Exercise without committing pins: (a) on the current tree, `task cue:deps:update`
      exits 0 and moves only what is newer on GHCR (expected: the catalog `v4.4.2` to `v4.4.4`),
      then `git checkout -- '*.cue'`; (b) with one module's catalog pin text-edited to an
      unpublished `v4.99.0`, the task exits non-zero and prints that module and CUE's error, then
      revert; (c) with the default read temporarily forced to `v2.0.0-alpha.13` (edit the grep
      target in a scratch copy of the script, not `loader.go`), the task exits non-zero naming
      the catalog that requires core `v2.0.0-beta.1`. Verify: the three outcomes as stated, and
      `git status --short` is clean afterwards apart from `Taskfile.yml` and `cue.yml`.
- [ ] 4.4 `task check` green, then commit
      `build: update test cue deps in one pass and fail loudly`.

## 5. Proof: bump the catalog pins to opm 4.4.4 (fixtures)

- [ ] 5.1 `task cue:deps:update`. Verify:
      `git diff --name-only | grep -v '/cue.mod/module.cue$'` prints nothing; the four modules
      (`modules/opm_platform`, `testdata/modules/web_app`, `testdata/parity`,
      `testdata/parity/opm_platform`) read `opmodel.dev/catalogs/opm@v4` `v4.4.4`; core still
      reads `v2.0.0-beta.1`. If `cue.dev/x/k8s.io` moved too, record it and keep it, unless a gate
      below fails because of it (design Risks).
- [ ] 5.2 Cross-cutting checks against GHCR: `task cue:check`, `task cue:catalog:drift`,
      `OPM_FLOW_TEST_FORCE=1 task cue:test:flow`,
      `OPM_FLOW_TEST_FORCE=1 go test ./opm/kernel -run 'TestParity|TestFlow' -count=1`. Verify:
      all green with no Go file edited. If a parity case diverges on `4.4.4`, stop: revert 5.1,
      record the divergence in design.md Risks, leave section 5 unchecked, and report. Sections 1
      to 4 stand alone.
- [ ] 5.3 `task check` green, then commit `test(fixtures): bump catalog pins to opm 4.4.4`.

## 6. Verify and archive

- [ ] 6.1 Whole-tree gates on the final tree: `task check`, `task cue:check`,
      `task cue:catalog:drift`, `OPM_FLOW_TEST_FORCE=1 task cue:test:flow`, the parity run of 5.2,
      and `go test -race ./opm/kernel ./opm/internal/renderstage -count=1`. Verify: all green.
- [ ] 6.2 `openspec validate derive-fixture-versions --strict` passes. Verify: the command
      prints that the change is valid.
- [ ] 6.3 `openspec archive derive-fixture-versions --yes`. Verify: `render-parity` gains the new
      requirement; `schema-dispatch` carries the MODIFIED text with all five original scenarios
      plus the two new ones; `openspec/specs/fixture-pin-maintenance/spec.md` exists with its
      Purpose; `openspec validate --all --strict` passes. There is no `enhancement.yaml`, so no
      delivery log runs.
- [ ] 6.4 Commit `chore(openspec): archive derive-fixture-versions`.
