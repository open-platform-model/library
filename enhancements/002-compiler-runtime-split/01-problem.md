# Problem Statement — Compiler / Runtime Kernel Split

## Current State

The OPM kernel today is a single deterministic transform pipeline:

```
loader.LoadReleaseFile        ->  cue.Value (release artifact)
module.ParseModuleRelease     ->  *module.Release          (validated, concrete)
compile.CompileModuleRelease  ->  *compile.CompileResult   (compiled + provenance)
```

`Kernel` (introduced by enhancement 001, slice 01) owns the shared substrate: `*cue.Context`, `*slog.Logger`, `trace.Tracer`, and a `Clock` interface placeholder. It exposes pure-transform methods only. Every method is a deterministic function of its inputs, and Constitution Principle I requires it stay that way:

> Loading, parsing, and rendering MUST be deterministic given identical inputs.
> Any I/O (filesystem, registry, network) MUST live at the edges of the library.

There is no surface for *executing* anything. The kernel describes resources to be created; some other layer is expected to apply them. That works for the resource side of the type system but breaks down for the operational side.

Catalog enhancement 010 (`#Op` & `#Action` Primitives) introduces atomic operations (`#Op` with `@op("...")` runtime dispatch) and composed flows (`#Action` with `$after` step ordering). These declare *what* should run; they require a runtime that interprets them and executes side effects. None of the three OPM frontends (CLI, operator, Crossplane composition function) has anywhere in the library to plug an Op executor.

## Gap 1: Op/Action Have No Executor

The catalog can publish `#DBMigration` as an Action. A module author can fill in concrete values. The kernel can validate the declaration against the Action's schema. But nothing in `pkg/` walks `#steps`, dispatches on `@op("...")`, captures `#out`, or honors `$after`. The schema is dead without a runtime.

The 010 design explicitly notes this:

> Runtime execution engine design (CLI or controller internals) — Non-Goal

— meaning the catalog-side enhancement deferred runtime design as out of scope. This enhancement picks it up.

Catalog-side enhancements following 010 introduce two distinct consumer constructs that share the Op/Action substrate but expose different invocation surfaces:

- **`#Workflow`** — on-demand named Action graphs, parallel to CUE's own `cmd` scripting concept. Invoked by the user (e.g., `opm workflow run db-migration`) or surfaced via CRD.
- **`#Lifecycle`** — phase-keyed Action map: `preInstall`, `install`, `postInstall`, `preUpgrade`, `upgrade`, `postUpgrade`, `preUninstall`, `uninstall`, `postUninstall`, and optionally a `downgrade` triplet. Driven by the deployment state machine (CLI's `opm install/upgrade/uninstall` walks the phase sequence; operator's reconcile loop picks the phase corresponding to the current desired-vs-observed state). The strictness of Op and Action types is the deliberate departure from Helm hooks, where unconstrained shell scripts make composition unsafe.

Each surface has a distinct invocation pattern but shares the same execution mechanics. The kernel must accommodate both without duplicating step execution logic.

## Gap 2: Single Kernel Forces a Determinism Compromise

Two paths exist if execution lives on the existing Kernel surface:

1. **Add `Kernel.RunAction(ctx, *Action)` directly.** Side effects now live on a type whose entire public contract is "deterministic given identical inputs." The wall is enforced by docstring; the next contributor adds I/O to a Compile path because nothing types-prevents it. Constitution I becomes aspirational, not enforced.
2. **Add `Kernel.Runtime() *Runtime` as a sub-object on the same struct.** Better — but still couples the embedding story. The Crossplane composition function can host pure CUE evaluation; it cannot host arbitrary process execution. Forcing it to import a `Runtime` type it cannot use bloats the binary and creates a configuration surface (executors, state store) the XR fn must explicitly null out.

Neither matches Principle I or the heterogeneous frontends.

## Gap 3: Frontend Capability Mismatch

The three embeddings have wildly different execution models:

| Frontend       | Capability                                                                  |
| -------------- | --------------------------------------------------------------------------- |
| `opm` CLI      | In-process, full local privileges, can shell out, hit network, run commands |
| `opm-operator` | ServiceAccount-scoped; submits Jobs/TaskRuns; long-running reconcile loop   |
| Crossplane fn  | Short-lived gRPC handler; no filesystem, no arbitrary code, no exec         |

If the kernel ships one `Runtime`, it ships one execution model. The smallest common denominator across the three is the empty set. A configurable runtime matrix inside the kernel duplicates per-frontend logic that already lives at the frontends.

## Gap 4: `Kernel.Clock` Is a Placeholder With No Consumer

Slice 01 of enhancement 001 added `Kernel.Clock` as a deterministic-time hook for "future workflow / lifecycle work." Nothing in the kernel reads it. Its only justification is anticipatory — and the anticipation points squarely at a future Runtime. Two outcomes are possible:

- **Delete it now (YAGNI).** Constitution VII is unambiguous about hypothetical consumers.
- **Reserve it on the future Runtime instead of the Kernel.** Determinism is a Compiler property; clocks are a Runtime concern (retry timing, timeouts, scheduled steps).

Either outcome reinforces the split. The Clock signal is itself evidence that the Kernel was already drifting toward a runtime concept it had no place to host.

## Concrete Example

A module declares a `db-migration` Action (concrete form from catalog 010 design):

```cue
myMigration: db.#DBMigration & {
    #steps: {
        migrate: {
            image:   "flyway/flyway:10"
            command: ["flyway", "-url=jdbc:postgresql://db:5432/app", "migrate"]
        }
        verify: url: "http://app:8080/health"
    }
}
```

Today:

- The CLI loads the module → `*module.Module` validates → `compile.CompileModuleRelease` produces `*core.Compiled` for any K8s resources the module declares. The Action declaration sits in the module's CUE but the kernel has no surface that returns it as an executable plan.
- The operator does the same on reconcile and reaches the same dead-end.
- The XR fn does the same and dead-ends identically.

Each frontend invents its own Action executor in its own codebase, with no shared interface, no shared state model, no shared output capture, no shared `$after` DAG walker. The duplication is exactly the failure mode the kernel is meant to prevent.

## Why Existing Workarounds Fail

**Run Actions as part of resource emission (e.g., emit a `Job` per step):** Loses the `$after` DAG semantics — the Action becomes a flat list of K8s Jobs that the cluster scheduler runs in arbitrary order. The runtime semantics that catalog 010 carefully designed (DAG, parallelism, output capture, retry) are erased. Lifecycle ordering across phases (install → upgrade → delete) cannot be expressed in K8s Job dependencies.

**Hand off to `cue cmd` per Action:** Ties OPM operation execution to CUE CLI availability. Operator and XR fn have no `cue` binary in scope. The hermetic boundary moves outside OPM's control, and the runtime dispatch contract (`@op("...")` → executor) is replaced by `cue cmd`'s own task system, which has no version dispatch and no shared state model.

**Per-frontend ad hoc executor (the de facto status quo):** Three executor implementations diverge on retry semantics, output capture format, error wrapping, and DAG walking. Module authors must test against each. The CLI's behavior becomes the de facto spec because no other reference exists. This is the Constitution III (Separation of Concerns) failure mode 001 was designed to fix on the resource side — and it would repeat on the operational side because no kernel-shared runtime exists.

**Wait for catalog 010 to land before designing the kernel side:** Catalog 010 has been Draft since 2026-04-11. Kernel-side design is a prerequisite for shipping any Op consumer. Decoupling means catalog and library can move in parallel: catalog publishes the schema; library publishes the runtime; each side ships when its own gates pass.
