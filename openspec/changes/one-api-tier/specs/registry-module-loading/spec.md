## ADDED Requirements

### Requirement: Acquire a module from the registry

The library SHALL provide one registry acquisition entry point, `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, that fetches a published `#Module` via CUE's native module machinery, loads it in memory as the main module (its own `cue.mod/module.cue` drives transitive resolution, its `kind`/`metadata` evaluate at the package root, no wrapper package is synthesized and no temporary directory is written), runs the acquisition shape gate, verifies the declared identity against the fetched coordinate, and returns a `*module.Module` whose staged source is attached: `Source.Root` the deterministic synthetic root, `Source.Overlay` the module's `.cue` files as bytes (its own `cue.mod/module.cue` included and nothing else). The staged source SHALL be the same tree used to build the module value, so no second fetch is required to reuse it. The registry mapping SHALL be the kernel's (`WithRegistry`), applied through the resolver and load configuration and never through `os.Setenv`. No value-only loader SHALL be exported from any package under `opm/`.

The acquired module's staged source SHALL be consumable by `SynthesizeInstance` to construct an instance inside the module's own main module, so transitive dependencies resolve via the module's own `cue.mod/module.cue`.

#### Scenario: Acquired module carries reusable staged source

- **WHEN** `Kernel.AcquireModuleFromRegistry(ctx, "<path>@v2", "v2.1.0")` is called against a registry serving the module
- **THEN** it returns a `*module.Module` whose staged source (overlay of bytes plus synthetic root) is populated and `HasSource()` reports true
- **AND** the staged source is the one used to build the module value (no second fetch is performed to obtain it)
- **AND** the process environment is not mutated

#### Scenario: Registry mapping is the kernel's

- **WHEN** a kernel constructed with `WithRegistry(mapping)` acquires a module
- **THEN** the fetch and the module's transitive imports resolve through `mapping` with no per-call argument

#### Scenario: No value-only loader

- **WHEN** a consumer searches every package under `opm/` for a registry loader returning a bare `cue.Value`
- **THEN** none exists; `Kernel.AcquireModuleFromRegistry` is the single entry point and the raw value is `Module.Package`

#### Scenario: Staged source drives transitive resolution during synthesis

- **WHEN** a module acquired via `AcquireModuleFromRegistry` is passed to `SynthesizeInstance`
- **THEN** the instance is staged inside the module's source tree and the module's own `cue.mod/module.cue` resolves the transitive (catalog) closure
- **AND** synthesis succeeds without the caller declaring the module's transitive dependencies

## REMOVED Requirements

### Requirement: Acquire a Module from the Registry with its staged source

**Reason**: Superseded by "Acquire a module from the registry": the requirement promised that the value-only loader entry points "remain available", and its scenario "Existing value-returning loader is unchanged" pinned that promise. Those entry points are removed by this change, so the requirement is restated without the promise rather than modified around it.

**Migration**: `Kernel.AcquireModuleFromRegistry(ctx, path, version)` is unchanged in signature; a caller of a value-only loader reads `Module.Package` from its result.

### Requirement: Load a Module from the Registry by Path and Version

**Reason**: The public loader function had one caller, the kernel; its behaviour (fetch via `mod/modconfig`, main-module load, no wrapper package, registry override without `os.Setenv`) is now specified as part of "Acquire a Module from the Registry with its staged source".

**Migration**: `registry.LoadModulePackageWithSource(ctx, cueCtx, path, version, opts)` becomes `Kernel.AcquireModuleFromRegistry(ctx, path, version)` on a kernel constructed with `WithRegistry`.
