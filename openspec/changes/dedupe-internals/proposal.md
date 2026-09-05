## Why

The same three helpers are written two to four times inside `opm/`, and one copy has already drifted. The cause is Go's `internal` rule, not carelessness: `opm/helper/loader/internal/stage` is visible only under `opm/helper/loader/`, so `opm/kernel` and `opm/internal/renderstage` each grew their own overlay walker and package-clause reader, and every package that touches `cue/load` grew its own `CUE_REGISTRY` env override. Separately, the registry loader and the synth helper both carry the core-v0/v1 module-address convention (major-free `metadata.modulePath` plus a snake-case leaf), a branch no module the v2 kernel can render ever takes: core v2's `#ModulePathType` requires the `@vN` suffix.

This is slice 3 of the eight-slice simplification plan reviewed on 2026-09-05, re-cut after exploration: the items that vanish for free when slice 2 folds the loaders into the kernel (the two file loaders that bypass `buildAndShapeGate`, the `validate.go` alias file) are left for that fold; the `runValidate` double unify goes into `cut-dead-surface` task 1.1, which already rewrites that function; and the `Source.Overlay` type change that would delete the `load.Source` reflection is a public-field break that belongs to slice 2's wave.

**Scope statement (Principle VIII).** Two new `opm/internal` packages that existing code calls, one dead branch deleted, three one-line cleanups. No public Go signature changes. One behaviour change on a helper-tier loader, called out below.

## What Changes

**New `opm/internal/cueenv`** (internal, no API change):

- `Override(registry, cacheDir string) []string`: nil when both are empty (so `load.Config` and `modconfig.Config` read the process environment unchanged), otherwise a copy of the process environment with `CUE_REGISTRY` and `CUE_CACHE_DIR` replaced or appended. Replaces `loaderfile.registryEnv`, `loaderregistry.registryEnv` (the drifted copy: it returned a full copy on empty), `renderstage.RegistryEnv` and `schema.mergeEnv`. Every load site can now honour a cache-dir override; only the schema loader could before. The kernel gains no new knob here (slice 2).

**New `opm/internal/sourcetree`** (internal, no API change):

- `PackageName(src *module.Source) (string, error)`, `OverlayFromDir(root string)`, `OverlayFromFS(fsys fs.FS, dir, keyRoot string)`, `WriteTo(dir string, src *module.Source) error`, `ReadFile(src *module.Source, path string) ([]byte, error)`, `Bytes(load.Source) ([]byte, error)`, `SyntheticRoot(modPath, version string) string`.
- Both overlay walkers apply one filter: `.cue` files only, the module's own `cue.mod/module.cue` included. This is what `cue/load` reads and what the kernel's on-disk walker already does; nothing in the workspace uses `@embed`.
- Replaces `opm/kernel/acquire.go` `overlayForRoot` and `packageNameOfDir`; `opm/helper/loader/internal/stage` (package deleted; its `SyntheticRoot` moves); `opm/internal/renderstage` `packageName`, `packageFiles`, `serveDir`'s overlay branch, `readSourceFile`, `sourceBytes`. `Bytes` keeps the reflection over `load.Source` verbatim until slice 2 changes the field type.
- **BREAKING (behaviour, helper tier)** The registry loader's staged overlay carries the fetched module's `.cue` files only; today it carries every file in the module zip. Observable only through `Module.Source.Overlay`; no consumer reads non-CUE entries (the cli's scaffold copies from its own fetch).

**Identity convention (BREAKING, `refactor(loader)!:`):**

- **BREAKING** `verifyModuleIdentity` compares `metadata.modulePath` with the fetched path by strict string equality in every case; the major-free carve-out and `parentPath` are deleted. A module published against core v0/v1 (major-free `modulePath`) now fails `Kernel.AcquireModuleFromRegistry` with `oerrors.IdentityError{Field: "path"}`. No core-v2 module is affected: `#ModulePathType` (`core/src/types.cue:67`) requires the suffix. The v2 kernel could not render such a module anyway; the only production caller is the cli's scaffold, which fetches v2 templates.
- `synth.moduleImportPath` returns `Metadata.ModulePath` unchanged; the major-free composition branch and `moduleSnakeName` are deleted.

**Small cleanups (internal):**

- CUE string literals in `opm/helper/platformmodule/generate.go` and `opm/helper/synth/render.go` are produced by `cuelang.org/go/cue/literal.String.Quote`, the quoting the render glue already uses; the hand-rolled `quote` and the Go `%q` sites go.
- `renderstage.Staged` shrinks to `{Dir, Skew}`, the two fields the kernel reads; `Instance`, `Platform`, `Promotion`, `InstanceImport`, `PlatformImport` were read only by one test whose assertions the unit tests for `Promote` and the import-path builder already make.
- `Taskfile.yml` `test`: the plain `go test` pass excludes `opm/kernel` and `opm/internal/renderstage`, which the `-race` pass runs anyway. About 4 s per run.

**Deferred on purpose:** `loader/file/module.go` and `platform.go` bypassing `buildAndShapeGate` and the `validate.go` alias file (slice 2 fold); `runValidate` double unify (`cut-dead-surface` task 1.1); `module.Source.Overlay` as bytes (slice 2); the three `major()` copies in test helpers (slice 8).

## SemVer classification

MAJOR on the alpha line (Principle VI: a behaviour change reachable through `Kernel.AcquireModuleFromRegistry`), carried by the identity commit alone; every other commit is an internal refactor (PATCH). Pre-GA, so no migration fragment (ADR-004). Downstream migration cost: zero source changes in `cli` and `opm-operator`.

## Affected packages and downstream consumers

- New: `opm/internal/cueenv`, `opm/internal/sourcetree`. Deleted: `opm/helper/loader/internal/stage`.
- Touched: `opm/kernel/acquire.go`, `opm/helper/loader/file/{instance,module,platform,build}.go`, `opm/helper/loader/registry/module.go`, `opm/helper/synth/render.go`, `opm/helper/platformmodule/generate.go`, `opm/internal/renderstage/{stage,modfile}.go`, `opm/schema/loader.go`, `Taskfile.yml`, `CLAUDE.md` layout table.
- `cli`, `opm-operator`: no source change. `catalog_opm`, `modules`, `core`: no impact.
- Sequencing: lands after `cut-dead-surface`, which edits `acquire.go` and `registry/module.go` too.

## Complexity justification

Net deletion of roughly 150 lines: two new packages of about 190 lines replace about 340 lines across seven files, plus the identity branch, its helper and two test cases. No new abstraction beyond a filesystem interface the fetched-module walker already needed.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `registry-module-loading`: the identity requirement drops the major-free carve-out (strict equality for every module; an older-line declaration is refused with the typed identity error); the in-memory load requirement states that the staged overlay carries the module's `.cue` files.
