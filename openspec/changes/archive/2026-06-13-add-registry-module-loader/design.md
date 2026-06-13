## Context

`opm/helper/loader/file.LoadModulePackage(ctx, dir, opts)` loads a `#Module` from a local directory: `load.Instances(["."], &load.Config{Dir: dir, Env: registryEnv(opts.Registry)})` → `BuildInstance` → shape gate. There is no equivalent for a module that lives only in an OCI registry. The `add-platform-materialize` change already gave the kernel a `registry` field + `WithRegistry` option and established the CUE-native pattern for registry I/O (`mod/modconfig` resolver + `cue/load` against `Config.Env`); this change reuses that substrate for module acquisition.

The motivating failure is the operator's wrapper-shim approach (see proposal.md). The key realization, confirmed by spike: a module fails to load *only* when it is wrapped/embedded by another package. Loaded **as the main module** — its own `cue.mod/module.cue` present, its `kind`/`metadata` at the package root — it evaluates cleanly, exactly as `cue eval .` does in the module's own directory. So the library primitive must fetch the module's real source and load *that*, never synthesize a wrapper.

Constraints that shape the design:

- **Principle I (kernel neutrality / determinism)** — no `os.Setenv`, no kernel-held module cache, no hidden env lookups; registry config arrives via the kernel's `registry` field; the synthetic load root must be deterministic.
- **Principle V (CUE-native resolution)** — fetch via `mod/modconfig` + load via `cue/load`; no hand-rolled OCI client or dependency walk.
- **Principle VII (YAGNI)** — module-only loader; no release/platform registry loaders, no new cache, until a consumer needs them.

## Goals / Non-Goals

**Goals:**

- A `loader/registry.LoadModulePackage` that turns `(modPath, version)` into a `cue.Value` for the module, with correct self-referential metadata, transitive deps resolved, and the module shape gate applied.
- A thin `Kernel.LoadModuleFromRegistry` wrapper mirroring `Kernel.LoadModulePackage`'s shape (raw `cue.Value`; caller decodes via `NewModuleFromValue`).
- No temporary directory and no wrapper package: load the fetched module in memory, as itself.
- Identical structural validation to `loader/file` (shared shape gate, shared sentinels).
- Full unit-testability via `modregistrytest`, no test-only backdoor in the public API.

**Non-Goals:**

- Loading `#ModuleRelease` / `#Platform` from a registry (add when needed).
- Fleshing out `loader/bytes` (different input: caller-supplied raw bytes, not a registry path).
- Any kernel-held module cache or eager fetch at construction.
- Changing core@v0's `#Module` self-reference (a published-schema contract; the consumer adapts).

## Decisions

### D1: New package `opm/helper/loader/registry`, sibling to `loader/file`

```go
// opm/helper/loader/registry
func LoadModulePackage(
    ctx context.Context,
    cueCtx *cue.Context,
    modPath, version string,
    opts LoadOptions,
) (cue.Value, error)
```

Mirrors `loader/file.LoadModulePackage`'s return shape `(cue.Value, error)` and its `LoadOptions{Registry}` (registry override).

