## 1. New opm/schema package (additive)

- [ ] 1.1 Create `opm/schema/doc.go` (package overview).
- [ ] 1.2 Create `opm/schema/paths.go` exporting every CUE path as a package-level `var`.
- [ ] 1.3 Create `opm/schema/metadata.go` with `ModuleMetadata`, `ReleaseMetadata`, `ProviderMetadata`, `PlatformMetadata`, `ReleaseView`.
- [ ] 1.4 Create `opm/schema/decode.go` with free-function decoders.
- [ ] 1.5 Create `opm/schema/context.go` with `ModuleReleaseContextData`, `ComponentContextData`, `BuildTransformerContext`.
- [ ] 1.6 Create `opm/schema/schemavalue.go` with `EmbeddedSchema`, `SchemaValue`, package-level `sync.Once` cache.
- [ ] 1.7 Create `opm/schema/consts.go` exporting `AnnotationDefaultNamespace`.
- [ ] 1.8 `task fmt && task vet` — package builds standalone (apis/core embed pattern not yet updated; expect a build error from the embed walk if `v1alpha2/*.cue` is still the pattern — verify only `opm/schema` itself builds).

## 2. Re-sync library/apis/core/ with new core repo

- [ ] 2.1 Delete `library/apis/core/v1alpha2/` (recursive, including testdata).
- [ ] 2.2 Copy `core/{blueprint,component,helpers_autosecrets,module,module_release,platform,resource,schemas,trait,transformer,types}.cue` from new core repo into `library/apis/core/*.cue` (flat layout, package `core`).
- [ ] 2.3 Update `library/apis/core/cue.mod/module.cue` to match new core repo's module name and language version.
- [ ] 2.4 Update `library/apis/core/embed.go`: change `//go:embed cue.mod/module.cue v1alpha2/*.cue` → `//go:embed cue.mod/module.cue *.cue`.
- [ ] 2.5 Re-run schema load via a tiny throwaway main if needed; confirm `schema.SchemaValue` returns a value whose `#ModuleRelease` exists. (Build runs first.)

## 3. Rewrite opm/module to drop APIVersion + use opm/schema

- [ ] 3.1 `module/module.go`: remove `APIVersion` field; replace `api.Lookup`/`b.Paths()` with `schema.X`; replace `b.DecodeModuleMetadata` with `schema.DecodeModuleMetadata`; drop `moduleMetadataFromAPI` helper (decoder now returns `*schema.ModuleMetadata` directly — keep `ModuleMetadata` as a local-typed alias or store the schema struct).
- [ ] 3.2 `module/release.go`: remove `APIVersion` field; same substitutions; keep `ReleaseView` methods (now satisfying `schema.ReleaseView`); drop `ReleaseMetadataFromAPI`.
- [ ] 3.3 Decide: keep `module.ModuleMetadata`/`module.ReleaseMetadata` as local Go types or alias `schema.ModuleMetadata`. (Recommendation: alias via type alias to avoid double copy.)
- [ ] 3.4 Update doc strings referencing `api.Paths()`, `binding`, `apiversion`.

## 4. Rewrite opm/platform to drop APIVersion + use opm/schema

- [ ] 4.1 `platform/platform.go`: remove `APIVersion` field; substitute `schema.DecodePlatformMetadata`; drop `platformMetadataFromAPI`.
- [ ] 4.2 Update doc strings.

## 5. Rewrite opm/compile to drop api.Binding parameter

- [ ] 5.1 `compile/match.go`: drop `binding` parameter from `Match`; drop `paths` arg from `lookupCandidates`, `candidateSatisfied`, `pairTransformer`; reference `schema.X` directly.
- [ ] 5.2 `compile/execute.go`: drop `binding` parameter from `executeTransforms` and `executePair`; replace `binding.BuildTransformerContext(...)` with `schema.BuildTransformerContext(...)`.
- [ ] 5.3 `compile/module.go`: drop `binding` parameter from `(*Module).Execute`; drop from `extractComponentSummaries`; update call to `executeTransforms`.
- [ ] 5.4 Update doc strings, remove `api.Binding`/`api.Paths` import references.

## 6. Rewrite opm/helper/loader/file to drop apiversion.Version return

- [ ] 6.1 `module.go`, `release.go`, `platform.go`: signatures collapse to `(cue.Value, error)`; drop `apiversion.Detect` calls; drop `apiversion` import.
- [ ] 6.2 Confirm `shapeGate` still runs as the only validation.

## 7. Rewrite opm/helper/platform/compose.go

- [ ] 7.1 Drop `api.Lookup(shell.APIVersion)`; use `schema.Registry.Selectors()` directly.
- [ ] 7.2 Drop `api` import.

