## ADDED Requirements

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
