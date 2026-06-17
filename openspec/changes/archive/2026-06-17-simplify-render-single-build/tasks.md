## 0. Precondition (gating — verify before happy-path work)

- [x] 0.1 Confirm `opmodel.dev/core@v0` resolves to a published version with author-supplied `#Module` identity (`modulePath!` / `version!`, not the `metadata.modulePath` self-cycle); record the version. **MET: `core@v0.5.0`** (commit `68e4520`, *feat(module): make #Module identity author-supplied*), published to `ghcr.io/open-platform-model` and present in the warm workspace cache. Verified on the kernel-pinned toolchain (`cue v0.17.0-alpha.1`): importing a published module into `#ModuleRelease.#module` loads with concrete `#module.metadata` on `v0.5.0` and fails with `field not allowed` on `v0.4.0`. Floor for the new mechanism is `core@v0` ≥ `v0.5.0`.

## 1. loader: extract shared evaluate-and-shape-gate

- [x] 1.1 In `opm/helper/loader/file`, factor a `buildAndShapeGate(ctx, root string, overlay map[string]load.Source /*nil = FS*/, spec shape)` that builds the `load.Config`, runs `load.Instances` + `BuildInstance`, and applies the shape gate — extracted from current `LoadReleasePackage` with zero behavior change.
- [x] 1.2 Re-point `LoadReleasePackage` at `buildAndShapeGate` (FS mode); confirm existing loader tests pass unchanged.

## 2. helper/synth: virtual-package renderers

- [x] 2.1 Add `renderModuleFile(in)` producing a fabricated `cue.mod/module.cue` (deps: resolved `core` version + module `path@version`), reusing the dependency-fabrication approach from `opm/helper/loader/registry`.
- [x] 2.2 Add `renderReleaseFile(in)` producing `release.cue` that imports `core` + the module, sets `metadata.{name,namespace,labels,annotations}` (regex-constrained literal formatting), and writes `#module: <import>`.
- [x] 2.3 Add `renderValuesFile(in)` producing `values.cue` from `in.Values` via `format.Node(in.Values.Syntax(...))`; emit no file when `!in.Values.Exists()`.

## 3. helper/synth: rewrite Release onto single-build

- [x] 3.1 Rewrite `Release` to assemble the overlay (2.1–2.3) and call `buildAndShapeGate` (overlay mode); keep required-input sentinel guards unchanged.
- [x] 3.2 Delete `buildReleaseScope`, the `userModule` overlay/`cue.Scope` compile, and `preparedModule.FillPath(schema.Config, in.Values)`.
- [x] 3.3 Update `opm/helper/synth/doc.go` to describe single-build construction (no scope/pre-merge); cross-reference ADR-003.

## 4. Tests

- [x] 4.1 Unit: required-input sentinels still returned (`Module`/`Name`/`Namespace`/`SchemaCache`); zero `Values` not replaced by `debugValues`.
- [x] 4.2 Unit: derived fields from schema unchanged — `metadata.uuid` stable + namespace-divergent; `components` fanned; `opm-secrets` present iff `#Secret`; standard labels stamped.
- [x] 4.3 Unit: no `cue.Scope` compile and no `#config` `FillPath` occur in the construction path (assert via the construction routine, not internals where possible). **Done structurally:** `buildReleaseScope` and the `FillPath(#config, Values)` pre-merge are deleted from `release.go`, and `TestRelease_AutoSecretsComponentInjected` proves the values merge happens in-build via the schema's `unifiedModule` (not a Go fill) — a `#Secret` supplied via `Values` surfaces the `opm-secrets` component with no Go-side `#config` fill.
- [x] 4.4 Integration: render a **real imported module** end-to-end (construction → `Kernel.Compile`) to concrete resources, as a **positive + negative pair**. Construction pair: `TestRelease_ImportedModule_ConstructsOnAuthorSuppliedCore` (renders on `core@v0.6.0`) + `TestRelease_ImportedModule_NegativeControlV040` (fails with `field not allowed` on `v0.4.0`), plus `TestRelease_HyphenatedNameImportsBySnakeCase` for the `nameSnakeCase` address. Full Compile-to-resources: `TestFlow_ImportedModule_SynthToCompile` (kernel) publishes a module whose component carries an inline `#Resource`, materializes a platform, runs `SynthesizeRelease → Match → Compile` to a concrete `Deployment`, AND compiles an authored `release.cue` importing the same module to the **same** Compiled set (single-build parity through `Kernel.Compile`).
- [x] 4.5 Parity: `synth.Release` output and an equivalent authored `release.cue` package compile to the same resources.
- [x] 4.6 Refresh the library release fixture to the `core@v0` import shape (retire the `core/v1alpha1` fixture form used in tests).
- [x] 4.7 Integration (authored path): a `LoadReleasePackage` test that loads an on-disk `release.cue` importing a **published** module into `#ModuleRelease.#module` (no `synth` involvement) and asserts concrete `#module.metadata.{modulePath,version,fqn}`. This is the path that rotted invisibly; pin it directly. (Verified working: on `v0.5.0` loads concretely; on `v0.4.0` `LoadReleasePackage` returns an error containing `field not allowed`.)
- [x] 4.8 Test harness: `opm/internal/registrytest` hardcodes `opmodel.dev/core@v0` at `v0.3.0` in `addModules`/`addCatalogs`. Add an opt-in core-version override (e.g. a `CoreVersion` field on `ModuleFixture`, defaulting to the existing `v0.3.0` so current callers are unaffected) so 4.4/4.7 fixtures can pin `v0.5.0` and `v0.4.0`. The in-memory registry serves only the `test.example` module/catalog fixtures; `core` still resolves from the public registry / warm cache.
- [x] 4.9 Gating + pitfalls for 4.4/4.7: gate on registry reachability and `-short` exactly like `kernel`'s flow test (`skipUnlessRegistry` + `OPM_FLOW_TEST_FORCE=1`), since `core@v0` resolves from GHCR. Build fixture module paths from a **clean** string (a sanitized core-version slug), NOT `t.Name()` — subtest names contain `@`/`#`/parens which are invalid in a CUE module path. Use distinct module paths per core version so the shared CUE module cache does not shadow one fixture with another.

## 5. Records

- [x] 5.1 `MIGRATIONS.md`: internal mechanism change, behavior-preserving; note imported-module releases now construct.
- [x] 5.2 Refresh `docs/design/release-cr-imported-module-closedness.md` — mark the self-cycle root cause fixed in `core`, point at this change and ADR-003.
- [x] 5.3 Confirm ADR-003 status line references this change.

## 6. Verify

- [x] 6.1 `task fmt && task vet && task lint`
- [x] 6.2 `task test` and `task cue:test:flow` (integration; requires registry with the fixed `core`)
- [x] 6.3 Confirm `cli/` and `opm-operator/` still build (no `opm/` signature change)
