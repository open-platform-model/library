## MODIFIED Requirements

### Requirement: SynthesizeRelease is documented as the recommended in-memory entry point

The package documentation and the `Kernel.SynthesizeRelease` godoc SHALL state that `SynthesizeRelease` is the recommended entry point for building a release from typed inputs, mirroring how `Kernel.LoadReleasePackage` is the recommended entry point for building a release from a directory-based CUE package. Callers that explicitly want the helper-level primitive MAY call `synth.Release` followed by `Kernel.ProcessModuleRelease` directly.

#### Scenario: Documentation directs callers to the kernel method

- **WHEN** a developer reads the godoc on `opm/helper/synth/`
- **THEN** the documentation states that `Kernel.SynthesizeRelease` is the recommended entry point
- **AND** notes that direct use of `synth.Release` is appropriate when the caller does not hold a `*Kernel`

#### Scenario: SynthesizeRelease godoc points to LoadReleasePackage

- **WHEN** a developer reads the `Kernel.SynthesizeRelease` godoc
- **THEN** the file-driven mirror it names is `Kernel.LoadReleasePackage`
- **AND** no reference to the removed `Kernel.LoadReleaseFile` remains

### Requirement: Backward-Compatible Method Wrappers

For every existing exported function in `opm/helper/loader/file/` and `opm/helper/platform/`, and the `*FromValue` constructors in `opm/module/` and `opm/platform/` that takes a `*cue.Context` (directly or via a `CueContextOwner` interface), the Kernel SHALL provide a method wrapper that sources `*cue.Context` from itself. The Kernel SHALL NOT wrap functions whose canonical implementation now lives on the Kernel itself (validation, layered values, module-release processing, and the values-file source loader); those are direct kernel methods, not wrappers. The Kernel SHALL NOT expose a `ValidateAndUnify` wrapper — the canonical replacement is `Kernel.ValidateConfigDetailed`.

#### Scenario: Loader method wrapper for module packages

- **WHEN** a caller invokes `k.LoadModulePackage(ctx, "./module", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the result is identical to calling `helper/loader/file.LoadModulePackage(k.CueContext(), "./module", loaderfile.LoadOptions{Registry: "..."})`
- **AND** any error returned is the same instance the underlying free function would return

#### Scenario: Loader method wrapper for release packages

- **WHEN** a caller invokes `k.LoadReleasePackage(ctx, "./release", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the result is identical to calling `helper/loader/file.LoadReleasePackage(k.CueContext(), "./release", loaderfile.LoadOptions{Registry: "..."})`
- **AND** any error returned is the same instance the underlying free function would return

#### Scenario: Helper-shaped functions remain callable

- **WHEN** existing downstream code calls `helper/loader/file.LoadModulePackage(cueCtx, dir, opts)` or `helper/loader/file.LoadReleasePackage(cueCtx, dir, opts)` directly
- **THEN** the call succeeds with the documented behavior
- **AND** the helper signatures continue to accept `*cue.Context` so non-kernel consumers can use them without importing `opm/kernel`

#### Scenario: Validation methods are not wrappers

- **WHEN** a developer reads `opm/kernel/validate.go`
- **THEN** the file contains the canonical implementation of `ValidateConfig`, `ValidateConfigPartial`, and `ValidateConfigDetailed` directly, with no `//nolint:staticcheck // SA1019:` exemptions for delegating to deleted helper packages

#### Scenario: ValidateAndUnify wrapper is gone

- **WHEN** a developer searches `opm/kernel/wrappers.go` (or the entire `opm/kernel/`) for `ValidateAndUnify`
- **THEN** no exported method or function with that name exists
- **AND** callers MUST use `k.ValidateConfigDetailed`

## ADDED Requirements

### Requirement: Kernel.LoadSourceFromFile auto-unwraps the values field

The `*Kernel.LoadSourceFromFile(path string)` method SHALL load the file at `path` as a CUE instance via `load.Instances`, evaluate it against the kernel's `*cue.Context`, and:

- If the evaluated value contains a top-level `values:` field whose `Exists()` is true and `Err()` is nil, the returned `Source.Value` SHALL be that field.
- Otherwise the returned `Source.Value` SHALL be the whole evaluated value.

The method SHALL set `Source.Origin` to the absolute path of the loaded file and `Source.Name` to its basename. The method SHALL NOT depend on `loaderfile.LoadValuesFile` (which is removed).

#### Scenario: Values file is auto-unwrapped

- **WHEN** a caller invokes `k.LoadSourceFromFile("./values.cue")` against a file containing `values: { foo: "bar" }`
- **THEN** the returned `Source.Value` is the inner `{ foo: "bar" }` value
- **AND** `Source.Origin` is the absolute path of `values.cue`
- **AND** `Source.Name` is `values.cue`

#### Scenario: File without values field passes through

- **WHEN** a caller invokes `k.LoadSourceFromFile("./flat.cue")` against a file with no top-level `values:` field
- **THEN** the returned `Source.Value` is the whole evaluated file value
- **AND** `Source.Origin` and `Source.Name` are populated as above

## REMOVED Requirements

### Requirement: Kernel.LoadReleaseFile wrapper

**Reason**: The underlying `loaderfile.LoadReleaseFile` is removed in this change. Callers move to `Kernel.LoadReleasePackage`, which is the kernel wrapper for the package-loader replacement.

**Migration**: `k.LoadReleaseFile(ctx, file, opts)` → `k.LoadReleasePackage(ctx, dir, opts)` where `dir` is the directory containing the release CUE files.

### Requirement: Kernel.LoadValuesFile wrapper

**Reason**: The underlying `loaderfile.LoadValuesFile` is removed in this change. The auto-unwrap-`values`-field behavior moves to `Kernel.LoadSourceFromFile`, which is the only consumer that needs it.

**Migration**: `k.LoadValuesFile(ctx, path)` → `k.LoadSourceFromFile(path)`. The returned shape changes from `cue.Value` to `Source`; callers that just want the value use `src.Value` from the returned `Source`.
