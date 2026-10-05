## MODIFIED Requirements

### Requirement: Goroutine Safety Contract

A single `Kernel` SHALL be safe for concurrent use across its own method calls: no operation shares evaluation state with another, because each creates its own `cue.Context` and drops its references to it on return, and the schema cache is memoized under synchronization. A value the operation returns (an artifact's `Package`, a `*kernel.Compiled`) keeps that context alive for as long as the caller holds it, and no longer: the context's lifetime is bounded by its holder (ADR-007). A consumer that needs concurrent operations uses one `Kernel` per process; the package documentation SHALL state this and SHALL NOT recommend one Kernel per goroutine.

`Kernel.Render` SHALL share nothing between renders: each render is its own CUE build in a fresh `cue.Context` whose references the kernel drops when `Render` returns, and no built value is retained by the kernel. A `*kernel.Compiled` the caller holds keeps its render's build alive until the caller releases it. Concurrency is across operations, never within one; a consumer rendering from several goroutines calls `Render` on one Kernel, with no shared built platform value and no mutex. The package documentation SHALL state this, SHALL NOT present any shared built value as a supported shape, and SHALL state that a render pool is sized by memory (about 61 MB plus 7.75 MB per component per concurrent render, 0019 experiment 08) rather than by core count. The retracted shared-materialized-platform model and its mutex stopgap SHALL NOT appear as supported shapes.

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

## ADDED Requirements

### Requirement: Each runtime contract has one home

Each runtime contract of the library SHALL be stated in the doc comment of the package, type or function that owns it; the rationale behind a contract SHALL live in an ADR under `adr/`, and its testable obligations in a requirement under `openspec/specs/`. `README.md`, `AGENTS.md` and the pages under `docs/` SHALL link to a contract's home instead of restating it, and MAY keep a short orientation sentence that names the home. A doc comment that the docs bundle publishes (any exported package under `opm/`) SHALL NOT carry an `ADR-NNN` pointer; a package that wants one keeps it in a non-doc comment. No committed file SHALL cite a session-local decision, one recorded only in an agent session's own log, as a source: it states the rule and cites a source a reader can open (an owner decision by its walkthrough id, an ADR, an enhancement decision such as `0012:D3`, a pull request or an archived change). Source: owner decision c4 (beta.1 walkthrough).

#### Scenario: The render contract is stated once

- **WHEN** a developer looks for the render gate's cause order outside `opm/`
- **THEN** `README.md`, `AGENTS.md` and `docs/getting-started.md` link to the `opm/kernel` package doc or the `RenderError` doc for it, and none lists the order itself

#### Scenario: The env-override rule is stated in its homes

- **WHEN** a developer searches the non-test Go code under `opm/` for `os.Setenv`
- **THEN** the hits are the `opm/internal/cueenv` package doc and the `OCILoader` type doc, and the acquire verbs and the loader options link to the package doc instead

#### Scenario: Published doc comments carry no ADR pointer

- **WHEN** a developer searches the doc comments of the exported packages under `opm/` for `ADR-`
- **THEN** none is found; ADR pointers appear only in non-doc comments

#### Scenario: No session-local citation

- **WHEN** a developer searches the repository outside `openspec/changes/` for a `Source:` line, a comment or a prose citation that names a session-local decision
- **THEN** none is found; each rule cites a source a reader can open

#### Scenario: Which artifacts carry a Source

- **WHEN** a developer reads the `module.Source` doc
- **THEN** it names Module (registry and directory acquire, overlay mode), Instance (synthesis in overlay mode; directory acquire on disk, or in overlay mode with values sources), Platform (directory acquire, on disk) and Catalog (registry and directory acquire, overlay mode), and makes no claim of a single cue/load overlay site
