# Tasks: cut-dead-surface

## 1. opm/kernel validation surface

- [ ] 1.1 `validate.go`, `source.go`: delete `ValidateConfig`, `ValidateConfigPartial`, `ValidateOption`, `Partial`, `validateConfig`; add unexported `validateSources(schema, sources, requireConcrete)` over `runValidate`; `ValidateConfigDetailed(schema, sources)` calls it with `true`; `acquire.go` extra-values pass and `attributeValuesError` call it with `false`/`true` as today.
- [ ] 1.2 `source.go`, `source_loader.go`: drop `Source.Name`; `LoadSourceFromBytes(origin, b)`; delete `LoadSourceFromString`; `LoadSourceFromFile` and `attributeValuesError` stop setting `Name`.
- [ ] 1.3 Tests: `helpers_test.go` with `mustSource(t, k, origin, src)`; rewrite the `LoadSourceFromString` call sites in `acquire_test.go`, `validate_test.go`, `source_loader_test.go`; single-value `ValidateConfig` cases become one-element `ValidateConfigDetailed` calls; partial-mode cases move to `validate_internal_test.go` (package `kernel`) against `validateSources(…, false)`; delete the two `Name` asserts.

## 2. opm/kernel instance processing and wrappers

- [ ] 2.1 `process.go`: `ProcessModuleInstance` becomes `processInstance(ctx, spec)`; delete the `ValidateConfig` call and the `FillPath(schema.Values, …)` branch; `bestEffortInstanceName(spec)` reads `schema.Metadata` only; update the two callers in `acquire.go` and `synth.go`.
- [ ] 2.2 `wrappers.go`: delete `NewInstanceFromValue`; `AcquireModuleFromRegistry` takes the `(cue.Value, *module.Source, error)` return; `acquire.go`: `ValuesFileName` becomes unexported.
- [ ] 2.3 Tests: `synth_test.go` and `kernel_test.go` cases that called `ProcessModuleInstance` run through `SynthesizeInstance` / `AcquireInstanceFromDir` (delete the zero-values no-fill case); `TestKernel_PrunedSurface` gains `ValidateConfig`, `ValidateConfigPartial`, `ProcessModuleInstance`, `NewInstanceFromValue`, `LoadSourceFromString` and its comment names `WithValues` / `SynthesizeInstance`; `acquire_test.go` uses the literal `"opm-values.cue"`.

## 3. opm/module, opm/platform, opm/schema

- [ ] 3.1 `module/module.go`, `module/instance.go`, `platform/platform.go`: delete both `CueContextOwner` interfaces and `module.NewInstanceFromValue`; `NewModuleFromValue(v)` and `NewPlatformFromValue(v)` take the value only; kernel wrappers in `wrappers.go` forward without the owner argument.
- [ ] 3.2 `schema/decode.go`: unexport the three decoders (`decodeModuleMetadata`, `decodeInstanceMetadata`, `decodePlatformMetadata`) and update their callers in `module`, `platform` and `kernel/process.go`; drop the `// Was: DecodeReleaseMetadata` breadcrumb.
- [ ] 3.3 Tests: `module/module_test.go`, `module/instance_test.go`, `platform/platform_test.go` drop the owner argument; `NewInstanceFromValue` cases move onto `NewModuleFromValue` or are deleted where kernel acquire tests already cover them; `helper/synth/instance_test.go` drops `stubOwner`.

## 4. opm/errors and opm/helper

- [ ] 4.1 `errors/match.go`: delete `UnresolvedDemand.UnstatedPosture` and its `Error()` clause; `errors/match_test.go` fixture updated. `errors/identity.go`: delete `Artifact`, `Error()` reads `identity mismatch at %s: …`; construction sites in `helper/loader/registry/module.go` and any message assertion in its tests updated.
- [ ] 4.2 `helper/loader/registry/module.go`: delete `StagedSource`; `LoadModulePackageWithSource` returns `(cue.Value, *module.Source, error)`; its tests destructure the new return.
- [ ] 4.3 `helper/platformmodule/generate.go`: delete `RootOption`, `WithCoreVersion`, `rootConfig`; `Roots(entries)` pins `schema.DefaultSchemaVersion()`; `closure_test.go` and `generate_test.go` build their pinned `[]Dep` directly.

## 5. Docs and verification

- [ ] 5.1 Grep `CLAUDE.md`, `README.md`, `docs/getting-started.md`, `opm/kernel/doc.go`, `opm/helper/synth/doc.go` for `ProcessModuleInstance`, `ValidateConfig(`, `ValidateConfigPartial`, `Partial()`, `LoadSourceFromString`, `CueContextOwner`, `NewInstanceFromValue`, `Source.Name` and rewrite each mention (values enter through `WithValues` and `SynthesizeInstance`; one validation primitive; constructors take the value).
- [ ] 5.2 Grep `../cli` and `../opm-operator` (non-test and test) for every removed name and for `identity mismatch`; confirm zero hits, then `go build ./...` in both with a temporary `replace` to the working tree and revert the replace.
- [ ] 5.3 `task fmt`, `task vet`, `task lint`, `task test` green; `go test -race ./opm/kernel/...` green.
