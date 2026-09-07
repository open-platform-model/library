# Design: one-api-tier

## Context

See `proposal.md` § Why. The import graph on `main` after `cut-dead-surface` and `dedupe-internals`:

```
  opm/kernel ---> opm/helper/loader/file      (LoadOptions, sentinels, buildAndShapeGate,
             |                                  BuildInstanceOverlayAt)
             +--> opm/helper/loader/registry  (LoadOptions copy, LoadModulePackageWithSource)
             +--> opm/helper/synth            (InstanceInput, Err*, Instance)
  opm/helper/synth ---> opm/helper/loader/file
  opm/helper/loader/{file,registry} ---> opm/helper/loader/internal/shape
  everything above ---> opm/internal/{cueenv,sourcetree,valuesfile}
```

Three `Source` producers exist: `sourceForDir` (on-disk, kernel), `sourcetree.OverlayFromFS` under a synthetic root (registry), and `synth.buildOverlay` (clone plus two files). Two consumers read a `Source`: `sourcetree` (`PackageName`, `ReadFile`, `WriteTo`) and `renderstage.serveDir`, which serves on-disk mode by pointing the render module's `replaceWith` at the real directory and overlay mode by writing every entry into the staging directory.

Constraints: `Render`'s stage, build and decode path is untouched; every existing test scenario keeps its coverage, re-homed where its subject moved; the tree is green after every commit; `cut-dead-surface` and `dedupe-internals` have landed (this change edits the same files).

## Goals / Non-Goals

**Goals:**

- One tier: every artifact a frontend can hold comes from an acquire verb or from the package constructor it already has a value for.
- One boundary, enforced by lint: nothing outside `opm/helper/` imports it.
- One registry knob and one values shape across the kernel.
- Clear the five items slices 1 and 3 parked here.
- One overlay writer, owned by `module.Source` and reachable by a frontend.

**Non-Goals:**

- Changing the `Source` model to "root plus layered overlay" (decision below; slice 6).
- A per-call `cue.Context` for acquisition or synthesis (slice 6).
- Retiring `Kernel.CueContext()`; its remaining production use is `SchemaCache().Get(k.CueContext())`, a `schema.Cache` API question.
- Any change to the render glue, `RenderDiagnostics` or the typed gate errors (slice 4).
- Consumer code: the cli and operator migrate in their own PRs (see Migration Plan).

## Research & Decisions

### `AcquireModuleFromDir` stamps overlay mode by reading the tree

**Context**: Synthesis accepts only overlay-mode sources (`HasSource` requires a non-empty overlay), and `synth.buildOverlay` clones that map to add its two files. A module acquired from a directory needs a `Source` synthesis accepts.
**Explored**: (A) read the directory's `.cue` files into an overlay, the cli's current `stageLocalModuleSource` moved down; (B) record only the path and teach synthesis, `sourcetree` and `renderstage.serveDir` a third shape, "real root plus a few layered files". B removes the tree read but needs a render-side mechanism: `serveDir` cannot `replaceWith` a real directory whose package only partly exists on disk, so either the tree is copied at render time (moving the read, not removing it) or the overlay is served through `load.Config.Overlay` at `Build`, which is the slice 6 spike. B is about 60 to 90 lines now against about 20 for A, and its measurable saving in this slice is cli-only (the operator never acquires a module from a directory).
**Decision**: A. `AcquireModuleFromDir` computes `sourceForDir(absDir)` and `sourcetree.OverlayFromDir(src.Root)` and stamps `{Root, Pkg, Overlay}`. `HasSource`, synthesis and `serveDir` are untouched.
**Rationale**: smallest diff, no spike, identical memory behaviour to what the cli does today. If slice 6 lands overlay serving at build time, "root plus layered files" becomes what `cue/load` does natively and B falls out of it; doing B now would write a branch slice 6 deletes.

### Synthesis refuses a subpackage module

**Context**: the synthesized `instance.cue` imports the module by `Metadata.ModulePath`, which resolves to the module's root package. `sourceForDir` can describe a subdirectory package (`Pkg` non-empty).
**Decision**: `AcquireModuleFromDir` stays general (scaffold, repair and the publish gate load module roots and read metadata; a subdirectory package is still a valid module value). `SynthesizeInstance` checks `Module.Source.Pkg == ""` after the `HasSource` gate and fails with a plain error naming the root-package requirement. No new sentinel: no consumer branches on it.

### The loaders and synthesis become `opm/internal/loader` and `opm/internal/synth`

