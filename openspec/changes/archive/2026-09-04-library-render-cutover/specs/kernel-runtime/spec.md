## MODIFIED Requirements

### Requirement: Goroutine Safety Contract

A single `Kernel` SHALL NOT be used concurrently across its own method calls: the owned `*cue.Context` (used by acquisition, synthesis and validation) is driven single-threaded, and sharing one `Kernel` between goroutines can race inside CUE evaluation. Callers needing concurrent operations SHALL construct one `Kernel` per goroutine; the package documentation SHALL state this and provide a one-Kernel-per-goroutine example.

`Kernel.Render` SHALL share nothing between renders: each render is its own CUE build in a fresh `cue.Context` that is released when `Render` returns, no built value is retained by the kernel, and the Kernel's own context is not used. Concurrency is across renders, never within one; a consumer rendering from several goroutines gives each goroutine its own Kernel and calls `Render`, with no shared platform value and no mutex. The package documentation SHALL state this, SHALL NOT present any shared built value as a supported shape, and SHALL state that a render pool is sized by memory (about 61 MB plus 7.75 MB per component per concurrent render, 0019 experiment 08) rather than by core count. The retracted shared-materialized-platform model and its mutex stopgap SHALL NOT appear as supported shapes.

#### Scenario: Documentation states the contract

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** it states that a single `Kernel` is not safe for concurrent use across its own method calls, shows one-Kernel-per-goroutine usage, and states that `Render` shares nothing between renders

#### Scenario: Documentation retracts the shared-platform model

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** no shared materialized platform, held built value or mutex-serialised render appears as a supported shape; the retraction is recorded in ADR-002's supersession header and the shares-nothing rule in ADR-005

#### Scenario: Repeated renders retain nothing

- **WHEN** `Render` is invoked repeatedly on one Kernel
- **THEN** each invocation builds in a fresh context, the results are byte-identical for identical inputs, and the kernel holds no built value between calls

#### Scenario: Race detector runs on the render packages

- **WHEN** the repository test task runs
- **THEN** `opm/kernel` and `opm/internal/renderstage` are additionally run under `go test -race`

### Requirement: Registry Configuration Option

The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the OCI registry mapping used for catalog and schema resolution: the render build's catalog imports (`Render`), registry module acquisition (`AcquireModuleFromRegistry`) and the schema cache. Absent the option, the kernel SHALL inherit `CUE_REGISTRY` from the process environment and SHALL NOT auto-apply a built-in default registry. The option MUST NOT mutate process environment state; the mapping is plumbed into each operation's load configuration (for a render, the staged module's `load.Config.Env`).

#### Scenario: Registry option used for resolution

- **WHEN** `kernel.New(WithRegistry("opmodel.dev=ghcr.io/open-platform-model"))` is called and `Render` runs against a platform whose `cue.mod` names a catalog under `opmodel.dev`
- **THEN** the catalog import resolves through that mapping
- **AND** the process environment is not mutated

#### Scenario: No default applied

- **WHEN** `kernel.New()` is called with no registry option
- **THEN** the kernel inherits the process `CUE_REGISTRY`
- **AND** applies no built-in default mapping

### Requirement: No Utility Methods on Kernel

The Kernel SHALL expose only the pipeline it runs (acquire, load, process, validate, synthesize, render) and SHALL NOT expose a finalization, constraint-stripping or other value-utility method.

#### Scenario: No finalization method on the Kernel

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/kernel`
- **THEN** neither `Finalize` nor `FinalizeValue` exists
- **AND** no method accepts a second, narrowed components value: the render build reads the imported instance's `components` once, for matching and execution alike

## ADDED Requirements

### Requirement: Tier-2 validation runs where values are applied

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema at the point they are filled into the instance, regardless of whether a Tier-1 helper validated them upstream. The Tier-2 entry point is `Kernel.ValidateConfig`, invoked by `Kernel.ProcessModuleInstance` (and so by `Kernel.SynthesizeInstance`). `Kernel.Render` SHALL NOT perform a second validation pass: the render build imports the instance as processed, which is already concrete.

#### Scenario: Kernel re-validates after Detailed

- **WHEN** a frontend that uses `k.ValidateConfigDetailed` supplies the resulting unified value to `k.ProcessModuleInstance`
- **THEN** the kernel performs full schema validation on the unified value via `ValidateConfig` before filling it
- **AND** any schema violation produces a CUE-native error walkable via `cueerrors.Errors`, wrapped with the instance name

#### Scenario: Kernel validates without Detailed

- **WHEN** a frontend skips `ValidateConfigDetailed` and feeds raw unified values to `ProcessModuleInstance` directly
- **THEN** the kernel still produces correct schema-validation errors
- **AND** the only loss is per-source attribution in error positions (`pos.Filename()` is empty unless the caller compiled with `cue.Filename(...)` themselves)

#### Scenario: Render does not re-validate

- **WHEN** a caller invokes `k.Render` with an instance returned by `ProcessModuleInstance` or `SynthesizeInstance`
- **THEN** no `#config` validation runs inside `Render`; the staged instance package is imported by the build as it was processed

## REMOVED Requirements

### Requirement: Phase-Explicit Methods on Kernel

**Reason**: `Match` and `Compile` are removed with the old pipeline (0019 D9/D10); `Render` is the single render verb and its input struct is specified by `single-build-render`.

**Migration**: `Compile` callers use `Render`; `Match` callers use `Render` and read `RenderDiagnostics.Pairs` and `Unmatched`, discarding `Compiled`.

### Requirement: Phase Input Structs

**Reason**: `MatchInput` and `CompileInput` are removed; `RenderInput` (instance, platform, runtime name, skew policy) is the only render input and is specified by `single-build-render`.

**Migration**: construct `kernel.RenderInput` with a source-carrying instance and platform.

### Requirement: Compile Rename

**Reason**: the compile pipeline and its `Compile` verb are deleted; there is no `opm/compile` package.

**Migration**: none beyond the `Render` migration above.

### Requirement: Materialize Method on Kernel

**Reason**: `opm/materialize` is deleted (0019 D5): the platform module's own imports replace pull-plus-index, and `#composedTransformers` is derived by core.

**Migration**: acquire the platform module from disk (`AcquirePlatformFromDir`) and pass it to `Render`; the render build resolves the catalogs.

### Requirement: Compile sources its cue.Context from the caller Kernel

**Reason**: there is no compile pipeline and no shared platform value; each render builds in its own fresh context (D8).

**Migration**: none.

### Requirement: Tier-2 Validation Always Runs

**Reason**: the requirement named `Kernel.Compile` as the pass that must not re-validate; `Compile` is deleted. Restated as "Tier-2 validation runs where values are applied" with `Render` in that role.

**Migration**: none; `ProcessModuleInstance` and `SynthesizeInstance` are unchanged.
