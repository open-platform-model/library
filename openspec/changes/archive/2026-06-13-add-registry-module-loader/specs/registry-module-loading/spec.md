# Delta: registry-module-loading

## ADDED Requirements

### Requirement: Load a Module from the Registry by Path and Version

The library SHALL provide `opm/helper/loader/registry.LoadModulePackage(ctx, cueCtx, modPath, version string, opts) (cue.Value, error)` that loads a `#Module` published in an OCI registry, identified by its major-qualified module path and version. (It mirrors the current `opm/helper/loader/file.LoadModulePackage` return shape `(cue.Value, error)`; the `opm/apiversion` package and loader-layer apiVersion detection were removed in commit `4276ec4`.) It SHALL fetch the module's source using CUE's native module machinery (`mod/modconfig`) and load it **as the main module** — its own `cue.mod/module.cue` drives transitive dependency resolution, and its `kind`/`metadata` are evaluated at the package root. It SHALL NOT synthesize a wrapper package that imports and embeds the target module.

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
