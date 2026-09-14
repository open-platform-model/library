## MODIFIED Requirements

### Requirement: Registry-Loaded Modules Pass the Module Shape Gate

The registry module loader SHALL evaluate and shape-gate the fetched module through the kernel's one evaluate-and-shape-gate routine, the same routine directory acquisition uses (concrete `kind == "Module"`; `metadata.name`, `metadata.modulePath`, `metadata.version` present and concrete), returning errors that wrap the same sentinels (`ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField`). The two acquisition paths SHALL differ only in where the package files come from: a fetched overlay under a synthetic root, or a directory on disk. It SHALL NOT perform full schema validation, which remains the Kernel's contract.

#### Scenario: Wrong artifact kind rejected

- **WHEN** the resolved registry artifact has a concrete `kind` other than `"Module"`
- **THEN** the loader returns a zero `cue.Value` and an error wrapping `ErrWrongKind`

#### Scenario: Missing identity field rejected

- **WHEN** the resolved module lacks a concrete `metadata.modulePath`
- **THEN** the loader returns an error wrapping `ErrMissingRequiredField`

#### Scenario: Registry and directory acquisition fail identically

- **WHEN** the same malformed module is acquired once from a registry and once from a directory
- **THEN** both acquisitions return an error wrapping the same `opm/errors` sentinel
- **AND** a well-formed module acquired both ways yields the same `metadata` values
