## ADDED Requirements

### Requirement: Registry Configuration Option

The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the OCI registry mapping used for catalog (and schema) resolution during `Materialize`. Absent the option, the kernel SHALL inherit `CUE_REGISTRY` from the process environment and SHALL NOT auto-apply a built-in default registry. The option MUST NOT mutate process environment state; the registry mapping is plumbed into the load configuration for the operation.

#### Scenario: Registry option used for resolution

- **WHEN** `kernel.New(WithRegistry("opmodel.dev=ghcr.io/open-platform-model"))` is called
- **THEN** catalog resolution during `Materialize` uses that mapping
- **AND** the process environment is not mutated

#### Scenario: No default applied

- **WHEN** `kernel.New()` is called with no registry option
- **THEN** the kernel inherits the process `CUE_REGISTRY`
- **AND** applies no built-in default mapping

### Requirement: Materialize Method on Kernel

The `Kernel` SHALL expose `(k *Kernel) Materialize(ctx context.Context, p *platform.Platform) (*MaterializedPlatform, error)` delegating to `opm/materialize`, using the kernel's configured registry and owned `*cue.Context`. Adding this method SHALL NOT change the signatures of existing phase methods in this slice.

#### Scenario: Delegates to materialize package

- **WHEN** a caller invokes `k.Materialize(ctx, plat)`
- **THEN** it returns the `*MaterializedPlatform` produced by `opm/materialize.Materialize` using the kernel's registry and context

#### Scenario: Existing phase signatures unchanged

- **WHEN** a developer reads `Match`, `Plan`, and `Compile` after this slice
- **THEN** their signatures are unchanged and still take `*platform.Platform`
