# ADR-007: Every verb shares nothing

## Status

Accepted (2026-09-08). Amends ADR-005: the shares-nothing rule it states for `Kernel.Render` now holds for every kernel verb, and its "one Kernel per goroutine" sentence is retired in favour of one Kernel per process. Implemented by `kernel-owns-no-build-context` (slice 6a of the simplification plan reviewed on 2026-09-05).

## Context

ADR-005 gave `Render` its own `cue.Context` per call and left the Kernel's long-lived context to acquisition, synthesis, validation and the schema cache. Three facts followed from that split.

The Kernel's context only grew. cue v0.17.1's runtime records every built package in its import index (`internal/core/runtime/imports.go`, filled by `AddInst`) and nothing removes an entry, so every verb that built into the Kernel's context added to a heap that shrank only when the Kernel was dropped. Measured on 2026-09-08 against the warm workspace cache, twenty acquisitions of `testdata/parity/opm_platform` (which imports the published `catalogs/opm` build) with each result dropped took the heap from 38.6 MB to 743.5 MB, about 37.1 MB per call; twenty acquire-and-synthesize cycles of the served render fixture module took it from 6.8 MB to 102.8 MB, about 5.1 MB per cycle (about 14 MB per cycle for the larger module measured on 2026-09-05). Dropping the Kernel returned the heap to under 2 MB. The operator keeps one Kernel for the process, so every platform generation and every render reconcile grew it. ADR-005's retention figures measured the render context only.

The shared context serialised every verb. A CUE runtime is not safe for concurrent builds, so the operator wrapped every acquire and synth in a mutex (`Store.kernelMu`) and the library documented one Kernel per goroutine, while `Render` needed neither.

The context leaked into the API. A value built in one context could not be assumed to unify with a value built in another, so anything that would meet a kernel-built value had to be compiled in the Kernel's context: `kernel.Source` carried a `cue.Value` that had to have been compiled there with `cue.Filename(Origin)`, `Kernel.CueContext()` existed to make that possible, and `schema.Cache.Get` took the caller's context. Both consumers reached for the accessor (the cli's `vet` and publish gate, the operator's startup smoke check and module renderer).

The cross-artifact verbs already read only `Metadata` and `Source` from their inputs, never `Package`: `internal/synth` stages the instance inside the module's source tree, and `Render` imports both inputs by source. `opm/internal/loader.LoadDir`, `FetchModule` and `opm/internal/synth.Instance` already took a `*cue.Context` parameter. Nothing about what a verb computes depended on which context it built in.

## Decision

Four rules, extending ADR-005's two from `Render` to the whole surface:

1. **Every verb creates and releases its own `cue.Context`.** Each acquire verb, `SynthesizeInstance`, `ValidateConfigDetailed` and `Render` calls `cuecontext.New()` at entry, builds in it, and returns. The `Kernel` struct holds no context and `Kernel.CueContext()` is gone. An artifact's `Package` keeps the context of the call that built it alive for as long as the caller holds the artifact, and nothing else does; memory held by a long-lived Kernel is bounded by the artifacts its caller holds, not by the number of operations it has run.
1. **A `Source` carries bytes and is compiled where it meets a schema.** `kernel.Source` is `{Origin string; Data []byte}`. `LoadSourceFromFile` reads the file and `LoadSourceFromBytes` wraps the bytes; both parse for syntax and evaluate nothing. The verb that uses a source compiles it with `cue.Filename(Origin)` in the context of the schema it is checked against: a file-backed source (an absolute origin naming an existing file) through cue/load at the file's directory, with the carried bytes overlaid, so imports resolve as they do for the file on disk; any other source from its bytes. The top-level `values:` unwrap applies after either. No position is lost, because the filename travels with the bytes.
1. **The schema cache owns a private context.** `schema.Cache.Get()` takes no argument: the cache creates its context on first use inside its `sync.Once`, hands it to `Loader.Load` (whose signature is unchanged) and never exposes it. A caller that must compile against the schema uses the returned value's `Context()`. The memo is the one long-lived evaluation state a Kernel legitimately owns.
1. **The concurrency contract inverts.** A single Kernel is safe for concurrent use across its method calls, because no operation shares evaluation state with another and the cache is memoized under synchronization. The documentation recommends one Kernel per process; the operator's mutex around acquire and synth is deleted. Artifacts cross Kernels: a module acquired with one Kernel synthesizes with a second and renders with a third, pinned by a test rather than assumed.

Rejected alternatives:

- **Rotate the shared context after N builds.** A bounded leak with a tuning knob, still not concurrency-safe, and a Kernel whose behaviour depends on how many calls preceded this one.
- **Construct a Kernel per operation.** Moves the cost to every consumer and forfeits the schema memo, which is the one thing worth keeping.
- **Keep `Source.Value` and require callers to compile in a context the kernel hands out.** Keeps the accessor this decision removes, and makes every consumer responsible for a runtime invariant the kernel can keep for free.
- **A `Source` that lazily compiles itself and caches per context.** State on a value type, keyed by context pointer; the two consumers already hold what a byte-carrying `Source` is (a file path, a Kubernetes object's raw bytes).
- **Compile sources in a one-off context and unify across contexts.** cue v0.17 documents cross-context unification as allowed, but compiling in the schema's own context costs nothing, keeps one runtime per operation, and does not rest on a guarantee the evaluator has revised before.

## Consequences

**Positive:** Retention is bounded by what a consumer holds: after this change the same twenty platform acquisitions move the heap by 0.1 MB and the twenty acquire-and-synthesize cycles by 0.0 MB, against 705 MB and 96 MB before. Concurrency is excluded from needing a mutex by construction, for every verb and not only for `Render`; `task test` runs `opm/kernel` under `-race` with eight goroutines acquiring and synthesizing on one Kernel. The accessor, the "compiled in the Kernel's context" contract text, and the operator's mutex and gate are deleted; a frontend needs no `cue.Context` of its own to hand values to the kernel.

**Negative:** Breaking on the alpha line: a public method and a struct field are removed, a public field changes type, a public method loses its parameter. Both consumers migrate in the same wave as slices 2, 4 and 5. A consumer that used to acquire once and unify the result with values it compiled itself now hands the kernel bytes instead; a consumer that caches a `*platform.Platform` pays its evaluation once and holds that context for as long as it holds the platform, which is the intended trade.

**Trade-off:** `ValidateConfigDetailed` compiles its sources in the schema value's own context (the module's or instance's, as acquired), so validating against one artifact from several goroutines at once shares that artifact's context; a consumer that needs that gives each goroutine its own acquired artifact. The kernel's own verbs never do this: each compiles the sources it receives in the context it created for the call.

**Relation to ADR-005 and ADR-006:** ADR-005 stated the rule for the render build and measured its retention; this ADR applies the same rule uniformly and measures the retention ADR-005 left out. ADR-006 (one CUE build per artifact, inputs entering by import) is what makes the rule affordable everywhere: a verb that imports its inputs by source has no reason to share a context with the verb that produced them.
