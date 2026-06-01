## MODIFIED Requirements

### Requirement: MaterializedPlatform Output Shape

The Materialize operation SHALL return a `*MaterializedPlatform` whose `Package` answers `LookupPath` for `#composedTransformers`, `#matchers.resources`, and `#matchers.traits`, and which records the resolved version per subscription path for diagnostics. Inputs SHALL NOT be mutated. The `Package` SHALL be consumed by the compile pipeline as **read-only input** — the `FillPath` argument / cross-read source — and SHALL NOT be used as the owner of the compile build context (which is sourced from the caller Kernel; see the `kernel-runtime` capability).

Under the v0.17 CUE toolchain, the returned `*MaterializedPlatform` SHALL be safe for concurrent **read-only** consumption: a platform materialized once MAY be rendered against simultaneously by multiple Kernels' compile pipelines (one Kernel per goroutine) without a mutex and without re-materialization. This is the basis of the materialize-once-reuse-many model the Platform-CR design depends on. Concurrent consumers SHALL NOT mutate the shared `Package`; the pipeline only looks up and fills *from* it, building results in each caller Kernel's own context.

#### Scenario: Composed transformers reachable

- **WHEN** Materialize succeeds
- **THEN** `mp.Package.LookupPath("#composedTransformers")` exists and contains the indexed transformers
- **AND** the resolved version for each subscription path is retrievable for diagnostics

#### Scenario: Idempotent and non-mutating

- **WHEN** Materialize is called twice with identical inputs
- **THEN** the returned platforms are semantically identical
- **AND** the input `*platform.Platform` is unchanged after each call

#### Scenario: Consumed as read-only compile input

- **WHEN** `Kernel.Compile` renders a release against a materialized platform
- **THEN** the platform's `Package` is read (looked up and filled-from) but the build context for the rendered output comes from the caller Kernel, not from `mp.Package.Context()`

#### Scenario: Safe for concurrent read-only sharing across Kernels

- **WHEN** a single `*MaterializedPlatform` is materialized once and then rendered against concurrently by multiple per-goroutine Kernels under the v0.17 toolchain
- **THEN** the shared `Package` is consumed read-only by every concurrent compile without mutation or re-materialization
- **AND** no data race occurs and each render produces the output correct for its own release
