## MODIFIED Requirements

### Requirement: Kernel Artifact Type Set

The kernel SHALL accept exactly four artifact types: `Module`, `ModuleInstance`, `Platform` and `Catalog`. `#ModuleDebug` SHALL NOT be a kernel artifact type. Debug values are carried as a `debugValues` field within `Module.Package`; whether they participate in the values stack is a frontend policy decision, not a kernel concern.

`Catalog` is admitted on the terms ADR-009 records and on no others: the kernel acquires it, reads it and derives from it, and every verdict about what it reads stays with the caller. A catalog is never rendered and never executed; the transformers it carries reach a render only through a platform that subscribes to it. The set SHALL NOT grow again without the candidate meeting, in writing, the four-part test ADR-009 states.

#### Scenario: No top-level ModuleDebug type

- **WHEN** a developer searches the kernel public API for `ModuleDebug`
- **THEN** no exported Go type with that name exists in any `opm/` package
- **AND** the version binding (`opm/api/<version>/`) exposes no `DecodeModuleDebugMetadata` or equivalent

#### Scenario: debugValues accessible via Module.Package

- **WHEN** a frontend reads debug overlays from a Module
- **THEN** the read goes through `Module.Package.LookupPath(binding.Paths().DebugValues)` (or directly through CUE if binding does not enumerate the path)
- **AND** the kernel never receives `debugValues` as a separate parameter

#### Scenario: Documentation explicitly retires the construct

- **WHEN** a developer reads `library/README.md` or `opm/module/` godoc
- **THEN** at least one prose section states that `#ModuleDebug` is not a kernel artifact and that debug overlays are a frontend layering concern

#### Scenario: The fourth type is a read, never a render

- **WHEN** a developer searches the kernel public API for a way to render or execute a `Catalog`
- **THEN** none exists: `*catalog.Catalog` is produced only by the two catalog acquire verbs, and `Kernel.Render` takes an instance and a platform
- **AND** what the catalog provides and what it requires are reported by methods on the artifact, which return data and refuse nothing

#### Scenario: The enumerated set is stated once and agrees everywhere

- **WHEN** a developer reads the kernel's accepted-kinds list in `README.md`, `CLAUDE.md` and this spec
- **THEN** all three enumerate the same four types
- **AND** each names ADR-009 as the record of why the fourth was admitted and of the test a fifth must pass