**Context**: `opm/internal/` is importable by every package under `opm/` and by nothing outside, the placement `dedupe-internals` used for `cueenv` and `sourcetree`.
**Explored**: keeping `opm/helper/synth` public "for a caller without a Kernel" (no such caller exists in either consumer; the publish gate calls the file loader directly only because it holds a `cue.Context`, and it builds a Kernel ten lines earlier); one internal package for loaders and synthesis together (the registry fetch pulls `mod/modconfig`, which synthesis does not need; two packages keep the dependency edges legible).
**Decision**: `opm/internal/loader` holds the shape gate and its specs (from `loader/internal/shape`), `LoadDir(ctx, root, pkg, overlay, env, spec)` (today's `buildAndShapeGate`, now the only load routine: the module and platform loaders that inlined the sequence go through it, deleting the slice 3 deferral and the `validate.go` alias file), and `FetchModule(ctx, cueCtx, path, version, env)` (today's `LoadModulePackageWithSource` with identity verification). `opm/internal/synth` holds `Input{Module, Name, Namespace, Values cue.Value, Labels, Annotations}`, `Instance(cueCtx, coreVersion, in)` and the file renderers; the kernel passes the merged values value and the resolved core version, so the package reads no cache. Tests move with their subjects.
**Rationale**: Principle III keeps loading, synthesis and the kernel in distinct packages; `internal` keeps them off the SemVer surface (Principle VI, VII).

### Sentinels live in `opm/errors`

**Context**: `opm/internal/loader` and `opm/internal/synth` cannot import `opm/kernel` (cycle), and a sentinel declared in an internal package is unreachable for `errors.Is` from a consumer.
**Decision**: `opm/errors/sentinels.go` declares `ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField`, `ErrMissingModule`, `ErrMissingName`, `ErrMissingNamespace`, `ErrMissingSource`, `ErrSchemaUnavailable`. The internal packages wrap them; the kernel wraps them; consumers already import `opm/errors` for the typed render causes. `ErrMissingSchemaCache` is deleted with its field.
**Alternative rejected**: re-exporting from `opm/kernel` (`var ErrWrongKind = loader.ErrWrongKind`). Two names for one value, and it puts loader failure classes on the kernel package rather than beside the other error types.

### One registry knob; `New` seeds the schema loader

**Context**: the mapping reaches the kernel through `WithRegistry`, through `LoadOptions.Registry` on every directory verb, and through `OCILoader.Registry` on `WithSchemaLoader`. The operator sets the first and second but not the third (`cmd/main.go:253`); its adapter already documents the per-call registry argument as ignored.
**Decision**: `New` applies the options, then, when `schemaLoader == nil`, builds `schema.OCILoader{Registry: k.registry}`. The acquire verbs and synthesis pass `cueenv.Override(k.registry, "")` to the internal loader; no `LoadOptions` type remains public. The `WithRegistry` doc names every operation the mapping governs.
**Rationale**: one concept, one knob. The seed is additive and fixes the operator's silent split; the parameter removal is the breaking half and every consumer passes the same string in both places today, so nothing is lost.
**Alternative rejected**: keeping `LoadOptions` and defaulting it from the kernel when empty. Non-breaking, but it leaves a second knob that can disagree with the first.

### `InstanceInput.Values` is `[]Source`; both values paths share one merge and one attribution pass

**Context**: synthesis renders `Values` to `values.cue` (`valuesfile.Render`), so the caller's `cue.Context` is irrelevant, yet the operator must compile bytes through `Kernel.CueContext()`. `loadInstanceWithValues` already unifies `[]Source`, renders the result, builds, and then checks the sources against `#config` so positions name the source.
**Decision**: `kernel.InstanceInput{Module, Name, Namespace, Values []Source, Labels, Annotations}`. `SynthesizeInstance` calls a shared `mergeSources(sources) (cue.Value, error)` (extracted from `loadInstanceWithValues`), passes the merged value to `internal/synth`, and after the build runs the same `validateSources(configSchema, sources, false)` pass `AcquireInstanceFromDir` runs. `AcquireOption`/`WithValues` become the variadic `values ...Source` (one option with one caller; Principle VII). `LoadSourceFromBytes`, kept by slice 1 for this, gets its consumer.
**Rationale**: the two ways values enter an instance now differ only in where the package files come from, which is the `instance-synthesis` shared-gate requirement taken to its end. The operator stops calling an "advanced, typically tests" accessor.

### `Source.Overlay` carries bytes

**Context**: every overlay the library builds starts as bytes (`OverlayFromDir`, `OverlayFromFS`, `valuesfile.Render`, the rendered `instance.cue`), yet the field stores `load.Source`, an opaque interface, and `sourcetree.Bytes` recovers the bytes by reflection to write them back. Slice 3 parked the type change here because it is a public field.
**Decision**: `Overlay map[string][]byte`. `loader.LoadDir` builds the `load.Config.Overlay` map with `load.FromBytes` at the one place cue/load is called with an overlay; `sourcetree.WriteTo` and `ReadFile` use the bytes directly; `sourcetree.Bytes` and its tests are deleted. Synthesis clones the map with `maps.Clone` and appends its two entries.
**Rationale**: no consumer reads the field, and the only consumer that constructed one is the cli walker this change deletes; the reflection was the cost of storing the wrong type.

