## ADDED Requirements

### Requirement: A held Render output keeps its build alive

The kernel SHALL retain no built value between calls. Each `*kernel.Compiled` that `Render` returns carries a `cue.Value` into that render's build, so a caller that holds one keeps the build alive until it releases it; retention is bounded by what the caller holds, the rule ADR-007 states for an acquired artifact's `Package`. ADR-005 and the `opm/kernel` package documentation SHALL state this rule and SHALL NOT state that a caller cannot obtain a built value to hold. ADR-005 SHALL record that Render output moves from `cue.Value` to bytes before GA at the latest. The bytes deadline is 0021:D8:R11; the retention rule is ADR-007 rule 1.

#### Scenario: The documentation states the holder-bounded rule

- **WHEN** a developer reads rule 2 of `adr/005-shares-nothing-renders.md` and the Goroutine safety section of the `opm/kernel` package documentation
- **THEN** both state that the kernel holds no built value between calls and that a held `*Compiled` keeps its render's build alive until the caller releases it
- **AND** neither states that a caller cannot obtain a built value to hold

#### Scenario: The move to bytes is recorded

- **WHEN** a developer reads the Status of `adr/005-shares-nothing-renders.md`
- **THEN** it states that Render output keeps `Compiled.Value` for now and moves to bytes before GA at the latest

## MODIFIED Requirements

### Requirement: Goroutine Safety Contract

A single `Kernel` SHALL be safe for concurrent use across its own method calls: no operation shares evaluation state with another, because each creates and releases its own `cue.Context`, and the schema cache is memoized under synchronization. A consumer that needs concurrent operations uses one `Kernel` per process; the package documentation SHALL state this and SHALL NOT recommend one Kernel per goroutine.

`Kernel.Render` SHALL share nothing between renders: each render is its own CUE build in a fresh `cue.Context` whose references the kernel releases when `Render` returns, and no built value is retained by the kernel. A `*kernel.Compiled` the caller holds keeps its render's build alive until the caller releases it. Concurrency is across operations, never within one; a consumer rendering from several goroutines calls `Render` on one Kernel, with no shared platform value and no mutex. The package documentation SHALL state this, SHALL NOT present any shared built value as a supported shape, and SHALL state that a render pool is sized by memory (about 61 MB plus 7.75 MB per component per concurrent render, 0019 experiment 08) rather than by core count. The retracted shared-materialized-platform model and its mutex stopgap SHALL NOT appear as supported shapes.

#### Scenario: Documentation states the contract

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** it states that a single `Kernel` is safe for concurrent use across its method calls, shows one Kernel shared by concurrent goroutines, and states that every operation shares nothing

#### Scenario: Documentation retracts the shared-platform model

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** no shared materialized platform, held built value or mutex-serialised render appears as a supported shape; the retraction is recorded in ADR-002's supersession header and the shares-nothing rule in ADR-005 and ADR-007

#### Scenario: Repeated renders retain nothing

- **WHEN** `Render` is invoked repeatedly on one Kernel
- **THEN** each invocation builds in a fresh context, the results are byte-identical for identical inputs, and the kernel holds no built value between calls

#### Scenario: Concurrent acquisitions on one Kernel

- **WHEN** several goroutines acquire artifacts and synthesize instances on one Kernel at the same time under the race detector
- **THEN** every call succeeds with the same result as the sequential run and the race detector reports nothing

#### Scenario: Race detector runs on the render packages

- **WHEN** the repository test task runs
- **THEN** `opm/kernel` and `opm/internal/renderstage` are additionally run under `go test -race`
