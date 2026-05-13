# Design Decisions — Compiler / Runtime Kernel Split

## Summary

Decision log for all architectural and design choices made during this enhancement. Each decision is numbered sequentially and recorded as it is made. Decisions are append-only — do not remove or renumber existing entries. If a decision is reversed, add a new decision that supersedes it.

---

### D1: Two parts in one Kernel substrate, with strict package separation

**Decision:** The Kernel becomes a substrate type holding shared dependencies (`cue.Context`, `opm/api` binding registry, logger, tracer) and exposes accessors only. Two sibling packages — `opm/compile` and `opm/runtime` — host the operational methods. Convenience constructors on `Kernel` may exist (e.g., `Kernel.Compiler()`, `Kernel.Runtime(opts)`) but only re-export the sibling packages. Frontends with strict import discipline construct from the sibling packages directly.

**Alternatives considered:**

- One monolithic Kernel with both surfaces as methods. Rejected — determinism wall is implicit, XR fn cannot avoid Runtime symbols, SemVer churn shared between unrelated concerns.
- Kernel and Runtime as fully unrelated types in unrelated packages, no convenience constructors. Rejected for ergonomics — three constructors at every embedding, no shared substrate sugar.

**Rationale:** The two parts share genuine substrate (cue.Context, binding registry) and would duplicate it if fully separated. A shared substrate with separate operational types preserves both ergonomics (one Kernel construction) and isolation (XR fn imports only `opm/compile`). The convenience methods are sugar; the package boundary is the load-bearing structure.

**Source:** Design discussion 2026-05-08, user selected this option after side-by-side comparison of all three candidate layouts.

---

### D2: Compiler is pure; Runtime is effectful; the wall lives at the package boundary

**Decision:** `opm/compile` MUST NOT import `opm/runtime`. CI lint enforces this with `go list -deps`. The Compiler has no method that may invoke I/O or read system state. The Runtime has no method that participates in deterministic compile output.

**Alternatives considered:**

- Document the wall in a constitution clause but allow imports. Rejected — past evidence shows docstring-enforced walls drift across contributors.
- Define separate interfaces (PureKernel vs EffectfulKernel) on the same struct. Rejected — type assertions wash out the wall at the call site, and the build cannot detect violations.

**Rationale:** Constitution Principle I is normative; the package boundary is the only mechanism that mechanically enforces it. A reviewer who proposes importing `opm/runtime` from `opm/compile` is rejected by the build, not by a code review heuristic.

**Source:** User decision 2026-05-08. Constitution Principle I (Kernel Neutrality & Determinism).

---

### D3: Runtime opts in via construction; XR fn never imports it

**Decision:** Runtime is constructed via `runtime.New(k, opts...)`. Construction returns an error if no Executors are registered for op types the Runtime is asked to handle (validated lazily on first `RunAction`). Frontends that cannot host execution (Crossplane composition function) do not import `opm/runtime` at all.

**Alternatives considered:**

- Always-available Runtime that no-ops when executors are missing. Rejected — silent no-op hides bugs; explicit construction makes the embedding decision visible at startup.
- Single executor registration on `Kernel` itself. Rejected — `Kernel` is the substrate; pushing executor config onto it leaks Runtime concerns into the shared surface.

**Rationale:** Constructor-time validation surfaces capability mismatches at startup, when they are easiest to diagnose. The XR fn's `opm/runtime` exclusion is enforceable by the module graph, not by runtime checks.

**Source:** Design discussion 2026-05-08.

---

### D4: Drop `Kernel.Clock` (YAGNI)

**Decision:** Slice 01 removes the `Kernel.Clock` field, the `Clock` interface, the `systemClock` struct, and the `WithClock` option from `opm/kernel`. When the Runtime ships in slice 05, it adds its own `runtime.Clock` interface and `runtime.WithClock` option. The Compiler never consults a clock.

**Alternatives considered:**

