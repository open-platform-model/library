## ADDED Requirements

### Requirement: Kernel verbs check cancellation at entry and between stages

`Kernel.AcquireModuleFromDir`, `Kernel.AcquireCatalogFromDir`, `Kernel.AcquirePlatformFromDir`, `Kernel.AcquireInstanceFromDir` and `Kernel.SynthesizeInstance` SHALL check their context after their argument checks and between their stages: after the directory is read, after the values sources are merged, after the package is built and before the next stage starts. The registry acquire verbs SHALL check it after the registry fetch returns, and `Kernel.Render` SHALL check it after the render build. When the context is done, the verb SHALL return the context's own error unwrapped (so `errors.Is(err, context.Canceled)` or `errors.Is(err, context.DeadlineExceeded)` holds) and no artifact.

A running `cue/load` or build SHALL NOT be interrupted; cancellation lands at the next stage boundary. The `opm/kernel` package doc and each of these verbs' godoc SHALL say so. Cancellation inside a stage is outside this requirement. Source: 0009:D9, as revised on 2026-10-03.

#### Scenario: A cancelled context stops each directory verb

- **WHEN** `AcquireModuleFromDir`, `AcquireCatalogFromDir`, `AcquirePlatformFromDir` or `AcquireInstanceFromDir` is called with a context that is already cancelled and a valid directory
- **THEN** it returns `context.Canceled` and a nil artifact

#### Scenario: A cancelled context stops synthesis

- **WHEN** `SynthesizeInstance` is called with a context that is already cancelled and a complete `InstanceInput`
- **THEN** it returns `context.Canceled` and a nil instance

#### Scenario: Argument errors come first

- **WHEN** `SynthesizeInstance` is called with a cancelled context and an `InstanceInput` with no `Module`
- **THEN** the error wraps `oerrors.ErrMissingModule`

#### Scenario: A fetch served from the cache still observes cancellation

- **WHEN** `AcquireModuleFromRegistry` is called with a cancelled context for a module the local cache already holds, so the fetch returns without a network request
- **THEN** it returns an error for which `errors.Is(err, context.Canceled)` holds, and no module

#### Scenario: The godoc states where cancellation lands

- **WHEN** a developer reads `go doc ./opm/kernel`
- **THEN** a Cancellation section says that the context is checked at entry and between stages, and that a running load or build is not interrupted
