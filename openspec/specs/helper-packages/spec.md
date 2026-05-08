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
