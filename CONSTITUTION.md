# Open Platform Model Library Constitution

## Purpose

This document is the reader-friendly reference for the principles that shape the design, implementation, validation, and change management of the OPM library. The library is governed by the normative constitutional source in `openspec/config.yaml`.

The library is the **kernel** of OPM. It provides the generic, reusable building blocks for loading, processing, validating, and rendering OPM `#Module`s, and will host the full `#Workflow` and `#Lifecycle` system. Every implementation of OPM — the CLI, the controller, and any future runtime — depends on this library and inherits its behavior. The principles below are written with that responsibility in mind.

## Design Principles

| # | Principle | Summary |
| ---- | --------- | ------- |
| **I** | [Kernel Neutrality & Determinism](#i-kernel-neutrality--determinism) | The library makes no runtime assumptions and behaves deterministically given its inputs |
| **II** | [Type Safety First](#ii-type-safety-first) | Inputs are validated at load time; strong Go and CUE types over open-ended data |
| **III** | [Separation of Concerns](#iii-separation-of-concerns) | Loader, module, provider, render, validate, errors, and core stay clearly split |
| **IV** | [Composability via Stable Contracts](#iv-composability-via-stable-contracts) | `pkg/` exposes a stable public API consumed by every OPM implementation |
| **V** | [CUE-Native Module Resolution](#v-cue-native-module-resolution) | The library owns CUE module and OCI plumbing on behalf of all downstream implementations |
| **VI** | [Semantic Versioning & Public API Discipline](#vi-semantic-versioning--public-api-discipline) | SemVer is contractual; downstream consumers depend on it |
| **VII** | [Simplicity & YAGNI](#vii-simplicity--yagni) | New abstractions must be justified by a real downstream need |
| **VIII** | [Small Batch Sizes](#viii-small-batch-sizes-iterative--incremental-delivery) | Changes must stay tiny, incremental, and independently verifiable |

---

### I. Kernel Neutrality & Determinism

The library is consumed by many downstream implementations (CLI, controller, future runtimes). It MUST NOT assume a particular runtime, process model, or operator persona.

- No global mutable state; no package-level singletons that hide behavior
- No `os.Exit`, no direct logging output, no shell invocation
- No hidden environment lookups; configuration arrives explicitly through function arguments
- Loading, parsing, and rendering MUST be deterministic given identical inputs
- Any I/O (filesystem, registry, network) MUST live at the edges of the library and accept caller-supplied configuration

If logging is needed, the library MUST accept a logger from the caller (via parameter or `context.Context`). The library MUST NOT decide on its own when to write to stdout, stderr, or a file.

This neutrality is what allows the same kernel to power a CLI, a Kubernetes controller, and any future implementation without surprising any of them.

---

### II. Type Safety First

All inputs into the library MUST be validated at load time. Invalid modules, releases, providers, or configuration MUST be rejected before processing or rendering begins.

- Validate in order: input types -> CUE schema -> module/release semantics -> render
- Use CUE for schema-level validation
- Prefer concrete Go types over `interface{}` / `any`
- `any` is acceptable only at genuine open-ended boundaries (for example, raw CUE values or rendered manifest payloads)
- Fail early so downstream implementations receive actionable errors before side effects occur

```text
inputs -> schema -> semantics -> render
```

---

### III. Separation of Concerns

The library MUST preserve clear package boundaries. Each package owns a single responsibility:

- `pkg/core/` — shared domain primitives (resource identity, rendered output)
- `pkg/errors/` — structured errors and sentinels (alias as `oerrors` in consumers)
- `pkg/loader/` — load CUE artifacts (modules, providers, releases) into `cue.Value`
- `pkg/module/` — module and release model, parsing, and helpers
- `pkg/provider/` — provider model
- `pkg/render/` — render pipeline (match, process, execute, finalize)
- `pkg/validate/` — configuration validation helpers

Domain logic belongs in focused packages, not aggregated into one monolithic API. Clear boundaries keep the library easier to test, evolve, and reuse across implementations.

```text
loader -> module/provider -> validate -> render -> core
```

---

### IV. Composability via Stable Contracts

The library is a kernel. Its public API is the contract that every downstream implementation depends on.

- Accept interfaces, return concrete structs (Go convention)
- Public surface lives in `pkg/`; non-public helpers live in `internal/`
- `pkg/` packages MUST NOT import command, controller, or runtime-specific concerns
- Output formatting and presentation MUST stay outside the library
- Functions accept `context.Context` for any I/O, longer workflows, or cancellation

Composition should come from explicit package contracts, not hidden coupling. Downstream implementations should be able to mix library packages freely without dragging unrelated dependencies along.

---

### V. CUE-Native Module Resolution

OPM modules are CUE modules. The library MUST use CUE's native module system for module acquisition, dependency resolution, and caching against OCI registries.

- Library callers (CLI, controller) rely on the library to load and resolve modules
- Custom OCI fetch logic, custom dependency resolvers, and custom caches do not belong in downstream implementations
- The library is the single place where CUE plumbing lives
- Future module sources (local paths, alternative registries, archives) MUST be expressed through CUE-native mechanisms where possible

Centralizing CUE-native resolution in the library guarantees that every OPM implementation behaves identically when loading the same module.

---

### VI. Semantic Versioning & Public API Discipline

The library MUST follow SemVer 2.0.0. Because multiple downstream implementations consume the library, public API stability is contractual.

- MAJOR: any breaking change to `pkg/` types, signatures, or behavior
- MINOR: additive changes that preserve existing behavior
- PATCH: bug fixes, performance improvements, internal refactors
- Commits SHOULD follow Conventional Commits v1: `type(scope): description`

Recommended commit types:

- `feat`
- `fix`
- `refactor`
- `docs`
- `test`
- `chore`

Recommended scopes match the package layout: `core`, `loader`, `module`, `provider`, `render`, `validate`, `errors`.

When a breaking change to `pkg/` is unavoidable, the change MUST call it out in the proposal and consider downstream migration cost explicitly.

---

### VII. Simplicity & YAGNI

Start with the simplest implementation that satisfies a real downstream need. New complexity MUST be justified by a concrete consumer requirement.

- Prefer direct functions over speculative abstractions
- Prefer explicit flow over hidden magic
- Defer extension points until at least one consumer actually requires them
- Defer plug-in style hooks, callbacks, and option-bag expansion until proven necessary

Every public symbol increases the SemVer surface and the burden of long-term support. The library should grow when downstream needs prove it, not in anticipation.

---

### VIII. Small Batch Sizes (Iterative & Incremental Delivery)

All work MUST be delivered in tiny, independently verifiable steps.

- Large requests SHOULD be split into smaller sequential changes
- Tiny changes produce focused, atomic commits
- A single change SHOULD ideally address one specific concern
- Validation SHOULD remain practical at every step

This principle applies to both planning and implementation. Large bundled changes hide risk, slow review, and weaken validation.

#### Execution Gate

Before beginning any implementation, the scope of the request MUST be evaluated against the small-batch principle.

If the request is too large, the required response is:

> "🛑 **Scope Warning**: This request is too large for a single safe iteration. I suggest we split it into the following smaller steps: [list 2-3 logical, tiny steps]. Should we start with step 1?"

---

## Technology Standards

- Language: Go (see `go.mod` for the pinned version)
- CUE Go SDK: `cuelang.org/go`
- Tests: `github.com/stretchr/testify`
- Build/test entrypoint: `Taskfile.yml`

The library has no `main` package and ships no binary. It is consumed as a Go module by other repositories in this workspace.

## Code Style Expectations

The library code SHOULD follow these defaults:

- Accept interfaces where useful, return concrete structs when practical
- Propagate `context.Context` through I/O, CUE evaluation, and longer workflows
- Wrap errors with context: `fmt.Errorf("loading module: %w", err)`
- Reuse `pkg/errors` types and sentinels where applicable
- Prefer concrete types over `map[string]any`
- No package-level mutable state; build fresh CUE contexts at the boundary

### Logging

- The library MUST NOT log directly to stdout or stderr
- If logging is needed, accept a logger from the caller (parameter or context)
- Log messages, when emitted by the caller's logger, SHOULD follow Go conventions: capitalized, no trailing period

### Imports

Standard Go grouping, with blank lines between groups:

1. standard library
2. external dependencies (including CUE)
3. local module imports (`github.com/open-platform-model/library/...`)

Let `gofmt` and `goimports` control formatting and grouping.

## Quality Gates

Before merge, the following checks SHOULD pass:

1. `task fmt`
2. `task vet`
3. `task lint`
4. `task test`

Equivalently, `task check` runs all four.

---

## OpenSpec Artifact Rules

These principles also shape how OpenSpec artifacts should be written for the library.

### Proposal

- Focus on WHY the change is needed and WHAT is in or out of scope
- Update the proposal when scope changes, intent clarifies, or the approach fundamentally shifts
- Identify affected `pkg/` packages and any downstream consumers (CLI, controller)
- State whether the change is MAJOR, MINOR, or PATCH under SemVer
- Any added complexity MUST include explicit justification
- Scope MUST remain small enough for a short implementation session

### Design

- Focus on HOW the change will be implemented
- Update the design when implementation reveals a better approach or constraints change
- Use RFC 2119 language: MUST, SHALL, SHOULD, MAY
- Include a `Research & Decisions` section whenever exploration was required
- Include Go pseudocode or CUE snippets where they clarify intent
- Explain the impact across loader / module / render / validate phases
- Call out any change to the public surface in `pkg/`

Recommended `Research & Decisions` shape:

```md
## Research & Decisions

### [Topic]
**Context**: [Why this decision was needed]
**Explored**: [What was investigated]
**Decision**: [Chosen option]
**Rationale**: [Why this option was selected]
```

### Specs

- Focus on WHAT behavior changes, not HOW it is implemented
- Update specs when requirements change or new observable behavior is introduced
- Use RFC 2119 language: MUST, SHALL, SHOULD, MAY
- Describe observable behavior such as returned errors, rendered output, and surfaced validation results
- Use `ADDED`, `MODIFIED`, and `REMOVED` sections for deltas
- Include scenarios such as valid load vs invalid load, render success vs render failure

### Tasks

- Focus on implementation steps
- Update tasks as work completes, blockers appear, or new work is discovered
- Break tasks into tiny chunks, ideally no more than 1-2 hours each
- If the list grows beyond roughly 10 items or spans multiple features, split it into another OpenSpec change
- Group tasks by package (`core`, `loader`, `module`, `provider`, `render`, `validate`, `errors`)
- Include validation gates as final tasks: `task fmt`, `task vet`, `task lint`, `task test`

---

## How Principles Work Together

These principles reinforce each other:

- Kernel neutrality keeps the library safe to embed in any runtime
- Type safety and CUE-native resolution make load behavior predictable across implementations
- Separation of concerns and composability keep the public API small and explainable
- SemVer and API discipline keep downstream upgrades safe
- Small batch sizes keep change quality high and validation practical

When principles appear to conflict, treat that as a design smell and document the trade-off explicitly.

## Further Reading

- `openspec/config.yaml` — normative constitutional source
- `Taskfile.yml` — build, lint, and test entrypoints
