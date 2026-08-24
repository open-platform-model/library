## 1. Comparator (opm/kernel, test helpers)

- [x] 1.1 Write the order-preserving encoder helper and its negative test: two `cue.Value`s with the same fields in different order MUST encode differently and the diff MUST report the first differing path (design D4). Nothing else is built until this passes.
- [x] 1.2 Add the `parityCase` row struct mirroring `#ParityCase` (`Name`, `Instance`, `Component`, `Transformer`, `Equality`, `ExpectedDivergence`) and the per-row assertion: agree when empty, diverge-as-named when set, fail with a "delete the entry" message when a recorded divergence stops reproducing (design D5).

## 2. Shipped-catalog fixtures (testdata/parity)

- [x] 2.1 Create `testdata/parity/cue.mod/module.cue` (`testing.opmodel.dev/library-parity@v0`, pins core `v2.0.0-alpha.4` and catalogs/opm `v2.0.0-alpha.3`), and copy `web_app/` and `opm_platform/` in with source-naming headers (design D2).
- [x] 2.2 Author `instance/instance.cue` (`#module: web_app` by import, `web-app-demo`/`default`, values from the experiment) and `oracle/render.cue` from experiment 01's glue; `cue vet -c ./oracle` exits 0 against GHCR with the workspace `CUE_REGISTRY` mapping.

## 3. Shipped-catalog cases (opm/kernel/parity_harness_test.go)

- [x] 3.1 Kernel side: load `web_app`, `opm_platform`, materialize, `LoadInstancePackage(instance/)`, `ProcessModuleInstance`, `Compile`; gate with `-short` skip, `skipUnlessRegistry`, `OPM_FLOW_TEST_FORCE` (design D1).
- [x] 3.2 Oracle side: `cue/load` the `oracle` package with the same registry mapping, read `pairs` and `rendered`.
- [x] 3.3 Assert pair sets equal (always-unify refusal reported as the single exemption), then run every `web_app` pair as an `output-fields-only` row; record in the table comments whether ordering diverged on this fixture (design D3, D14 note).

## 4. Probe cases (opm/kernel/parity_harness_test.go, registrytest)

- [x] 4.1 Build the probe catalog with `registrytest` at a `UniquePath`: `names-probe` (emits `#component.#names.dns.fqdn` and `resourceName`) and `instance-probe` (emits `#moduleInstance.metadata.name`), plus a probe platform subscribing to it.
- [x] 4.2 Write the probe oracle module into `t.TempDir()` (same glue, catalog import swapped, `cue.mod` pinned to the served version) and run both renderers through the shared registry mapping.
- [x] 4.3 Add the two rows with `ExpectedDivergence` set (`FinalizeValue strips #names from #component`, `#moduleInstance is never filled`) and confirm each reproduces as named while the oracle renders concretely.

## 5. Validation

- [x] 5.1 `task fmt`, `task vet`, `task lint`, `task test` (harness included; existing suite unchanged; no exported identifier added under `opm/`).
- [x] 5.2 Add `testdata/parity` to the Taskfile `cue vet` coverage alongside the existing testdata modules, and note the harness in `CLAUDE.md`'s test inventory if one is kept there.
