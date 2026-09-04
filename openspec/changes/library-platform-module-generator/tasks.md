# Tasks: library-platform-module-generator

## 1. opm/helper/platformmodule (lift)

- [x] 1.1 Create `opm/helper/platformmodule/doc.go` and `generate.go`: `Entry`, `Dep`, `Input{Name, Type, ModulePath, Entries, Deps}`, `Files`, `Generate`, `Roots(entries, ...RootOption)` with `WithCoreVersion`; core pin default from a new `schema.DefaultSchemaVersion()` accessor; `platform.cue`/`cue.mod/module.cue` rendering byte-identical to the operator's (positional `cat<N>` aliases, sorted entries, modfile canonical format, no default markers).
- [x] 1.2 Port `generate_test.go` from the operator with its golden expectations; add cases for `ModulePath` input, the default and explicit core pin, and every refusal (empty name/type, duplicate entry, empty path/version, double pin, closure missing core or an entry).
- [x] 1.3 Create `closure.go`: `ModFileSource`, `RegistryConfig{Registry, ClientType, Env}`, `NewRegistry(RegistryConfig)`, `Closure(ctx, src, roots)` (breadth-first, max version per path, roots participate, local replacements skipped, context cancellation, unpublished build error naming path and version).
- [x] 1.4 Port `closure_test.go` (fixture module-file graph, no registry): transitive pin, root wins, unpublished refusal, cancellation.
- [x] 1.5 Create `write.go`: `Files.WriteTo(dir)` (parents created, path escape refused, 0o644 files) + test.
- [x] 1.6 `build_test.go`: in-process `registrytest` serving core and a catalog whose module file requires a transitive dependency; derive the closure, write, `AcquirePlatformFromDir`, assert the stamped `version` readout and the transitive pin (the operator's `registry_test` scenario without GHCR).

## 2. opm/kernel acquire option

- [x] 2.1 `opm/kernel/acquire.go`: `AcquireOption` + `WithValues(sources ...Source)`; on-disk path unchanged when no option is given.
- [x] 2.2 Overlay construction: read the module root's files into a `load.Source` overlay (same inclusion rules `cue/load` applies), unify the values sources as `ValidateConfigDetailed` does, render `values.cue` with the package's own package clause through the renderer shared with `synth` (extract; no duplicate), build with `loaderfile.BuildInstanceOverlayAt`, return `Source{Root, Pkg, Overlay}`.
- [x] 2.3 Tests: layered values merge (two sources, unset field), `Package` equals the no-option acquire minus `values`, caller directory unchanged, conflicting source fails with attributable positions, and a `Render` of the layered instance reflecting the values (fixture under `testdata/render/`).

## 3. Docs

- [x] 3.1 `opm/helper/doc.go` (new subpackage entry), `CLAUDE.md` layout table (`helper/platformmodule/`), `docs/getting-started.md` (platform from coordinates: entries → closure → write → `AcquirePlatformFromDir`; instance file with extra values).
- [x] 3.2 `openspec/specs` deltas are the contract; no ADR (no architectural decision beyond ADR-006, which this reuses).

## 4. Verification

- [x] 4.1 `task fmt`, `task vet`, `task lint`, `task test` green; `go test -race ./opm/kernel/... ./opm/helper/platformmodule/...`.
- [x] 4.2 Diff the helper's generated files for the operator's sample Platform input against `opm-operator/internal/platformmodule` output at the current operator `main` (byte-equal), so `operator-platformmodule-lift` can rely on it. Result (operator `main` at `ab2d9f5`, closure derived against GHCR by both generators: identical, `cue.dev/x/k8s.io@v0 v0.10.0`, `k8s@v1 v1.0.0-alpha.2`, `opm@v4 v4.0.1`, `core@v2 v2.0.0-alpha.7`): `cue.mod/module.cue` byte-equal; `platform.cue` byte-equal from line 2 on, line 1 being the generator attribution header (design § "The platform.cue header names no frontend").
