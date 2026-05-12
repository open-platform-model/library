## Context

`pkg/helper/loader/file/` currently exposes three loaders with three different shapes:

```
LoadModulePackage(ctx, dir)                                  → (val, ver, err)
LoadReleaseFile(ctx, file, opts)                             → (val, dir, ver, err)
LoadPlatformFile(ctx, file, opts)                            → (val, dir, err)
LoadValuesFile(ctx, file)                                    → (val, err)
```

`LoadModulePackage` uses `load.Instances([]string{"."}, cfg)` — a CUE package load. `LoadReleaseFile` uses `load.Instances([]string{<basename>}, cfg)` — a single-file load. The two paths cannot share fixtures: a module is a directory of `.cue` files, a release is one `.cue` file. Yet a release is just as much a CUE artifact, with a schema-bound apiVersion and registry-resolved imports. The split is historical, not principled.

`LoadValuesFile` is even more of an anomaly. It loads a single file and, if the result has a top-level `values:` field, returns that field instead of the whole value. The only kernel-side consumer is `Kernel.LoadSourceFromFile` (`pkg/kernel/source_loader.go:49`), which delegates straight into it. The standalone helper exists because the kernel `Source` type was introduced later as a wrapper; the underlying file-load logic was never folded in.

The change unifies module and release loading around the package semantics, and inlines the values-file logic where it actually belongs.

## Goals / Non-Goals

**Goals:**

- Make `LoadModulePackage` and `LoadReleasePackage` two instances of the same idea: "load a directory of CUE files as a package, detect the apiVersion, return the value." Same return shape, same options.
- Inline the auto-extract-`values`-field behavior into `Kernel.LoadSourceFromFile` so the magic lives next to the `Source` abstraction that needs it, not in a generically-named helper.
- Surface registry-override support on both package loaders. Modules and releases both import other CUE modules from a registry; both need `LoadOptions`.

**Non-Goals:**

- Touching `LoadPlatformFile`. Platforms today are single-file top-level artifacts. Symmetry with `LoadPlatformPackage` is YAGNI.
- Touching `LoadProvider`. Not part of the user-requested scope.
- Reshaping the `LoadOptions` struct beyond the existing `Registry string` field. Future option-bag growth is a separate decision.
- Bytes loader (`pkg/helper/loader/bytes/`) implementation.
- CLI / operator migration. Library is currently the only in-tree consumer; downstream repos update on their own cadence.

## Decisions

### D1. Symmetric package-loader signatures

Both loaders SHALL share the shape:

```go
func LoadModulePackage(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, apiversion.Version, error)
func LoadReleasePackage(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, apiversion.Version, error)
```

`LoadOptions` carries the registry override (already used by `LoadReleaseFile` and `LoadPlatformFile` today). Neither loader returns a `parentDir`: the directory passed in is already the parent. The file-based `LoadPlatformFile` returns `parentDir` because it accepts a file path that may resolve to a non-`platform.cue` filename in some parent directory; that ambiguity does not exist for package loaders.

**Alternative considered**: Keep `LoadModulePackage` unchanged (no `LoadOptions`) and only add it to `LoadReleasePackage`. Rejected — modules can also import other modules (transformers, traits) from a registry, and the lack of `LoadOptions` is a latent gap (commented in `pkg/kernel/flow_integration_test.go:68`). Symmetric shape is worth a one-time breaking-signature cost.

**Alternative considered**: Variadic options (`opts ...LoadOption`). Rejected per Principle VII — the options struct has one field today; adding fields can happen additively without an option-bag refactor.

### D2. Removal, not deprecation, for `LoadReleaseFile` and `LoadValuesFile`

The user-confirmed direction is hard removal. The library has no out-of-tree consumers today; the CLI and operator repos will be updated to consume the new surface. A deprecation shim would carry weight for no benefit.

**Alternative considered**: Add `LoadReleasePackage` first, leave `LoadReleaseFile` deprecated for one release, then remove. Rejected — there is no migration window we need to honor; the deprecation cycle costs developer attention without buying safety.

### D3. `Kernel.LoadSourceFromFile` absorbs the values-field auto-unwrap

Today `Kernel.LoadSourceFromFile` delegates to `loaderfile.LoadValuesFile`, which contains this branch:

```go
if valuesField := val.LookupPath(cue.ParsePath("values")); valuesField.Exists() && valuesField.Err() == nil {
    return valuesField, nil
}
return val, nil
```

After the change, `Kernel.LoadSourceFromFile` performs the same load locally (via `load.Instances`) and keeps the auto-unwrap. The semantics for existing callers (`vet.go`, `render/values.go`) are unchanged. The branch lives next to the `Source` abstraction it serves.

