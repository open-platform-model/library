# Tasks — rename-release-to-instance (enhancement 0002 / L1)

> One atomic PR on a dedicated branch (`branch-per-change`). A pure rename has no separately-mergeable intermediate state, so "independently testable" means at the group/gate level, not per checkbox. Order matters: the gate (§1) is blocking, then pin core (§2), then rename inward-out, then verify, then publish.

## 1. Pre-flight gate (BLOCKING)

- [x] 1.1 Create the branch off `main` once `opmodel.dev/core@v1` `v1.0.0-alpha.1` is confirmed published. → branch `rename-release-to-instance` off `main`; core tags `v1.0.0-alpha.1` present on ghcr.
- [x] 1.2 Gate check. **PASSED:** a `core@v1`-importing fixture (declaring `language: version: "v0.17.0-alpha.1"`) resolves and `cue vet`s under the `v0.17.0-alpha.1` toolchain — no language-floor block. `#ModuleInstance` exists (`kind` defaults `"ModuleInstance"`); `#ModuleRelease` is `undefined field` (hard rename, no alias). NOTE: resolves from PublicRegistry (ghcr), not the workspace `localhost:5000` mapping — test tasks (§5) need core@v1 reachable.

## 2. Axis 2 — pin core@v1 + CUE language version

- [x] 2.1 `opm/schema/loader.go`: `DefaultSchemaModule "opmodel.dev/core@v0"` → `@v1` (+ `// Was:` breadcrumb); update doc comment in `loader.go:52`.
- [x] 2.2 `opm/schema/doc.go`, `opm/kernel/kernel.go`, `opm/kernel/doc.go`, `opm/materialize/doc.go`, `opm/internal/schematest/schematest.go`: doc-comment `core@v0` → `@v1` references.
- [x] 2.3 Bump CUE `language: version:` to `v0.17.0-alpha.1`: `opm/helper/synth/render.go` (`synthLanguageVersion`), `opm/internal/registrytest/registrytest.go` (×2, currently `v0.16.0`), and the non-frozen fixture `cue.mod/module.cue` files under `testdata/`, `modules/opm_platform/`, `testdata/modules/web_app/`, `docs/design/repro-hidden-field/`. **Do not touch** `enhancements/004` and `006` experiments.
- [x] 2.4 Migrate every test-fixture `import core "opmodel.dev/core@v0"` → `@v1` and inline `deps: "opmodel.dev/core@v0"` generators in `registrytest.go` (verify each still resolves).

## 3. Axis 1+3 — rename the kernel surface (inward-out)

