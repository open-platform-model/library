## Why

The library can load a `#Module` from a **local directory** (`opm/helper/loader/file.LoadModulePackage`) but has no way to load a module **published in an OCI registry by `path@version`**. Every consumer that starts from a registry reference — the operator's `ModuleRelease` render path today, a future CLI `opm module` against a published module, the planned Crossplane composition function — must bridge that gap itself.

The operator bridged it with a synthetic *wrapper* CUE package that depends on the target module and pulls it in (`import mod "<path>"`, then references `mod`). That technique is broken three ways, all of which trace back to the wrapper, not the module:

1. **Self-reference collapse.** Embedding the module at the wrapper's package root re-evaluates `#Module`; core@v0's self-referential metadata (`modulePath: metadata.modulePath`, `version: metadata.version`) resolves to bottom and the author-set values are rejected as `field not allowed`.
2. **Missing transitive deps.** The wrapper's `cue.mod/module.cue` lists only the direct dependency, so the programmatic loader cannot resolve the target's catalog/core imports (`cannot find package opmodel.dev/catalogs/opm/resources`). The `cue` CLI auto-resolves these, which masked the problem during the operator's design.
3. **Root shape-gate mismatch.** Binding the module off-root to dodge (1) leaves the package root without `kind`/`metadata`, so the loader's shape gate fails (`expected kind "Module", found no kind field`).

This is exactly the custom OCI-acquisition / dependency-resolution plumbing that **Principle V (CUE-Native Module Resolution)** says must NOT live in downstream implementations: *"The library is the single place where CUE plumbing lives. Custom OCI fetch logic, custom dependency resolvers, and custom caches do not belong in downstream implementations."* The fix is to give the library a first-class "load a published module by path" primitive and have consumers call it.

A spike (in the operator, `experiments/moduleacquire-spike/`) confirmed the correct technique: fetch the module's own source from the registry and load it **as the main module** — exactly the situation `cue eval .` is in. Its own `cue.mod/module.cue` then drives transitive resolution, and its `kind`/`metadata` sit at the package root, so all three failures disappear with no wrapper, no field-binding trick, and no hand-rolled dependency walk.

## What Changes

This change is **additive (MINOR)**. No existing signature changes.

- **NEW `opm/helper/loader/registry`** — `LoadModulePackage(ctx, cueCtx, modPath, version string, opts) (cue.Value, apiversion.Version, error)`: fetch the published module's source from the registry via CUE's native module machinery (`mod/modconfig` → `Registry.Fetch`), then load it in-memory as the main module via `cue/load` with an `Overlay` (no temp directory, no wrapper package). Because the module is loaded as itself, its own `cue.mod/module.cue` resolves transitive deps and its `kind`/`metadata` are at the package root. Runs the **same** module shape gate as `loader/file` before returning.
- **MODIFIED `Kernel`** — `(k *Kernel) LoadModuleFromRegistry(ctx context.Context, modPath, version string) (cue.Value, error)` delegating to the new package, using the kernel's configured registry (the `registry` field + `WithRegistry` option already added by `add-platform-materialize`) and owned `*cue.Context`. Mirrors the existing `Kernel.LoadModulePackage` two-step (returns a raw `cue.Value`; callers decode via `Kernel.NewModuleFromValue`). Existing phase methods unchanged.
- **Shape-gate reuse** — extract the module shape gate and its sentinels (`ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField`) to a shared internal location so both `loader/file` and `loader/registry` enforce the identical structural contract. The sentinels stay re-exported from `loader/file` with **unchanged identity** (`errors.Is` callers unaffected — non-breaking).

Out of scope (separate work): fleshing out `opm/helper/loader/bytes` for caller-supplied raw bytes (a different input shape — Crossplane gRPC payloads, fuzzing); a registry loader for `#ModuleRelease`/`#Platform` (add when a consumer needs it — YAGNI); any kernel-held module cache (consumers rely on CUE's on-disk module cache, per Principle I).

## Capabilities

### New Capabilities

- `registry-module-loading`: loading a published `#Module` from an OCI registry by `path@version` into a `cue.Value` — CUE-native fetch, in-memory main-module load (no temp dir, no wrapper), transitive-dependency resolution via the module's own `cue.mod/module.cue`, and module shape-gate validation.

### Modified Capabilities

- `kernel-runtime`: `Kernel` gains a `LoadModuleFromRegistry` method. Additive only — existing methods and signatures are unchanged.
- `helper-packages`: a new `opm/helper/loader/registry` subpackage is added under the helper boundary, and the module shape gate is shared between `loader/file` and `loader/registry` (sentinel identity preserved).

## Impact

- **New package**: `opm/helper/loader/registry`.
- **`opm/kernel/`**: adds `LoadModuleFromRegistry` (thin wrapper, mirrors `LoadModulePackage`).
- **`opm/helper/loader/file/` + new shared internal shape package**: behavior-preserving refactor to share the shape gate; sentinels re-exported with unchanged identity.
- **Tests**: `mod/modregistrytest`-backed, mirroring `add-platform-materialize` — push a real core@v0 module that imports a catalog at an inline registry, load it, assert decoded metadata (`name`/`version`/`modulePath`) and that a catalog-importing module resolves. The Overlay-load technique and the negative `load.Config.FS` result are **verified by spike** (design.md § Research & Decisions, operator `experiments/moduleacquire-spike/`); they must be re-confirmed in-library against the pinned `cuelang.org/go` version before claiming done.
- **SemVer**: **MINOR**. Additive; `cli/` and `opm-operator/` are unaffected until they choose to call `LoadModuleFromRegistry`. Migration cost for existing callers is zero.
- **Unblocks**: `opm-operator` change `fix-moduleacquire-core-v0`, which deletes its wrapper shim and calls `Kernel.LoadModuleFromRegistry`.
- **Sequencing**: follows `add-platform-materialize` (shipped — provides the kernel `registry` field + `WithRegistry`) and `replace-embedded-schema-with-oci-loader` (shipped — the OCI substrate). Precedes the operator's `fix-moduleacquire-core-v0`.
