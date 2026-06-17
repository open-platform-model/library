# ADR-003: Construct artifacts by single-build CUE evaluation, not cross-build FillPath of closed values

## Status

Accepted — implemented by OpenSpec changes `simplify-render-single-build` (render path) and `rewrite-materialize-single-build` (materialize path).

## Context

The kernel assembles OPM artifacts (`#ModuleRelease`, `#Platform`) from pieces that originate in **separate CUE evaluations**: the core schema is loaded into a per-Kernel `*schema.Cache`; a module is loaded from the registry as its own `cue.Value`; caller values are compiled from bytes; catalogs are pulled one build at a time. The kernel then stitches these together in Go via `FillPath` and `cue.Scope`.

CUE closedness identity is **per-build**. Two `#Image` definitions produced by two separate `BuildInstance` calls are distinct closed types even though they are the same source text in the same `*cue.Context`. The moment Go unifies a value carrying closed type `#Image`-from-build-A into a slot typed by `#Image`-from-build-B, CUE's closedness check rejects fields it should accept — `field not allowed` — because the two closed identities do not match. Pure whole-program CUE (`cue eval`/`cue export` over imports) never hits this: every definition is resolved once, so there is exactly one `#Image`.

This is not a hypothetical. The same failure mode has been worked around in three independent places:

1. **`synth.Release` scope trick** (`opm/helper/synth/release.go`, `buildReleaseScope` + `cue.Scope`). A published, closed `#Module` could not be unified into `#ModuleRelease.#module` directly. Aggravated by a now-fixed self-cycle in `core` `#Module.metadata` (`modulePath: metadata.modulePath`, fixed in `core` commit making identity author-supplied — `modulePath!: #ModulePathType`).
2. **`synth.Release` value pre-merge** (`opm/helper/synth/release.go`, `preparedModule.FillPath(schema.Config, in.Values)`). Caller values had to be merged into the module's `#config` **in Go before** the module entered release compilation, because the registry-loaded module's closed `res.#Image` / `res.#Secret` types differ at the CUE-runtime level from the same types reached through the cache-resolved schema. The schema already expresses this merge in CUE (`let unifiedModule = #module & {#config: values}`); the Go pre-merge re-implements it solely to dodge the cross-build closure mismatch.
3. **`Materialize` `Composed` escape hatch** (`opm/materialize/types.go` `MaterializedPlatform.Composed`; `opm/materialize/materialize.go` `FillPath(schema.ComposedTransformers, …)`). `FillPath`-ing the composed transformer map into the closed, independently-built `#Platform` value corrupts lazy in-expression resolution of output-local hidden fields in transformers (documented in `docs/design/transformer-output-hidden-field-scope-bug.md` §12). The fix exposes a **separate open** `Composed` value and routes the executor to read transforms from it rather than from the closed `Package`.

Each was fixed locally with a different mechanism. Each passed `cue vet` and unit tests with stubs, and only failed when a *real* multi-source assembly ran — the regressions were invisible until the exact combination occurred. There is no stated invariant preventing the next one.

The registry loader already demonstrates the alternative. `opm/helper/loader/registry` builds a **virtual CUE package** in memory (`load.Config.Overlay`, FS nil) and lets CUE resolve transitive imports as one build. `TestOverlayResolvesDepsButFSPinningFails` exists specifically so this approach is "never simplified to FS-pinning" — the team has already learned, the hard way, that single-build evaluation is the correct shape for cross-module assembly.

## Decision

**Construct OPM artifacts by generating CUE source and evaluating it in a single build — the way `cue eval`/`cue export` does — and do not assemble a closed value by `FillPath`/`cue.Scope`-stitching values that originate in separate `BuildInstance` evaluations.**

Concretely:

- Where the kernel needs to build an artifact from typed inputs plus a referenced module/catalog, it SHALL synthesize a virtual package (an overlay carrying `cue.mod/module.cue` + the artifact source that *imports* the dependency + values as a source file) and evaluate it through `load.Instances` → `BuildInstance` — the same path the file loaders use. Overlay and on-disk packages are treated identically, because CUE treats them identically.
- Caller-supplied values that participate in a closed schema slot SHALL enter the build **as a source file in that build** (rendered from the already-parsed value via `format.Node`, never string-interpolated from raw input), not unified across a build boundary afterward.
- A frontend-driven artifact (an in-memory CR) and an author-written package SHALL converge on **one** evaluate-and-shape-gate function whose only parameter is *where the package files come from* (overlay vs. directory). A bug in artifact construction then surfaces in both paths or neither.
- Alternatives rejected: smarter per-seam `FillPath` workarounds (the status quo — each is a local patch that does not generalize and leaves the next seam undiscovered); requiring authors to inline everything (loses the reuse-a-published-artifact ergonomic the model is built for).

This decision depends, for the release path, on the `core` schema fix that makes `#Module` identity author-supplied; import-based construction is only sound once re-evaluating an imported `#Module` no longer resolves its identity self-references to bottom.

## Consequences

**Positive:** The recurring `field not allowed` / hidden-field-corruption class is retired at its root rather than patched per site — there is exactly one `#Image`, one `#Secret`, one transformer identity per build. `synth.Release` sheds both the scope trick and the Go value pre-merge; the schema's own `unifiedModule = #module & {#config: values}` does the work. The two render entry points (`ModuleRelease` CR synthesis, authored `Release` package) collapse to one mechanism, so render bugs can no longer hide in just one path. The kernel's behavior converges on `cue eval`, which is the mental model authors already have.

**Negative / trade-off:** Synthesis must fabricate a correct `cue.mod/module.cue` (the dependency set and versions) for the virtual package — machinery copied from the registry loader, not invented, but a real moving part. Rendering values to a source file adds a `format.Node` step and a small surface to get escaping right (mitigated by rendering from the parsed value, not raw input). For materialize, the principle is clear but the **mechanism is a larger rewrite** (single-evaluation composition of N pulled catalogs) and is scoped as its own change with an explicit spike; this ADR states the target, not a finished materialize design. Removing the `Composed` escape hatch is a public-surface change on `MaterializedPlatform` and is handled under SemVer in the materialize change.

**Scope note:** This invariant governs how the kernel *assembles closed values across evaluations*. It does not forbid `FillPath` outright — filling concrete inputs into an open value within a single build (e.g. `#component`, `#context` in the executor) is unaffected. The line is **cross-build unification of closed definitions**, which is the failure mode.
