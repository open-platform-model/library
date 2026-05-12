## Context

The library today produces a `*module.Release` along two code paths:

1. **File-driven** — `Kernel.LoadReleaseFile` parses a hand-authored `release.cue`, then `module.NewReleaseFromValue` constructs the typed struct. This is what the CLI's `opm module test` and `opm module compile` flows use.
2. **In-memory** — `Kernel.ProcessModuleRelease` accepts a pre-built spec `cue.Value` plus a `module.Module` and `values`, validates, fills, and returns a `*module.Release`. This is used by integration tests today, but the "pre-built spec" half is hand-rolled per call site via `CompileString` + `FillPath`, bypassing the schema and forcing the test author to hardcode the UUID (see `pkg/kernel/flow_integration_test.go:112-150`).

Two future consumers force a real synthesis path:

- **CLI**: `opm release` family — invoking from `(module, name, namespace, values)` rather than `release.cue`.
- **`opm-operator`**: reconciler converting a `ModuleRelease` CR into the kernel artifact for compilation.

The CUE schema is the system of record: `apis/core/v1alpha2/module_release.cue:19` derives the release UUID via `cue_uuid.SHA1(OPMNamespace, "<module.uuid>:<name>:<namespace>")`, the `components` field is fanned by a CUE comprehension over the unified module, and the standard `module-release.opmodel.dev/{name,uuid}` labels are stamped by the schema. Any synthesis path that bypasses unification with `#ModuleRelease` must reimplement all of these in Go — which guarantees drift the moment the schema evolves.

The version binding (`pkg/api/v1alpha2/binding.go`) already exposes the embedded CUE filesystem via `EmbeddedSchema() fs.FS` per the `api-version-dispatch` capability, but no production code consumes it. This slice gives that filesystem its first runtime consumer.

## Goals / Non-Goals

**Goals:**

- Make synthesis from `(Module, name, namespace, values, labels, annotations)` a first-class library operation reachable through a single `Kernel` method.
- Keep CUE the source of truth for derived fields (UUID, components, schema-stamped labels) by unifying inputs against the embedded `#ModuleRelease` definition.
- Establish a small, version-agnostic helper boundary at `pkg/helper/synth/` that future synthesis helpers (e.g. synthesising `#Provider` or `#Platform` artifacts) can extend.
- Surface the loaded schema package as a reusable primitive on the `Binding` interface so future helpers (validation, dry-run renderers) do not each re-implement `load.Instances` over the embed FS.

**Non-Goals:**

- Implementing the CLI command or operator reconciler — both are downstream slices that consume this helper.
- Replacing the file-driven loader. `Kernel.LoadReleaseFile` stays unchanged; synthesis is an additional path, not a replacement.
- Synthesis of `#ModuleReleaseMap` (multi-release bundles).
- Bytes loader (`pkg/helper/loader/bytes/`) — still a doc-only skeleton.
- Adding values-overlay convenience (debug values, file-based overlays). Values is caller-supplied; layering is a separate concern.
- v1alpha3 binding work. The slice ships the v1alpha2 implementation and the interface extension.

## Decisions

### D1. Package location: `pkg/helper/synth/` peer to `pkg/helper/loader/`

The `pkg/helper/loader/` tree is named for *loading* — reading existing artifact bytes from a source (filesystem, byte buffer). Synthesis is creation from typed inputs, not parsing. Co-locating under `loader/` would conflate verbs and force the package doc to apologise for the location. A peer directory keeps each helper's purpose legible.

**Alternative considered:** `pkg/helper/loader/synth/`. Rejected because the loader-package doc (`pkg/helper/loader/file/release.go:25`) explicitly says "Load... a #ModuleRelease from a standalone .cue file." Synthesis doesn't fit that contract.

**Alternative considered:** Put the function on `Kernel` only, with no helper package. Rejected because the helper-package convention (`helper-packages` spec) is the documented landing zone for "opinionated frontend conveniences that wrap kernel primitives" — synthesis fits that exact description, and keeping it as a free function makes it reusable without a `Kernel` instance (e.g. for unit-testing transformer behavior in tight loops).

### D2. `Binding.SchemaValue(*cue.Context) (cue.Value, error)` is the schema-load primitive

