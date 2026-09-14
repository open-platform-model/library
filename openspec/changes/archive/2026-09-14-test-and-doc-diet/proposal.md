## Why

The kernel diet's last slice. The 2026-09-05 review found about 1,500 test lines that pinned a retired schema line, repeated other tests or duplicated helpers, plus package docs describing shapes that no longer exist. Slices 01 to 03 and 6b already removed the largest items (the v1-pinned synth suite, the `registrytest` CoreVersion plumbing, the double `task test` run). What is left, re-verified on 2026-09-13, is a set of tests that repeat stronger neighbours, absence pins that grow by one name per prune, fixture text that can drift from the generator it imitates, and doc rot the reader has to route around.

## What Changes

- **Absence pins become one exact-set surface test.** `TestKernel_NoFinalizeMethod`, `TestKernel_PrunedSurface` (23 must-not-exist names) and `TestKernel_NoLoadModuleFromRegistryMethod` in `opm/kernel`, plus the `NewModuleFromValue` reflection check in `opm/module`, are replaced by one test asserting the exact exported method set of `*kernel.Kernel`. The spec scenarios they cite (`kernel-runtime` "No finalization method", `single-build-render` "Old entry points are gone", `artifact-types` "No kernel constructor wrappers") stay satisfied: an exact set proves every absence. `TestKernel_NoContextAccessor` stays; it checks signatures, not names.
- **Tests that repeat stronger neighbours go.** `component_fill_test.go` and `instance_fill_test.go` duplicate the positive half of `parity_probe_test.go`, which is hermetic; the probe's `-short` skip is a leftover from when it needed GHCR and is dropped if a spike shows the probe is fast enough. `integration_validate_test.go` repeats `synth_schema_test.go`. `integration_live_test.go`'s one assertion (the `web_app` fixture's `debugValues` satisfy its `#config`) moves into `flow_integration_test.go`, which acquires the same module under the same GHCR gate.
- **Fixtures come from the code they imitate.** `writeCatalogPlatform` calls `platformmodule.Generate` and `Files.WriteTo` instead of hand-writing the module and platform files; `writeTemp{Module,Instance,Platform}Dir`, defined in both `opm/kernel` and `opm/internal/loader` tests, move to `opm/internal/schematest`; `majorOf`, `registrytest.coreMajor` and `synth.major` collapse onto one exported `registrytest.Major` for tests (production `synth.major` stays). `schema/loader_test.go` stops resolving the retired `opmodel.dev/core@v1` line to prove major-only resolution.
- **Doc rot.** `opm/helper/doc.go` drops the "planned subpackage" and slice-process paragraphs and cites the legacy enhancement as `legacy:001`; the six `// Was: Release…` breadcrumbs from the 0002 rename go; the CLAUDE.md "Render contract" section points at the `opm/kernel` package doc instead of restating it.

**Not in this change:** the three test kernel constructors (they differ in where the registry mapping comes from); `opm/kernel/doc.go` itself (consumer godoc, kept); any non-test code path.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. Test and doc changes only; `.openspec.yaml` sets `skip_specs: true`. Every spec scenario the retired tests cited remains covered by the exact-set surface test.

## Impact

**SemVer:** none. No `opm/` signature, behaviour or dependency changes; nothing for the CLI or the controller to pick up. Commits are `test(...)` and `chore(...)`, so no release is cut.

**Packages:** `opm/kernel` (tests), `opm/module` (one test), `opm/schema` (one test), `opm/internal/schematest` and `opm/internal/registrytest` (test helpers), `opm/helper` and four packages' doc comments, `CLAUDE.md`.

**Complexity justification (Principle VII):** net negative, roughly 350 test lines and 30 doc lines removed; two small exported test helpers added in packages that exist for that purpose.
