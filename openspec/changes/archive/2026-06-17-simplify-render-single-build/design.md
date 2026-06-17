# Design — simplify-render-single-build

Governing principle: **ADR-003** (construct artifacts by single-build CUE evaluation, not cross-build `FillPath` of closed values). This change applies ADR-003 to the release-render path.

## Research & Decisions

### The two render paths diverge only at release-value construction
**Context**: Bugs kept appearing in one render path but not the other; the goal is to make a render bug surface in both paths or neither.
**Explored**: Traced both operator renderers and both library entry points. `KernelModuleRenderer` → `moduleacquire.Acquire` → `Kernel.SynthesizeRelease` (`synth.Release`) → `Kernel.Compile`. `KernelReleaseRenderer` → `Kernel.LoadReleasePackage` → `Kernel.Compile`. Everything from `Kernel.Compile` (FinalizeValue → Match → Execute → emit) is already shared and identical.
**Decision**: The only divergence is *how the `#ModuleRelease` `cue.Value` is built* — Go-synthesized vs. CUE-loaded. Converge that single box; leave the shared downstream untouched.
**Rationale**: Minimizes blast radius and directly targets the divergence. The inputs to the two CRs are legitimately different (typed fields + registry ref vs. authored CUE package) and SHOULD stay different; only the construction *mechanism* unifies.

### Both `synth.Release` workarounds exist to dodge cross-build closedness
**Context**: `synth.Release` is materially more complex than `LoadReleasePackage`; we need to know whether that complexity is essential.
**Explored**: Read `opm/helper/synth/release.go`. Found (a) `buildReleaseScope` + `cue.Scope` injecting the module as a `userModule` value to avoid unifying a closed `#Module` into `#ModuleRelease.#module`; and (b) `preparedModule.FillPath(schema.Config, in.Values)` pre-merging values in Go because the registry-loaded module's closed `res.#Image`/`res.#Secret` differ at CUE-runtime from the cache-resolved schema's. Confirmed the schema *already* expresses the values merge in CUE: `core/src/module_release.cue` has `let unifiedModule = #module & {#config: values}`.
**Decision**: Both workarounds are non-essential dodges of the ADR-003 failure mode. Once the module is *imported* into a single build (one `#Image`), both are removable: the scope trick because there is no closed-into-closed fill, and the pre-merge because the schema's `unifiedModule` does it.
**Rationale**: Deleting them is the simplification; keeping either would re-create a cross-build seam.

### The core self-cycle fix is the unblock, and it gates this change
**Context**: Import-based construction (`#module: <import>`) previously failed with `field not allowed`.
**Explored**: `core` declared `#Module.metadata.modulePath: metadata.modulePath` (a self-cycle CUE never registers as a permitted field of the closed struct). Re-evaluating an imported `#Module` resolved identity to bottom → admission rejected the concrete identity. `core` commit `feat(module): make #Module identity author-supplied` changes it to `modulePath!: #ModulePathType` / `version!: #VersionType`.
**Decision**: This change is gated on that fix being **published** in `opmodel.dev/core@v0`. Tests pin a `core` version carrying author-supplied identity; runtime asserts the resolved schema exposes the author-supplied shape before relying on import construction.
**Rationale**: Without the fix, the new mechanism reproduces the very error it removes. The fix is in `core` (separate repo), so this change cannot land its happy path until `core` publishes — call it out as a hard precondition, not an assumption.
**Status (resolved)**: The precondition is **MET** — `core@v0.5.0` (commit `68e4520`) is published to `ghcr.io/open-platform-model` with the author-supplied identity. Empirically confirmed on the kernel-pinned toolchain (`cue v0.17.0-alpha.1`): importing a published module into `#ModuleRelease.#module` loads with concrete `#module.metadata` on `v0.5.0` and still fails with `field not allowed` on `v0.4.0`. The new mechanism's `core@v0` floor is therefore `v0.5.0`.

### Single-build construction via in-memory overlay is already the blessed pattern
**Context**: Need a mechanism to construct the release as one evaluation that resolves the module import.
**Explored**: `opm/helper/loader/registry` builds a virtual package with `load.Config.Overlay` (FS nil) and lets CUE resolve transitive deps; `TestOverlayResolvesDepsButFSPinningFails` guards it against regressing to FS-pinning. `LoadReleasePackage` is plain `load.Instances` → `BuildInstance` → shape gate — i.e. `cue eval` semantics.
**Decision**: `synth.Release` builds an in-memory overlay package — `cue.mod/module.cue` (fabricated deps) + `release.cue` (imports `core` + module, sets metadata, `#module: <import>`) + `values.cue` (rendered from `in.Values`) — and evaluates it through the *same* loader path `LoadReleasePackage` uses. Factor the build+shape-gate into one shared function.
**Rationale**: Reuses proven machinery, makes overlay and on-disk packages literally share code, and yields exact `cue eval` parity — the stated correctness bar ("a release from a CUE package should behave the same in the kernel as `cue eval`/`export` on that package").

