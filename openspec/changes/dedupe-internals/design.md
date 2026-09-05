# Design: dedupe-internals

## Context

See `proposal.md` § Why. The duplicates and their visibility problem:

```
                              may import              loader/internal/stage   internal/renderstage
   opm/kernel                                               no                      yes
   opm/internal/renderstage                                 no                      -
   opm/helper/loader/registry                               yes                     yes
   opm/helper/synth                                         no                      yes
```

Anything placed under `opm/internal/` is importable by every package under `opm/`, which is the only placement that lets the kernel, the render stage and both loaders share one implementation. `opm/internal/valuesfile` already follows that pattern.

Constraints: no public Go signature changes; `Kernel.AcquireInstanceFromDir` with extra values, `AcquireModuleFromRegistry`, `SynthesizeInstance` and `Render` produce identical results before and after; `cut-dead-surface` lands first (it edits `acquire.go` and `registry/module.go`).

## Goals / Non-Goals

**Goals:**

- One env-override helper, one set of source-tree helpers, one CUE quoting call.
- Delete the identity branch no renderable module takes.
- Keep every existing test scenario's coverage, re-homed where its subject moved.

**Non-Goals:**

- Folding `helper/loader/file` or `helper/synth` into the kernel (slice 2).
- Changing `module.Source.Overlay`'s type (slice 2).
- A kernel cache-dir knob (slice 2); `cueenv` merely makes one possible.
- Exporting a `Source.WriteTo` for the cli's scaffold (slice 5).

## Decisions

### `cueenv.Override` returns nil when nothing is overridden

**Context**: the four copies disagree on the empty case: three return nil (cue/load then reads the process environment itself), one returns a full copy. `modconfig.Config.Env` and `load.Config.Env` both document nil as "use `os.Environ()`".
**Decision**: `Override(registry, cacheDir string) []string` returns nil when both arguments are empty; otherwise a fresh copy of `os.Environ()` with each non-empty override replacing an existing `KEY=` entry or appended. No `os.Setenv`, ever; the package doc states it once instead of four times. The four sites become `cueenv.Override(opts.Registry, "")`, `cueenv.Override(l.Registry, l.CacheDir)` and `cueenv.Override(registry, "")` for the render build. `TestRegistryEnv` (`renderstage/stage_test.go:172`), the registry loader's internal env test and the schema loader's `mergeEnv` cases move into `cueenv` as one table test.

### `sourcetree` owns walking, reading, naming and writing a `module.Source`

**Context**: the tree helpers are split three ways and the kernel reaches `readSourceFile` only through `renderstage`.
**Decision**: one package, seven functions:

| Function | Replaces |
| --- | --- |
| `PackageName(src)` | `kernel.packageNameOfDir`, `renderstage.packageName` + `packageFiles` |
| `OverlayFromDir(root)` | `kernel.overlayForRoot` |
| `OverlayFromFS(fsys, dir, keyRoot)` | `stage.OverlayFromSource` (the caller passes `loc.FS`, `loc.Dir`, `SyntheticRoot(...)`) |
| `SyntheticRoot(modPath, version)` | `stage.SyntheticRoot` (moved verbatim) |
| `WriteTo(dir, src)` | the overlay branch of `renderstage.serveDir` |
| `ReadFile(src, path)` | `renderstage.readSourceFile` |
| `Bytes(load.Source)` | `renderstage.sourceBytes` (moved verbatim, reflection included, with its tests) |

`renderstage.serveDir` stays as a six-line wrapper choosing on-disk root versus `WriteTo` into the staging dir. `kernel.loadInstanceWithValues` computes `sourceForDir(absDir)` first and calls `PackageName(src)`, which reads the same directory `packageNameOfDir` read. The "fetched module is empty" refusal stays at the registry loader (`sourcetree` cannot import `loader/internal/shape`; the loader checks `len(overlay) == 0` and wraps `shape.ErrInvalidPackage`).
**Alternative rejected**: a `module.Source` method set. The walkers need `os`/`fs` I/O and `load` types that `opm/module` deliberately does not import.

### One walker filter: `.cue` files only

**Context**: the kernel's walker copies `.cue` files; the registry walker copies every file in the archive. cue/load reads only `.cue` files unless `@embed` is used, and nothing in the workspace uses it. A walker that copies everything from an on-disk root would also swallow `.git` and `node_modules`, the reason the cli's copy carries a skip list.
**Decision**: both walkers include a regular file iff its name ends in `.cue`. `cue.mod/module.cue` is covered by the suffix. Directory names are not filtered, so a vendored `cue.mod/pkg/**/*.cue` is included as cue/load would read it. The registry-side narrowing is the one behaviour change and is recorded in the spec delta.

### Identity: strict equality only

**Context**: `verifyModuleIdentity` has a major-free branch and `synth.moduleImportPath` its inverse, both encoding the 0003 convention for modules published against core v0/v1. Core v2 rejects a major-free `modulePath` at the schema, and the v2 kernel's glue cannot render a v0/v1 module.
**Decision**: `verifyModuleIdentity` compares `declaredPath == modPath`; `parentPath` is deleted. `moduleImportPath` becomes `m.Metadata.ModulePath`; `moduleSnakeName` is deleted. No new sentinel: a module that somehow reaches synth with a major-free path fails the build with CUE's own import error, which names the path. The two carve-out tests in `registry/module_test.go:134-160` collapse into one refusal test; the spec scenario keeps its name with a refusal body.

### Quoting through `literal.String.Quote`

**Decision**: `platformmodule.quote` is deleted and its two call sites use `literal.String.Quote`; `synth/render.go` replaces `%q` with the same call for `name`, `namespace`, labels, annotations and the two import paths. The output for every value in play (DNS names, SemVer, module paths) is byte-identical to today's, so golden tests do not change; the gain is that a value with a non-ASCII or non-printable rune is quoted the way CUE parses it, not the way Go does.

### `Staged` keeps what the kernel reads

**Decision**: `Staged{Dir string; Skew []VersionRow}`. `stage_test.go:114` (`Promotion.Replacements`) becomes an assertion in `TestPromote_*` on the `Promotion` value `Promote` returns; `:128-129` (the two import paths) become a case in the import-path builder's test. `Stage` drops its trailing struct assembly.

### `task test` runs each package once

**Decision**: the first command becomes `go test $(go list ./... | grep -v -e /opm/kernel -e /opm/internal/renderstage)`; the `-race` command is unchanged. Same coverage, one build fewer per package.

## Risks / Trade-offs

- [`modconfig.Config.Env` nil does not mean process environment in cue v0.17.1] → verified against the SDK source at implementation; the `cueenv` test builds a registry with `Env: nil` and asserts resolution still honours `CUE_REGISTRY` from the process.
- [A published module relies on a non-CUE file] → only `@embed` could; none in the workspace; the spec delta records the narrowing so a future module that needs it changes the spec first.
- [A cli user acquires a v1-line module on the v2 cli] → they now get `IdentityError` naming both paths at acquire instead of a render failure later; the v2 kernel could not render it either way.
- [Two changes touch `acquire.go` and `registry/module.go`] → `cut-dead-surface` lands first; this change rebases onto it.

## Migration Plan

One PR, commits per package. The identity commit is `refactor(loader)!:` and carries the MAJOR; the rest are `refactor(<pkg>):`. Consumers re-pin at their next `fix(deps)` wave with no source change. Rollback is a revert of the release.

## Open Questions

None.
