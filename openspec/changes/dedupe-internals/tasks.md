# Tasks: dedupe-internals

## 1. opm/internal/cueenv

- [x] 1.1 Create `opm/internal/cueenv` with `Override(registry, cacheDir string) []string` (nil when both empty; otherwise a copy of `os.Environ()` with each non-empty override replacing an existing `KEY=` entry or appended) and a table test covering empty, replace and append for both variables.
- [x] 1.2 Replace the four copies with `cueenv.Override`: `opm/helper/loader/file/instance.go` (`registryEnv`, also used by `module.go`, `platform.go`, `build.go`), `opm/helper/loader/registry/module.go` (`registryEnv`, both the `modconfig` and the `load.Config` use), `opm/internal/renderstage/stage.go` (`RegistryEnv`, and its caller in `opm/kernel/render.go`), `opm/schema/loader.go` (`mergeEnv`); delete the copies and move `TestRegistryEnv`, the registry loader's internal env test and the schema `mergeEnv` cases into `cueenv`.

## 2. opm/internal/sourcetree

- [x] 2.1 Create `opm/internal/sourcetree` with `PackageName`, `OverlayFromDir`, `OverlayFromFS`, `SyntheticRoot`, `WriteTo`, `ReadFile`, `Bytes` (`Bytes` and `SyntheticRoot` moved verbatim); `.cue`-suffix filter in both walkers; tests: package-name cases (zero, one, two clauses; overlay and on-disk), walker filter (non-CUE file excluded, nested `cue.mod/pkg/*.cue` included, synthetic-root rekeying), `WriteTo` round trip, the `Bytes` cases from `renderstage/modfile_test.go:110-126`.
- [x] 2.2 `opm/kernel/acquire.go`: `loadInstanceWithValues` computes `sourceForDir` first, calls `sourcetree.PackageName(src)` and `sourcetree.OverlayFromDir(src.Root)`; delete `overlayForRoot` and `packageNameOfDir`.
- [x] 2.3 `opm/helper/loader/registry/module.go`: build the overlay with `sourcetree.OverlayFromFS(loc.FS, loc.Dir, sourcetree.SyntheticRoot(modPath, version))`, keep the empty-overlay refusal wrapping `shape.ErrInvalidPackage` at the call site; delete `opm/helper/loader/internal/stage`; add a loader test that a fixture archive with a non-CUE file yields an overlay without it.
- [x] 2.4 `opm/internal/renderstage`: `serveDir` becomes on-disk root or `sourcetree.WriteTo`; `ReadModFile` uses `sourcetree.ReadFile`; `packageName` callers use `sourcetree.PackageName`; delete `packageName`, `packageFiles`, `readSourceFile`, `sourceBytes` and their now-moved tests.
- [x] 2.5 `renderstage.Staged` becomes `{Dir, Skew}`; `Stage` drops the struct assembly; `stage_test.go:114` moves onto the `Promotion` value in `TestPromote_*`, `:128-129` onto the import-path builder's test.

## 3. Identity convention

- [x] 3.1 `opm/helper/loader/registry/module.go`: `verifyModuleIdentity` compares `declaredPath == modPath` only; delete `parentPath`; collapse the two carve-out tests at `module_test.go:134-160` into one that asserts a major-free declaration fails with `IdentityError{Field: "path"}` carrying both paths; update the D11 comment block.
- [x] 3.2 `opm/helper/synth/render.go`: `moduleImportPath` returns `m.Metadata.ModulePath`; delete `moduleSnakeName` and the 0003-convention doc; remove any synth test that constructs a major-free module path.
- [x] 3.3 Re-home the retired v1 synth integration scenarios on core-v2 fixtures in `opm/kernel/synth_schema_test.go`: bad name fails at unification, empty values not backfilled from `debugValues`, schema-derived fields (uuid, components, labels, annotations, namespace divergence), synth-versus-authored instance-value parity.
- [x] 3.4 `opm/internal/registrytest`: delete the now-unset `CoreVersion` fixture fields and `coreVersionOr`; every fixture declares `DefaultCoreVersion`.

## 4. Quoting and test task

- [x] 4.1 `opm/helper/platformmodule/generate.go`: delete `quote`, use `literal.String.Quote` at both sites; `opm/helper/synth/render.go`: replace every `%q` of a CUE literal (`name`, `namespace`, map keys and values, the two import paths) with `literal.String.Quote`; golden tests unchanged.
- [x] 4.2 `Taskfile.yml` `test`: first command excludes `opm/kernel` and `opm/internal/renderstage` via `go list ./... | grep -v`; race command unchanged; `task test` still runs every package at least once.

## 5. Docs and gates

- [x] 5.1 `CLAUDE.md` § Repository Layout: add `opm/internal/cueenv` and `opm/internal/sourcetree`, remove `helper/loader/internal/stage`, note the single-sourced env override; `opm/helper/loader/registry` package doc drops the "mirrors file.registryEnv" sentence; `opm/helper/synth/render.go` and `registry/module.go` doc comments drop the v0/v1 convention text.
- [x] 5.2 `task fmt`, `task vet`, `task lint`, `task test` green; `go test -race ./opm/kernel/... ./opm/internal/renderstage/...` green; `go build ./...` in `../cli` and `../opm-operator` against a temporary `replace` to the working tree, then revert the replace.
