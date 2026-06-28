## Why

`synth.Instance` fabricates a `cue.mod/module.cue` that declares only `{core, module}` and never tidies it. CUE's `load.Instances` does **not** re-derive a dependency's transitive closure at build time — the main module must already declare the full closure (that closure is computed by `cue mod tidy` at module *publish* time and written into the module's own `cue.mod/module.cue`). So the moment a module imports a catalog subpackage (e.g. `opmodel.dev/catalogs/opm/blueprints/workload`), synthesis fails with `cannot find module providing package …`, even against a correctly-configured registry. This is the verified root cause of library#31 (deterministic, non-k8s reproduction confirmed).

The sibling capability `registry-module-loading` already solved this exact problem: it loads the module **as the main module** so the module's own (already-tidied) `cue.mod/module.cue` drives transitive resolution. `synth.Instance` should construct the instance the same way instead of fabricating a deps-incomplete main module.

## What Changes

- `synth.Instance` constructs the instance by overlaying the instance source (and rendered values) **into the fetched module's own source tree** (under a synthetic subpath), reusing the module's published, already-tidied `cue.mod/module.cue` verbatim. CUE's loader resolves the full transitive closure — no fabricated deps list, nothing to keep in sync with `cue mod tidy`.
- The fabricated-`module.cue` path is removed: `renderModuleFile` (and its `corePath`/`major`/`normalizeVersion`/`moduleImportPath` dep-list helpers) are deleted. The instance source no longer imports the module by registry `path@version`; it references the module's own package, which is local to the build.
- The registry module loader surfaces the **staged module source** (the deterministic overlay + synthetic root it already builds) so the acquire step can hand it to synth, rather than discarding it after building the `cue.Value`. The module's source travels with the acquired `*module.Module` (or an adjacent typed result).
- `Kernel.SynthesizeInstance` / `Kernel.LoadModuleFromRegistry` thread the staged source from acquire into synth.
- **Not** done: no vendoring/forking of CUE's internal `modload.Tidy` (it is in `cuelang.org/go/internal/` and unimportable); no Go-side reimplementation of MVS/tidy. The module is already tidied at publish time — we reuse that artifact.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities

- `instance-synthesis`: the construction requirement changes from "fabricate a `cue.mod/module.cue` declaring `{core, module}` deps + import the module by `path@version`" to "synthesize the instance inside the module's own staged source root, reusing the module's tidied `cue.mod/module.cue`." The public `Instance(ctx, in InstanceInput) (cue.Value, error)` signature and `InstanceInput` field set stay the same; for every input that succeeds today the result stays observably equivalent (same `metadata.uuid`, `components`, labels), and inputs that previously failed on transitive catalog imports now succeed.
- `registry-module-loading`: additively surface the staged module source (overlay + synthetic root, or `module.SourceLoc`) the loader already builds, so synthesis can reuse it without a second fetch and without re-deriving dependencies.

## Impact

- **Affected `opm/` packages:** `opm/helper/synth` (construction rewrite; `render.go` shrinks), `opm/helper/loader/registry` (surface staged source), `opm/module` (carry source on `*module.Module`, or a new typed acquire result), `opm/kernel` (`SynthesizeInstance` / `LoadModuleFromRegistry` plumbing).
- **Downstream consumers:** `opm-operator` render path (`moduleacquire.Acquire` → `SynthesizeInstance`) and the CLI consume `LoadModuleFromRegistry` / `SynthesizeInstance`; if the acquire return shape changes, both update. Document in `MIGRATIONS.md`.
- **SemVer:** target **MINOR** (additive: synth resolves strictly more inputs; loader gains a source-surfacing path). Becomes **MAJOR** only if `LoadModuleFromRegistry`'s existing return signature is changed rather than extended — design.md decides between "extend `*module.Module`" (MINOR) and "new return type" (MAJOR).
- **Tests:** the `instance-synthesis` spec's "Imported-module render coverage" requirement gains a case where the module imports a catalog subpackage (the exact #31 shape) and renders end-to-end — this is the regression guard that the old fabricated-deps path could not satisfy.
