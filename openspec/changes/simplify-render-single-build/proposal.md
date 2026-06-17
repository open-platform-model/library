## Why

The kernel has two entry points that turn inputs into a `#ModuleRelease` and feed the same `Kernel.Compile` pipeline:

- **`ModuleRelease` (Go-synthesized)** — `synth.Release` builds the release value from typed inputs + a registry-loaded `*module.Module`.
- **`Release` (authored package)** — `LoadReleasePackage` evaluates an author-written `release.cue` verbatim.

They diverge at exactly one box — *how the `#ModuleRelease` value is constructed* — and that divergence is a defect, not a design choice. `synth.Release` constructs the value by stitching pieces from **separate CUE evaluations** in Go, which forced two workarounds onto it (see ADR-003):

1. a `cue.Scope` / `userModule` trick to avoid unifying a closed `#Module` into `#ModuleRelease.#module`, and
2. a Go-side pre-merge of caller `Values` into the module's `#config` to dodge a cross-build `res.#Image` / `res.#Secret` closure mismatch.

The authored `Release` path, meanwhile, hits the *same* underlying closedness wall from the other side: a `release.cue` that imports a published module (`#module: <import>`) failed to load with `field not allowed` (documented in `docs/design/release-cr-imported-module-closedness.md`). Its only fixture is still on the retired `core/v1alpha1` schema and no test renders a real imported module, so the regression was invisible.

The `core` schema fix that makes `#Module` identity author-supplied (`modulePath!` / `version!`, replacing the `modulePath: metadata.modulePath` self-cycle) removes the root blocker. With it, both paths can construct the release as **one CUE build that imports the module** — `cue eval` semantics — eliminating both Go workarounds and converging the two paths onto a single mechanism. This change does that for the render path.

## What Changes

- Rewrite `synth.Release` (`opm/helper/synth/release.go`) to construct the `#ModuleRelease` by **synthesizing a virtual CUE package and evaluating it in a single build** via the loader path, instead of `buildReleaseScope` + `cue.Scope` + Go value pre-merge. The virtual package (an in-memory `load.Config.Overlay`) carries a fabricated `cue.mod/module.cue` (deps: `opmodel.dev/core@<resolved>`, `<module path>@<version>`), a `release.cue` that imports both and writes `#module: <import>` + caller metadata, and a `values.cue` rendered from the caller's `Values` (via `format.Node`, never string-interpolated). Reuse the dependency-fabrication approach already proven in `opm/helper/loader/registry` (`TestOverlayResolvesDepsButFSPinningFails`).
- **Delete the scope trick** (`buildReleaseScope`, the `userModule` overlay, the `cue.Scope` compile) and **delete the Go value pre-merge** (`preparedModule.FillPath(schema.Config, in.Values)`). The schema's own `let unifiedModule = #module & {#config: values}` performs the merge in CUE.
- Factor the shared **evaluate → shape-gate** step so `synth.Release` (overlay source) and `LoadReleasePackage` (on-disk source) run the identical build-and-validate code. The only difference becomes *where the package files come from*.
- Preserve the public Go signature of `synth.Release` / `Kernel.SynthesizeRelease` and the observable outputs (schema-derived `metadata.uuid`, fanned `components`, `opm-secrets`, stamped labels). Behavior is preserved for all inputs that work today; **additionally**, imported-module releases now construct correctly.
- Add the missing coverage: an integration test that renders a **real imported module** end-to-end through the shared path (the gap that let the `Release` path rot), and a test asserting `synth.Release` and an equivalent authored package produce the same compiled output.
- Records: `MIGRATIONS.md` (internal mechanism change, behavior-preserving), refresh `docs/design/release-cr-imported-module-closedness.md` to mark the root cause fixed and point at this change, and reference ADR-003.

## Capabilities

### Modified Capabilities

- `release-synthesis`: `synth.Release` constructs the release through single-build CUE evaluation of a synthesized package (per ADR-003), not cross-build `FillPath`/`Scope` stitching. Derived-fields-from-schema and no-implicit-values-fallback requirements are retained; the "values pre-merged into module `#config` in Go" mechanism is replaced by in-build CUE unification, and import-based module references are now first-class.

## Impact

- **Affected packages**: `opm/helper/synth` (rewrite), `opm/helper/loader/file` (extract shared evaluate/shape-gate), possibly `opm/helper/loader/internal/shape` (no change expected — reuse as-is). Downstream `opm-operator` `KernelModuleRenderer` / `KernelReleaseRenderer` call the same entry points and need no change; the authored `Release` path simply starts working for imports.
- **SemVer**: no break to `opm/` Go signatures → **MINOR at most** (additive capability: imported-module releases construct). Internal construction mechanism changes; outputs for currently-working inputs are unchanged.
- **Hard dependency / gating**: requires the `core` `#Module` author-supplied-identity fix to be **published** and resolvable via `opmodel.dev/core@v0`. Until then, import-based construction still hits `field not allowed`. This change MUST verify the resolved `core` version carries author-supplied identity before relying on it (and its tests pin a `core` version that does).
- **Complexity (Principle VII)**: net **removal** — two workarounds deleted, two paths merged into one; the new cost is virtual-package fabrication, reused from the registry loader.
- **Out of scope**: the materialize path (its `Composed`/`FillPath` seam is the sibling change `rewrite-materialize-single-build`); any `core` schema edit (lands in `core/`); operator-side changes.
