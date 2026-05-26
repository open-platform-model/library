## 1. New opm/schema package (additive)

- [x] 1.1 Create `opm/schema/doc.go` (package overview).
- [x] 1.2 Create `opm/schema/paths.go` exporting every CUE path as a package-level `var`.
- [x] 1.3 Create `opm/schema/metadata.go` with `ModuleMetadata`, `ReleaseMetadata`, `ProviderMetadata`, `PlatformMetadata`, `ReleaseView`.
- [x] 1.4 Create `opm/schema/decode.go` with free-function decoders.
- [x] 1.5 Create `opm/schema/context.go` with `ModuleReleaseContextData`, `ComponentContextData`, `BuildTransformerContext`.
- [x] 1.6 Create `opm/schema/schemavalue.go` with `EmbeddedSchema`, `SchemaValue`, package-level `sync.Once` cache.
- [x] 1.7 Create `opm/schema/consts.go` exporting `AnnotationDefaultNamespace`.
- [x] 1.8 `task fmt && task vet` — package builds standalone (apis/core embed pattern not yet updated; expect a build error from the embed walk if `v1alpha2/*.cue` is still the pattern — verify only `opm/schema` itself builds).

## 2. Re-sync library/apis/core/ with new core repo

- [x] 2.1 Delete `library/apis/core/v1alpha2/` (recursive, including testdata).
- [x] 2.2 Copy every `*.cue` file from `core/src/` (the new core repo's source directory) into `library/apis/core/*.cue` (flat layout, package `core`). As of 2026-05-26 the file set is: `blueprint.cue`, `catalog.cue`, `component.cue`, `helpers_autosecrets.cue`, `module.cue`, `module_context.cue`, `module_release.cue`, `platform.cue`, `resource.cue`, `schemas.cue`, `trait.cue`, `transformer.cue`, `types.cue`. `catalog.cue` and `module_context.cue` were introduced by enhancement 0001's `core/` slice (D19/D25 and D1–D4 respectively) — copy whatever is in `core/src/` at sync time rather than relying on this enumerated list. Skip `INDEX.md` (generated doc).
- [x] 2.3 Copy `core/src/cue.mod/module.cue` verbatim — the module identifier is `opmodel.dev/core@v0` per enhancement 0001 D12.
- [x] 2.4 Update `library/apis/core/embed.go`: change `//go:embed cue.mod/module.cue v1alpha2/*.cue` → `//go:embed cue.mod/module.cue *.cue`.
- [x] 2.5 Re-run schema load via a tiny throwaway main if needed; confirm `schema.SchemaValue` returns a value whose `#ModuleRelease` exists. (Build runs first.) The re-synced schema is the post-0001 `core` shape: `#Module` has `#ctx` (no `#defines`); `#Platform.#registry` is path-keyed `#Subscription`; `#FQNType` uses SemVer; `#Catalog` is a top-level definition. Consumer fixtures that still use the old shapes will fail unification — see task 12 for the quarantine + D10 in `design.md` for the scope boundary.

## 3. Rewrite opm/module to drop APIVersion + use opm/schema

- [x] 3.1 `module/module.go`: remove `APIVersion` field; replace `api.Lookup`/`b.Paths()` with `schema.X`; replace `b.DecodeModuleMetadata` with `schema.DecodeModuleMetadata`; drop `moduleMetadataFromAPI` helper (decoder now returns `*schema.ModuleMetadata` directly — keep `ModuleMetadata` as a local-typed alias or store the schema struct).
- [x] 3.2 `module/release.go`: remove `APIVersion` field; same substitutions; keep `ReleaseView` methods (now satisfying `schema.ReleaseView`); drop `ReleaseMetadataFromAPI`.
- [x] 3.3 Decide: keep `module.ModuleMetadata`/`module.ReleaseMetadata` as local Go types or alias `schema.ModuleMetadata`. (Recommendation: alias via type alias to avoid double copy.)
- [x] 3.4 Update doc strings referencing `api.Paths()`, `binding`, `apiversion`.

## 4. Rewrite opm/platform to drop APIVersion + use opm/schema

- [x] 4.1 `platform/platform.go`: remove `APIVersion` field; substitute `schema.DecodePlatformMetadata`; drop `platformMetadataFromAPI`.
- [x] 4.2 Update doc strings.

## 5. Rewrite opm/compile to drop api.Binding parameter

- [x] 5.1 `compile/match.go`: drop `binding` parameter from `Match`; drop `paths` arg from `lookupCandidates`, `candidateSatisfied`, `pairTransformer`; reference `schema.X` directly.
- [x] 5.2 `compile/execute.go`: drop `binding` parameter from `executeTransforms` and `executePair`; replace `binding.BuildTransformerContext(...)` with `schema.BuildTransformerContext(...)`.
- [x] 5.3 `compile/module.go`: drop `binding` parameter from `(*Module).Execute`; drop from `extractComponentSummaries`; update call to `executeTransforms`.
- [x] 5.4 Update doc strings, remove `api.Binding`/`api.Paths` import references.

## 6. Rewrite opm/helper/loader/file to drop apiversion.Version return

- [x] 6.1 `module.go`, `release.go`, `platform.go`: signatures collapse to `(cue.Value, error)`; drop `apiversion.Detect` calls; drop `apiversion` import.
- [x] 6.2 Confirm `shapeGate` still runs as the only validation.

## 7. Rewrite opm/helper/platform/compose.go

- [x] 7.1 Drop `api.Lookup(shell.APIVersion)`; use `schema.Registry.Selectors()` directly.
- [x] 7.2 Drop `api` import.

## 8. Rewrite opm/helper/synth/release.go

- [x] 8.1 Drop `api.Lookup(in.Module.APIVersion)` + `binding.SchemaValue(ctx)`; replace with `schema.SchemaValue(ctx)`.
- [x] 8.2 Replace `binding.Paths().Config` with `schema.Config`.
- [x] 8.3 Drop `api` import; retain `ErrSchemaUnavailable`.

## 9. Rewrite opm/kernel

- [x] 9.1 `process.go`: drop `api.Lookup(mod.APIVersion)`; use `schema.X` paths; call `schema.DecodeReleaseMetadata`; drop `APIVersion` from constructed `*module.Release`.
- [x] 9.2 `phases.go::Validate`: drop binding lookup; use `schema.Config` directly.
- [x] 9.3 `phases.go::Match`: drop apiVersion-mismatch check; drop binding param to `compile.Match`.
- [x] 9.4 `phases.go`: delete `DetectAPIVersion` method.
- [x] 9.5 `compile.go::compileModuleRelease`: drop apiVersion-mismatch check; drop binding param to `compile.Match`.
- [x] 9.6 `compile.go::moduleFromRelease`: drop `api.Lookup`; use `schema.Module` path directly.
- [x] 9.7 `wrappers.go`: drop `apiversion.Version` from return signatures; drop `apiversion` import.
- [x] 9.8 Update doc strings.

## 10. Rewrite cmd/flow-inspect/main.go

- [x] 10.1 Drop `_ "opm/api/v1alpha2"` blank import.
- [x] 10.2 Drop `api.Lookup(modVer)` and replace `paths := binding.Paths()` with direct `schema.X` references.
- [x] 10.3 Drop `modVer` variable (`LoadModulePackage` no longer returns it).
- [x] 10.4 Remove `apiVersion: "opmodel.dev/v1alpha2"` line from the inline release-skeleton CUE string.
- [x] 10.5 Update `printMatcherIndex` signature: drop `paths api.Paths` param.
- [x] 10.6 Replace `api.Paths` references in helpers with `schema` package vars; drop `api` import.

## 11. Delete opm/api and opm/apiversion packages

- [x] 11.1 `rm -rf opm/api opm/apiversion`.
- [x] 11.2 Confirm `git grep "opm/api\\|opm/apiversion" -- '*.go'` returns nothing (excluding archived openspec/changes/).
- [x] 11.3 Confirm `git grep "apiversion\\." -- '*.go'` returns nothing.

## 12. Rewrite tests

- [x] 12.1 `opm/module/module_test.go`: drop `apiVersion` literal from CUE fixtures, drop `apiversion.V1alpha2` references, drop blank `_ "opm/api/v1alpha2"` import; rewrite `TestNewModuleFromValue_UnknownAPIVersion` / `MissingAPIVersion` (likely delete — concept gone).
- [x] 12.2 `opm/module/release_test.go`: same treatment.
- [x] 12.3 `opm/platform/platform_test.go`: same treatment.
- [x] 12.4 `opm/kernel/phase_test.go`: delete `TestKernel_DetectAPIVersion` and `TestKernel_DetectAPIVersion_Unknown`.
- [x] 12.5 `opm/kernel/{kernel_test,synth_test,flow_integration_test,flow_synth_integration_test,validate_typed_test}.go`: drop blank imports + apiVersion literals; rewrite as needed.
- [x] 12.6 `opm/compile/compile_test.go`: drop binding fixtures and blank imports.
- [x] 12.7 `opm/helper/loader/file/{module,release,platform,validate}_test.go`: drop apiversion.Version expectations.
- [x] 12.8 `opm/helper/platform/compose_test.go`: drop blank import + APIVersion-on-Platform usage.
- [x] 12.9 `opm/helper/synth/release_test.go`: drop blank import + APIVersion usage.
- [x] 12.10 Verify there are no leftover references to `apiVersion: "opmodel.dev/v1alpha2"` in any `_test.go` CUE string fixtures.

## 12a. Quarantine consumer fixtures the re-sync breaks

Per D10 in `design.md`, post-0001 `core` schema breaks three consumer fixtures whose rewrites belong to enhancement 0001's library slice, not Part B. Quarantine them so the test suite stays green at the end of Part B.

- [x] 12a.1 `library/modules/opm_platform/platform.cue` uses the old `#registry: {opm: {#module: …, enabled: true}}` Module-valued shape and imports `opmodel.dev/core/v1alpha2@v1`. The new `#Platform.#registry` is path-keyed `[Path=#ModulePathType]: #Subscription`. Either: (a) delete the file and re-introduce it under enhancement 0001's library slice in the `#Subscription` shape, or (b) gate its build with a build tag the rest of the library does not set. Pick one and document the choice in the commit message.
- [x] 12a.2 `library/testdata/modules/web_app/{module,components}.cue` imports `opmodel.dev/core/v1alpha2@v1` and references the `opmodel.dev/modules/opm` catalog at MAJOR-only `@v1`. Both paths and FQN shapes are pre-0001. Quarantine the package (same options as 12a.1); enhancement 0001's library slice rewrites it against `opmodel.dev/core@v0` + the repackaged `opmodel.dev/catalogs/opm@0.1.0` (D23) once that catalog tag exists.
- [x] 12a.3 `opm/kernel/flow_integration_test.go` and `flow_synth_integration_test.go` consume the two fixtures above. Mark them `t.Skip("quarantined — see openspec/changes/remove-api-binding-dispatch design.md D10; re-enabled by enhancement 0001 library slice")` so `task test` stays clean. Do not delete — the skip leaves an obvious marker for the 0001 library-slice author.
- [x] 12a.4 Confirm `task cue:test:flow` either succeeds against the (skipped) integration tests or is itself skipped via its existing `OPM_FLOW_TEST_FORCE`-gated guard.

## 13. Validation gates

- [x] 13.1 `task fmt`.
- [x] 13.2 `task vet`.
- [x] 13.3 `task lint`.
- [x] 13.4 `task test`.
- [x] 13.5 Manual: `go run ./cmd/flow-inspect -stages module` (requires local registry per the tool's existing contract) — confirms cmd builds and at minimum loads a module.

## 14. Spec sync

- [x] 14.1 `openspec verify --change remove-api-binding-dispatch` clean.
- [x] 14.2 Archive the change (`/opsx:archive`) once `task check` passes.
