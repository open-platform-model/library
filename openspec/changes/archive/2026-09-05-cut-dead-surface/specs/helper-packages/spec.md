## MODIFIED Requirements

### Requirement: Registry Loader Under Helper

The library SHALL provide a `opm/helper/loader/registry` subpackage, sibling to `opm/helper/loader/file`, that loads a published `#Module` from an OCI registry by `path@version`. It SHALL expose:

- `LoadModulePackageWithSource(ctx, cueCtx, modPath, version string, opts) (cue.Value, *module.Source, error)`: fetch the module's source via CUE's native module machinery, load it in memory as the main module with registry override via `opts.Registry`, verify its declared identity against the fetched coordinate, and return the built value together with the staged source tree as the artifact `Source` type (overlay mode: synthetic root plus overlay carrying the module's files, its own `cue.mod/module.cue` included) so a follow-on build can reuse the module's own `cue.mod`.
- `LoadOptions` carrying the registry override, the same shape as `loader/file.LoadOptions`.

The package SHALL NOT define a loader-specific result type: the staged tree is `*module.Source`, the type every artifact carries. The package SHALL NOT expose a second, value-only entry point: dropping the staged source is the caller's choice to discard the second return.

The package SHALL be opt-in under the `opm/helper/` boundary: a frontend MAY skip it and resolve registry modules another way.

#### Scenario: Registry loader present under helper

- **WHEN** a developer reads `opm/helper/loader/registry`
- **THEN** the package exposes `LoadModulePackageWithSource` for loading a published module by path and version
- **AND** it lives under `opm/helper/`, marking it opt-in convenience over the kernel core contract

#### Scenario: Staged source is the artifact Source type

- **WHEN** `LoadModulePackageWithSource` succeeds
- **THEN** the second return is a non-nil `*module.Source` in overlay mode whose `Overlay` carries the module's files and whose `Root` is the synthetic root the overlay keys sit under
- **AND** `Kernel.AcquireModuleFromRegistry` stores it on `Module.Source` unchanged
- **AND** no `StagedSource` type exists in the package

#### Scenario: Value-only loader is gone

- **WHEN** a developer searches `opm/helper/loader/registry` and `opm/kernel` for `LoadModulePackage(` and `LoadModuleFromRegistry`
- **THEN** neither symbol exists; `Kernel.AcquireModuleFromRegistry` is the single kernel entry point for a published module
