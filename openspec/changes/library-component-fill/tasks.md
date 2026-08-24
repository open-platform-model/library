## 1. Fill from the unstripped value (opm/compile, opm/kernel)

- [ ] 1.1 `executePair`: look the component up in `schemaComponents` and fill `#component` with it; drop the `dataComponents` lookup. Update the function comment (flow steps 2 and 3).
- [ ] 1.2 `Module.Execute`: keep the signature; document `dataComponents` as ignored and deprecated (removed with `FinalizeValue`, 0019 step 4). `Kernel.compileModuleInstance`: remove the `FinalizeValue` call and pass `schemaComponents` for both arguments; update `Kernel.Compile`'s "(Validate + Match + Execute + Finalize)" doc.

## 2. Flow fixture repair (testdata/modules/web_app, opm/kernel)

- [ ] 2.1 Add `testdata/modules/web_app/instance/instance.cue`: import-authored `#ModuleInstance` (`web-app-demo` / `default`, `#module` by import, `values` = today's `debugValues`, no `uuid`). `cue vet ./...` in the fixture module passes.
- [ ] 2.2 `flow_integration_test.go`: replace the skeleton + `FillPath` construction with `LoadInstancePackage(instance/)` + `ProcessModuleInstance`; delete the `debugValues` plumbing the skeleton needed. Assert `components.web.#names.dns.fqdn == "web.default.svc.cluster.local"` on the processed instance and that `metadata.uuid` is concrete.

## 3. Regression test (opm/kernel, hermetic)

- [ ] 3.1 Add a `registrytest`-served catalog test whose transformer emits `#component.#names.resourceName` and `.dns.fqdn`, rendered through `Kernel.Compile` against an import-authored instance; assert the concrete values (design D3).

## 4. Parity harness bookkeeping (opm/kernel)

- [ ] 4.1 Run `TestParity_Probes`; delete `ExpectedDivergence` on the `names-probe :: web` row when the harness reports it no longer reproduces.
- [ ] 4.2 Run `TestParity_ShippedCatalog`; if `worker env order is hoisted by finalization` now passes, delete that subtest, drop `divergenceFinalizeHoisting` from the worker row, and add the D14 ordering note to `MIGRATIONS.md` (`## Unreleased — Additive`). Otherwise record in `design.md` why it still reproduces.

## 5. Validation

- [ ] 5.1 `task fmt`, `task vet`, `task lint`, `task test` (includes `composed_open_test.go`, the CUE canary pair, and the full parity harness against GHCR).
- [ ] 5.2 `task cue:vet` (the `web_app` fixture module with its new `instance` package) and confirm no exported identifier under `opm/` changed.
