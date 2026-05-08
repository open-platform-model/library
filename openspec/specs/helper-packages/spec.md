# helper-packages Specification

## Purpose
The `pkg/helper/` subdirectory is the opt-in convenience boundary of the OPM library. Subpackages under `pkg/helper/` are opinionated frontend conveniences that wrap kernel primitives for specific embedding patterns; a frontend MAY skip them and call the kernel directly. Anything outside `pkg/helper/` is part of the kernel core contract that every frontend (CLI, controller, Crossplane composition function, future runtimes) MUST honour. This boundary keeps the kernel's public surface small and lets the helper layer evolve independently. Future kernel-redesign slices add subpackages here (loaders, layered values, Platform composition); each new helper requires its own slice.

## Requirements

### Requirement: Helper Boundary at pkg/helper/

The library SHALL maintain a `pkg/helper/` subdirectory whose subpackages are opt-in, opinionated frontend conveniences. Anything outside `pkg/helper/` SHALL be considered part of the kernel core contract.

#### Scenario: Helper boundary documented

- **WHEN** a developer reads `pkg/helper/doc.go`
- **THEN** the file documents that anything under `pkg/helper/` is opt-in and a frontend MAY skip it
- **AND** documents that anything outside `pkg/helper/` is part of the kernel contract

### Requirement: Loader Reorganization Under Helper

The filesystem-coupled loader SHALL live at `pkg/helper/loader/file/`. The package SHALL retain the public API of the prior `pkg/loader/` package: `LoadModulePackage`, `LoadReleaseFile`, `LoadValuesFile`, `LoadProvider`, and any associated types (`LoadOptions`).

#### Scenario: New loader path callable

- **WHEN** a caller invokes `loaderfile.LoadReleaseFile(ctx, path, opts)` (where `loaderfile` is the alias for `pkg/helper/loader/file`)
- **THEN** the function performs filesystem-based release loading exactly as the previous `pkg/loader/` package did
- **AND** the symbol identity, return shape, and error semantics are unchanged from the prior package

#### Scenario: Bytes loader skeleton present

- **WHEN** a developer reads `pkg/helper/loader/bytes/`
- **THEN** the package has a doc-only file describing intent (in-memory loading for Crossplane fn / tests / fuzzing)
- **AND** the package exposes no functions yet

### Requirement: Deprecation Shim at pkg/loader/

The old `pkg/loader/` import path SHALL retain a thin re-export package. Each function SHALL be a deprecated alias that delegates to its `pkg/helper/loader/file/` counterpart.

#### Scenario: Old import path still compiles

- **WHEN** existing downstream code imports `pkg/loader` and calls `loader.LoadReleaseFile(ctx, path, opts)`
- **THEN** the call succeeds with behavior identical to `loaderfile.LoadReleaseFile(...)`
- **AND** the function carries a `// Deprecated:` doc comment pointing to `pkg/helper/loader/file`

#### Scenario: Shim removal scheduled

- **WHEN** the next MAJOR release after this slice is planned
- **THEN** the shim is scheduled for removal
- **AND** CHANGELOG and documentation note the removal cycle

### Requirement: Helper Layout for Future Subpackages

Future opt-in helpers SHALL follow the `pkg/helper/<name>/` convention. Subpackages SHALL be added by their owning slices (e.g. `pkg/helper/platform/` by slice 10) and not as part of this slice.

#### Scenario: Platform helper landing place

- **WHEN** slice 10 (`add-platform-composition-helper`) is implemented
- **THEN** `pkg/helper/platform/` is the directory it occupies
- **AND** the convention is consistent with `pkg/helper/values/` and `pkg/helper/loader/file/`

### Requirement: Compose Function

The library SHALL expose `func Compose(k *kernel.Kernel, shell *platform.Platform, modules []*module.Module) (*platform.Platform, error)` in `pkg/helper/platform/`. The function SHALL produce a fully-composed Platform by FillPath-injecting each Module into `shell.Package` at `binding.Paths().Registry[<id>]`, evaluating the result, and returning a new `*Platform`.

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

The Kernel SHALL expose `(k *Kernel) ComposePlatform(shell *Platform, modules []*Module) (*Platform, error)` delegating to `pkg/helper/platform.Compose`.

#### Scenario: Kernel method matches helper

- **WHEN** a caller invokes `k.ComposePlatform(shell, modules)`
- **THEN** the result is identical to calling `helper/platform.Compose(k, shell, modules)` directly