### Values enter as a source file, not a cross-build unify
**Context**: Caller `Values` is a `cue.Value` compiled in the kernel context; naive post-build unification would re-introduce a cross-build seam, and raw-string interpolation would be a CUE-injection risk.
**Explored**: `synth.renderReleaseSource` already only string-formats regex-constrained `name`/`namespace`/labels and keeps `values` as a value. CUE's `format.Node(value.Syntax(...))` renders a parsed value back to canonical source.
**Decision**: Render `in.Values` to a `values.cue` overlay file via `format.Node` so it joins the single build; never interpolate raw input as source. Identity strings (`name`, `namespace`) stay regex-constrained literal formatting.
**Rationale**: Keeps everything in one build (no seam) and one trust boundary (rendered from an already-parsed value, not attacker-controlled text).

### Coverage is part of the change, not a follow-up
**Context**: The `Release` path rotted invisibly because no test rendered a real imported module and its fixture is on the retired schema.
**Decision**: Ship (a) an integration test rendering a real imported module end-to-end through the shared path, (b) a parity test asserting `synth.Release` output and an equivalent authored `Release` package compile to the same resources, and (c) a direct `LoadReleasePackage` test for the authored-`Release` import path that rotted (tasks 4.4/4.7). The integration tests carry a **negative control** pinned to `core@v0.4.0` so they fail loudly if the self-cycle ever returns, rather than passing vacuously. Update the stale `test/fixtures` shape (operator-side fixture refresh is tracked in `opm-operator`; library adds its own fixture).
**Rationale**: The unified mechanism only stays unified if a test pins both paths to the same outcome; the negative control guarantees the test is exercising the re-admission path, not a happy accident of schema evolution.

## Implementation sketch (Go pseudo-code)

```go
// synth.Release (rewritten): build a virtual package, evaluate once.
func Release(ctx *cue.Context, in ReleaseInput) (cue.Value, error) {
    // ...required-input guards unchanged (sentinels)...
    overlay := map[string]load.Source{
        "/synth/cue.mod/module.cue": load.FromString(renderModuleFile(in)),   // deps: core@resolved, module@version
        "/synth/release.cue":        load.FromString(renderReleaseFile(in)),  // import core, mod; #module: mod; metadata
        "/synth/values.cue":         load.FromBytes(renderValuesFile(in)),    // format.Node(in.Values.Syntax())
    }
    val, err := buildAndShapeGate(ctx, "/synth", overlay, releaseShape)       // SHARED with LoadReleasePackage
    // no buildReleaseScope, no FillPath(Config, Values)
    return val, err
}

// LoadReleasePackage (unchanged behavior): same shared core, on-disk source.
func LoadReleasePackage(ctx *cue.Context, dir string, opts LoadOptions) (cue.Value, error) {
    return buildAndShapeGate(ctx, dir, nil /*FS*/, releaseShape)
}
```

## Phase impact

- **loader**: extract `buildAndShapeGate` (build `load.Config` from overlay-or-dir, `BuildInstance`, run shape gate). No behavior change to `LoadReleasePackage`.
- **helper/synth**: rewrite `Release`; delete `buildReleaseScope`, `renderReleaseSource`'s scope coupling, and the pre-merge. Add module-file / values-file renderers (fabrication reused from `loader/registry`).
- **kernel**: `SynthesizeRelease` wrapper unchanged in signature; still chains into `ProcessModuleRelease`.
- **Public surface (`opm/`)**: signatures stable. The shape-gate extraction stays within existing packages.

## Open questions (resolve during implementation, none blocking the design)

1. Can `synth.Release` resolve the module via the same registry env the loader uses, or does the fabricated `cue.mod/module.cue` need the resolved `core` version threaded from `*schema.Cache.ResolvedVersion()`? (Likely the latter — pin `core` explicitly in the fabricated module file.) **Resolved:** the fabricated `cue.mod` pins `core` from `ResolvedVersion()`, and the module is resolved from the process `CUE_REGISTRY`.
2. Does removing the Go pre-merge expose any values-defaulting behavior that `ProcessModuleRelease` previously relied on? (Parity test is the guard.) **Resolved:** yes — since `synth.Release` now bakes values into the build, `SynthesizeRelease` calls `ProcessModuleRelease` with the **zero** value (no re-fill), matching how a file-loaded release whose values live in the package is processed.
3. Overlay path naming: synthesize under a stable virtual root; confirm CUE module resolution is happy with an in-memory `cue.mod`. **Resolved:** stable root `/opm-synth-release`; the synth package is the main module (rebuilt per call), never cached by path.

> **Module import path (resolved post-design).** This design assumed the importable module path is `metadata.modulePath/name@version`. That is wrong for modules whose registry leaf differs from `metadata.name` (e.g. `zot-registry-ttl` published at `…/zot_registry_ttl`). The implementation imports by `metadata.modulePath/metadata.nameSnakeCase@<major>` — the canonical address established by the **OPM Module Publishing Workflow** enhancement (`enhancements/0003`), which depends on `core`'s `metadata.nameSnakeCase` (shipped in `core@v0.6.0`). See `opm/helper/synth/render.go:moduleImportPath` / `moduleSnakeName`.