- Keep `Kernel.Clock` as a placeholder for the future Runtime. Rejected — Constitution VII (YAGNI) is unambiguous about hypothetical consumers, and the placeholder has been unused since slice 01 of enhancement 001 introduced it.
- Move `Clock` to the Compiler in case future renders need deterministic time. Rejected — there is no concrete render that needs time, and a clock on a deterministic-by-contract type is an active hazard.

**Rationale:** The Clock field's only justification was anticipation of Runtime work. Now that the Runtime is being designed concretely, Clock relocates to where it earns its keep. Keeping it on the Kernel during the gap between deletion and Runtime introduction would violate YAGNI.

**Source:** User decision 2026-05-08 — "If Clock is not used, maybe we should remove it. Generally i want to follow YAGNI."

---

### D5: Compiler emits `ActionInvocation`s as part of its deterministic output

**Decision:** `CompileResult` gains an `ActionInvocations []*core.ActionInvocation` field. The Compiler walks Action declarations during compile, resolves the `$after` DAG, and produces a fully concrete plan. The Compiler does not execute. The frontend (or an opt-in Runtime) consumes the plan.

**Alternatives considered:**

- Compiler returns Action declarations unresolved; Runtime resolves the DAG. Rejected — DAG resolution is a deterministic function of the declaration; placing it in the Runtime forces every embedding to repeat the same logic and would split the validation surface across packages.
- Compiler returns a callback (`func(Executor) error`). Rejected — callbacks couple compile-time and runtime types; serialization across processes (operator → Job, XR fn → response) becomes impossible.

**Rationale:** `ActionInvocation` is data; data crosses process boundaries cleanly. Resolving the DAG at compile time means the Runtime is a pure step executor, not a planner. This separation simplifies retries (re-execute a known plan), persistence (serialize the plan to CRD status), and dry-run (compile and inspect without running).

**Source:** Design discussion 2026-05-08.

---

### D6: Frontend orchestrates the Compile↔Run loop; neither side calls the other

**Decision:** The library exposes the two operations as separate public methods. The frontend writes the loop: compile → apply rendered → run actions → snapshot → recompile. No call from `Compiler` ever reaches `Runtime`, and vice versa.

**Alternatives considered:**

- A `Drive(ctx, rel, provider) error` orchestrator in `opm/kernel` that runs the full loop. Rejected — the loop's termination, retry policy, and parallelism choices differ per frontend. CLI runs to completion synchronously; operator runs partial steps per reconcile cycle; XR fn does not run actions at all.
- Runtime polls a Compiler-provided plan iterator. Rejected — couples the two via a shared iterator type and reintroduces an implicit dependency.

**Rationale:** The two halves serve different lifecycles. Forcing one orchestrator on all three frontends makes the library opinionated about reconciliation timing in ways that conflict with Kubernetes operators and stateless function handlers.

**Source:** Design discussion 2026-05-08.

---

### D7: Library imports Op/Action schema from catalog 010; defines no schema itself

**Decision:** This enhancement adds no CUE definitions. The library decodes `#Op`, `#Action`, and `#Step` from `apis/core/v1alpha2/` after catalog 010 publishes them there. `opm/api/v1alpha2` adds `Paths.Steps`, `Paths.After`, `Paths.OpType`, and `DecodeAction` per the existing Binding pattern.

**Alternatives considered:**

- Library publishes its own `#Op`/`#Action` definitions to avoid blocking on catalog. Rejected — duplicates the schema and forces catalog and library to stay in lockstep manually.
- Library accepts arbitrary CUE values as Action declarations and skips schema decoding. Rejected — typed decoding is the contract that lets the Compiler's `ActionInvocation` carry a deterministic shape.

**Rationale:** Catalog 010 is the single source of truth for Op/Action schemas, parallel to how catalog 014 is the source of truth for `#Platform`. The library is the runtime side; the catalog is the schema side. Publish order: catalog publishes, library decodes.

**Source:** Design discussion 2026-05-08. Mirrors the catalog/library division established by enhancement 001.

---

