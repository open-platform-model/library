# helper-packages Specification

## Purpose
The `opm/helper/` subdirectory is the opt-in convenience boundary of the OPM library. Subpackages under `opm/helper/` are opinionated frontend conveniences that wrap kernel primitives for specific embedding patterns; a frontend MAY skip them and call the kernel directly. Anything outside `opm/helper/` is part of the kernel core contract that every frontend (CLI, controller, Crossplane composition function, future runtimes) MUST honour. This boundary keeps the kernel's public surface small and lets the helper layer evolve independently. Future kernel-redesign slices add subpackages here (loaders, layered values, Platform composition); each new helper requires its own slice.

## Requirements

### Requirement: Helper Boundary at opm/helper/

The library SHALL maintain a `opm/helper/` subdirectory whose subpackages are opt-in, opinionated frontend conveniences. Anything outside `opm/helper/` SHALL be considered part of the kernel core contract.

#### Scenario: Helper boundary documented

- **WHEN** a developer reads `opm/helper/doc.go`
- **THEN** the file documents that anything under `opm/helper/` is opt-in and a frontend MAY skip it
- **AND** documents that anything outside `opm/helper/` is part of the kernel contract

### Requirement: Loader Reorganization Under Helper

The filesystem-coupled loader SHALL live at `opm/helper/loader/file/`. The package SHALL expose the following public API:

- `LoadModulePackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#Module`, with registry override via `opts.Registry`.
- `LoadReleasePackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#ModuleRelease`, with registry override via `opts.Registry`.
- `LoadPlatformPackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#Platform`, with registry override via `opts.Registry`.
- `LoadProvider` and any associated types unchanged.

`LoadOptions` SHALL carry the registry override applied to `load.Config.Env`. All three package loaders SHALL accept the same `LoadOptions` value and share the same signature shape so that a caller can pass identical options through to module, release, and platform loads.

#### Scenario: Release package load

- **WHEN** a caller invokes `loaderfile.LoadReleasePackage(ctx, "./release-dir", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads every `.cue` file in `./release-dir` that shares the package as a single CUE package
- **AND** detects the apiVersion via `apiversion.Detect`
- **AND** returns the evaluated `cue.Value` and detected `apiversion.Version`
- **AND** an unrecognised or missing apiVersion wraps `apiversion.ErrUnknownAPIVersion`

#### Scenario: Platform package load

- **WHEN** a caller invokes `loaderfile.LoadPlatformPackage(ctx, "./platform-dir", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads every `.cue` file in `./platform-dir` that shares the package as a single CUE package and returns the detected `apiversion.Version`

#### Scenario: Module package load with registry override

- **WHEN** a caller invokes `loaderfile.LoadModulePackage(ctx, "./module", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads the module's CUE package using the supplied registry override
- **AND** module imports resolved from the registry succeed when the registry serves the required dependencies

#### Scenario: Multi-file release package

- **WHEN** a release directory contains both `release.cue` and `values.cue` declaring the same package name
- **AND** the caller invokes `loaderfile.LoadReleasePackage(ctx, dir, opts)`
- **THEN** the returned `cue.Value` reflects the unification of both files
- **AND** apiVersion detection succeeds against the unified value

#### Scenario: Bytes loader skeleton present

- **WHEN** a developer reads `opm/helper/loader/bytes/`
- **THEN** the package has a doc-only file describing intent (in-memory loading for Crossplane fn / tests / fuzzing)
- **AND** the package exposes no functions yet

### Requirement: Helper Layout for Future Subpackages

Future opt-in helpers SHALL follow the `opm/helper/<name>/` convention. Subpackages SHALL be added by their owning slices (e.g. `opm/helper/platform/` by slice 10) and not as part of the originating slice that established the convention. Past examples of helper subpackages in this convention SHALL reflect the current package layout; subpackages that have been collapsed into the kernel (such as the previous `opm/helper/values/`) SHALL NOT appear as exemplars.

#### Scenario: Platform helper landing place

- **WHEN** slice 10 (`add-platform-composition-helper`) is implemented
- **THEN** `opm/helper/platform/` is the directory it occupies
- **AND** the convention is consistent with `opm/helper/loader/file/`

#### Scenario: Values helper subpackage no longer exists

- **WHEN** a developer searches `opm/helper/` for a `values` subpackage
- **THEN** no `opm/helper/values/` directory exists
- **AND** the canonical implementation of layered values validation lives at `Kernel.ValidateConfigDetailed` in `opm/kernel/`

### Requirement: Compose Function

The library SHALL expose `func Compose(k *kernel.Kernel, shell *platform.Platform, modules []*module.Module) (*platform.Platform, error)` in `opm/helper/platform/`. The function SHALL produce a fully-composed Platform by FillPath-injecting each Module into `shell.Package` at `binding.Paths().Registry[<id>]`, evaluating the result, and returning a new `*Platform`.

#### Scenario: Successful composition

- **WHEN** a caller invokes `Compose(k, shell, []*Module{m1, m2})` where `shell` has an empty `#registry` and `m1`, `m2` register transformers without conflict
- **THEN** the returned `*Platform` has `Package` carrying a `#registry` with two entries (keyed by `m1.Metadata.Name` and `m2.Metadata.Name`)
- **AND** the computed views (`#composedTransformers`, `#matchers`, `#knownResources`, `#knownTraits`) include contributions from both Modules

#### Scenario: ID derived from module metadata name

- **WHEN** a Module is registered
- **THEN** the `#registry` key is the Module's `metadata.name` (kebab-case per catalog 014 D16)

#### Scenario: enabled defaults to true

- **WHEN** a Module is registered without explicit enable/disable instruction
- **THEN** the `#ModuleRegistration.enabled` field is set to `true` explicitly

#### Scenario: Inputs not mutated

- **WHEN** `Compose` is called twice with the same inputs
- **THEN** both calls return semantically identical `*Platform` values
- **AND** the input `shell` and `modules` are unchanged after each call

### Requirement: Multi-Fulfiller Error Surface

When two registered Modules' transformers claim the same primitive FQN (violating catalog 014 D13), `Compose` SHALL return a non-nil `*MultiFulfillerError`. The error type carries `FQN`, `ConflictingModules`, and `ConflictingTransformers` fields for structured attribution; these MAY be empty when the underlying CUE diagnostic does not surface enough structure to extract them safely. In that fallback case the raw CUE error is preserved on the value and reachable via `errors.Unwrap`, so frontends can still surface a useful diagnostic. Richer extraction (e.g. re-evaluating against `#PlatformBase` to read `#matchers._invalid`) is a follow-up; the initial slice ships the type and the detection, not necessarily the parser.

#### Scenario: Multi-fulfiller failure

- **WHEN** two Modules each register a transformer with `requiredResources["<fqn>"]`
- **THEN** `Compose` returns an error whose chain contains a `*MultiFulfillerError` (verifiable via `errors.As`)
- **AND** the wrapped CUE diagnostic (returned by `Unwrap`) describes the multi-fulfiller violation
- **AND** structured fields (`FQN`, `ConflictingModules`, `ConflictingTransformers`) MAY be empty when classification fell back to wrapping the raw error

### Requirement: Kernel Convenience Method

The Kernel SHALL expose `(k *Kernel) ComposePlatform(shell *Platform, modules []*Module) (*Platform, error)` delegating to `opm/helper/platform.Compose`.

#### Scenario: Kernel method matches helper

- **WHEN** a caller invokes `k.ComposePlatform(shell, modules)`
- **THEN** the result is identical to calling `helper/platform.Compose(k, shell, modules)` directly
