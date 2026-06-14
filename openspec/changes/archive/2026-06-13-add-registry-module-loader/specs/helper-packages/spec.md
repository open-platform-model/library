## ADDED Requirements

### Requirement: Registry Loader Under Helper

The library SHALL provide a `opm/helper/loader/registry` subpackage, sibling to `opm/helper/loader/file`, that loads a published `#Module` from an OCI registry by `path@version`. It SHALL expose:

- `LoadModulePackage(ctx, cueCtx, modPath, version string, opts) (cue.Value, error)` — fetch the module's source via CUE's native module machinery and load it in memory as the main module, with registry override via `opts.Registry`. (Mirrors the current `loader/file.LoadModulePackage` return shape `(cue.Value, error)`.)
- `LoadOptions` carrying the registry override, the same shape as `loader/file.LoadOptions`.

The package SHALL be opt-in under the `opm/helper/` boundary: a frontend MAY skip it and resolve registry modules another way.

#### Scenario: Registry loader present under helper

- **WHEN** a developer reads `opm/helper/loader/registry`
- **THEN** the package exposes `LoadModulePackage` for loading a published module by path and version
- **AND** it lives under `opm/helper/`, marking it opt-in convenience over the kernel core contract

### Requirement: Shared Module Shape Gate Across Loaders

The module shape gate SHALL be single-sourced so that `opm/helper/loader/file` and `opm/helper/loader/registry` validate a `#Module` identically. The sentinels `ErrInvalidPackage`, `ErrWrongKind`, and `ErrMissingRequiredField` SHALL remain exposed from `opm/helper/loader/file` with unchanged identity, so existing `errors.Is` callers are unaffected. Extracting the gate to a shared location SHALL be behavior-preserving for `loader/file`.

#### Scenario: Identical gate for both loaders

- **WHEN** `loader/file.LoadModulePackage` and `loader/registry.LoadModulePackage` each load a package whose `kind` is not `"Module"`
- **THEN** both return an error wrapping the same `ErrWrongKind` sentinel value

#### Scenario: Sentinel identity preserved

- **WHEN** a frontend that previously called `errors.Is(err, loaderfile.ErrWrongKind)` is recompiled against this slice
- **THEN** that check continues to compile and behave identically
