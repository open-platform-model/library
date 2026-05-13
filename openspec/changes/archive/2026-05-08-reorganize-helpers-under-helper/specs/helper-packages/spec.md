## ADDED Requirements

### Requirement: Helper Boundary at opm/helper/

The library SHALL maintain a `opm/helper/` subdirectory whose subpackages are opt-in, opinionated frontend conveniences. Anything outside `opm/helper/` SHALL be considered part of the kernel core contract.

#### Scenario: Helper boundary documented

- **WHEN** a developer reads `opm/helper/doc.go`
- **THEN** the file documents that anything under `opm/helper/` is opt-in and a frontend MAY skip it
- **AND** documents that anything outside `opm/helper/` is part of the kernel contract

### Requirement: Loader Reorganization Under Helper

The filesystem-coupled loader SHALL live at `opm/helper/loader/file/`. The package SHALL retain the public API of the prior `opm/loader/` package: `LoadModulePackage`, `LoadReleaseFile`, `LoadValuesFile`, `LoadProvider`, and any associated types (`LoadOptions`).

#### Scenario: New loader path callable

- **WHEN** a caller invokes `loaderfile.LoadReleaseFile(ctx, path, opts)` (where `loaderfile` is the alias for `opm/helper/loader/file`)
- **THEN** the function performs filesystem-based release loading exactly as the previous `opm/loader/` package did
- **AND** the symbol identity, return shape, and error semantics are unchanged from the prior package

#### Scenario: Bytes loader skeleton present

- **WHEN** a developer reads `opm/helper/loader/bytes/`
- **THEN** the package has a doc-only file describing intent (in-memory loading for Crossplane fn / tests / fuzzing)
- **AND** the package exposes no functions yet

### Requirement: Deprecation Shim at opm/loader/

The old `opm/loader/` import path SHALL retain a thin re-export package. Each function SHALL be a deprecated alias that delegates to its `opm/helper/loader/file/` counterpart.

#### Scenario: Old import path still compiles

- **WHEN** existing downstream code imports `opm/loader` and calls `loader.LoadReleaseFile(ctx, path, opts)`
- **THEN** the call succeeds with behavior identical to `loaderfile.LoadReleaseFile(...)`
- **AND** the function carries a `// Deprecated:` doc comment pointing to `opm/helper/loader/file`

#### Scenario: Shim removal scheduled

- **WHEN** the next MAJOR release after this slice is planned
- **THEN** the shim is scheduled for removal
- **AND** CHANGELOG and documentation note the removal cycle

### Requirement: Helper Layout for Future Subpackages

Future opt-in helpers SHALL follow the `opm/helper/<name>/` convention. Subpackages SHALL be added by their owning slices (e.g. `opm/helper/platform/` by slice 10) and not as part of this slice.

#### Scenario: Platform helper landing place

- **WHEN** slice 10 (`add-platform-composition-helper`) is implemented
- **THEN** `opm/helper/platform/` is the directory it occupies
- **AND** the convention is consistent with `opm/helper/values/` and `opm/helper/loader/file/`