Adding the method to the existing `Binding` interface centralises schema loading next to the version-specific path constants and decoders. Each binding implements it once, caches the result, and exposes a `cue.Value` whose definitions (`#ModuleRelease`, `#Module`, etc.) callers reach with `LookupPath`. The synth helper becomes a thin "look up `#ModuleRelease`, fill known paths, return" pass — version-specific schema mechanics live where the rest of version-specific machinery lives.

**Alternative considered (Q1.A):** Have the synth helper consume `binding.EmbeddedSchema() fs.FS` directly and call `load.Instances` itself. Rejected — it duplicates schema-load logic in every consumer of the embed FS, and the load arguments (overlay path, package name) are version-specific knowledge that should live on the binding.

**Alternative considered:** A new top-level helper like `pkg/helper/schema/` exposing `LoadSchemaPackage(binding) (cue.Value, error)`. Rejected — it would not eliminate the per-binding knowledge of overlay paths; it just moves it to a switch on `binding.Version()`.

**Caching strategy:** `sync.Once` per binding instance, keyed implicitly on the first `*cue.Context` passed in. The library's "one Kernel per process" guidance (`kernel-runtime` spec, Goroutine Safety Contract) makes this safe in practice. Documented in the method docstring as "first-context-wins; pass the same `*cue.Context` for the binding's lifetime."

**Breaking change:** This is an additive method on an exported interface, which under Go semantics is a breaking change for out-of-tree implementers. The library has none today, and per Principle VI we mark the slice MAJOR.

### D3. Values is caller-supplied; no debug-values fallback

Earlier discussion considered defaulting `ReleaseInput.Values` to `Module.debugValues` when caller passed the zero `cue.Value`. Rejected because `debugValues` is documented as "author-supplied example values used by build/validation tooling" (`pkg/module/module.go:5-13`). Treating it as a production default would let a controller silently deploy a CR with no values into the dev-test overlay the module author wrote — a class of surprise we should not bake into the kernel.

Instead: `ReleaseInput.Values` is a `cue.Value` field. When the zero value is passed, the synth helper does not fill `paths.Values`; the spec then flows to `ProcessModuleRelease`, which fails the concreteness check when `#config` has no defaults. Frontends that *want* debug-values behavior (e.g. CLI `--use-debug-values` flag) layer it explicitly on the caller side before constructing `ReleaseInput`.

### D4. Schema unification via `cue.Scope`, not `CompileString` skeleton

`synth.Release` produces a `#ModuleRelease` value through CUE compile-time unification so every schema-driven derivation runs automatically:

- `metadata.uuid` is computed by `uuid.SHA1(OPMNamespace, "<module.uuid>:<name>:<namespace>")` (module_release.cue:19).
- `components` is fanned out of `unifiedModule.#components` (module_release.cue:50-57).
- `opm-secrets` component is auto-added when the module config contains `#Secret` instances.
- Standard labels (`module-release.opmodel.dev/{name,uuid}`) are stamped (module_release.cue:23-27).

The `flow_integration_test.go` skeleton pattern is rejected for production use precisely because it bypasses each of these.

**Implementation: scope-driven compilation, not `releaseDef.FillPath`.** An earlier draft called for `binding.SchemaValue(ctx).LookupPath("#ModuleRelease").FillPath(paths.Module, mod.Package)`. That path is rejected by CUE's Go API because `#Module.metadata` declares `modulePath: metadata.modulePath` — a self-referential constraint that evaluates to bottom when `#Module` is lifted out of its package and re-unified through `FillPath`. Caller-supplied `modulePath` / `version` then fail closed-struct admission as "field not allowed".

The implemented path:

1. Extend the binding's schema package with a non-hidden `userModule: _` field via `schemaPkg.Unify(...)` (the package value is open at the file level, so this admission is legal).
2. `FillPath(userModule, in.Module.Package)` to bind the caller's module into the combined scope.
3. Compile a small CUE source that references `#ModuleRelease` and `userModule` via `cue.Scope(combined)`. The release source is `{ #ModuleRelease, metadata: {...}, #module: userModule }` — exactly what a hand-authored `release.cue` would write, so admission is checked against the original definitions rather than a re-evaluated copy.

This restores compile-time semantics for the closed-struct check and lets every schema derivation run as listed above.

