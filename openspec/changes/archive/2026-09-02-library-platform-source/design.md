# Design: library-platform-source

## Context

See `proposal.md` § Why. Enhancement 0019 D9's render build consumes source trees, not evaluated values; this slice threads the trees through the artifact types so the render-build slice has them to consume. Current state: `module.Source` exists (registry-staged, overlay-only, stamped by `Kernel.AcquireModuleFromRegistry`); `synth.Instance` builds a staged tree in `buildOverlay` and discards it after the build; `platform.Platform` and `module.Instance` carry no source.

## Goals / Non-Goals

**Goals**

- Every artifact the render build will import (instance, platform) carries a `Source` describing where its package lives, in both overlay and on-disk modes.
- Zero behavior change: no existing code path reads the new fields; every existing test stays green untouched (except signature-following edits at `synth.Instance` call sites).

**Non-Goals**

- No render-module staging, no `cue.mod` promotion, no skew comparison, no per-render `cue.Context` — all `library-render-build` (D9/D13/D7/D18/D8).
- No `Materialize` changes and no deletion; the old pipeline runs unchanged until the render-build wave re-pins core.
- No `Source` for directory-loaded `module.Module` (the module reaches the render build inside the instance's tree; add it later if a consumer appears — Principle VII).
- No broadening of the synth gate: `synth.Instance` still requires a registry-staged, overlay-carrying module (`ErrMissingSource`). Letting synth stage inside an on-disk module root is a real possibility the broadened `Source` type admits, deferred until a consumer needs it.

## Research & Decisions

### One Source type, broadened, instead of per-artifact source structs

**Context**: Platform needs a source; instance needs a source; `module.Source` already exists with narrower documented semantics ("fetched from a registry").
**Explored**: a new neutral package (rejected: new public package for one struct); duplicating a `platform.Source` struct (rejected: three copies of one concept, and the render-build slice would need conversions); broadening `module.Source`.
**Decision**: broaden `module.Source` (docs + new `Pkg string` field + documented `Overlay == nil` on-disk mode) and alias it as `platform.Source = module.Source`.
**Rationale**: the alias mirrors the existing `platform.PlatformMetadata = schema.PlatformMetadata` re-export pattern, keeps call sites reading `platform.Source`, and adds no conversion surface. `Pkg` is required because the synthesized instance package lives in a subdirectory (`opm-synth-instance`) of the module's root, while dir-loaded artifacts are root packages; the render build needs to know which package inside the tree to import. Additive for existing users: zero value `Pkg` means `.` and `HasSource()` is untouched.

### `synth.Instance` returns the tree; the Kernel stamps it

**Context**: The staged tree exists inside `synth.Instance` (`buildOverlay`) and is discarded after `BuildInstanceOverlayAt`.
**Explored**: a parallel `synth.InstanceStaged` function keeping the old signature (rejected: permanent duplicate surface for a pre-GA nicety, Principle VII); stamping inside `synth` onto the eventual `*Instance` (impossible: synth returns a `cue.Value`; the `*Instance` is constructed later by `ProcessModuleInstance`).
**Decision**: `synth.Instance` returns `(cue.Value, *module.Source, error)`; `Kernel.SynthesizeInstance` sets `inst.Source` after `ProcessModuleInstance` succeeds.
**Rationale**: the tree is a byproduct synth already computed; returning it costs nothing and re-deriving it later would duplicate `buildOverlay`. The break is confined to direct `synth.Instance` callers; `Kernel.SynthesizeInstance` (the recommended entry point, and the only caller in `cli`/`opm-operator` per a workspace grep task below) keeps its signature. Pre-GA, no migration fragment (ADR-004; consumers migrate in the same wave, and none is expected to need to).

Sketch:

```go
// synth
spec, src, err := synth.Instance(k.cueCtx, in)   // src: {Root, Pkg: synthPkgDir, Overlay: augmented}
// kernel
inst, err := k.ProcessModuleInstance(ctx, spec, *in.Module, cue.Value{})
inst.Source = src
```

### Acquire methods on the Kernel, not loader signature changes

**Context**: Dir-loaded platforms and instances need `Source` stamped somewhere; the file loaders return bare `cue.Value`s.
**Explored**: adding source returns to `loaderfile.Load*Package` (rejected: breaks three helper signatures to plumb a value the kernel can compute itself — the source of a dir-load IS the dir); stamping inside `NewPlatformFromValue` (impossible: a value carries no path).
**Decision**: two new kernel methods, `AcquirePlatformFromDir` and `AcquireInstanceFromDir`, composing the existing loader + constructor/validated-entry-point + a `Source{Root: absDir}` stamp. Naming follows `AcquireModuleFromRegistry`: "Acquire" = returns a typed, source-carrying artifact.
**Rationale**: keeps every existing signature stable, gives frontends one obvious call for the render-build input shape, and keeps the loader tree doing exactly one job (evaluate + gate). `AcquireInstanceFromDir` goes through `ProcessModuleInstance` (zero values) rather than `NewInstanceFromValue` so the acquired instance is validated where applied — the same bar `SynthesizeInstance` output meets; a file-loaded instance package must already be concrete, which `LoadInstancePackage`'s contract already implies.

## Public surface changes (`opm/`)

- `opm/module`: `Source.Pkg` field (additive), broadened `Source` docs, `Instance.Source` field (additive).
- `opm/platform`: `Source` alias + `Platform.Source` field (additive).
- `opm/helper/synth`: `Instance` signature change (**breaking**, helper boundary).
- `opm/kernel`: `AcquirePlatformFromDir`, `AcquireInstanceFromDir` (additive); `SynthesizeInstance` behavior gains the stamp (signature unchanged).

## Risks / Trade-offs

- [`Instance.Source.Overlay` retains the module's cloned file map in memory for the instance's lifetime] → acceptable: it is the same data already retained on `Module.Source`, and the render-build slice is its consumer; a caller that never renders can nil the field.
- [Alias couples `opm/platform` to `opm/module` by import] → no cycle (`module` does not import `platform`); the coupling is one type alias, the same weight as the existing schema re-exports.
- [A direct `synth.Instance` caller outside this repo breaks] → verification task greps `cli` and `opm-operator`; expected zero hits (both use the kernel wrapper). If a hit appears, the fix is mechanical at the call site in that repo's own wave.
- [`AcquireInstanceFromDir` enforces concreteness where a hypothetical caller wanted a draft instance] → deliberate: it is the validated acquire; draft flows keep using `LoadInstancePackage` + their own processing.

## Migration Plan

Single PR, conventional commits per package (`feat(module): …`, `feat(platform): …`, `feat(kernel): …`, `refactor(helper)!: return staged source from synth.Instance` or fold into one `feat!` commit per repo convention). No migration fragment (pre-GA, ADR-004). Rollback is a revert; nothing external consumes the new surface until `library-render-build`.

## Open Questions

None. The one deferred possibility (synth staging inside an on-disk module root) is recorded as a Non-Goal, not an unknown.
