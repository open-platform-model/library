## 1. De-risking spike (re-verify in-library; settle before building)

- [x] 1.1 Stand up a `modregistrytest`-backed test (mirror `add-platform-materialize`'s substrate, incl. the read-only-cache chmod-walk cleanup helper): push a real core@v0 `#Module` that imports a small `#Catalog`/resource package, plus the catalog + a stand-in `core` at fixed versions. Confirm `modconfig.NewRegistry(...).Fetch(ctx, module.MustNewVersion(path, version))` returns a usable `SourceLoc`.
- [x] 1.2 Confirm the **Overlay** load (D2): read the `SourceLoc` files into a `load.Config.Overlay` at a synthetic root with `FS` nil + `Env` registry mapping; `load.Instances(["."], cfg)` → `BuildInstance` resolves the catalog import and decodes correct `metadata.{name,modulePath,version}`. Record the outcome in design.md § Research & Decisions. Also assert the negative result (an `FS`-pinned variant fails on the transitive dep) so the footgun is pinned by a test.

> Note: `opm/apiversion` was deleted (commit `4276ec4`) and `loader/file` now returns `(cue.Value, error)`. This change mirrors current `loader/file` — no apiVersion detection, signature `(cue.Value, error)`.

## 2. Shared shape gate (refactor, behavior-preserving)

- [x] 2.1 Extract `shapeGate`, `artifactSpec`, `moduleSpec`, and the sentinels (`ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField`) from `opm/helper/loader/file` into `opm/helper/loader/internal/shape` (or resolve Q3 to an exported `ShapeGateModule`).
- [x] 2.2 Re-export the sentinels from `loader/file` as identity-preserving aliases (`var ErrWrongKind = shape.ErrWrongKind`, etc.). Keep `releaseSpec`/`platformSpec` gating wired through the shared package.
- [x] 2.3 Confirm `loader/file`'s existing tests pass unchanged, including any `errors.Is(err, loaderfile.ErrWrongKind)` assertions (add one if absent).

## 3. registry loader package

- [x] 3.1 Create `opm/helper/loader/registry` with `LoadOptions{Registry string}` (same shape as `loader/file.LoadOptions`).
- [x] 3.2 Implement `LoadModulePackage(ctx, cueCtx, modPath, version, opts) (cue.Value, error)`: fetch via `mod/modconfig` (D3), build a deterministic synthetic-root `Overlay` from the `SourceLoc`, `load.Instances` + `cueCtx.BuildInstance`, run the shared module shape gate (D4). Mirror current `loader/file` (which returns `(cue.Value, error)` with no apiVersion detection). Use `module.NewVersion` and wrap parse errors (do not `Must*` on caller input).
- [x] 3.3 Add a load-bearing comment citing the spike's negative `load.Config.FS` result so the Overlay approach is not "simplified" to FS-pinning.
- [x] 3.4 Resolve D3a: reuse a kernel-held `modconfig.Registry` if cleanly available, else construct per call from the registry string.

## 4. kernel wiring

- [x] 4.1 Add `(*Kernel) LoadModuleFromRegistry(ctx, modPath, version string) (cue.Value, error)` delegating to `registry.LoadModulePackage` with `k.CueContext()` and the kernel's configured `registry` (D5).
- [x] 4.2 Confirm existing phase/loader method signatures are unchanged (additive slice).

## 5. Tests (modregistrytest-backed)

- [x] 5.1 Happy path: load a published core@v0 module by `path@version`; assert decoded `metadata.{name,modulePath,version}` equal the author-set values (the fields that regressed under the wrapper approach) and no `field not allowed`.
- [x] 5.2 Transitive deps: the module imports a catalog/resource package; assert it resolves through the Overlay load (the case that broke the wrapper).
- [x] 5.3 Shape gate: a registry artifact whose `kind != "Module"` returns an error wrapping `ErrWrongKind`; a module missing a required identity field wraps `ErrMissingRequiredField`. Assert `errors.Is` reaches the `loader/file`-exported sentinels.
- [x] 5.4 Unresolvable `path@version` surfaces a wrapped load/fetch error; inputs not mutated; no process-env mutation.

## 6. Docs + validation gates

- [x] 6.1 `MIGRATIONS.md`: add an *Unreleased — next MINOR* entry for `opm/helper/loader/registry` + `Kernel.LoadModuleFromRegistry` (additive), noting the shape-gate sentinel re-export preserves identity.
- [x] 6.2 Add a short note to `library/CLAUDE.md` (loader/registry exists; "load a published module by path" lives in the library, not consumers — Principle V).
- [x] 6.3 `task fmt`.
- [x] 6.4 `task vet`.
- [x] 6.5 `task lint`.
- [x] 6.6 `task test`.
