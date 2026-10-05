## 1. Measure the base and pin the fail-closed boundary

Library tests run with an absolute private `TMPDIR`
(`export TMPDIR=$(mktemp -d -p <session scratchpad>)`). memprobe is a scratchpad copy of
`claude-stuff/kernel-plan-beta1/memprobe` whose `go.mod` `replace` names this worktree. Every run
uses `CASES="r1-nil r2-nil" RUNS=5 MODULE=fixtures/cert_manager ./run.sh <label>`, the setup g4
used (archived `2026-10-05-drop-failed-pairs-from-render-glue` design.md, "Measurements"). None of
it is committed. Every commit task stages the files it names with `git add <file>`.

- [ ] 1.1 Before any code edit, run memprobe at the base and record the medians of `render_peak_heap`, `render_peak_live`, `vmhwm` and `user_s` in design.md under "Measurements / Before", together with the base commit.
- [ ] 1.2 Spike (design D3): add `testdata/render/scenarios/no_resources`, a plain CUE package (not a core `#ModuleInstance`) whose `components.web` carries `metadata.name` and no `#resources`. Add `TestRender_ComponentWithoutResourcesRefuses` in `opm/kernel/render_test.go`: build a `*module.Instance` struct literal with `Metadata.Name` set and `Source` naming that package on disk, render it on `platform`, and assert that `Render` returns a non-nil error that is not a `*RenderError` and a nil result. It passes at the base, because the matcher reads `#resources` unguarded. If the package cannot be staged, record why in design.md D3 and assert the same on `RenderForTest`'s built value instead.
- [ ] 1.3 Add a row for `no_resources` to `testdata/render/scenarios/README.md`.
- [ ] 1.4 `task check` green, then commit `test(kernel): pin render refusal on a component without resources`.

## 2. render: report every contract a render requires

- [ ] 2.1 `opm/internal/renderstage/render.cue.tmpl`: add the hidden `_demand` struct and `diagnostics.requiredContracts` (design D1), with the comment from D1. `#resources` is read unguarded, and `#traits` behind `!= _|_`.
- [ ] 2.2 `opm/kernel/render_decode.go`: add `RequiredContracts []string` (json `requiredContracts`) to `glueDiagnostics`. Copy it onto `RenderDiagnostics` in `decodeRenderDiagnostics`, and normalise nil to `[]string{}`.
- [ ] 2.3 `opm/kernel/render.go`: add `RenderDiagnostics.RequiredContracts` with the godoc from design D2. Also say there that the field is absent on a plain error and that an errored `#traits` reads as no traits, as it does for matching. `opm/kernel/doc.go`: one sentence in the diagnostics paragraph naming the field as the instance's contract demand.
- [ ] 2.4 Add `testdata/render/scenarios/empty`, an instance of a module with no components, and its README row. Add `opm/kernel/render_demand_test.go` with these tests:
  - happy path: resources and traits, sorted, with a key shared by two components listed once;
  - a component with no `#traits`;
  - `empty`: success, `RequiredContracts` non-nil and of length 0;
  - `missing`: the field is set on the `*RenderError`;
  - `unprovided` under `SkipUnprovided`: the omitted `ledger` component's keys are listed.
- [ ] 2.5 In the same file, add the parity test from design D4. `walkDeclaredContracts` is copied from opm-operator `internal/render/demand.go` at its `origin/main`, with the commit cited in a comment. The test compares the helper's result with `RequiredContracts` for every scenario package, the happy-path instance and the parity harness instances, and asserts the exclusion list (`unstated`, `no_resources`).
- [ ] 2.6 `opm/kernel/render_glue_shape_test.go`: through `RenderForTest`, assert that `diagnostics.requiredContracts` exists on the built value and is a concrete list.
- [ ] 2.7 `task check` green, then commit `feat(render): report every contract a render requires`.

## 3. module: instance module metadata, values and module debugValues accessors

- [ ] 3.1 `opm/module/instance.go`: add `ModuleMetadata()` (design D6, using `decodeModuleMetadata` on `Package.LookupPath(schema.Module)`) and `Values()` (design D7), each with godoc. Update the `Instance` doc comment so that it no longer says every read goes through `Package.LookupPath` alone.
- [ ] 3.2 `opm/module/module.go`: add `DebugValues()` (design D7). Change the package doc's debug-overlay paragraph to say the overlay is read with `Module.DebugValues()`.
- [ ] 3.3 `opm/schema/paths.go`: the `DebugValues` comment says frontends read it through `Module.DebugValues()`. The path itself stays (SD1: cli reads it at `origin/main`).
- [ ] 3.4 Tests in `opm/module/instance_test.go` and `opm/module/module_test.go`:
  - `ModuleMetadata()` on a well-formed instance (Name, Version, ModulePath, FQN and UUID equal the embedded `#module.metadata`), with no `#module` (nil), with undecodable metadata (nil), and on a nil receiver (nil, no panic);
  - `Values()` on a well-formed instance (equal to `Package.LookupPath(schema.Values)`), with no `values`, and on a nil receiver;
  - `DebugValues()` present, absent (`Exists() == false`) and on a nil receiver.
  In `opm/kernel`, add one acquisition test that `SynthesizeInstance` and `AcquireInstanceFromDir` instances both return a non-nil `ModuleMetadata()` whose `Name` matches the module.
- [ ] 3.5 `opm/kernel/flow_integration_test.go:66`: read `mod.DebugValues()` in place of the raw lookup.
- [ ] 3.6 `README.md:32`: the migration line reads "read `mod.DebugValues()`" in place of `mod.Package.LookupPath(schema.DebugValues)` (SD14).
- [ ] 3.7 `task check` green, then commit `feat(module): add instance module metadata, values and module debugValues accessors`.

## 4. Measure after and run the full suite

- [ ] 4.1 Re-run memprobe at the head with the same knobs. Fill in "Measurements / After" in design.md in the same table shape, with the ratio for each column. If cert_manager `r1-nil` `render_peak_heap` or `user_s` rose by more than 5% over the base, stop and report it to the supervisor.
- [ ] 4.2 Run the full non-short suite with the network tests forced: `OPM_FLOW_TEST_FORCE=1 go test -race ./opm/kernel/... ./opm/internal/renderstage/...`, then `OPM_FLOW_TEST_FORCE=1 go test ./...`. `TestParity_*`, `TestRender_InventoryParity` and the flow test must run, not skip. Record the result in design.md "Verification".
- [ ] 4.3 Run the consumer build locally against a fresh clone of cli `main` and of opm-operator `main` (`GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> . <scratch work dir>`). Both must build and vet. Record the consumer commits in design.md "Verification". Then run `task api:diff`: it must list no incompatible change charged to this branch.
- [ ] 4.4 `task check` green, then commit `chore(openspec): record expose-render-demand-and-instance-reads measurements`.

## 5. Verify and archive

- [ ] 5.1 Run `openspec verify` for `expose-render-demand-and-instance-reads` (the repo's openspec-verify-change skill). Verify: no CRITICAL finding.
- [ ] 5.2 At PR time, not in the implement stage: run `openspec archive expose-render-demand-and-instance-reads --yes`. Verify the following:
  - the main `single-build-render` spec carries the new requirement;
  - `artifact-types` carries "Module and platform constructors from cue.Value" without the duplicate scenario, the renamed instance-accessor requirement, the modified debugValues scenario and "Module exposes its debug values";
  - `openspec validate --specs --strict` passes.
- [ ] 5.3 Run the gates green, then commit `chore(openspec): archive expose-render-demand-and-instance-reads` (the archive rides the implementing PR).
