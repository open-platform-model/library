# registry-module-loading Specification

## Purpose
The library is the single place where CUE module-acquisition plumbing lives (Principle V — CUE-Native Module Resolution). This capability gives the library a first-class primitive for loading a `#Module` that is published in an OCI registry, identified by `path@version`, so that consumers (operator render path, a future CLI, the planned Crossplane composition function) never hand-roll OCI fetch logic, wrapper-package shims, or dependency walks. The module is fetched via CUE's native module machinery and loaded **as the main module** — its own `cue.mod/module.cue` drives transitive resolution and its `kind`/`metadata` evaluate at the package root — which preserves core@v0's self-referential metadata that the wrapper approach broke.
## Requirements
### Requirement: Load a Module from the Registry by Path and Version

The library SHALL provide `opm/helper/loader/registry.LoadModulePackage(ctx, cueCtx, modPath, version string, opts) (cue.Value, error)` that loads a `#Module` published in an OCI registry, identified by its major-qualified module path and version. It SHALL fetch the module's source using CUE's native module machinery (`mod/modconfig`) and load it **as the main module** — its own `cue.mod/module.cue` drives transitive dependency resolution, and its `kind`/`metadata` are evaluated at the package root. It SHALL NOT synthesize a wrapper package that imports and embeds the target module. (It mirrors the current `opm/helper/loader/file.LoadModulePackage` return shape `(cue.Value, error)`; the `opm/apiversion` package and loader-layer apiVersion detection were removed in commit `4276ec4`.)

The loader SHALL apply the registry override carried by `opts` to the load configuration (via `load.Config.Env`) and SHALL NOT mutate process environment state. It SHALL build the value in the supplied `*cue.Context`.

#### Scenario: Published module is loaded by path and version

- **WHEN** `registry.LoadModulePackage(ctx, cueCtx, "testing.opmodel.dev/modules/hello@v0", "v0.0.2", opts)` is called with `opts.Registry` mapping the host to a registry serving the module
- **THEN** it returns the module's evaluated `cue.Value` and a nil error
- **AND** the process environment is not mutated

#### Scenario: Self-referential core@v0 metadata is preserved

- **WHEN** the loaded module is authored against `opmodel.dev/core@v0`, whose `#Module` derives `modulePath`/`version` from themselves
- **THEN** the returned value's `metadata.modulePath` and `metadata.version` equal the author-set values
- **AND** no `field not allowed` admission error occurs

#### Scenario: Transitive dependencies resolve

- **WHEN** the loaded module imports catalog resource definitions (for example `opmodel.dev/catalogs/opm/resources`)
- **THEN** those imports resolve from the registry via the module's own `cue.mod/module.cue`
- **AND** loading succeeds without the caller declaring the module's transitive dependencies

#### Scenario: Unresolvable module surfaces a load error

- **WHEN** the loader is called with a `path@version` not present in the registry
- **THEN** it returns a zero `cue.Value` and an error identifying the failed fetch or load

### Requirement: In-Memory Load Without a Temporary Directory

The registry module loader SHALL load the fetched module in memory and SHALL NOT write the module's source to a temporary directory. It SHALL inject the fetched module's files via `load.Config.Overlay` under a deterministic synthetic root, leaving `load.Config.FS` nil so the module's transitive dependencies resolve through the registry and CUE module cache.

#### Scenario: No temporary directory created

- **WHEN** a module is loaded from the registry
- **THEN** no temporary directory is created or left behind for the module's source
- **AND** the module's transitive dependencies still resolve

### Requirement: Registry-Loaded Modules Pass the Module Shape Gate

The registry module loader SHALL validate the built value with the same module shape gate `opm/helper/loader/file` applies (concrete `kind == "Module"`; `metadata.name`, `metadata.modulePath`, `metadata.version` present and concrete), returning errors that wrap the same sentinels (`ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField`). It SHALL NOT perform full schema validation, which remains the Kernel/Binding layer's contract.

#### Scenario: Wrong artifact kind rejected

- **WHEN** the resolved registry artifact has a concrete `kind` other than `"Module"`
- **THEN** the loader returns a zero `cue.Value` and an error wrapping `ErrWrongKind`

#### Scenario: Missing identity field rejected

- **WHEN** the resolved module lacks a concrete `metadata.modulePath`
- **THEN** the loader returns an error wrapping `ErrMissingRequiredField`

### Requirement: Acquire a Module from the Registry with its staged source

The library SHALL provide a source-carrying acquire entrypoint — `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)` — that fetches and loads a published `#Module` (the same fetch + main-module staging `LoadModulePackage` performs) and returns a `*module.Module` whose staged source (the deterministic overlay + synthetic root the loader builds from the fetched module, or the underlying `module.SourceLoc`) is attached and reachable by callers. The staged source SHALL be the same artifact used to build the module value, so no second fetch is required to reuse it. The existing `LoadModulePackage` / `LoadModuleFromRegistry` entrypoints SHALL remain available with their current `(cue.Value, error)` shape (this requirement is additive).

The acquired module's staged source SHALL be consumable by `instance-synthesis` to construct an instance inside the module's own main module, so transitive dependencies resolve via the module's own `cue.mod/module.cue`.

#### Scenario: Acquired module carries reusable staged source

- **WHEN** `Kernel.AcquireModuleFromRegistry(ctx, "<path>@v0", "v0.1.0")` is called against a registry serving the module
- **THEN** it returns a `*module.Module` whose staged source (overlay + synthetic root) is populated
- **AND** the staged source is the one used to build the module value (no second fetch is performed to obtain it)
- **AND** the process environment is not mutated

#### Scenario: Existing value-returning loader is unchanged

- **WHEN** a caller uses the existing `LoadModuleFromRegistry` / `registry.LoadModulePackage` entrypoint
- **THEN** it still returns `(cue.Value, error)` with the current behavior
- **AND** no caller is forced to migrate to the source-carrying entrypoint

#### Scenario: Staged source drives transitive resolution during synthesis

- **WHEN** a module acquired via `AcquireModuleFromRegistry` is passed to `synth.Instance`
- **THEN** the instance is staged inside the module's source tree and the module's own `cue.mod/module.cue` resolves the transitive (catalog) closure
- **AND** synthesis succeeds without the caller declaring the module's transitive dependencies

