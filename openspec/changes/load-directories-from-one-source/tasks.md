# Tasks: load-directories-from-one-source

Worktree `library/.claude/worktrees/load-directories-from-one-source`, branch
`refactor/load-directories-from-one-source` (from `origin/main`). Seed `.cue-cache` by copying
the main checkout's (`cp -a`), never symlinking. Every command runs inside the worktree with
the registry env exported on two lines and a private `TMPDIR` for the tests:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run from a known cross-process
cache race; rerun `go test ./opm/helper/platformmodule -count=1` alone before treating it as a
finding. Commit bodies never start a line with `word(` and carry no bare at-sign (write "the
embed attribute"). The only trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`.
Design decisions are LS1 to LS7 in design.md. Code and test comments cite no LSn, task or
section number (openspec/config.yaml and AGENTS.md bar design-local ids in comments).

## 1. Pin today's directory acquisition behaviour (kernel, loader, sourcetree tests; design LS2, LS5, LS6)

Tests only; every one passes on `origin/main` before any code moves.

- [x] 1.1 `opm/kernel/acquire_test.go`: one table test,
      `TestKernel_AcquireFromDir_PathErrors`, over the four directory verbs (module, catalog,
      platform, instance with no values) plus instance with one values source. For a missing
      path assert the exact text `accessing <label> directory "<abs path>": ` as a prefix and
      `errors.Is(err, fs.ErrNotExist)`, and that the message does not contain
      `reading module tree`; for a regular file assert `<label> path "<abs path>" is not a
      directory`. A nil artifact in every case. Verify:
      `go test ./opm/kernel -run TestKernel_AcquireFromDir_PathErrors -count=1` green.
- [x] 1.2 `opm/kernel/acquire_test.go`: `TestKernel_AcquireModuleFromDir_EmbedsNonCUEFile`
      writes a module root (`cue.mod/module.cue` at language `v0.17.0`, a `module.cue`
      declaring the embed extern and a `data` field embedding `data.json`, and `data.json`)
      and a `sub` package (its own package name) embedding `sub/d.json`. Acquire the root and
      `sub`; assert each `Package` carries its embedded value and each `Source.Overlay` holds
      only `.cue` keys (design LS5). Verify: the test is green.
- [x] 1.3 `opm/internal/sourcetree/sourcetree_test.go`: extend
      `TestOverlayFromDir_CueFilesOnly` to assert the absent-root error starts with
      `reading module tree `; add `TestOverlayFromDir_Symlinks` (skip on Windows): a symlinked
      `.cue` file is read through the link under its own key, and a symlinked directory is
      not descended into (design LS6). Verify: `go test ./opm/internal/sourcetree -count=1`
      green.
- [x] 1.4 `opm/internal/loader/load_test.go`: `TestLoadDir_MissingPath` and
      `TestLoadDir_NotADirectory` also assert the full message prefix (`accessing platform
      directory "` and `platform path "`). Verify: `go test ./opm/internal/loader -count=1`
      green.
- [x] 1.5 `task check` green, then commit
      `test: pin directory acquisition errors and embedded files` (scope-neutral: the commit
      spans the kernel, loader and sourcetree packages).

## 2. Read a directory overlay through os.DirFS (sourcetree; design LS6)

- [x] 2.1 `opm/internal/sourcetree/sourcetree.go`: `OverlayFromDir(root)` returns
      `OverlayFromFS(os.DirFS(root), ".", root)` wrapped in `reading module tree %s: %w`;
      the hand-written `filepath.WalkDir` goes. Update its doc comment to say it is
      `OverlayFromFS` over the directory; drop imports nothing else uses. Rewrite the package
      doc's filter paragraph (its claim that nothing uses the embed attribute is false once
      1.2 lands): non-CUE files such as embedded data are not carried in the overlay and are
      read through the host layer when `Root` is the real directory. Correct the
      `acquire_test.go` comment "cue/load does not read it" the same way. Verify:
      `go test ./opm/internal/sourcetree ./opm/internal/renderstage -count=1` green, and the
      section 1 tests unchanged.
- [x] 2.2 `task check` green, then commit
      `refactor(sourcetree): read a directory overlay through os.DirFS`.

## 3. LoadDir takes a Source and load options (loader, synth; design LS1, LS2)

No behaviour change: the kernel still builds module and catalog from disk in this section.

- [x] 3.1 `opm/internal/loader/load.go`: add `Options{Env []string}` and
      `CheckDir(dir string, spec ArtifactSpec) error` (today's two messages, LS2). Change
      `LoadDir` to `LoadDir(cueCtx *cue.Context, src *module.Source, opts Options, spec
      ArtifactSpec)` with the two modes of the LS1 table; on-disk mode calls `CheckDir` on
      `Root` joined with `Pkg`. A nil `src` or empty `Root` returns the plain error
      `source carries no module root` (no sentinel wrap). Rewrite the `LoadDir` doc comment for the new arguments
      (only the lines the signature makes false). Doc comments on `Options` and `CheckDir`.
- [x] 3.2 `opm/internal/loader/registry.go`: `FetchArtifact` builds
      `src := &opmmodule.Source{Root: synthRoot, Overlay: overlay}` once, loads it with
      `Options{Env: env}`, and returns that same pointer.
- [x] 3.3 `opm/internal/synth/instance.go`: `Instance` builds
      `&module.Source{Root: moduleRoot, Pkg: synthPkgDir, Overlay: overlay}` once, loads it
      with `loader.Options{Env: in.Env}`, and returns that same pointer.
- [x] 3.4 `opm/kernel/acquire.go`: the six `LoadDir` calls pass a `Source` and
      `loader.Options{Env: k.loadEnv()}`, keeping today's mode at each site (on-disk
      `&module.Source{Root: absDir}` for module, catalog, platform, no-values instance and
      `attributeValuesError`; the values overlay `src` for the layered instance).
