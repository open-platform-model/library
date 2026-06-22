## MODIFIED Requirements

### Requirement: MaterializedPlatform Output Shape

The Materialize operation SHALL return a `*MaterializedPlatform` that exposes the composed transformers and the matcher reverse index as **native first-class fields built in the owner `*cue.Context`**: a `Transformers` value answering `LookupPath` for each indexed transformer FQN (each carrying a `#transform`), and a `Matchers` value answering `LookupPath` for `resources` and `traits`. It SHALL record the resolved version per subscription path for diagnostics. Inputs SHALL NOT be mutated.

The Materialize operation SHALL NOT `FillPath` the composed transformer map or the matcher index onto the closed `c.#Platform` value, and the returned `*MaterializedPlatform` SHALL NOT expose a `Composed` field nor a `Package` field for these purposes. The original closed platform spec (carrying `#registry`, metadata) SHALL remain reachable via `Source.Package` for diagnostics and registry inspection. As a consequence of building the surfaces natively (rather than filling them into a closed value), reading a transformer's `#transform` directly off `Transformers` — including output-local hidden fields — SHALL evaluate to concrete values; no separate open map and no read-from-`Package` prohibition is required.

Selection of multiple versions of the same catalog `path@major` (e.g. a `range` admitting `0.5.0` and `0.5.1`) SHALL be preserved: each selected version's transformers are indexed under their distinct version-bearing FQNs and remain independently reachable in `Transformers`.

The native surfaces SHALL be consumed by the compile pipeline as **read-only input** — looked up and filled-*from* — and SHALL NOT be used as the owner of the compile build context (which is sourced from the caller Kernel; see the `kernel-runtime` capability).

Under the v0.17 CUE toolchain, the returned `*MaterializedPlatform` SHALL be safe for concurrent **read-only** consumption: a platform materialized once MAY be rendered against simultaneously by multiple Kernels' compile pipelines (one Kernel per goroutine) without a mutex and without re-materialization. This is the basis of the materialize-once-reuse-many model the Platform-CR design depends on. Concurrent consumers SHALL NOT mutate the shared `Transformers`/`Matchers`; the pipeline only looks up and fills *from* them, building results in each caller Kernel's own context.

#### Scenario: Composed transformers reachable

- **WHEN** Materialize succeeds
- **THEN** `mp.Transformers.LookupPath(<tfFQN>)` exists for each indexed transformer
- **AND** the resolved version for each subscription path is retrievable for diagnostics

#### Scenario: Transforms render concrete off the native surface

- **WHEN** a composed transformer declaring output-local hidden fields is rendered by reading its `#transform` directly off `mp.Transformers`
- **THEN** the output evaluates to concrete values (no `non-concrete value _` from corrupted lazy resolution)
- **AND** no separate `Composed` map and no closed `Package` twin is required or exposed to obtain concrete output

#### Scenario: Matcher index reachable on the native surface

- **WHEN** Materialize succeeds
- **THEN** `mp.Matchers.LookupPath("resources")` and `mp.Matchers.LookupPath("traits")` exist and contain the reverse index built from the selected transformers

#### Scenario: Multi-version composition preserved

- **WHEN** a subscription filter selects two versions of the same catalog path (e.g. `0.5.0` and `0.5.1`)
- **THEN** `mp.Transformers` contains a distinct entry for each version's transformers under its version-bearing FQN
- **AND** both are independently reachable for matching by exact FQN

#### Scenario: Idempotent and non-mutating

- **WHEN** Materialize is called twice with identical inputs
- **THEN** the returned platforms are semantically identical
- **AND** the input `*platform.Platform` is unchanged after each call

#### Scenario: Consumed as read-only compile input

- **WHEN** `Kernel.Compile` renders a release against a materialized platform
- **THEN** the platform's `Transformers`/`Matchers` are read (looked up and filled-from) but the build context for the rendered output comes from the caller Kernel, not from `mp.Transformers.Context()`

#### Scenario: Safe for concurrent read-only sharing across Kernels

- **WHEN** a single `*MaterializedPlatform` is materialized once and then rendered against concurrently by multiple per-goroutine Kernels under the v0.17 toolchain
- **THEN** the shared `Transformers`/`Matchers` are consumed read-only by every concurrent compile without mutation or re-materialization
- **AND** no data race occurs and each render produces the output correct for its own release

> Note: the prior `MaterializedPlatform.Composed` field (the open map the executor was required to read instead of `Package`) and `MaterializedPlatform.Package` (the closed twin onto which the composed map was filled) are removed. Their roles are replaced respectively by `Transformers` (the canonical transform surface, concrete by construction) and `Source.Package` (the closed spec, read only for `#registry`/metadata/diagnostics). The *Transforms render concrete off the native surface* scenario is the observable guarantee that the closedness corruption is eliminated structurally rather than worked around.
