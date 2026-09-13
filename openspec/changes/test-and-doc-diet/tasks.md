## 1. Spike: the parity probe without its -short skip (opm/kernel)

- [x] 1.1 Remove the `testing.Short()` skip from `TestParity_Probes` in `opm/kernel/parity_probe_test.go` and time `go test ./opm/kernel/ -run TestParity_Probes -count=1` on the warm workspace cache; if it exceeds about five seconds, restore the skip with a message that names the measured cost, and record the number under "Retire the fill tests" in `design.md` either way.
- [x] 1.2 Delete `opm/kernel/component_fill_test.go` and `opm/kernel/instance_fill_test.go`; verify `go test ./opm/kernel/ -run 'TestParity_Probes|TestRender'` passes and `grep -rn 'names-regression' opm/kernel/*_test.go` hits only `parity_probe_test.go`.
- [x] 1.3 `task check` green, then commit `test(kernel): retire the fill tests in favour of the parity probe`.

## 2. One exact-set surface test (opm/kernel, opm/module)

- [x] 2.1 Add `TestKernel_ExportedSurface` to `opm/kernel/kernel_test.go`: collect the exported method names of `reflect.TypeOf(&kernel.Kernel{})`, drop names ending in `ForTest`, sort, and assert equality with the literal list from `design.md` (ten names); delete `TestKernel_NoFinalizeMethod`, `TestKernel_PrunedSurface` and, in `opm/kernel/registry_loader_test.go`, `TestKernel_NoLoadModuleFromRegistryMethod`, and the `MethodByName("NewModuleFromValue")` assertion in `opm/module/module_test.go`; keep `TestKernel_NoContextAccessor`; verify `go test ./opm/kernel/ ./opm/module/ -run 'Surface|NoContext|NewModuleFromValue'` passes and fails when a name is added to the literal.
- [x] 2.2 Delete `opm/kernel/integration_validate_test.go`; move the `debugValues`-satisfy-`#config` assertion from `opm/kernel/integration_live_test.go` into `TestFlow_WebApp_OnOpmPlatform` right after the module acquire, then delete `integration_live_test.go`; verify `go test ./opm/kernel/ -run 'TestFlow_WebApp|TestKernel_SynthesizeInstance'` passes (the flow test is GHCR-gated; run it with the registry reachable) and `ls opm/kernel/integration_*_test.go` lists only `integration_fixtures_test.go` and `integration_test.go`.
- [x] 2.3 `task check` green, then commit `test(kernel): pin the exported surface as one exact set`.

## 3. Fixtures from the generator and shared helpers (opm/kernel, opm/internal/schematest, opm/internal/registrytest, opm/schema)

- [x] 3.1 Rewrite `writeCatalogPlatform` in `opm/kernel/integration_fixtures_test.go` to build a `platformmodule.Input` (name `hermetic`, type `kubernetes`, the existing module path, one enabled entry for the served catalog, deps for core at `registrytest.DefaultCoreVersion` and the catalog) and write `platformmodule.Generate`'s files with `Files.WriteTo`; verify `go test ./opm/kernel/ -run TestIntegration` passes and the hand-written `module:`/`#registry` template text is gone from the file.
- [x] 3.2 Add `WriteModuleDir`, `WriteInstanceDir`, `WritePlatformDir` and `WritePkgDir` to `opm/internal/schematest` (bodies from `opm/internal/loader/load_test.go:31-52`) and replace both definitions sets (`opm/kernel/kernel_test.go:110-124`, `opm/kernel/acquire_test.go:46`, `opm/internal/loader/load_test.go:31-52`) with calls; verify `grep -rn 'func writeTemp' opm --include='*_test.go'` is empty and `go test ./opm/kernel/ ./opm/internal/loader/` pass.
- [x] 3.3 Export `registrytest.Major(version string) string`, make `coreMajor`'s callers use it, and replace `majorOf` in `opm/kernel/integration_fixtures_test.go` with it; change `opm/schema/loader_test.go:97` to load `opmodel.dev/core@v2`; verify `go test ./opm/internal/registrytest/ ./opm/kernel/ ./opm/schema/` pass and `grep -rn 'core@v1' opm --include='*_test.go'` is empty.
- [x] 3.4 `task check` green, then commit `test(kernel): build fixtures through the generator and shared helpers`.

## 4. Doc trim (opm/helper, opm/module, opm/kernel, opm/schema, opm/internal/synth, CLAUDE.md)

- [ ] 4.1 Trim `opm/helper/doc.go` per `design.md` "Doc trim" (boundary, `platformmodule`, one line per folded subpackage, `legacy:001`); delete the six `// Was: Release…` breadcrumbs (`opm/module/instance.go:21,44`, `opm/kernel/synth.go:98`, `opm/kernel/render.go:81`, `opm/schema/metadata.go:37`, `opm/internal/synth/render.go:43`); verify `grep -rn '// Was:' opm --include='*.go' | grep -v _test` is empty and `go vet ./...` passes.
- [ ] 4.2 Replace the body of CLAUDE.md "### Render contract" with a pointer to the `opm/kernel` package doc (`go doc ./opm/kernel`) and the two agent-only rules it carries (tests serve catalogs from `opm/internal/registrytest`; the parity harness is the oracle); verify the section is under fifteen lines and every rule it dropped is present in `opm/kernel/doc.go`.
- [ ] 4.3 `task check` green, then commit `chore(library): trim stale package docs and rename breadcrumbs`.