- [x] 3.5 `opm/internal/loader/load_test.go`: `loadDir` and the `:140` call build an on-disk
      `Source` through one helper. Add `TestLoadDir_OverlaySubpackage` (an overlay `Source`
      with a non-empty `Pkg` builds `./<Pkg>`) and `TestLoadDir_NilSource` (the plain
      `source carries no module root` error, not `oerrors.ErrInvalidPackage`). Verify: `grep -rn 'LoadDir(' opm` shows only the new
      shape; `go test ./opm/internal/... ./opm/kernel -count=1` green.
- [x] 3.6 `task check` green, then commit
      `refactor(loader): take a module.Source and load options in LoadDir`.

## 4. Acquire directories through one read-then-build helper (kernel; design LS3, LS4, LS7)

- [x] 4.1 `opm/kernel/acquire.go`: add `dirSource` and `acquireDir` (LS3): `filepath.Abs`
      (today's `<verb>: resolving <label> directory: %w`), `loader.CheckDir` first,
      `sourceForDir`, and with the overlay flag one `sourcetree.OverlayFromDir(src.Root)`
      (wrapped `<verb>: %w`); then `LoadDir` on that `Source`, its error returned unwrapped.
- [x] 4.2 `AcquireModuleFromDir` and `AcquireCatalogFromDir` call `acquireDir(..., true)` and
      stamp the `Source` they built from; `AcquirePlatformFromDir` and the no-values branch of
      `AcquireInstanceFromDir` call `acquireDir(..., false)`. Delete `overlaySourceForDir`.
- [x] 4.3 `loadInstanceWithValues` (LS4): take the authored `Source` from
      `dirSource(..., true)` (its own `os.Stat` goes), merge the sources, read the package
      name from the authored `Source`, render the values file, build from a new `Source` with
      the same `Root`/`Pkg` and `maps.Clone` of the authored overlay plus `opm-values.cue`,
      and return that one. `attributeValuesError` takes the authored `*module.Source` instead
      of `absDir` and builds from it; nothing else in its body changes.
- [x] 4.4 Rewrite only the doc-comment lines this section makes false: `AcquireModuleFromDir`
      and `AcquireCatalogFromDir` say the package is built from the overlay they stamp;
      `loadInstanceWithValues` and `attributeValuesError` name the authored overlay. Verify:
      `grep -rn overlaySourceForDir opm` prints nothing, and each directory verb reaches
      `OverlayFromDir` at most once and `LoadDir` once per build (read the diff: the number
      of reads has no runtime probe, design LS3/LS4, so this inspection is its check).
- [x] 4.5 Tests in `opm/kernel/acquire_test.go` and `acquire_catalog_test.go`:
      `TestKernel_AcquireCatalogFromDir_Subpackage` (catalog from a subdirectory: `Root` the
      module root, `Pkg` `sub`, overlay spans the root); in
      `TestKernel_AcquireInstanceFromDir_WithSources_ConflictAttributed` (or a sibling)
      assert the attributed error still names the source's `Origin` when the author's
      directory also holds an `opm-values.cue` of its own (LS4). Expose `dirSource` through
      `opm/kernel/export_test.go` and add `TestDirSource_BuildsFromTheBytesReadFirst`: take
      an overlay `Source` from it, rewrite a `.cue` file on disk, build that `Source` with
      `loader.LoadDir`, and assert the value read first wins. The section 1 tests pass
      unchanged.
- [x] 4.5a Review follow-up: `TestKernel_AcquireFromDir_SubpackageBuildsFromTheStampedRoot`
      acquires a module and a catalog from a subdirectory and pins the load error
      (`loading <label> package from <root> (./sub): `) and the build error (`building <label>
      package from <root>/sub: `). It fails if the verbs build from the package directory on
      disk. `LoadDir` names the package directory in build and gate errors in both modes.
- [x] 4.6 Cross-cutting checks (LS7): `go test -race ./opm/kernel ./opm/internal/renderstage
      -count=1`, the parity tests (`go test ./opm/kernel -run Parity -count=1`) and
      `task cue:test:flow`. Verify: all green (the flow test may skip when the registry is
      unreachable; say so if it does). If `TestKernel_AcquireModuleFromDir_EmbedsNonCUEFile`
      passed in section 1 but fails now, stop and report to the supervisor before adding any
      disk fallback.
- [x] 4.7 `task check` green, then commit
      `refactor(kernel): acquire directories through one read-then-build helper`.

## 5. Verify

- [x] 5.1 Whole-tree gates on the final tree: `task check`. Verify: green.
- [x] 5.2 `openspec validate load-directories-from-one-source --strict` passes.
- [ ] 5.3 Archive only when the supervisor says the PR is being opened, on this branch so the
      archive rides the implementing PR: `openspec archive load-directories-from-one-source
      --yes`, then `openspec validate --all --strict`, then commit
      `chore(openspec): archive load-directories-from-one-source`. There is no
      `enhancement.yaml`, so no delivery log runs.
