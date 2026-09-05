# Tasks: cut-dead-surface

## 1. opm/kernel validation surface

- [x] 1.1 `validate.go`, `source.go`: delete `ValidateConfig`, `ValidateConfigPartial`, `ValidateOption`, `Partial`, `validateConfig`; add unexported `validateSources(schema, sources, requireConcrete)` over `runValidate`; `ValidateConfigDetailed(schema, sources)` calls it with `true`; `acquire.go` extra-values pass and `attributeValuesError` call it with `false`/`true` as today.
- [x] 1.2 `source.go`, `source_loader.go`: drop `Source.Name`; `LoadSourceFromBytes(origin, b)`; delete `LoadSourceFromString`; `LoadSourceFromFile` and `attributeValuesError` stop setting `Name`.
- [x] 1.3 Tests: `helpers_test.go` with `mustSource(t, k, origin, src)`; rewrite the `LoadSourceFromString` call sites in `acquire_test.go`, `validate_test.go`, `source_loader_test.go`; single-value `ValidateConfig` cases become one-element `ValidateConfigDetailed` calls; partial-mode cases move to `validate_internal_test.go` (package `kernel`) against `validateSources(…, false)`; delete the two `Name` asserts.

## 2. opm/kernel instance processing and wrappers

- [x] 2.1 `process.go`: `ProcessModuleInstance` becomes `processInstance(ctx, spec)`; delete the `ValidateConfig` call and the `FillPath(schema.Values, …)` branch; `bestEffortInstanceName(spec)` reads `schema.Metadata` only; update the two callers in `acquire.go` and `synth.go`.
- [x] 2.2 `wrappers.go`: delete `NewInstanceFromValue`; `AcquireModuleFromRegistry` takes the `(cue.Value, *module.Source, error)` return; `acquire.go`: `ValuesFileName` becomes unexported.
- [x] 2.3 Tests: `synth_test.go` and `kernel_test.go` cases that called `ProcessModuleInstance` run through `SynthesizeInstance` / `AcquireInstanceFromDir` (delete the zero-values no-fill case); `TestKernel_PrunedSurface` gains `ValidateConfig`, `ValidateConfigPartial`, `ProcessModuleInstance`, `NewInstanceFromValue`, `LoadSourceFromString` and its comment names `WithValues` / `SynthesizeInstance`; `acquire_test.go` uses the literal `"opm-values.cue"`.

## 3. opm/module, opm/platform, opm/schema

- [x] 3.1 `module/module.go`, `module/instance.go`, `platform/platform.go`: delete both `CueContextOwner` interfaces and `module.NewInstanceFromValue`; `NewModuleFromValue(v)` and `NewPlatformFromValue(v)` take the value only; kernel wrappers in `wrappers.go` forward without the owner argument.
- [x] 3.2 `schema/decode.go`: delete it; each decoder moves, unexported, beside its one caller (`decodeModuleMetadata` in `module/module.go`, `decodePlatformMetadata` in `platform/platform.go`, `decodeInstanceMetadata` in `kernel/process.go`, since Go cannot call an unexported `opm/schema` function from another package); drop the `// Was: DecodeReleaseMetadata` breadcrumb; `schema/doc.go` and `metadata.go` stop naming the decoders.
- [x] 3.3 Tests: `module/module_test.go`, `module/instance_test.go`, `platform/platform_test.go` drop the owner argument; `NewInstanceFromValue` cases move onto `NewModuleFromValue` or are deleted where kernel acquire tests already cover them; `helper/synth/instance_test.go` drops `stubOwner`.

## 4. opm/errors and opm/helper

- [x] 4.1 `errors/match.go`: delete `UnresolvedDemand.UnstatedPosture` and its `Error()` clause; `errors/match_test.go` fixture updated. `errors/identity.go`: delete `Artifact`, `Error()` reads `identity mismatch at %s: …`; construction sites in `helper/loader/registry/module.go` and any message assertion in its tests updated.
- [x] 4.2 `helper/loader/registry/module.go`: delete `StagedSource`; `LoadModulePackageWithSource` returns `(cue.Value, *module.Source, error)`; its tests destructure the new return.
- [x] 4.3 `helper/platformmodule/generate.go`: delete `RootOption`, `WithCoreVersion`, `rootConfig`; `Roots(entries)` pins `schema.DefaultSchemaVersion()`; `closure_test.go` and `generate_test.go` build their pinned `[]Dep` directly.

## 5. Docs and verification

- [x] 5.1 Grep `CLAUDE.md`, `README.md`, `docs/getting-started.md`, `opm/kernel/doc.go`, `opm/helper/synth/doc.go` for `ProcessModuleInstance`, `ValidateConfig(`, `ValidateConfigPartial`, `Partial()`, `LoadSourceFromString`, `CueContextOwner`, `NewInstanceFromValue`, `Source.Name` and rewrite each mention (values enter through `WithValues` and `SynthesizeInstance`; one validation primitive; constructors take the value).
- [x] 5.2 Grep `../cli` and `../opm-operator` (non-test and test) for every removed name and for `identity mismatch`; confirm zero hits, then `go build ./...` in both with a temporary `replace` to the working tree and revert the replace. Result: zero non-test hits; one test literal, `opm-operator/internal/reconcile/resolution_test.go` sets `IdentityError.Artifact` (its `go vet` fails against the working tree; `go build ./...` is green in both). Built through a scratch `-modfile` copy, so neither consumer's `go.mod` was touched.
- [x] 5.3 `task fmt`, `task vet`, `task lint`, `task test` green; `go test -race ./opm/kernel/...` green.

## 6. Verify-pass cleanups

- [x] 6.1 `validate.go`: fold `runValidate` and `appendSchemaErrors` (each single-caller once the variants left, and the latter's bool return was never read) into `validateSources`; `walkDisallowed` stays separate (it recurses).
- [x] 6.2 `process.go`: `processInstance(spec)` is a free function, since it reads neither the Kernel nor a context; callers in `acquire.go` and `synth.go` updated.
- [x] 6.3 Tests: delete `TestIntegration_Validate` and its `buildModule` helper (every case is in `validate_test.go`), `source_test.go` (the `Source.Name` absence is pinned in `TestKernel_PrunedSurface`), and the two `kernel_test.go` cases that duplicated `acquire_test.go` / `validate_test.go`.
- [x] 6.4 Docs: `README.md`, `opm/schema/doc.go` and `.claude/skills/security-audit/SKILL.md` stop naming the deleted decoders file; one `acquire_test.go` comment reworded.