**Values handling deviates from the schema's `#config: values` path.** The release schema is written so caller `values` flow into `#config` via `let unifiedModule = #module & {#config: values}`. Going through that path triggers a second closure conflict: the user's module embeds resource definitions (e.g. `res.#Image`, `res.#Secret`) loaded from the registry-backed CUE module, while the embedded schema's evaluation reaches the *same FQN* types through its own load. At the CUE-runtime level these are distinct closed structs even though they declare identical fields, so propagating `release.values` back into `#config` produces "field not allowed" on harmless defaults like `image.pullPolicy` once `ProcessModuleRelease` fills validated values with their schema-supplied defaults.

To sidestep that, `synth.Release` pre-merges `in.Values` into `mod.Package` at `paths.Config` *before* the module enters the scope. The schema then evaluates `#config: values` against an already-merged module, the components/auto-secrets comprehensions run as designed, and `release.values` stays open so downstream `ProcessModuleRelease` can fill validated values without colliding with cross-load type identity. The visible behaviour matches the spec: identical (module, name, namespace) produce identical UUIDs, secret instances trigger `opm-secrets`, caller labels coexist with schema-stamped labels.

### D5. Kernel wrapper is the recommended entry point

`Kernel.SynthesizeRelease(ctx, synth.ReleaseInput) (*module.Release, error)` chains `synth.Release` into `Kernel.ProcessModuleRelease`. This mirrors how every other kernel operation is exposed — the free function lives in the helper package, the kernel method anchors it to the runtime instance. Callers that already hold a `*Kernel` get one call; callers building their own pipeline (rare) can use the helper directly.

## Risks / Trade-offs

**[R1] Schema-load failure at runtime → embed mismatch**: The schema is embedded at build time; a load failure is a programming error, not a runtime input error. Mitigation: return a wrapped error from `Binding.SchemaValue`; document that callers may treat it as fatal. A failing `task test` against the v1alpha2 binding catches the case before release.

**[R2] `sync.Once` cache assumes one `*cue.Context` per process**: If a downstream consumer constructs multiple `*Kernel`s with distinct contexts and reaches the shared binding singleton, the cached `cue.Value` is bound to the first context. Mitigation: document the contract in the `SchemaValue` docstring; the existing `kernel-runtime` Goroutine Safety Contract already steers callers toward one Kernel per goroutine, which transitively means one context.

**[R3] Caller-supplied empty `Values` produces a confusing error**: `ProcessModuleRelease` will reject the spec with a concreteness error that may not obviously point at "you forgot to pass values". Mitigation: the synth helper's docstring states the expectation; the kernel wrapper's docstring repeats it. No special-case error wrapping — surfacing CUE's own diagnostic stays closer to the failure site.

**[R4] Multiple-binding cache contention**: Calling `SchemaValue` from multiple goroutines on the same binding is safe (`sync.Once`), but the *first* call pays the load cost on the calling goroutine. Mitigation: document; consider a pre-warm helper in a future slice if measurement shows it matters. Not addressed in this slice (YAGNI).

**[R5] Interface extension is a SemVer breaking change**: Any out-of-tree `Binding` implementation breaks. Mitigation: there are none today (this is the only place we know of where users might implement it, and the docs do not advertise it as an extension point). The slice is labeled MAJOR; the CHANGELOG entry lists the new method explicitly.

## Migration Plan

1. Land the interface change (`Binding.SchemaValue`) and the v1alpha2 implementation in one commit. Both must move together to keep the binding registry valid.
2. Land `pkg/helper/synth/` in a follow-up commit so the helper builds against the new method without merge-window race conditions.
3. Land the `Kernel.SynthesizeRelease` wrapper last; it depends on the helper.
4. CHANGELOG entry under the next MAJOR. No downstream migration code needed because no consumer in-tree calls `Binding` methods directly outside the registered v1alpha2 binding.

Rollback strategy: revert the three commits in reverse order. Because each is additive on top of an empty downstream, no data migration is involved.

## Open Questions

- Should `synth.Release` accept a pre-loaded schema `cue.Value` to allow callers to pre-warm or test-inject? Defer — no consumer needs it yet; can be added without breaking the existing signature by introducing a `Release(ctx, in, ...Option)` shape later.
- Future v1alpha3 may shift `#ModuleRelease` to a different path. The synth helper looks the definition up by literal name `#ModuleRelease` — fine for now, may need a `Paths.ReleaseDefinition` entry once a second binding lands.