- [x] 3.1 `opm/core`: `Resource.Release()` → `Instance()` (`resource.go`), `Compiled.Release` field → `Instance` (`compiled.go`); `// Was:` breadcrumbs.
- [x] 3.2 `opm/schema`: `ReleaseMetadata` → `InstanceMetadata` (`metadata.go`), `ReleaseView` → `InstanceView` + its `ReleaseName/ReleaseUUID` methods → `InstanceName/InstanceUUID`, `DecodeReleaseMetadata` → `DecodeInstanceMetadata` (`decode.go`), `ModuleReleaseContextData` → `ModuleInstanceContextData` (`context.go`).
- [x] 3.3 **Wire (Axis 3)** — `opm/schema/paths.go`: `ContextModuleReleaseMetadata` → `ContextModuleInstanceMetadata` with CUE def `moduleReleaseMetadata` → `moduleInstanceMetadata`; `opm/schema/context.go:92` `FillPath("#moduleReleaseMetadata")` → `"#moduleInstanceMetadata"`. These MUST match core@v1.
- [x] 3.4 `opm/module`: `git mv release.go instance.go` (+ `release_test.go` → `instance_test.go`); `type Release` → `Instance`, `ReleaseMetadata` alias → `InstanceMetadata`, methods `ReleaseName/ReleaseUUID` → `InstanceName/InstanceUUID`, `NewReleaseFromValue` → `NewInstanceFromValue`; `// Was:` breadcrumbs. (`Namespace`, `ModuleFQN`, `ModuleVersion`, `Labels`, `Annotations`, `ConfigSchema`, `MatchComponents` keep their names.)
- [x] 3.5 `opm/helper/loader/file`: `git mv release.go instance.go` (+ test); `LoadReleasePackage` → `LoadInstancePackage`; **Axis 3** `opm/helper/loader/internal/shape/shape.go` `ExpectedKind "ModuleRelease"` → `"ModuleInstance"`.
- [x] 3.6 `opm/helper/synth`: `git mv release.go instance.go` (+ `release_test.go`, `release_integration_test.go`); `func Release` → `Instance`, `ReleaseInput` → `InstanceInput`, `renderReleaseFile` → `renderInstanceFile`; **Axis 3** the `module-release.opmodel.dev/{name,uuid}` assertions in `release_integration_test.go` → `module-instance.opmodel.dev/...`.
- [x] 3.7 `opm/kernel`: `ProcessModuleRelease` → `ProcessModuleInstance` (`process.go`), `SynthesizeRelease` → `SynthesizeInstance` (`synth.go`), `LoadReleasePackage`/`NewReleaseFromValue` wrappers (`wrappers.go`), `ValidateReleaseValues*` → `ValidateInstanceValues*` (`validate_typed.go`), `compileModuleRelease` (`compile.go`), `moduleFromRelease`/`releaseDisplayName`/`bestEffortReleaseName` helpers (`phases.go`, `process.go`); input-struct field `ModuleRelease *module.Release` → `ModuleInstance *module.Instance` (`inputs.go`, `phases.go`); `// Was:` breadcrumbs on every exported identifier.
- [x] 3.8 `opm/compile`, `opm/errors`: release-context references through `execute.go`, `module.go`, `match.go`, `match.go` error strings.
- [x] 3.9 `cmd/flow-inspect/main.go`: kind literal `"ModuleRelease"` → `"ModuleInstance"` and release-named locals.

## 4. Fixtures, docs, breadcrumbs

- [x] 4.1 Sweep `**/*_test.go` + the ~24 kind fixtures: kind literals `"ModuleRelease"` → `"ModuleInstance"`, type/method references, label assertions; `git mv` any remaining `*release*` test files.
- [x] 4.2 `MIGRATIONS.md`: append an entry recording the `Release` → `Instance` rename recipe (type/method/func table). Leave historical software-release prose untouched.
- [x] 4.3 `README.md`, `CLAUDE.md`, `docs/**`: update artifact-table and prose references (`#ModuleRelease`/`module.Release`/`ProcessModuleRelease`); add "Renamed from …" notes where a doc section names the old construct (D12).

## 5. Validation gates

- [x] 5.1 `task fmt` — formatted.
- [x] 5.2 `task vet` — passes.
- [x] 5.3 `task lint` — golangci-lint clean.
- [x] 5.4 `task test` — unit tests pass.
- [x] 5.5 `task cue:test:flow` — plan→match→compile integration green against core@v1 (exercises the Axis 3 wire path end-to-end; this is what catches a missed `#moduleInstanceMetadata` / `ExpectedKind`).
- [x] 5.6 **Residual-literal gate**: grep for `ModuleRelease`, `moduleRelease`, `module-release`, `\.Release\b`, `ReleaseMetadata`, `ReleaseView`, `synth\.Release`, `LoadReleasePackage` across `opm/` + `cmd/`, excluding the software-release allowlist (`CHANGELOG.md`, `release-please*`, `MIGRATIONS.md` history, `TestPublicRegistry_Value`). Must be empty.

## 6. Archive & publish

- [ ] 6.1 `openspec validate rename-release-to-instance --strict` (re-run) → `openspec-verify-change`.
- [ ] 6.2 Bulk-archive the six spec deltas (`openspec-bulk-archive-change`) — including the `release-synthesis` → `instance-synthesis` capability dir move.
- [ ] 6.3 Conventional Commit(s) (`feat(module)!: rename Release artifact family to Instance`); open the atomic PR referencing enhancement `0002` / slice L1.
- [ ] 6.4 On merge, publish the `v1.0.0-alpha.N` library tag (release-please); record the slice as a `history` event in `enhancements/0002/config.yaml` (`slice: "library#<PR>"`). This tag unblocks O*/X*.
