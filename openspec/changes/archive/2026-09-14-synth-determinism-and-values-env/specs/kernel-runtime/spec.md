## MODIFIED Requirements

### Requirement: Registry Configuration Option

The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the one OCI registry mapping every kernel operation uses for catalog, module and schema resolution: the render build's catalog imports (`Render`), registry module acquisition (`AcquireModuleFromRegistry`), directory acquisition (`AcquireModuleFromDir`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`), instance synthesis (`SynthesizeInstance`), the compilation of file-backed values sources on every path that accepts `Source` values (`ValidateConfigDetailed`, `AcquireInstanceFromDir` with trailing values, `SynthesizeInstance`), and the default schema cache. No acquire verb SHALL take a per-call registry override. Absent the option, the kernel SHALL inherit `CUE_REGISTRY` from the process environment and SHALL NOT auto-apply a built-in default registry. The option MUST NOT mutate process environment state; the mapping is plumbed into each operation's load configuration.

#### Scenario: Registry option used for resolution

- **WHEN** `kernel.New(WithRegistry("opmodel.dev=ghcr.io/open-platform-model"))` is called and `Render` runs against a platform whose `cue.mod` names a catalog under `opmodel.dev`
- **THEN** the catalog import resolves through that mapping
- **AND** the process environment is not mutated

#### Scenario: Directory acquisition uses the kernel mapping

- **WHEN** a kernel constructed with `WithRegistry(mapping)` acquires a platform or instance from a directory whose imports resolve from `opmodel.dev`
- **THEN** those imports resolve through `mapping` with no per-call argument

#### Scenario: Values source compilation uses the kernel mapping

- **WHEN** a kernel constructed with `WithRegistry(mapping)` compiles a file-backed `Source` whose file imports a package that only `mapping` routes
- **THEN** the import resolves through `mapping` on `ValidateConfigDetailed`, on `AcquireInstanceFromDir` with that source as a trailing value, and on `SynthesizeInstance` with it in `InstanceInput.Values`
- **AND** the process environment is not mutated

#### Scenario: No per-call registry parameter

- **WHEN** a consumer inspects the signatures of the acquire verbs and `SynthesizeInstance`
- **THEN** none takes a load-options or registry argument, and no `LoadOptions` type is exported from `opm/`

#### Scenario: No default applied

- **WHEN** `kernel.New()` is called with no registry option
- **THEN** the kernel inherits the process `CUE_REGISTRY`
- **AND** applies no built-in default mapping
