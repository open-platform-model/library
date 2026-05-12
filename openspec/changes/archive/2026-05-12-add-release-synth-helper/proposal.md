## Why

Today the only way to obtain a `#ModuleRelease` artifact is to load a hand-authored `release.cue` from disk via `pkg/helper/loader/file/`. Downstream consumers that need to deploy a module from in-memory identity inputs — the CLI's anticipated `opm release deploy <module>` command and the `opm-operator` reconciler turning a `ModuleRelease` CR into a kernel artifact — cannot produce a release without writing a stub file first. A first-class synthesis path that unifies `(Module, name, namespace, values, labels, annotations)` against the embedded `#ModuleRelease` schema removes the file detour while preserving every schema-driven derivation (UUID, components fan-out, label stamping).

## What Changes

- Add `pkg/helper/synth/` subpackage with `synth.Release(ctx, ReleaseInput) (cue.Value, error)` that builds a `#ModuleRelease` CUE value by unifying inputs against the schema definition loaded from the version binding's embedded filesystem.
- Add `Binding.SchemaValue(*cue.Context) (cue.Value, error)` method to the `pkg/api.Binding` interface. Each binding lazily loads its embedded schema package and caches the resulting `cue.Value`. The v1alpha2 binding implements it over its existing `EmbeddedSchema() fs.FS`.
- Add `Kernel.SynthesizeRelease(ctx, synth.ReleaseInput) (*module.Release, error)` wrapper that chains `synth.Release` into `Kernel.ProcessModuleRelease`, returning a fully constructed `*module.Release` in one call.
- The synthesis path never falls back to `Module.debugValues`. `ReleaseInput.Values` is caller-supplied; an empty value is filled as-is and concreteness is enforced downstream by `ProcessModuleRelease`. Frontends that want a debug-values overlay layer it explicitly on the caller side.
- **BREAKING (`pkg/api`)**: Adding `SchemaValue` to the `Binding` interface breaks any out-of-tree binding implementation. In-tree there is only the v1alpha2 binding, which we update. MAJOR per Principle VI.

## Capabilities

### New Capabilities
- `release-synthesis`: the `pkg/helper/synth/` package — building a `#ModuleRelease` artifact from in-memory inputs by unifying against the embedded schema, and the kernel-anchored wrapper that drives it through `ProcessModuleRelease`.

### Modified Capabilities
- `api-version-dispatch`: extends the `Binding` interface with `SchemaValue`. Bindings must expose their loaded schema package as a `cue.Value` so callers (synth helper, future validation helpers) can unify against schema definitions without re-running `load.Instances` themselves.
- `kernel-runtime`: adds `Kernel.SynthesizeRelease` as the recommended entry point for in-memory release construction, mirroring the existing `Kernel.LoadReleaseFile` / `Kernel.ProcessModuleRelease` surface.

## Impact

**Affected packages**

- `pkg/api/` — new method on `Binding` interface; bumps the SemVer surface.
- `pkg/api/v1alpha2/` — implements `SchemaValue` (lazy `sync.Once` cache keyed on the first `*cue.Context`).
- `pkg/helper/synth/` — new package.
- `pkg/kernel/` — new `SynthesizeRelease` method (additive on `Kernel`).

**Affected downstream consumers**

- CLI: future `opm release deploy <module>` or equivalent command. No code changes in this slice; the helper unblocks the follow-up CLI slice.
- `opm-operator`: future controller reconciliation path. No code changes in this slice.

**SemVer classification**

- MAJOR overall, driven by the `pkg/api.Binding` interface extension. The kernel and synth additions are themselves additive but ride the major bump.

**Out of scope**

- Bytes loader (`pkg/helper/loader/bytes/`) implementation.
- CLI command wiring.
- Operator reconciler wiring.
- Multi-release synthesis (`#ModuleReleaseMap`).
- v1alpha3 binding work.
