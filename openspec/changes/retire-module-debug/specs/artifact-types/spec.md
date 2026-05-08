## MODIFIED Requirements

### Requirement: Kernel Artifact Type Set

The kernel SHALL accept exactly three artifact types: `Module`, `ModuleRelease`, and `Platform`. `#ModuleDebug` SHALL NOT be a kernel artifact type. Debug values are carried as a `debugValues` field within `Module.Package`; whether they participate in the values stack is a frontend policy decision, not a kernel concern.

#### Scenario: No top-level ModuleDebug type

- **WHEN** a developer searches the kernel public API for `ModuleDebug`
- **THEN** no exported Go type with that name exists in any `pkg/` package
- **AND** the version binding (`pkg/api/<version>/`) exposes no `DecodeModuleDebugMetadata` or equivalent

#### Scenario: debugValues accessible via Module.Package

- **WHEN** a frontend reads debug overlays from a Module
- **THEN** the read goes through `Module.Package.LookupPath(binding.Paths().DebugValues)` (or directly through CUE if binding does not enumerate the path)
- **AND** the kernel never receives `debugValues` as a separate parameter

#### Scenario: Documentation explicitly retires the construct

- **WHEN** a developer reads `library/README.md` or `pkg/module/` godoc
- **THEN** at least one prose section states that `#ModuleDebug` is not a kernel artifact and that debug overlays are a frontend layering concern
