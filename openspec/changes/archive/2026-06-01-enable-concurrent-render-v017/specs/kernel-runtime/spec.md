## MODIFIED Requirements

### Requirement: Goroutine Safety Contract

A single `Kernel` SHALL NOT be used concurrently across its own method calls — the owned `*cue.Context` is driven single-threaded, and sharing one `Kernel` between goroutines can race inside CUE evaluation. Callers needing concurrent operations SHALL construct one `Kernel` per goroutine; the package documentation SHALL state this and provide a one-Kernel-per-goroutine example.

Under the v0.17 CUE toolchain, a `*MaterializedPlatform` produced by one `Kernel` SHALL be safe to share **read-only** across goroutines and other Kernels: many per-goroutine Kernels MAY render distinct `ModuleRelease`s concurrently against a single platform that was materialized once, with no mutex and no re-materialization. This is sound because the compile pipeline builds every value it constructs in the **caller** Kernel's `*cue.Context` and only cross-*reads* the shared platform (see the "Compile sources its cue.Context from the caller Kernel" requirement). The package documentation SHALL describe this concurrent-render model and provide an example of rendering against a shared platform.

#### Scenario: Documentation states the contract

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** the documentation explicitly states that a single `Kernel` is not safe for concurrent use across its own method calls
- **AND** the documentation provides an example showing one-Kernel-per-goroutine usage in a multi-worker scenario

#### Scenario: Documentation states the shared-platform concurrency model

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** the documentation states that, under the v0.17 toolchain, a `*MaterializedPlatform` materialized once is safe to be read concurrently by many per-goroutine Kernels' `Compile` calls without a mutex or re-materialization
- **AND** the documentation provides an example of per-goroutine Kernels rendering against one shared materialized platform

#### Scenario: Concurrent rendering against a shared platform is race-clean and correct

- **WHEN** one Kernel materializes a platform once, and N other goroutines each construct their own Kernel and concurrently `Compile` a distinct `ModuleRelease` against that single shared `*MaterializedPlatform`, executed under the race detector
- **THEN** no data race is reported
- **AND** each goroutine's `CompileResult` contains the output expected for its own release, with no cross-contamination between concurrent renders
