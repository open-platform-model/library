# ADR-008: The kernel plans, the caller runs

## Status

Accepted (2026-09-14). Amends the Purpose statement in `CONSTITUTION.md` and `openspec/config.yaml`, which said the library will host the full `#Workflow` and `#Lifecycle` system. Constrains enhancement 0009 (workspace root, `enhancements/0009`), whose D1, D3 and D4 were recorded before ADR-005 and ADR-007 and describe a kernel shape that no longer exists. No implementing change yet: core carries no lifecycle vocabulary, so there is nothing to plan from until the schema grows one.

## Context

The Constitution has said since the library was founded that it will host the full `#Workflow` and `#Lifecycle` system. No code has ever pointed at that sentence, and two later decisions narrowed what it can mean.

ADR-005 and ADR-007 fixed the kernel's shape. Every verb creates a `cue.Context`, builds in it and drops it; the Kernel retains nothing but the schema memo; concurrency is across operations rather than inside one. A kernel that holds no evaluation state between calls cannot hold run state between calls either, unless run state is something other than evaluation state.

Enhancement 0009 designs an execution half against the pre-ADR-005 kernel. Its sketch places the plan types, the planner, the executor port and a runner in a kernel-core `opm/flow/`, with backends in the opt-in helper tier. D3 calls the library a pure planner and orchestrator that performs no side effects of its own, and D4 gives each frontend its own backend registry. The design text still names `opm/compile/` and `[]*core.Compiled`, both deleted, so the entry needs revision whatever this decision says.

The planner half is uncontroversial. A function from an instance and a phase to a plan value is the shape `Render` already has: inputs in, decoded data out, nothing retained.

The runner half is the open question. A loop that walks a dependency graph, threads results between steps and blocks on wait steps is a process model, and Principle I says the library must not assume one.

One asymmetry between the two first-party consumers decides it. A controller reconcile is level-triggered and must return promptly. It cannot block on a wait step while holding a work-queue slot, so its wait is a requeue, and its step state has to be reconstructed on each pass from the custom resource it owns. A cli invocation is one-shot and can block for as long as the operation takes. A blocking runner in the kernel therefore serves the cli and is unusable by the operator. That is the shape the 2026-09-11 kernel audit found below the render line, re-verified on 2026-09-14: ordering, digests, prune guards and object conversion each written twice, with the apply-time guard in one frontend and the delete-time guard in the other, and neither holding both.

Two limits already constrain any answer. Enhancement 0009:OQ6 records that a caller's context reaches phase boundaries and registry fetches but never a running CUE evaluation, so cancellation can only ever land between steps. And core has no lifecycle vocabulary today: no ordering, dependency, phase, hook or health field on any construct.

This ADR carries no measurements. Unlike ADR-005 and ADR-007 it settles a boundary before there is code to measure. What it rests on is the consumer asymmetry above and the duplication the audit found below the render line.

## Decision

Four rules.

1. **The kernel plans; the caller runs.** Planning is a pure function from artifacts to a plan value, the shape `Render` already has. The library ships no loop that drives a plan to completion.
1. **A plan advances one step per call, and the state belongs to the caller.** Advancing is a pure function of a plan and a state value that returns the next state and the action the caller is to perform. The kernel keeps nothing between calls. The state MUST be serialisable, so a controller can carry it in a custom resource across reconciles and a one-shot frontend can hold it in memory.
1. **The kernel names an action; it never performs one.** No I/O, no waiting, no clock, no process spawning, in line with Principle I. If the library ever ships executor backends they live under `opm/helper/`, where the opt-in fence and its depguard rule already are.
1. **Lifecycle facts are data off a build.** Ordering edges, hook declarations and wait conditions are emitted by the CUE build and decoded in Go, the way render verdicts already are. The kernel derives no ordering of its own.

Rejected alternatives:

- **A blocking runner in the kernel, 0009:D3 and D4 as written.** It is usable by one of the two first-party consumers. The operator would keep the plan, ignore the runner and write its own resumable walk, which reproduces below the plan exactly the duplication this ADR exists to prevent above it.
- **A plan the kernel emits and each frontend walks with its own loop.** Safest against ADR-005 and ADR-007, and it leaves the sequencing rules in two places. The audit already shows what happens to a rule with two homes: the cli lost weight-ordered apply when the render path changed under it, and nothing failed.
- **Run state held by the Kernel, keyed by instance.** Reintroduces exactly the retention ADR-007 removed, with a new eviction question and a new concurrency question, so that a caller need not pass a value it already has.
- **A goroutine-driven runner with callbacks.** Moves the process model into the library and hides it behind an interface. A controller cannot use callbacks that outlive its reconcile, and a caller cannot cancel between steps without a channel protocol the kernel would then own.

## Consequences

**Positive:** Sequencing semantics get one definition that both frontends execute, which is the property the render half already has and the apply half does not. Cancellation is trivial and needs no mechanism: a caller that stops calling has stopped, which is the only cancellation OQ6 says is available anyway. ADR-005 and ADR-007 are untouched, so nothing about context lifetime or concurrency has to be revisited. A plan and its transitions are testable without a cluster, a registry or a clock.

**Negative:** Each frontend writes the loop that calls the step function, so a small amount of code is duplicated by design; what is not duplicated is any decision the loop makes. The public surface grows by a plan type, a state type and a step function, and every one of them is SemVer surface under Principle VI. The step function is more API than emitting a plan alone would be.

**Trade-off:** Rule 2 requires the state to be serialisable, which bounds what a step may carry to the next one. A step cannot hand a live `cue.Value`, an open connection or a file handle to its successor; it hands data. This bears directly on 0009:OQ2, which asks whether steps pass typed data or only ordering: whatever that question answers, the answer has to survive a round trip through a custom resource.

**What this does not decide:** the executor artifact form (0009:OQ1), whether steps pass typed data or only ordering edges (OQ2), the run-state and idempotency model for on-demand workflows (OQ3), and where `#Lifecycle` and `#Workflow` attach on `#Module` (OQ4). All four remain open and none is blocked by this decision.

**Relation to ADR-005, ADR-006 and ADR-007:** those three settle how the kernel evaluates, what it retains and how it may be called concurrently. This one settles what the kernel does with time. The first three are why the answer is a step function rather than a loop: a kernel that drops its evaluation state at the end of every call has nowhere to keep a half-finished run, and the caller does.

**Relation to enhancement 0012:** ordering is the first lifecycle fact that already exists, and it is already duplicated. Delivering ordering as plan data under these rules is the cheapest test of them, and it has a live defect behind it rather than a schema that has yet to be written.