**Alternative considered**: Drop the auto-unwrap; require callers to do `.LookupPath("values")` themselves. Rejected per the user's directive — keep the magic. The auto-unwrap is a load-bearing convention for OPM values files.

**Alternative considered**: Split into `LoadSourceFromFile` (raw) plus a `LoadValuesSource` (with magic). Rejected — only one shape is used in the codebase today; introducing a second is speculative.

### D4. Release-package fixtures replace single-file release fixtures

Release tests today materialize a single `release.cue` file in a temp dir. After the change, the same temp dir layout works — a directory containing a single `release.cue` file is still a valid CUE package. New multi-file fixtures (e.g. `release.cue` + `values.cue` in one package) are added to lock the multi-file behavior. No existing fixture is invalidated.

### D5. Wrapper updates follow the existing convention

`Kernel.LoadModulePackage` gains the `opts loaderfile.LoadOptions` parameter (breaking on the kernel surface). `Kernel.LoadReleasePackage` is added with the same shape. `Kernel.LoadReleaseFile` and `Kernel.LoadValuesFile` wrappers are removed. The kernel-runtime "Backward-Compatible Method Wrappers" requirement still holds for the loaders that remain.

## Risks / Trade-offs

**[R1] Cross-repo breakage at the moment of merge**: Downstream CLI / operator builds break the instant the library merges. Mitigation: user has explicitly accepted this — the CLI / operator repos will be updated in their own commits as a follow-up. No deprecation shim is required because there are no out-of-tree consumers.

**[R2] `LoadOptions` on `LoadModulePackage` is a one-time API churn**: Every existing test/caller passes a fourth parameter. Mitigation: there are few in-tree callers (kernel wrapper, two tests); the churn is mechanical. The benefit — symmetric loaders, registry support on modules — is a one-time payoff.

**[R3] Auto-unwrap-`values`-field semantics moves**: The behavior is identical, but the location changes. A reader looking for "where does the values field get extracted?" today finds it in `loaderfile.LoadValuesFile`; after the change they find it in `Kernel.LoadSourceFromFile`. Mitigation: godoc on `LoadSourceFromFile` documents the auto-unwrap explicitly. The branch carries a `// Why:` comment naming the convention.

**[R4] Multi-file release packages introduce a new failure mode**: A release-package directory with two `.cue` files using different package names is a CUE package-name conflict, which surfaces as a load error. Mitigation: the same failure already exists for module packages; the error message from `load.Instances` is informative. No additional handling needed in the loader.

**[R5] Stale `Kernel.LoadReleaseFile` references after the refactor lands**: Synth shipped while the file wrapper still existed, so its godoc, CHANGELOG entry, and kernel-runtime spec requirement all name `Kernel.LoadReleaseFile` as the "file-driven mirror" anchor. After the wrapper is removed those references point at nothing.

Concrete sites:

1. `pkg/kernel/synth.go` — `Kernel.SynthesizeRelease` godoc says *"it mirrors how [Kernel.LoadReleaseFile] is the recommended entry point for the file-driven path."*
2. `CHANGELOG.md` — the Unreleased `add-release-synth-helper` entry says *"Mirrors how `Kernel.LoadReleaseFile` is the recommended entry point for the file-driven path."*
3. `openspec/specs/kernel-runtime/spec.md` after synth archives — Requirement *"SynthesizeRelease is documented as the recommended in-memory entry point"* and its scenarios name `LoadReleaseFile` as the mirror.

Mitigation: this refactor includes mechanical rewrites of (1)–(3) to use `Kernel.LoadReleasePackage`. The synth `proposal.md` and `design.md` are historical artefacts and stay untouched (they record the state at synth's authoring time).

## Migration Plan

1. Land `add-release-synth-helper` first. Synth lands against the existing loader surface; this refactor follows.
2. Land this change as a single commit on the library: rewrite `release.go`, modify `module.go`, update `kernel/wrappers.go`, update `kernel/source_loader.go`, update fixtures and tests.
3. Update CLI repo to consume the new surface (separate commit, separate repo).
4. Update operator repo if needed (separate commit, separate repo).

Rollback strategy: revert the library commit. CLI / operator stay on the previous library version until updated. Because the library has no out-of-tree consumers, the rollback surface is tiny.

## Open Questions

- Should `LoadOptions` grow a `Stdin io.Reader` or `Files map[string][]byte` field at some point to support in-memory packages? Defer — that is the territory of `pkg/helper/loader/bytes/`, which is still a skeleton.
- Should `LoadReleasePackage` validate the package contains exactly one `#ModuleRelease` (vs. zero or more)? Defer to the apiVersion detection step; an empty package fails `apiversion.Detect`, a multi-release package becomes an `apiversion.Detect` ambiguity. Both are caller errors with informative messages today.