## 8. Rewrite opm/helper/synth/release.go

- [ ] 8.1 Drop `api.Lookup(in.Module.APIVersion)` + `binding.SchemaValue(ctx)`; replace with `schema.SchemaValue(ctx)`.
- [ ] 8.2 Replace `binding.Paths().Config` with `schema.Config`.
- [ ] 8.3 Drop `api` import; retain `ErrSchemaUnavailable`.

## 9. Rewrite opm/kernel

- [ ] 9.1 `process.go`: drop `api.Lookup(mod.APIVersion)`; use `schema.X` paths; call `schema.DecodeReleaseMetadata`; drop `APIVersion` from constructed `*module.Release`.
- [ ] 9.2 `phases.go::Validate`: drop binding lookup; use `schema.Config` directly.
- [ ] 9.3 `phases.go::Match`: drop apiVersion-mismatch check; drop binding param to `compile.Match`.
- [ ] 9.4 `phases.go`: delete `DetectAPIVersion` method.
- [ ] 9.5 `compile.go::compileModuleRelease`: drop apiVersion-mismatch check; drop binding param to `compile.Match`.
- [ ] 9.6 `compile.go::moduleFromRelease`: drop `api.Lookup`; use `schema.Module` path directly.
- [ ] 9.7 `wrappers.go`: drop `apiversion.Version` from return signatures; drop `apiversion` import.
- [ ] 9.8 Update doc strings.

## 10. Rewrite cmd/flow-inspect/main.go

- [ ] 10.1 Drop `_ "opm/api/v1alpha2"` blank import.
- [ ] 10.2 Drop `api.Lookup(modVer)` and replace `paths := binding.Paths()` with direct `schema.X` references.
- [ ] 10.3 Drop `modVer` variable (`LoadModulePackage` no longer returns it).
- [ ] 10.4 Remove `apiVersion: "opmodel.dev/v1alpha2"` line from the inline release-skeleton CUE string.
- [ ] 10.5 Update `printMatcherIndex` signature: drop `paths api.Paths` param.
- [ ] 10.6 Replace `api.Paths` references in helpers with `schema` package vars; drop `api` import.

## 11. Delete opm/api and opm/apiversion packages

- [ ] 11.1 `rm -rf opm/api opm/apiversion`.
- [ ] 11.2 Confirm `git grep "opm/api\\|opm/apiversion" -- '*.go'` returns nothing (excluding archived openspec/changes/).
- [ ] 11.3 Confirm `git grep "apiversion\\." -- '*.go'` returns nothing.

## 12. Rewrite tests

- [ ] 12.1 `opm/module/module_test.go`: drop `apiVersion` literal from CUE fixtures, drop `apiversion.V1alpha2` references, drop blank `_ "opm/api/v1alpha2"` import; rewrite `TestNewModuleFromValue_UnknownAPIVersion` / `MissingAPIVersion` (likely delete — concept gone).
- [ ] 12.2 `opm/module/release_test.go`: same treatment.
- [ ] 12.3 `opm/platform/platform_test.go`: same treatment.
- [ ] 12.4 `opm/kernel/phase_test.go`: delete `TestKernel_DetectAPIVersion` and `TestKernel_DetectAPIVersion_Unknown`.
- [ ] 12.5 `opm/kernel/{kernel_test,synth_test,flow_integration_test,flow_synth_integration_test,validate_typed_test}.go`: drop blank imports + apiVersion literals; rewrite as needed.
- [ ] 12.6 `opm/compile/compile_test.go`: drop binding fixtures and blank imports.
- [ ] 12.7 `opm/helper/loader/file/{module,release,platform,validate}_test.go`: drop apiversion.Version expectations.
- [ ] 12.8 `opm/helper/platform/compose_test.go`: drop blank import + APIVersion-on-Platform usage.
- [ ] 12.9 `opm/helper/synth/release_test.go`: drop blank import + APIVersion usage.
- [ ] 12.10 Verify there are no leftover references to `apiVersion: "opmodel.dev/v1alpha2"` in any `_test.go` CUE string fixtures.

## 13. Validation gates

- [ ] 13.1 `task fmt`.
- [ ] 13.2 `task vet`.
- [ ] 13.3 `task lint`.
- [ ] 13.4 `task test`.
- [ ] 13.5 Manual: `go run ./cmd/flow-inspect -stages module` (requires local registry per the tool's existing contract) — confirms cmd builds and at minimum loads a module.

## 14. Spec sync

- [ ] 14.1 `openspec verify --change remove-api-binding-dispatch` clean.
- [ ] 14.2 Archive the change (`/opsx:archive`) once `task check` passes.