### D8: `RuntimeSnapshot` is the only Runtime → Compiler channel, and it is an immutable `cue.Value`

**Decision:** The Runtime exposes `Snapshot() RuntimeSnapshot` where `RuntimeSnapshot` is a `cue.Value`. The Compiler accepts it via `compile.WithRuntimeSnapshot(snap)`. The Compiler may read fields from the snapshot but cannot mutate it. No other channel exists between Runtime and Compiler.

**Alternatives considered:**

- Pass a `*Runtime` reference to the Compiler so it can lazily query state. Rejected — couples the Compiler to the Runtime's lifecycle and breaks the import-discipline wall.
- Use a typed Go struct for the snapshot. Rejected — Action `#out` schemas are open and per-Op; a typed struct would force the library to know every Op's output shape at compile time.

**Rationale:** A `cue.Value` is the lowest-common-denominator data carrier in the kernel. It naturally serializes to YAML/JSON for inspection, persists to CRD status fields, and survives process boundaries. Immutability is enforced by CUE's evaluation semantics — the Compiler cannot rebind an existing path in a `cue.Value`.

**Source:** Design discussion 2026-05-08.

---

### D9: Per-frontend Executor maps; library ships only the interface and reference implementations

**Decision:** `opm/runtime` defines the `Executor` interface and the DAG walker. Reference implementations live in `opm/runtime/local`, `opm/runtime/k8s`, and `opm/runtime/grpc`. Frontends register the executors they need at construction time. The library does not auto-register any executor; an empty Runtime errors on first `RunAction`.

**Alternatives considered:**

- Auto-register a default `local.Exec` executor when the Runtime is constructed. Rejected — opaque defaults make embedding bugs invisible and constrain the operator embedding (which must use `k8s.Job`, not `local.Exec`).
- Ship only the interface; require every frontend to implement its own executors. Rejected — leaves the CLI without a reference implementation and forces every consumer to reimplement the well-known op set from catalog 010.

**Rationale:** Reference implementations exist to bootstrap consumers; explicit registration keeps the embedding decision visible. The shipped reference set covers the catalog 010 well-known ops; future ops add new executors without breaking existing registrations.

**Source:** Design discussion 2026-05-08. Follows the existing pattern in `opm/api/v1alpha2` where the binding is opt-in via `init()`.

---

### D10: Defer cross-step `#out` wiring; track as OQ1

**Decision:** This enhancement does not design cross-step output wiring. Each step's inputs are concrete CUE supplied by the producer of the `ActionInvocation`. When a real consumer demands cross-step references (likely the operator's restore orchestration), a follow-up enhancement designs the wiring mechanism.

**Alternatives considered:**

- Ship a `$inputs: { from: "stepName.#out.value" }` reference syntax in slice 04. Rejected — catalog 010 D4 deferred this on the schema side; library cannot lead the catalog.
- Use `#ctx`-based reference resolution from catalog enhancement 016. Rejected — that enhancement's design is itself in flux; coupling Runtime feedback to it now creates a chain dependency.

**Rationale:** Catalog 010 explicitly deferred this. The library's runtime can only honor what the schema declares. When catalog adds the wiring syntax, the library Runtime grows the corresponding execution semantics in a follow-up.

**Source:** Catalog 010 D4. This enhancement carries the deferral forward.

---

### D11: Lifecycle phases are a typed enum in `opm/compile`; canonical order shipped as a constant; downgrade gated on catalog

**Decision:** Slice 09 introduces `compile.LifecyclePhase` as a typed string enum with constants for `PhasePreInstall`, `PhaseInstall`, `PhasePostInstall`, `PhasePreUpgrade`, `PhaseUpgrade`, `PhasePostUpgrade`, `PhasePreUninstall`, `PhaseUninstall`, `PhasePostUninstall`. A `compile.CanonicalOrder() []LifecyclePhase` helper returns the install→upgrade→uninstall trajectory ordering. Downgrade phases (`PhasePreDowngrade`, `PhaseDowngrade`, `PhasePostDowngrade`) are added only if and when catalog `#Lifecycle` includes them.

