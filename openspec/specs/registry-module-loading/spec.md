# registry-module-loading Specification

## Purpose
The library is the single place where CUE module-acquisition plumbing lives (Principle V — CUE-Native Module Resolution). This capability gives the library a first-class primitive for loading a `#Module` that is published in an OCI registry, identified by `path@version`, so that consumers (operator render path, a future CLI, the planned Crossplane composition function) never hand-roll OCI fetch logic, wrapper-package shims, or dependency walks. The module is fetched via CUE's native module machinery and loaded **as the main module** — its own `cue.mod/module.cue` drives transitive resolution and its `kind`/`metadata` evaluate at the package root — which preserves core@v0's self-referential metadata that the wrapper approach broke.
## Requirements

### Requirement: In-Memory Load Without a Temporary Directory

The registry module loader SHALL load the fetched module in memory and SHALL NOT write the module's source to a temporary directory. It SHALL inject the fetched module's CUE files (every `.cue` file under the module root, the module's own `cue.mod/module.cue` included, and nothing else) via `load.Config.Overlay` under a deterministic synthetic root, leaving `load.Config.FS` nil so the module's transitive dependencies resolve through the registry and CUE module cache. The staged overlay the loader returns on the module's `Source` SHALL be that same set of files.

#### Scenario: No temporary directory created

- **WHEN** a module is loaded from the registry
- **THEN** no temporary directory is created or left behind for the module's source
- **AND** the module's transitive dependencies still resolve

#### Scenario: Staged overlay carries the module's CUE files only

- **WHEN** a fetched module's archive contains `.cue` files, a `cue.mod/module.cue`, and non-CUE files such as a license or a readme
- **THEN** the staged overlay on `Module.Source` holds every `.cue` file and `cue.mod/module.cue`, each keyed under the synthetic root
- **AND** the non-CUE files are not present in the overlay
- **AND** a fetch that stages no `.cue` file fails the load with an error wrapping `ErrInvalidPackage` (a defensive branch: a module zip CUE's fetch accepts always carries `cue.mod/module.cue`, so the empty case is exercised at the walker, not through a registry)

### Requirement: Registry-Loaded Modules Pass the Module Shape Gate

The registry module loader SHALL validate the built value with the same module shape gate `opm/helper/loader/file` applies (concrete `kind == "Module"`; `metadata.name`, `metadata.modulePath`, `metadata.version` present and concrete), returning errors that wrap the same sentinels (`ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField`). It SHALL NOT perform full schema validation, which remains the Kernel/Binding layer's contract.

#### Scenario: Wrong artifact kind rejected

- **WHEN** the resolved registry artifact has a concrete `kind` other than `"Module"`
- **THEN** the loader returns a zero `cue.Value` and an error wrapping `ErrWrongKind`

#### Scenario: Missing identity field rejected

- **WHEN** the resolved module lacks a concrete `metadata.modulePath`
- **THEN** the loader returns an error wrapping `ErrMissingRequiredField`

### Requirement: Module Identity Verification

After the artifact shape gate passes, the registry module loader SHALL verify the acquired artifact's declared identity against the coordinate it was fetched by: `metadata.modulePath` SHALL equal the requested module path (both in the full major-suffixed form, compared as strings without recomposition), and `metadata.version` SHALL equal the fetched tag with its `v` prefix stripped. A disagreement SHALL fail the load with a typed identity error naming both the declared and the fetched value. The verification SHALL sit on the shared load path so every entrypoint, and every frontend calling through the kernel, inherits one implementation.

There SHALL be no alternative verification for a major-free `metadata.modulePath`: the OPM schema the library consumes requires the major-suffixed form, so a declaration without the suffix cannot equal the fetched path and is refused with the same typed error. The library SHALL NOT compose a module address from a parent path and a name.

The kernel SHALL NOT write or correct either value: the schema declares identity, the reader verifies it (the kernel is a verifier, never a stamper).

#### Scenario: Address mismatch refused at acquire

- **WHEN** a published module is fetched by `opmodel.dev/modules/demo@v1` but declares `metadata.modulePath: "opmodel.dev/modules/other@v1"`
- **THEN** the load fails with a typed error carrying both paths

#### Scenario: Version mismatch refused at acquire

- **WHEN** a published module is fetched by tag `v2.0.2` but declares `metadata.version: "2.0.0"`
- **THEN** the load fails with a typed error carrying both versions

#### Scenario: Older-line parent-path declaration verified by convention

- **WHEN** a module fetched by `testing.opmodel.dev/x/modules/hello@v0` declares the major-free `metadata.modulePath: "testing.opmodel.dev/x/modules"` (the core-v0/v1 shape)
- **THEN** the load fails with the typed identity error for the `path` field, carrying the declared parent path and the fetched path
- **AND** no publishing-convention fallback is attempted

#### Scenario: Honest artifact loads

- **WHEN** the declared path and version match the fetched coordinate
- **THEN** the load proceeds exactly as before this requirement

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