### The overlay writer is a method on `Source`

**Context**: the cli's scaffold needs a fetched module's tree on disk and re-fetches it to get one (`cli/internal/scaffold/scaffold.go:219-268`), while `sourcetree.WriteTo` already writes exactly that tree for the render stage. `sourcetree` imports `module` for the `Source` type, so a method on `Source` cannot delegate to it.
**Explored**: (A) a kernel verb, `Kernel.WriteSource(src, dir)`: no kernel state is involved, wrong altitude; (B) exporting from `sourcetree`: internal by design, unreachable from a consumer; (C) `(*Source).WriteTo(dir) ([]string, error)` with the loop in `opm/module`, `sourcetree.WriteTo` deleted and `serveDir` calling the method.
**Decision**: C, landing right after the byte overlay (task 4.2): over `map[string][]byte` the loop is about fifteen lines with no reflection, whereas over `load.Source` it would need the `Bytes` reflection this change deletes, which is why it cannot lead the task list. The method validates every entry before writing (nil receiver, on-disk mode since `Root` already is the directory, an entry outside `Root`), creates parent directories, and returns the dir-relative paths written, sorted, so a caller never iterates the overlay itself and the overlay's element type stays invisible to it. The name matches `platformmodule.Files.WriteTo(dir string)`; `go vet`'s standard-method check applies only when the first parameter is an `io.Writer`.
**Rationale**: one writer, owned by the type that holds the data; the scaffold's second fetch, its `modconfig` dependency and its walk are deleted for a method the render stage already needed.

### The boundary is a lint rule

**Decision**: `.golangci.yml` gains a `depguard` rule: files under `opm/kernel`, `opm/module`, `opm/platform`, `opm/schema`, `opm/errors`, `opm/core`, `opm/compat` and `opm/internal/**` may not import `github.com/open-platform-model/library/opm/helper/**`. `task lint` is already a merge gate.
**Alternative rejected**: a test that shells out to `go list -deps`. Kernel neutrality forbids shell in library code and the lint gate already runs on every PR.

### Commit order: additive first

**Decision**: tasks 1.x land the additive surface (`AcquireModuleFromDir`, the seeded schema loader) with the helper packages still in place, so a reviewer can stop there with a shippable library. Tasks 2.x onward are the breaking wave; consumers re-pin once at its end.

## Risks / Trade-offs

- [Test migration volume: about 75 `LoadOptions{}` sites, 24 `InstanceInput{}` sites and 31 raw-loader calls in library tests] → mechanical edits done per package as its subject moves; each task's verification is that package's tests green; the parity harness and flow tests pin end-to-end behaviour.
- [Operator behaviour change: core schema resolves through the `WithRegistry` mapping instead of the process `CUE_REGISTRY`] → both are set to the same value in every deployment today; `verifyCoreSchema` at startup proves resolution; called out in the operator PR.
- [`AcquireModuleFromDir` on a directory with no `cue.mod`] → `sourceForDir` makes it its own root; the module value loads as today; synthesis then fails at the module import with CUE's own error naming the path. Tests use `testdata` modules with a `cue.mod`.
- [The publish gate in the cli loses its bare-context loader] → `publish.Options` gains the `*kernel.Kernel` `RunPublish` already builds; the gate calls `AcquireModuleFromDir` and discards the module. Consumer-side, in the cli PR.
- [Two internal packages instead of one] → the edge `synth -> loader` stays (synthesis builds through `LoadDir`); `loader` does not import `synth`. Documented in each package doc.

## Migration Plan

1. Library PR: tasks in order; `task check` and the race pass green; `go build ./...` in `../cli` and `../opm-operator` against a temporary `replace` to prove the consumer edits are the ones the proposal lists.
2. Release-please cuts the next alpha.
3. cli PR: the thirteen sites in `proposal.md` § Impact, `stageLocalModuleSource` and `copyFetched` deleted, `publish.Options.Kernel` added.
4. operator PR: the four sites plus the integration helper; values through `LoadSourceFromBytes`; note the schema-resolution change.
5. Rollback: consumers pin the previous alpha; no data or on-disk format changes.

## Open Questions

None that change the specs or tasks. Whether `Kernel.CueContext()` survives past slice 7 depends on giving `schema.Cache` a context-free accessor, which this change does not touch.