> **Note (resolved during implementation):** earlier drafts of this design specified `(cue.Value, apiversion.Version, error)` to mirror `loader/file`. Commit `4276ec4` (`feat(kernel)!: drop api binding dispatch…`) deleted the `opm/apiversion` package and collapsed `loader/file.Load{Module,Release,Platform}Package` to `(cue.Value, error)`. To genuinely mirror the current `loader/file`, this loader returns `(cue.Value, error)` and performs no apiVersion detection. (The library's main specs `helper-packages`/`kernel-runtime` still describe the old signature — they are stale relative to shipped code and out of scope for this change.) It takes an explicit `*cue.Context` so the built value shares the caller's (kernel's) context — the same contract `add-platform-materialize` established for cross-value `FillPath`.

**Why a new package, not `loader/file`:** `file` is filesystem-coupled by name and contract; a registry fetch is a different I/O edge. **Why not `loader/bytes`:** `bytes` is for caller-supplied raw buffers (Crossplane gRPC, fuzzing) — a different input shape; conflating "fetch from registry" with "here are the bytes" muddies both. Keeping `registry` focused mirrors the existing `file` package and respects Principle III.

### D2: In-memory load via `load.Config.Overlay`, NOT `load.Config.FS`

The module's source is fetched as a `module.SourceLoc{FS, Dir}` (D3). Load it by reading those files into a `load.Config.Overlay` keyed under a deterministic synthetic absolute root (e.g. `/opm-registry-module/<sanitized path@version>`), with `Dir` and `ModuleRoot` set to that root and `FS` left nil so the loader uses the real OS filesystem + registry/cache for the module's transitive dependencies:

```go
overlay := map[string]load.Source{}  // synthRoot/<rel> -> FromBytes(file)
cfg := &load.Config{
    Dir:        synthRoot,
    ModuleRoot: synthRoot,
    Overlay:    overlay,
    Env:        registryEnv(opts.Registry),   // deps resolve via registry/cache
}
insts := load.Instances([]string{"."}, cfg)
val := cueCtx.BuildInstance(insts[0])
```

**Spike-confirmed negative result (do not retry):** pinning `load.Config.FS` to the fetched module's `SourceLoc.FS` (the obvious approach) **fails** — with `FS` set, the loader reads *all* source through that single FS, so transitive deps in other cache directories are unreadable (`cannot find package opmodel.dev/catalogs/opm/resources`). `FS`-pinning only works for a self-contained virtual tree that already vendors the entire closure. `Overlay` (with `FS` nil) injects only the target module's files while leaving normal registry/cache dep-resolution intact. See § Research & Decisions.

**Why no temp dir:** the operator spike's Path A (copy the fetched files to a temp dir, then reuse `file.LoadModulePackage`) also works and is simpler, but it writes to disk every acquisition, must guarantee cleanup on all paths, and leaks dirs on crash. Overlay avoids all of that — preferable for a long-running controller and aligned with the library's in-memory-loading direction (`loader/bytes` doc). Both were measured at ~13 ms warm; perf is not the deciding factor (§ Research & Decisions).

### D3: Fetch with CUE-native `mod/modconfig` (Principle V)

```
modconfig.NewRegistry(&modconfig.Config{Env: registryEnv(opts.Registry)})
  → reg.Fetch(ctx, module.MustNewVersion(modPath, version))   // module.SourceLoc{FS, Dir}
```

`Fetch` downloads if necessary and returns the extracted module's source location (the modcache returns `OSDirFS(extractDir)` with `Dir: "."`). This is the same `mod/modconfig` substrate the schema loader and materialize already use — one place for CUE plumbing.

**Open (D3a):** whether to construct the `modconfig.Registry` per call or reuse one the kernel already holds for materialize. Lean toward reuse (avoid rebuilding the resolver per acquisition), but confirm the kernel exposes/holds a reusable `Registry` of the right shape; if not, construct per call from the kernel's `registry` string (cheap relative to the fetch). Resolve in implementation.

### D4: Shared module shape gate; sentinels keep their identity

`loader/file` runs `shapeGate(val, moduleSpec)` after build and exposes `ErrInvalidPackage` / `ErrWrongKind` / `ErrMissingRequiredField`. `loader/registry` MUST apply the *same* module gate so a registry-loaded module is validated identically to a directory-loaded one.

Extract the shape gate (`shapeGate`, `artifactSpec`, `moduleSpec`, the sentinels) into a shared package importable by both loaders — `opm/helper/loader/internal/shape` (Go `internal` rules allow `opm/helper/loader/file` and `.../registry` to import `.../loader/internal/shape`). Re-export the sentinels from `loader/file` as aliases (`var ErrWrongKind = shape.ErrWrongKind`) so their **identity is unchanged** and existing `errors.Is` callers keep working — this keeps the refactor non-breaking (MINOR, not MAJOR). The `helper-packages` spec's requirement that the sentinels are exposed from `loader/file` still holds.

**Alternative considered:** export `ShapeGateModule(cue.Value) error` from `loader/file` and have `registry` call it (no internal package). Rejected as the primary because it grows `loader/file`'s public surface for an internal need (Principle VII) and risks an import-direction smell; the internal-package extraction keeps both loaders thin and the gate single-sourced. Either keeps sentinel identity intact.

### D5: `Kernel.LoadModuleFromRegistry` is a thin wrapper

```go
// opm/kernel
func (k *Kernel) LoadModuleFromRegistry(ctx context.Context, modPath, version string) (cue.Value, error) {
    return registry.LoadModulePackage(ctx, k.cueCtx, modPath, version, registry.LoadOptions{Registry: k.registry})
}
```

Returns a raw `cue.Value`, exactly as the existing `Kernel.LoadModulePackage` wrapper does; callers decode with `k.NewModuleFromValue`. This keeps the kernel's two-step load→decode contract uniform across directory and registry sources.

### D6: Version input format

Accept `version` as a CUE module version string (`vX.Y.Z`) and `modPath` as the major-qualified module path (`<host>/<path>@vN`), passed straight to `module.MustNewVersion(modPath, version)` (use `NewVersion` and wrap parse errors rather than `Must*` panicking on caller input). Document the expected forms on the function godoc.

## Research & Decisions

### Overlay-vs-FS load + temp-dir-vs-memory (verified by spike, operator repo)

**Context:** D2's choice between staging the fetched module on disk vs. loading it in memory, and the non-obvious failure mode of `load.Config.FS`, needed empirical confirmation before committing the library API shape.

**Explored:** `opm-operator/experiments/moduleacquire-spike/spike_test.go`, run against the live local registry (`testing.opmodel.dev=localhost:5000`) loading the real `testing.opmodel.dev/modules/hello@v0 v0.0.2` module (a core@v0 module that imports `opmodel.dev/catalogs/opm/resources`). Two paths, shared `Fetch`:

- **Path A (temp dir):** fetch → copy `SourceLoc` files to a fresh temp dir → `Kernel.LoadModulePackage(dir)` → `NewModuleFromValue`. **Works** — decoded `name=hello version=0.0.2 modulePath=testing.opmodel.dev/modules`. Reuses the existing loader incl. its shape gate.
- **Path B (in-memory Overlay):** fetch → build a `load.Config.Overlay` at a synthetic root → `load.Instances` → `BuildInstance` → `NewModuleFromValue`. **Works** — identical decoded metadata, no temp dir.

**Confirmed findings:**

- The original bug, the transitive-dep failure, and the root shape-gate failure are ALL artifacts of the wrapper approach. Loading the module as the main module eliminates all three.
- `module.SourceLoc` from modcache is `{FS: OSDirFS(extractDir), Dir: "."}`. With `load.Config.FS` set, both `Dir` and `ModuleRoot` must point at the module root within that FS.
- **`load.Config.FS` pinned to the single fetched module FS FAILS on transitive deps** (`cannot find package opmodel.dev/catalogs/opm/resources`): the loader reads all source — including deps — through that one FS, and deps live in separate cache dirs. This drove D2 to `Overlay` (with `FS` nil).
- `load.Config.Overlay` keyed under a synthetic non-existent absolute root, with `FS` nil and `Env` carrying the registry mapping, loads the target module's files from the overlay while resolving deps normally through the registry/cache. Clean, no disk write.
- Warm latency over 20 iterations: Path A ≈ 13.9 ms/call, Path B ≈ 12.5 ms/call — essentially equal; cold cost is the shared `Fetch` and identical for both. Perf does not decide A vs B; operational cleanliness (no temp dir / cleanup / leak) favors B.

**Rationale:** Confirms D2 (Overlay, not FS; in-memory, not temp dir) and D3 (CUE-native fetch) on real infrastructure with a real catalog-importing module.

**Re-verified in-library during implementation (resolved):**

- The Overlay load + `Fetch` flow was re-confirmed against the library's pinned `cuelang.org/go v0.17.0-alpha.1` and a `modregistrytest`-served module (a real core@v0 `#Module` importing an inline `#Catalog`). `modconfig.NewRegistry(...).Fetch(...)` returns a usable `SourceLoc`; the Overlay load decodes correct `metadata.{name,modulePath,version}` and resolves the transitive catalog dep. See `opm/helper/loader/registry/module_test.go` (`TestLoadModulePackage_HappyPathAndTransitiveDeps`) and the kernel-level `TestKernel_LoadModuleFromRegistry`.
- The negative `load.Config.FS` result is pinned by `module_internal_test.go` (`TestOverlayResolvesDepsButFSPinningFails`): Overlay (FS nil) resolves the catalog dep, while FS-pinning fails with `cannot find package`. In-library the *first* unreadable import under FS-pinning is `opmodel.dev/core@v0` (also outside the pinned FS), not the catalog — same footgun, slightly different first error. A load-bearing comment in `module.go` cites this so the approach is not "simplified" to FS-pinning.
- D4's shared shape gate was implemented in `opm/helper/loader/internal/shape` and exercised for both wrong-kind (`ErrWrongKind`) and missing-field (`ErrMissingRequiredField`) registry loads, asserting `errors.Is` reaches the `loader/file`-exported (re-exported, identity-preserved) sentinels.
- D3a resolved: the kernel exposes no reusable `modconfig.Registry`, so the loader constructs one per call from the registry string (cheap relative to `Fetch`; mirrors materialize's per-call `NewResolver`).
- Synthetic-overlay-root behavior confirmed on Linux (the CI/dev target). Portability across other OSes remains exercised only there.

## Risks / Trade-offs

- **`load.Config.Overlay` synthetic-root behavior is subtle** → the `FS`-pin footgun is recorded (D2); add a test asserting a catalog-importing module resolves through Overlay, and a comment in the loader citing the negative `FS` result so it is not "simplified" to `FS`-pinning.
- **Shape-gate extraction could break `errors.Is` callers** → re-export sentinels as aliases with unchanged identity (D4); add a test asserting `errors.Is(err, loaderfile.ErrWrongKind)` still holds for a wrong-kind registry load. Keep the extraction behavior-preserving (MINOR).
- **CUE version drift** → the spike used `v0.17.0-alpha.1`; `load.Config.FS`/`Overlay` semantics are evolving. Re-verify in-library; pin findings to the library's CUE version.
- **Per-call registry construction cost** → bounded by the `Fetch` (which dominates); reuse the kernel's resolver if cleanly available (D3a), else accept per-call construction.
- **Determinism (Principle I)** → synthetic root derived deterministically from `path@version`; no `os.Setenv`; registry mapping plumbed via `Config.Env` only.

## Migration Plan

Additive — no caller migration. `cli/` and `opm-operator/` keep compiling unchanged; they adopt `LoadModuleFromRegistry` when ready (the operator's `fix-moduleacquire-core-v0` is the first adopter and deletes its wrapper shim). Add a `MIGRATIONS.md` entry under *Unreleased — next MINOR* introducing the new package + kernel method (additive) and noting the sentinel re-export keeps identity. Rollback: revert the new package + method; `loader/file` behavior is unchanged by the (behavior-preserving) shape-gate extraction.

## Open Questions

- **D3a:** Reuse a kernel-held `modconfig.Registry` (shared with materialize) or construct one per `LoadModuleFromRegistry` call? Lean reuse; confirm the kernel exposes a suitable `Registry` during implementation.
- **Q2:** Should `LoadModuleFromRegistry` return `(cue.Value, error)` (matches `LoadModulePackage`) or `(*module.Module, error)` (one-step convenience)? Lean `cue.Value` for contract uniformity; revisit if every caller immediately decodes.
- **Q3:** Does the shared shape gate belong in `opm/helper/loader/internal/shape` (D4 primary) or as an exported `ShapeGateModule` on `loader/file`? Decide at implementation; both preserve sentinel identity.
