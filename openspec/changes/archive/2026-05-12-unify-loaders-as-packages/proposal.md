## Why

The filesystem loader surface treats modules and releases inconsistently. `LoadModulePackage` loads an entire CUE package from a directory — the idiomatic way to load CUE — but `LoadReleaseFile` is a single-file loader that only resolves `release.cue` from a directory. This forces release authors into a single-file pattern, even when a multi-file CUE package (e.g. `release.cue` + `values.cue` + `overlay.cue` sharing one package name) would be more natural. Modules and releases are both CUE artifacts with a schema-bound `apiVersion`; they should load the same way.

`LoadValuesFile` is a parallel oddity: it loads a CUE file and auto-extracts a `values` field when present. The function name does not advertise that magic, and its only kernel consumer (`Kernel.LoadSourceFromFile`) immediately delegates to it. The standalone helper exists because of a refactor leftover, not because callers need both shapes.

## What Changes

- **REMOVED**: `loaderfile.LoadReleaseFile` and the `Kernel.LoadReleaseFile` wrapper. The single-file release path is no longer supported. Callers move to `LoadReleasePackage`.
- **REMOVED**: `loaderfile.LoadValuesFile` and the `Kernel.LoadValuesFile` wrapper. The auto-extract-`values`-field behavior moves into `Kernel.LoadSourceFromFile`, which is the only consumer that needs it.
- **ADDED**: `loaderfile.LoadReleasePackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)`. Mirrors `LoadModulePackage`: loads the entire CUE package from a directory, detects the apiVersion, returns the raw `cue.Value`. Accepts `LoadOptions` for registry override (releases import modules from a registry).
- **MODIFIED**: `loaderfile.LoadModulePackage` gains a `LoadOptions` parameter (registry override). Modules that import other modules (transformers, traits) from a registry need this; the asymmetry today is a latent gap.
- **MODIFIED**: `Kernel.LoadModulePackage` wrapper picks up the new `LoadOptions` parameter.
- **ADDED**: `Kernel.LoadReleasePackage` wrapper mirroring `Kernel.LoadModulePackage`.
- **MODIFIED**: `Kernel.LoadSourceFromFile` absorbs the auto-extract-`values`-field behavior. Existing callers see no behavior change.
- **BREAKING (`opm/helper/loader/file`, `opm/kernel`)**: removal of `LoadReleaseFile`, `LoadValuesFile`, and the change to `LoadModulePackage`'s signature are all source-incompatible. MAJOR per Principle VI.

## Capabilities

### Modified Capabilities

- `helper-packages`: drop the `LoadReleaseFile` / `LoadValuesFile` requirements; add `LoadReleasePackage`; require `LoadModulePackage` to accept `LoadOptions`; note that values-file auto-unwrap lives on `Kernel.LoadSourceFromFile`.
- `kernel-runtime`: remove the `Kernel.LoadReleaseFile` and `Kernel.LoadValuesFile` wrapper requirements; add `Kernel.LoadReleasePackage`; update `Kernel.LoadSourceFromFile` to own the values-field auto-unwrap behavior.

## Impact

**Affected packages**

- `opm/helper/loader/file/` — `release.go` rewritten (release-package loader, no values helper). `module.go` signature changes for `LoadOptions`.
- `opm/kernel/` — `wrappers.go` loses two wrappers, gains one; `source_loader.go` absorbs the values-unwrap logic.
- Tests under both packages updated; existing `release.cue` fixtures become `release/` package fixtures (a `release.cue` file in a dir still works because a dir is still a package).

**Affected downstream consumers**

The library has no other in-tree consumers at this time (per user direction). The CLI and operator repos will be updated separately to consume the new surface.

**SemVer classification**

- MAJOR. Three source-incompatible changes in `opm/`: two removed functions, one modified signature.

**Ordering**

- This change lands **after** `add-release-synth-helper`, which is now implemented. Synth shipped without depending on the file-loader path, but four landed artefacts reference `Kernel.LoadReleaseFile` by name and become stale once the wrapper is removed:
  1. `opm/kernel/synth.go` godoc on `Kernel.SynthesizeRelease`.
  2. The Unreleased CHANGELOG entry for `add-release-synth-helper`.
  3. The `add-release-synth-helper` kernel-runtime delta requirement "SynthesizeRelease is documented as the recommended in-memory entry point" — folds into the main `kernel-runtime` spec when synth archives.
  4. Synth's `proposal.md` / `design.md` — historical artefacts, left as-is.

  This refactor includes mechanical updates to (1)–(3) so the `LoadReleasePackage` mirror anchor lands coherent.

- Synth's other landed surface (`Binding.SchemaValue`, v1alpha2 pointer-receiver switch, `opm/helper/synth/`, `Kernel.SynthesizeRelease`) is orthogonal to file loaders and unaffected.

**Out of scope**

- `LoadPlatformFile` — left untouched. Platforms are typically single-file artifacts; symmetry there is YAGNI until a consumer needs it.
- `LoadProvider` — unchanged.
- Bytes loader (`opm/helper/loader/bytes/`) — still a doc-only skeleton.
- CLI / operator migration — happens in their own repos.
