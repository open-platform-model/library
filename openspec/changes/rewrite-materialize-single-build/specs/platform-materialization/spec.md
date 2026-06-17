## MODIFIED Requirements

### Requirement: MaterializedPlatform Output Shape

The Materialize operation SHALL return a `*MaterializedPlatform` whose `Package` answers `LookupPath` for `#composedTransformers`, `#matchers.resources`, and `#matchers.traits`, and which records the resolved version per subscription path for diagnostics. Inputs SHALL NOT be mutated.

The `Package` SHALL be produced by **single-build CUE evaluation** that composes the platform together with the selected catalog builds (per ADR-003), NOT by `FillPath`-ing a separately-built `#composedTransformers` map onto an independently-built closed `#Platform` value. As a consequence, the `Package` SHALL render each composed transformer's `#transform` — including output-local hidden fields — concretely when read directly; consumers SHALL NOT require a separate open map to read transforms. The `MaterializedPlatform` SHALL NOT expose a `Composed` field for this purpose (removed, or deprecated as an alias of `Package` for one release per the change's SemVer decision).

The `Package` SHALL be consumed by the compile pipeline as **read-only input** — looked up and filled-from — and SHALL NOT be used as the owner of the compile build context (which is sourced from the caller Kernel; see `kernel-runtime`).

Under the v0.17 CUE toolchain, the returned `*MaterializedPlatform` SHALL remain safe for concurrent **read-only** consumption: a platform materialized once MAY be rendered against simultaneously by multiple Kernels' compile pipelines (one Kernel per goroutine) without a mutex and without re-materialization. Concurrent consumers SHALL NOT mutate the shared `Package`.

#### Scenario: Composed transformers reachable

- **WHEN** Materialize succeeds
- **THEN** `mp.Package.LookupPath("#composedTransformers")` exists and contains the indexed transformers
- **AND** the resolved version for each subscription path is retrievable for diagnostics

#### Scenario: Transforms render concrete off Package

- **WHEN** a composed transformer declaring output-local hidden fields is rendered by reading its `#transform` directly off `mp.Package`
- **THEN** the output evaluates to concrete values (no `non-concrete value` from corrupted lazy resolution)
- **AND** no separate `Composed` map is required to obtain concrete output

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

> Note: the current `MaterializedPlatform.Composed` field and the "executor MUST read transforms from `Composed`, not `Package`" rule are implementation-level (code + comments in `opm/materialize/types.go` and `opm/compile/execute.go`), not a named requirement in this capability. They are eliminated by the single-build composition mandated above; the *Transforms render concrete off Package* scenario is the observable guarantee that replaces them.
