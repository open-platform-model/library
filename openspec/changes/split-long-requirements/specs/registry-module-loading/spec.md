## REMOVED Requirements

### Requirement: In-Memory Load Without a Temporary Directory
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The registry module loader loads the fetched module in memory and writes no temporary directory", "The loader injects the fetched module's CUE files under a deterministic synthetic root and leaves load.Config.FS nil" and "The synthetic root is an absolute path that does not exist on disk and the loader refuses the acquire when anything exists at it".

### Requirement: Registry-Loaded Modules Pass the Module Shape Gate
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The registry module loader evaluates and shape-gates the fetched module through the kernel's one routine", "The registry and directory acquisition paths differ only in where the package files come from" and "The registry module loader performs no full schema validation".

### Requirement: Module Identity Verification
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The loader verifies a module's declared identity against its fetched coordinate", "The identity verification sits on the single module entrypoint of the shared fetch routine", "The identity verification is scoped to modules", "The identity check is distinguishable in the code rather than by convention", "A major-free metadata.modulePath has no alternative verification and no module address is composed from a parent path and a name" and "The kernel is a verifier and never a stamper of the declared identity".

### Requirement: Acquire a module from the registry
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Each artifact kind has one registry acquisition entry point", "One fetch routine serves every artifact kind", "AcquireModuleFromRegistry fetches a published module via CUE's native module machinery", "AcquireModuleFromRegistry loads the module in memory as the main module", "AcquireModuleFromRegistry runs the shape gate and verifies the declared identity", "AcquireModuleFromRegistry returns the module with its staged source attached", "The staged source is the tree used to build the module value", "The registry mapping of module acquisition is the kernel's", "No value-only loader is exported from any package under opm" and "The acquired module's staged source is consumable by SynthesizeInstance".

## ADDED Requirements

### Requirement: The registry module loader loads the fetched module in memory and writes no temporary directory
The registry module loader SHALL load the fetched module in memory and SHALL NOT write the module's source to a temporary directory.

#### Scenario: No temporary directory created

- **WHEN** a module is loaded from the registry
- **THEN** no temporary directory is created or left behind for the module's source
- **AND** the module's transitive dependencies still resolve

### Requirement: The loader injects the fetched module's CUE files under a deterministic synthetic root and leaves load.Config.FS nil
The registry module loader SHALL inject the fetched module's CUE files (every `.cue` file under the module root, the module's own `cue.mod/module.cue` included, and nothing else) via `load.Config.Overlay` under a deterministic synthetic root, leaving `load.Config.FS` nil so the module's transitive dependencies resolve through the registry and CUE module cache. The staged overlay the loader returns on the module's `Source` SHALL be that same set of files.

#### Scenario: Staged overlay carries the module's CUE files only

