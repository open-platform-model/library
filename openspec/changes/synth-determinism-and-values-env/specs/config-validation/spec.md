## ADDED Requirements

### Requirement: File-Backed Sources Resolve Imports Through the Kernel Mapping

When a `Source` is file-backed (its `Origin` is an absolute path naming an existing file), the operation that compiles it SHALL load the file at its directory with the kernel's registry mapping (`WithRegistry`) applied to the load configuration, exactly as directory acquisition applies it. Absent the option, the load SHALL read the process `CUE_REGISTRY` unchanged. This SHALL hold on every path that compiles sources: `ValidateConfigDetailed`, `AcquireInstanceFromDir` with trailing values, `SynthesizeInstance`, and the kernel's internal per-source attribution pass. A source compiled from bytes carries no imports and is unaffected. The process environment SHALL NOT be mutated.

#### Scenario: Values file importing a registry module

- **WHEN** a kernel constructed with `WithRegistry(mapping)` compiles a file-backed `Source` whose file imports a package served only through `mapping`
- **THEN** the import resolves and the source compiles
- **AND** the same source compiled by a kernel constructed without the option, in a process whose `CUE_REGISTRY` does not route that path, fails at the import

#### Scenario: Same mapping on every compiling path

- **WHEN** the file-backed source above is passed to `ValidateConfigDetailed`, as a trailing value to `AcquireInstanceFromDir`, and as `InstanceInput.Values` to `SynthesizeInstance`, on the kernel constructed with `WithRegistry(mapping)`
- **THEN** each path resolves the import through `mapping`
- **AND** the process `CUE_REGISTRY` is unchanged afterwards