**Alternatives considered:**

- Plain string keys without an enum. Rejected — frontends would scatter string literals across reconcile state machines and CLI command wiring; typos surface as silent map-misses at runtime instead of build errors.
- Phase enum in `opm/api/v1alpha2`. Rejected — phase names are kernel-runtime concepts that frontends use directly; placing them under a versioned binding adds an indirection without value. The binding still carries `Paths.LifecyclePhase` for CUE-level path lookup, but the Go enum is shared kernel surface.
- Hardcode all phases including downgrade up-front. Rejected — catalog 010's downstream consumer enhancements have not committed to downgrade as in-scope; library defines what catalog publishes, not what catalog might publish.

**Rationale:** Type safety at the frontend integration boundary catches misnamed phases at build time. Canonical order belongs in the library because it is the same for every embedding; per-frontend re-derivation would diverge. Downgrade exclusion respects catalog leadership of schema design.

**Source:** User decision 2026-05-08 — phase set explicitly enumerated; downgrade flagged as conditional.

---

### D12: `CompileResult` exposes Workflows and Lifecycle as separately-keyed maps; Runtime API treats invocations uniformly

**Decision:** `CompileResult` carries:

```go
Workflows map[string]*core.ActionInvocation              // keyed by Workflow FQN
Lifecycle map[LifecyclePhase]*core.ActionInvocation      // keyed by phase enum
```

The Runtime's `RunAction(ctx, *core.ActionInvocation)` method takes a single invocation regardless of source. The frontend selects the right invocation from the right map and feeds it to the Runtime.

**Alternatives considered:**

- Single flat `[]*core.ActionInvocation` slice with a `Source` discriminator field on each invocation. Rejected — frontends always filter by source first (CLI on install walks Lifecycle phases; on `workflow run` walks Workflows); type-system separation makes the filter trivial and missing-key errors compile-time.
- Two separate `RunWorkflow` / `RunLifecycle` methods on Runtime. Rejected — the execution mechanics are identical (DAG walk + Op dispatch). Splitting the method duplicates the implementation surface for no gain.
- Keep Workflows as a slice (no key) since user invokes by name from CLI. Rejected — name lookup is the dominant access pattern; a map directly encodes it.

**Rationale:** Separation of concerns at the result type boundary. The Runtime stays consumer-agnostic (one method, one type). The Compiler stays declarative (emit all known invocations). The frontend stays in control of invocation policy.

**Source:** Design discussion 2026-05-08.

---

### D13: Workflow is on-demand-named; Lifecycle is phase-driven; both share Op/Action mechanics

**Decision:** The library treats `#Workflow` and `#Lifecycle` as parallel consumers of the Op/Action substrate. Workflows are invoked by name (frontend trigger: user request, CRD field, scheduled job). Lifecycle phases are driven by the deployment state machine (frontend trigger: install/upgrade/uninstall transitions, reconcile state). The kernel never decides when either runs; it only emits what is declared. A single `#Action` declaration may be referenced as a step inside both a Workflow and a Lifecycle phase — the resulting `ActionInvocation`s are independent, but the underlying Action FQN is shared.

**Alternatives considered:**

- Unify Workflow and Lifecycle into a single "Pipeline" construct distinguished by tag. Rejected — invocation semantics genuinely differ (named vs phase-keyed) and forcing one shape makes both awkward.
- Lifecycle as a special case of Workflow (named workflows that the kernel auto-runs on phase transitions). Rejected — couples kernel emission to deployment state, breaking Compiler determinism.

**Rationale:** Each construct's invocation pattern matches a real frontend need. The Compiler's job is to surface both clearly; the frontend's job is to drive each appropriately. Sharing the underlying Op/Action mechanics means the Runtime's executor map and DAG walker serve both consumers without specialization.

**Source:** User decision 2026-05-08 — explicit framing of Workflow as `cue cmd`-parallel and Lifecycle as Helm-hook-parallel-with-typed-substrate.