- **WHEN** a fetched module's archive contains `.cue` files, a `cue.mod/module.cue`, and non-CUE files such as a license or a readme
- **THEN** the staged overlay on `Module.Source` holds every `.cue` file and `cue.mod/module.cue`, each keyed under the synthetic root
- **AND** the non-CUE files are not present in the overlay
- **AND** a fetch that stages no `.cue` file fails the load with an error wrapping `ErrInvalidPackage` (a defensive branch: a module zip CUE's fetch accepts always carries `cue.mod/module.cue`, so the empty case is exercised at the walker, not through a registry)

### Requirement: The synthetic root is an absolute path that does not exist on disk and the loader refuses the acquire when anything exists at it
The synthetic root SHALL be an absolute path on every operating system (on Windows it carries the current volume) that does not exist on disk. Because cue/load reads a real directory beneath an overlay root, the loader SHALL refuse the acquire, with an error naming the path and before anything is built, when anything (a directory, a file or a symlink) exists at the synthetic root. The same holds for every artifact kind the loader fetches from a registry, a catalog included.

#### Scenario: An existing synthetic root is refused

- **WHEN** a directory holding a `.cue` file of the module's package, or a plain file, exists at the synthetic root of a module version and that version is fetched from the registry
- **THEN** the load returns an error naming the synthetic root, and no value and no source
- **AND** once nothing exists at the root, the same fetch succeeds

#### Scenario: An existing synthetic root refuses a catalog acquire

- **WHEN** a directory holding a `.cue` file, or a plain file, exists at the synthetic root of a catalog version and that version is fetched from the registry with the catalog spec
- **THEN** the load returns an error naming the synthetic root, and no value and no source
- **AND** once nothing exists at the root, the same fetch succeeds

### Requirement: The registry module loader evaluates and shape-gates the fetched module through the kernel's one routine
The registry module loader SHALL evaluate and shape-gate the fetched module through the kernel's one evaluate-and-shape-gate routine, the same routine directory acquisition uses (concrete `kind == "Module"`; `metadata.name`, `metadata.modulePath`, `metadata.version` present and concrete), returning errors that wrap the same sentinels (`ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField`).

#### Scenario: Wrong artifact kind rejected

- **WHEN** the resolved registry artifact has a concrete `kind` other than `"Module"`
- **THEN** the loader returns a zero `cue.Value` and an error wrapping `ErrWrongKind`

#### Scenario: Missing identity field rejected

- **WHEN** the resolved module lacks a concrete `metadata.modulePath`
- **THEN** the loader returns an error wrapping `ErrMissingRequiredField`

### Requirement: The registry and directory acquisition paths differ only in where the package files come from
The two acquisition paths (Requirement: The registry module loader evaluates and shape-gates the fetched module through the kernel's one routine) SHALL differ only in where the package files come from: a fetched overlay under a synthetic root, or a directory on disk.

#### Scenario: Registry and directory acquisition fail identically

- **WHEN** the same malformed module is acquired once from a registry and once from a directory
- **THEN** both acquisitions return an error wrapping the same `opm/errors` sentinel
- **AND** a well-formed module acquired both ways yields the same `metadata` values

### Requirement: The registry module loader performs no full schema validation
The registry module loader SHALL NOT perform full schema validation, which remains the Kernel's contract.

#### Scenario: Full schema validation stays the Kernel's contract

- **WHEN** the registry module loader evaluates and shape-gates a fetched module
- **THEN** the registry module loader does not perform full schema validation, which remains the Kernel's contract

### Requirement: The loader verifies a module's declared identity against its fetched coordinate
After the artifact shape gate passes, the registry module loader SHALL verify the acquired artifact's declared identity against the coordinate it was fetched by: `metadata.modulePath` SHALL equal the requested module path (both in the full major-suffixed form, compared as strings without recomposition), and `metadata.version` SHALL equal the fetched tag with its `v` prefix stripped. A disagreement SHALL fail the load with a typed identity error naming both the declared and the fetched value.

#### Scenario: Address mismatch refused at acquire

- **WHEN** a published module is fetched by `opmodel.dev/modules/demo@v1` but declares `metadata.modulePath: "opmodel.dev/modules/other@v1"`
- **THEN** the load fails with a typed error carrying both paths

#### Scenario: Version mismatch refused at acquire

- **WHEN** a published module is fetched by tag `v2.0.2` but declares `metadata.version: "2.0.0"`
- **THEN** the load fails with a typed error carrying both versions

#### Scenario: Honest artifact loads

- **WHEN** the declared path and version match the fetched coordinate
- **THEN** the load proceeds exactly as before this requirement

### Requirement: The identity verification sits on the single module entrypoint of the shared fetch routine
The verification (Requirement: The loader verifies a module's declared identity against its fetched coordinate) SHALL sit on the single module entrypoint of the shared fetch routine, so every frontend acquiring a module through the kernel inherits one implementation.

#### Scenario: Every frontend inherits one verification

- **WHEN** a frontend acquires a module through the kernel
- **THEN** the verification sits on the single module entrypoint of the shared fetch routine, so every frontend acquiring a module through the kernel inherits one implementation

### Requirement: The identity verification is scoped to modules
The verification (Requirement: The loader verifies a module's declared identity against its fetched coordinate) SHALL be scoped to modules. The kind-parameterized fetch routine underneath the module entrypoint SHALL NOT run it, so an artifact of another kind traverses the shared fetch without a coordinate check; a kind whose requirements call for one states that requirement itself.

#### Scenario: A non-module kind traverses the shared fetch unchecked

- **WHEN** an artifact of a kind other than `Module` is fetched through the shared fetch routine
- **THEN** no coordinate identity check runs and no identity error can be raised by that routine
- **AND** the module entrypoint above it still runs the check for every module

### Requirement: The identity check is distinguishable in the code rather than by convention
The two behaviours (Requirement: The identity verification is scoped to modules) SHALL be distinguishable in the code rather than by convention: the check lives in the module entrypoint, not in the routine both kinds share.

#### Scenario: The check is found in the module entrypoint

- **WHEN** the identity check is located in the code
- **THEN** the check lives in the module entrypoint, not in the routine both kinds share

### Requirement: A major-free metadata.modulePath has no alternative verification and no module address is composed from a parent path and a name
There SHALL be no alternative verification for a major-free `metadata.modulePath`: the OPM schema the library consumes requires the major-suffixed form, so a declaration without the suffix cannot equal the fetched path and is refused with the same typed error (Requirement: The loader verifies a module's declared identity against its fetched coordinate). The library SHALL NOT compose a module address from a parent path and a name.

#### Scenario: Older-line parent-path declaration verified by convention

- **WHEN** a module fetched by `testing.opmodel.dev/x/modules/hello@v0` declares the major-free `metadata.modulePath: "testing.opmodel.dev/x/modules"` (the core-v0/v1 shape)
- **THEN** the load fails with the typed identity error for the `path` field, carrying the declared parent path and the fetched path
- **AND** no publishing-convention fallback is attempted

### Requirement: The kernel is a verifier and never a stamper of the declared identity
The kernel SHALL NOT write or correct either value (Requirement: The loader verifies a module's declared identity against its fetched coordinate): the schema declares identity, the reader verifies it (the kernel is a verifier, never a stamper).

#### Scenario: The kernel does not stamp the declared identity

- **WHEN** the kernel verifies the declared identity of an acquired module
- **THEN** the kernel does not write or correct either value, because the schema declares identity and the reader verifies it

### Requirement: Each artifact kind has one registry acquisition entry point
The library SHALL provide exactly one registry acquisition entry point per artifact kind.

#### Scenario: Each artifact kind has one registry acquisition entry point

- **WHEN** a developer lists the registry acquisition entry points of the library
- **THEN** there is exactly one registry acquisition entry point per artifact kind

### Requirement: One fetch routine serves every artifact kind
Every kind's entry point SHALL be served by one fetch routine, parameterized by the shape it gates to. Fetching and staging a published CUE module artifact reads no `kind`, so a second fetch implementation SHALL NOT exist for any kind.

#### Scenario: One fetch routine serves every kind

- **WHEN** a developer searches `opm/internal/loader` for the code that resolves a coordinate, fetches it and stages its files
- **THEN** exactly one routine does it, taking the shape to gate to as a parameter
- **AND** each kind's registry entry point calls that routine with its own shape rather than repeating the fetch

### Requirement: AcquireModuleFromRegistry fetches a published module via CUE's native module machinery
For modules that entry point (Requirement: Each artifact kind has one registry acquisition entry point) SHALL be `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which fetches a published `#Module` via CUE's native module machinery.

#### Scenario: Acquisition fetches a published module

- **WHEN** Kernel.AcquireModuleFromRegistry is called for a published module
- **THEN** it fetches the module via CUE's native module machinery

### Requirement: AcquireModuleFromRegistry loads the module in memory as the main module
For modules that entry point (Requirement: Each artifact kind has one registry acquisition entry point) SHALL be `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which loads it in memory as the main module (its own `cue.mod/module.cue` drives transitive resolution, its `kind`/`metadata` evaluate at the package root, no wrapper package is synthesized and no temporary directory is written).

#### Scenario: Acquisition loads the module in memory as the main module

- **WHEN** Kernel.AcquireModuleFromRegistry is called for a published module
- **THEN** it loads the module in memory as the main module, with no wrapper package synthesized and no temporary directory written

### Requirement: AcquireModuleFromRegistry runs the shape gate and verifies the declared identity
For modules that entry point (Requirement: Each artifact kind has one registry acquisition entry point) SHALL be `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which runs the acquisition shape gate, verifies the declared identity against the fetched coordinate.

#### Scenario: Acquisition runs the shape gate and verifies the identity

- **WHEN** Kernel.AcquireModuleFromRegistry is called for a published module
- **THEN** it runs the acquisition shape gate and verifies the declared identity against the fetched coordinate

### Requirement: AcquireModuleFromRegistry returns the module with its staged source attached
For modules that entry point (Requirement: Each artifact kind has one registry acquisition entry point) SHALL be `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which returns a `*module.Module` whose staged source is attached: `Source.Root` the deterministic synthetic root, `Source.Overlay` the module's `.cue` files as bytes (its own `cue.mod/module.cue` included and nothing else).

#### Scenario: Acquired module carries reusable staged source

- **WHEN** `Kernel.AcquireModuleFromRegistry(ctx, "<path>@v2", "v2.1.0")` is called against a registry serving the module
- **THEN** it returns a `*module.Module` whose staged source (overlay of bytes plus synthetic root) is populated and `HasSource()` reports true
- **AND** the staged source is the one used to build the module value (no second fetch is performed to obtain it)
- **AND** the process environment is not mutated

### Requirement: The staged source is the tree used to build the module value
The staged source (Requirement: AcquireModuleFromRegistry returns the module with its staged source attached) SHALL be the same tree used to build the module value, so no second fetch is required to reuse it.

#### Scenario: The staged source is the one used to build the module value

- **WHEN** a module is acquired from the registry
- **THEN** the staged source is the same tree used to build the module value, so no second fetch is required to reuse it

### Requirement: The registry mapping of module acquisition is the kernel's
The registry mapping (Requirement: AcquireModuleFromRegistry fetches a published module via CUE's native module machinery) SHALL be the kernel's (`WithRegistry`), applied through the resolver and load configuration and never through `os.Setenv`.

#### Scenario: Registry mapping is the kernel's

- **WHEN** a kernel constructed with `WithRegistry(mapping)` acquires a module
- **THEN** the fetch and the module's transitive imports resolve through `mapping` with no per-call argument

### Requirement: No value-only loader is exported from any package under opm
No value-only loader SHALL be exported from any package under `opm/`.

#### Scenario: No value-only loader

- **WHEN** a consumer searches every package under `opm/` for a registry loader returning a bare `cue.Value`
- **THEN** none exists; `Kernel.AcquireModuleFromRegistry` is the single module entry point and the raw value is `Module.Package`

### Requirement: The acquired module's staged source is consumable by SynthesizeInstance
The acquired module's staged source SHALL be consumable by `SynthesizeInstance` to construct an instance inside the module's own main module, so transitive dependencies resolve via the module's own `cue.mod/module.cue`.

#### Scenario: Staged source drives transitive resolution during synthesis

- **WHEN** a module acquired via `AcquireModuleFromRegistry` is passed to `SynthesizeInstance`
- **THEN** the instance is staged inside the module's source tree and the module's own `cue.mod/module.cue` resolves the transitive (catalog) closure
- **AND** synthesis succeeds without the caller declaring the module's transitive dependencies
