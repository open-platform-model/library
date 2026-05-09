# Design — Compiler / Runtime Kernel Split

## Design Goals

- The Compiler stays a pure function from `*ModuleRelease + *Provider/*Platform + values → CompileResult`. Same determinism contract as today's `render.ProcessModuleRelease`, with `CompileResult` extended to include `[]*ActionInvocation`.
- A new Runtime executes Op/Action declarations against a pluggable `Executor` map. Effects (process exec, HTTP, time) live exclusively here.
- The two share the existing `Kernel` substrate (`cue.Context`, `pkg/api` binding registry, logger, tracer) without sharing methods or state.
- Frontends opt in: a Compiler is always available; a Runtime is constructed only when the embedding can host execution. The Crossplane composition function can build the Kernel without importing `pkg/runtime`.
- The determinism wall is enforced at the package boundary: `pkg/compile` does not import `pkg/runtime`. CI lint catches violations at build time.
- The Runtime's progress is observable to the Compiler only through an immutable `RuntimeSnapshot` (a `cue.Value`). The Compiler never calls into the Runtime; the Runtime never calls into the Compiler. The frontend is the conductor of the loop.
- The `Kernel.Clock` field is removed before the Runtime ships. Time concerns relocate to the Runtime, where retries and timeouts justify them.
- Workflow and Lifecycle constructs (catalog enhancements following 010) consume the same Op/Action substrate but emit invocations through distinct surfaces: Workflow as on-demand named graphs (parallel to CUE's `cmd`), Lifecycle as a phase-keyed map of graphs (parallel to Helm hooks but with strict types). The Compiler emits both shapes from the same walker; the Runtime executes both with the same DAG mechanics; the frontend selects which to drive.

## Non-Goals

- Implementing Workflow or Lifecycle constructs. Catalog enhancements for those are prerequisites for slices 08 and 09; this enhancement does not design them.
- Implementing Op CUE schemas. Those live in `catalog/enhancements/010-op-action-primitives/`. The library imports them via the existing `apis/core/v1alpha2/` embed pattern once catalog publishes.
- Defining cross-step `#out` wiring. Catalog 010 D4 deferred this; this enhancement follows. The first slice that ships Action execution treats each step's input as concrete CUE supplied by the producer of the Action invocation.
- Persistent state stores. Slice 05 ships an in-memory `StateStore`. Each frontend wires its own persistence (CLI = file, operator = CRD status, XR fn = none). Persistence backend selection is per-frontend.
- Concurrent Action execution. Catalog 010 allows it via `$after`-derived parallelism, but slice 05 ships sequential execution. Parallelism is a per-Runtime configuration that lands when a real consumer demands it.
- A `KernelPool` for shared `cue.Context` reuse across goroutines. The existing one-Kernel-per-goroutine guidance from 001 carries forward unchanged.

## High-Level Approach

The Kernel becomes a substrate that provides shared state but exposes no operational methods. Two sibling packages, `pkg/compile` and `pkg/runtime`, host the two halves. Both consume `*Kernel` as a dependency. Neither imports the other.

```
                          ┌──────────────────────────────┐
                          │        pkg/kernel            │
                          │  Kernel { cueCtx, logger,    │
                          │           tracer }           │
                          │  + accessors                 │
                          └──────────────┬───────────────┘
                                         │
                       ┌─────────────────┴─────────────────┐
                       │                                   │
         ┌─────────────▼───────────┐         ┌─────────────▼───────────┐
         │    pkg/compile          │         │     pkg/runtime         │
         │  Compiler { *Kernel }   │         │  Runtime { *Kernel,     │
         │  pure                   │         │            Executor[],  │
         │  CompileResult {        │         │            StateStore } │
         │    Compiled,            │         │  effectful              │
         │    ActionInvocations,   │         │  RunAction              │
         │    Warnings }           │         │  Snapshot()             │
         └─────────────────────────┘         └─────────────────────────┘
                       ▲                                   │
                       │                                   │
                       │              ┌────────────────────┘
                       │              │
                       │              ▼ (frontend wires the loop)
                       │   ┌─────────────────────┐
                       │   │   ActionInvocation  │
                       │   │   (data only)       │
                       │   └─────────────────────┘
                       │              │
                       │              ▼ (after Runtime executes)
                       │   ┌─────────────────────┐
                       └───┤   RuntimeSnapshot   │
                           │   (cue.Value)       │
                           └─────────────────────┘
```

Convenience constructors `Kernel.Compiler()` and `Kernel.Runtime(opts ...RuntimeOption)` may live in `pkg/kernel` for ergonomics, but they re-export the sibling packages. Frontends that want strict import discipline (XR fn) construct `compile.NewCompiler(k)` directly without naming `pkg/runtime`.

## Type Layout

### `pkg/kernel`

Substrate. Drops `Clock` in slice 01.

```go
package kernel

type Kernel struct {
    cueCtx *cue.Context
    logger *slog.Logger
    tracer trace.Tracer
}

func New(opts ...Option) *Kernel
func WithLogger(*slog.Logger) Option
func WithTracer(trace.Tracer) Option
// (no WithClock — see Decision D4)

// Accessors only. No transform methods.
func (k *Kernel) CueContext() *cue.Context
func (k *Kernel) Logger() *slog.Logger
func (k *Kernel) Tracer() trace.Tracer
```

### `pkg/compile` (current `pkg/render` after 001 slice 06)

Pure transform. Same shape as today's pipeline; one new field on the result, one new option.

```go
package compile

type Compiler struct { /* holds *Kernel */ }

func NewCompiler(k *kernel.Kernel) *Compiler

type CompileResult struct {
    Compiled  []*core.Compiled                              // unchanged: K8s/etc resources
    Workflows map[string]*core.ActionInvocation             // NEW (slice 08): keyed by Workflow FQN
    Lifecycle map[LifecyclePhase]*core.ActionInvocation     // NEW (slice 09): keyed by phase
    Warnings  []string
}

// LifecyclePhase is a typed string enum for #Lifecycle phase keys.
// Added in slice 09. Phase names match catalog #Lifecycle field names exactly.
type LifecyclePhase string

const (
    PhasePreInstall    LifecyclePhase = "preInstall"
    PhaseInstall       LifecyclePhase = "install"
    PhasePostInstall   LifecyclePhase = "postInstall"
    PhasePreUpgrade    LifecyclePhase = "preUpgrade"
    PhaseUpgrade       LifecyclePhase = "upgrade"
    PhasePostUpgrade   LifecyclePhase = "postUpgrade"
    PhasePreUninstall  LifecyclePhase = "preUninstall"
    PhaseUninstall     LifecyclePhase = "uninstall"
    PhasePostUninstall LifecyclePhase = "postUninstall"
    // Optional downgrade triplet — gated on catalog inclusion (OQ6)
)

// CanonicalOrder returns the install→upgrade→uninstall trajectory ordering.
// Frontends use this when driving phases sequentially (e.g., `opm install`
// walks PhasePreInstall, PhaseInstall, PhasePostInstall in order).
// The kernel does not enforce sequence — emission is per-Compile.
func CanonicalOrder() []LifecyclePhase

type CompileOption func(*compileOptions)
func WithRuntimeSnapshot(s RuntimeSnapshot) CompileOption  // NEW: feedback loop

func (c *Compiler) Compile(
    ctx context.Context,
    rel *module.Release,
    p *provider.Provider,
    runtimeName string,
    opts ...CompileOption,
) (*CompileResult, error)
```

`ActionInvocation` is added to `pkg/core` (slice 03):

```go
package core

type ActionInvocation struct {
    APIVersion apiversion.Version
    FQN        string                  // Action's metadata.fqn
    Steps      []StepNode
    DAG        ActionDAG               // resolved $after edges
}

type StepNode struct {
    Name     string
    Type     string                    // "op" | "action"
    OpType   string                    // matches @op("...") if Type=="op"
    Inputs   cue.Value                 // concrete fill of the step
    OutShape cue.Value                 // #out schema for runtime to populate
    After    []string                  // raw $after declarations
    Children []StepNode                // nested Action expansion if Type=="action"
}

type ActionDAG struct {
    Edges map[string][]string          // step name -> dependents
    Roots []string                     // step names with no $after
}
```

### `pkg/runtime` (new)

Effectful orchestration. Consumes `*Kernel` for shared state; consumes its own `Executor` map for op dispatch.

```go
package runtime

type Runtime struct { /* holds *Kernel, Executor map, StateStore, Clock */ }

type Option func(*Runtime)
func WithExecutor(opType string, e Executor) Option
func WithStateStore(s StateStore) Option
func WithClock(c Clock) Option

func New(k *kernel.Kernel, opts ...Option) (*Runtime, error)

type Executor interface {
    OpType() string                                              // matches @op("...")
    Run(ctx context.Context, in cue.Value) (cue.Value, error)    // returns #out
}

type StateStore interface {
    Save(invocationID string, st *ActionState) error
    Load(invocationID string) (*ActionState, error)
}

type Clock interface { Now() time.Time }

func (r *Runtime) RunAction(
    ctx context.Context,
    inv *core.ActionInvocation,
) (*ActionResult, error)

// Coroutine form for operator / multi-call drivers.
func (r *Runtime) Step(
    ctx context.Context,
    st *ActionState,
) (*StepResult, *ActionState, error)

// Snapshot returns an immutable view of completed step outputs.
// Suitable to feed back into Compiler.WithRuntimeSnapshot.
func (r *Runtime) Snapshot() RuntimeSnapshot

type RuntimeSnapshot cue.Value

type ActionState struct { /* per-step status, captured #out, retry counters */ }
type StepResult  struct { /* single-step outcome */ }
type ActionResult struct { /* terminal Action outcome */ }
```

### `pkg/runtime/local`, `pkg/runtime/k8s`, `pkg/runtime/grpc`

Reference Executor implementations. Slice 06 ships `local`. The other two land when their consumers (operator, XR fn) actively need them — YAGNI.

## Determinism Wall

Enforced by package import discipline, not by docstring.

| Boundary                          | Where it lives                       |
| --------------------------------- | ------------------------------------ |
| CUE evaluation                    | `pkg/kernel.Kernel.CueContext` — both sides use it |
| Reading `@op("...")` attribute    | `pkg/runtime` only                   |
| `Executor.Run` side effects       | `pkg/runtime` Executors only         |
| Time / retry scheduling           | `pkg/runtime.Clock` only             |
| Step state mutation               | `pkg/runtime.StateStore` only        |
| `RuntimeSnapshot` as Compile input| `pkg/compile.WithRuntimeSnapshot`    |
| Compiler may call Runtime         | NO                                   |
| Runtime may call Compiler         | NO                                   |

`pkg/compile` MUST NOT import `pkg/runtime`. CI lint enforces this with a single `go list -deps` rule. Any future refactor that proposes importing `pkg/runtime` from `pkg/compile` is rejected at the import layer, not at code review.

## Consumer Surfaces: Workflow and Lifecycle

Two catalog-side consumer constructs invoke the Op/Action substrate through different patterns:

| Consumer    | Invocation pattern                              | CompileResult field                                | Frontend trigger                                                  |
| ----------- | ----------------------------------------------- | -------------------------------------------------- | ----------------------------------------------------------------- |
| `#Workflow` | On-demand, named (parallel to CUE `cmd`)        | `Workflows map[string]*ActionInvocation` (FQN key) | User request (`opm workflow run <name>`), CRD field, scheduler    |
| `#Lifecycle`| Phase-keyed (parallel to Helm hooks, typed)     | `Lifecycle map[LifecyclePhase]*ActionInvocation`   | Deployment state machine (CLI install/upgrade, operator reconcile)|

Both surfaces share:

- The same `#Action` substrate from catalog 010 — a single `#Action` definition (e.g., `#DBMigration`) can be a step inside both a Workflow and a Lifecycle phase. The `ActionInvocation`s emitted for each are independent (different inputs, different invocation IDs).
- The same Runtime API — `RunAction(ctx, *ActionInvocation)` does not care which surface produced the invocation.
- The same `$after` DAG mechanics, output capture, and step state model.

What differs is invocation policy: who decides when to run, and how the result feeds back. The Compiler emits *all* declared invocations from both surfaces deterministically; the frontend selects which to drive.

## Compiler ↔ Runtime Feedback Loop

Two artifacts cross the wall:

1. `*core.ActionInvocation` (one at a time, sourced from `Workflows` or `Lifecycle`) — Compiler → Runtime. A deterministic plan: which Action to run, with what concrete inputs, in what DAG order. Identical Compile inputs always yield identical invocations.
2. `RuntimeSnapshot` (cue.Value) — Runtime → Compiler. An immutable view of completed step `#out` values. The next Compile reads it via `WithRuntimeSnapshot`. Compile remains pure given (rel, provider, values, snapshot).

The frontend's loop terminates when a Compile yields no new invocations to drive — the deployment is in steady state. This pattern matches Crossplane's composition function model and Kubernetes' reconcile loop semantics.

```go
// CLI install command — drives the install phase sequence
k := kernel.New(kernel.WithLogger(logger))
cmp := compile.NewCompiler(k)
rt, _ := runtime.New(k,
    runtime.WithExecutor("exec", local.Exec()),
    runtime.WithExecutor("http.get", local.HttpGet()),
)

snap := rt.Snapshot()
res, err := cmp.Compile(ctx, rel, provider, "opm-cli",
    compile.WithRuntimeSnapshot(snap))
if err != nil { return err }

if inv, ok := res.Lifecycle[compile.PhasePreInstall]; ok {
    if _, err := rt.RunAction(ctx, inv); err != nil { return err }
}
apply(res.Compiled)
if inv, ok := res.Lifecycle[compile.PhaseInstall]; ok {
    if _, err := rt.RunAction(ctx, inv); err != nil { return err }
}
if inv, ok := res.Lifecycle[compile.PhasePostInstall]; ok {
    if _, err := rt.RunAction(ctx, inv); err != nil { return err }
}

// Operator reconcile — drives one phase per reconcile cycle
res, _ := cmp.Compile(ctx, rel, provider, "opm-controller",
    compile.WithRuntimeSnapshot(rt.Snapshot()))
phase := r.currentPhase(release)  // controller's state machine picks this
if inv, ok := res.Lifecycle[phase]; ok {
    rt.RunAction(ctx, inv)
}

// CLI on-demand workflow — `opm workflow run db-migration`
res, _ := cmp.Compile(ctx, rel, provider, "opm-cli")
inv, ok := res.Workflows["opmodel.dev/opm/v1alpha2/actions/database/db-migration@v1"]
if !ok { return errors.New("workflow not declared by module") }
rt.RunAction(ctx, inv)
```

## Embedding Stories

| Frontend       | Builds Compiler? | Builds Runtime? | Workflow trigger                       | Lifecycle trigger                                              | Executors                                                       |
| -------------- | ---------------- | --------------- | -------------------------------------- | -------------------------------------------------------------- | --------------------------------------------------------------- |
| `opm` CLI      | Yes              | Yes             | `opm workflow run <fqn>`               | `opm install/upgrade/uninstall` walks `CanonicalOrder()` slice  | `local.Exec`, `local.HttpGet/Post`, `local.WaitFor`             |
| `opm-operator` | Yes              | Yes             | CRD-triggered (optional surface)       | Reconcile state machine picks phase per cycle                  | `k8s.Job` for exec, `local.HttpGet/Post`, `k8s.Wait` for conditions |
| Crossplane fn  | Yes              | NO              | Surfaced as deferred-action signal     | Surfaced as deferred-action signal                             | n/a — Compiler-only deployment, no Action execution             |

XR fn imports only `pkg/kernel` and `pkg/compile`. `pkg/runtime` is not in its module graph. Any Workflow or Lifecycle invocation declared by a module is visible to the XR fn through `CompileResult.Workflows` / `CompileResult.Lifecycle`, but the XR fn does not execute them; it surfaces them in its composition response so a downstream operator or workflow consumer drives execution.

## Slice Dependency Graph

```
catalog/010 publish ─────┐
                         ▼
                add-action-decoder-paths (02)
                         │
                         ▼
                add-action-invocation-core-type (03)
                  │                │
                  ▼                ▼
   emit-action-invocations    add-runtime-package-skeleton (05)
   -from-compile (04)               │
        │                           ▼
        │                     add-local-op-executors (06)
        │
        │                     add-runtime-snapshot-feedback (07)  [needs 05]
        │
        ├───▶ add-workflow-decoder  (08) ──── needs catalog #Workflow
        │
        └───▶ add-lifecycle-decoder (09) ──── needs catalog #Lifecycle

drop-unused-kernel-clock (01) — independent prerequisite cleanup
```

Slices 01–04 ship purely declarative changes (no I/O introduced). Slice 05 introduces the Runtime scaffolding without external Executors. Slice 06 is the first slice that performs side effects. Slice 07 enables cross-cycle data flow between Runtime and Compiler. Slices 08 and 09 add the typed consumer surfaces (Workflow and Lifecycle); each gates only on slice 04 plus its corresponding catalog construct, so they may land in either order or in parallel once their catalog prerequisites publish.

## Before / After

### Today

```go
result, err := compile.CompileModuleRelease(ctx, rel, plat, "opm-cli")
// result.Compiled = K8s resources
// no Action execution surface
```

### After this enhancement

```go
k := kernel.New()
cmp := compile.NewCompiler(k)
rt, _ := runtime.New(k, runtime.WithExecutor("exec", local.Exec()))

snap := rt.Snapshot()
res, err := cmp.Compile(ctx, rel, provider, "opm-cli",
    compile.WithRuntimeSnapshot(snap))
// res.Compiled  = K8s resources (unchanged)
// res.Workflows = {"opmodel.dev/.../db-migration@v1": <inv>, ...}
// res.Lifecycle = {PhasePreInstall: <inv>, PhasePostInstall: <inv>, ...}

// CLI install: walk lifecycle phases in canonical order
for _, phase := range compile.CanonicalOrder()[:3] {  // pre/install/post
    if inv, ok := res.Lifecycle[phase]; ok { rt.RunAction(ctx, inv) }
    if phase == compile.PhaseInstall { apply(res.Compiled) }
}

// CLI on-demand: user invokes a named workflow
inv := res.Workflows["opmodel.dev/.../db-migration@v1"]
rt.RunAction(ctx, inv)

// Next Compile reads rt.Snapshot() — cross-cycle data flow without
// breaking determinism.
```

## File Layout

```
pkg/
  kernel/         Kernel struct, accessors. No Clock after slice 01.
  compile/        (renamed from pkg/render in 001 slice 06)
                  Compiler, CompileResult, WithRuntimeSnapshot.
  core/           ActionInvocation, StepNode, ActionDAG (added in slice 03)
  runtime/        (NEW)
    runtime.go    Runtime, RunAction, Step, Snapshot
    executor.go   Executor interface
    state.go      StateStore interface, in-memory impl
    dag.go        $after DAG walker
    clock.go      Clock interface (relocated from pkg/kernel)
  runtime/local/  Reference executors — exec, http, wait
  runtime/k8s/    (deferred) Reference executors for in-cluster exec
  runtime/grpc/   (deferred) Reference executors for delegated exec
```
