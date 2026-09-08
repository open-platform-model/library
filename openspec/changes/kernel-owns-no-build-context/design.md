# Design: kernel-owns-no-build-context

## Context

See `proposal.md` § Why. Evaluation state on `main` today:

```
  Kernel (process lifetime in the operator)
  +------------------------------------------------------------+
  | registry string                                            |
  | schemaCache ---- built into cueCtx once, memoized          |
  | cueCtx <-- AcquireModuleFromRegistry / AcquireModuleFromDir|
  |        <-- AcquirePlatformFromDir / AcquireInstanceFromDir |
  |        <-- SynthesizeInstance                              |
  |        <-- LoadSourceFromFile / LoadSourceFromBytes        |
  |        <-- CueContext()  (cli publish gate, operator smoke)|
  +------------------------------------------------------------+
  Render: cuecontext.New() per call, released on return (ADR-005)
```

After this change:

```
  Kernel
  +------------------------------------------------+
  | registry string                                |
  | schemaCache: private context, built once       |
  +------------------------------------------------+
  every verb:  cuecontext.New() -> build -> return artifact -> context released
  artifact.Package pins ITS OWN runtime for the artifact's lifetime
  Source{Origin, Data}: compiled where it meets a schema, in that schema's context
```

Constraints: no verb changes what it computes; the parity oracle and every render test pass unchanged; `opm/internal/loader.LoadDir`, `FetchModule` and `opm/internal/synth.Instance` already take a `*cue.Context` parameter, so they do not change; the cross-artifact verbs (`SynthesizeInstance`, `Render`) already read only `Metadata` and `Source` from their inputs (`internal/synth/instance.go:171-203`, `render.go:174-181`), never `Package`.

## Goals / Non-Goals

**Goals:**

- Memory held by a long-lived Kernel is bounded by the artifacts its caller holds, not by the number of operations it has run.
- One Kernel is safe for concurrent use across all its methods, with no mutex anywhere.
- No kernel input or output is bound to a context the caller cannot see.

**Non-Goals:**

- Reducing what acquisition evaluates, or caching anything across calls.
- Changing `Loader.Load(ctx)`; the cache supplies the context.
- Changing the render staging or serving path (`overlay-served-in-memory`).

## Decisions

### Every verb creates and releases its own context

**Context**: the growth is the runtime's import index, which has no eviction API; the retention is per context.
**Explored**: (A) rotate the shared context after N builds (a bounded leak with a tuning knob, still not concurrency-safe); (B) a context per verb call, as `Render` already does; (C) ask consumers to construct a Kernel per operation (moves the cost to every consumer and forfeits the schema memo).
**Decision**: B. Each verb calls `cuecontext.New()` at entry and passes it down; the Kernel struct loses `cueCtx`.
**Rationale**: identical to ADR-005's reasoning for `Render`, now applied uniformly; creation is microseconds, the expensive memo (the core schema) stays cached, and a returned value pins exactly the runtime that built it, which is the lifetime the caller controls.

### `Source` carries bytes and is compiled where it meets a schema

**Context**: `Source.Value` is compiled in the Kernel's context and later unified with a module's `#config` (`validate.go:61,69`). With per-verb contexts that becomes a cross-context unification, which CUE does not guard.
**Explored**: keeping `Source.Value` and requiring callers to compile it in a context the kernel hands out (the accessor this change removes); a `Source` that lazily compiles itself and caches per context (state on a value type, and a map keyed by context pointer).
**Decision**: `Source{Origin string; Data []byte}`. `LoadSourceFromFile` reads the file, `LoadSourceFromBytes` wraps the bytes, both parse with `cue/parser` so a syntax error surfaces at load, positioned at `Origin`. One unexported `compileSource(ctx, s)` does what the two helpers did at load time: a file-backed source (an absolute `Origin` that exists) goes through `load.Instances` with the file overlaid at its own directory so imports resolve exactly as today, any other source through `ctx.CompileBytes(data, cue.Filename(origin))`; the top-level `values:` unwrap applies after either. `ValidateConfigDetailed` compiles in `schema.Context()`; `mergeSources(ctx, sources)` takes the verb's context; `attributeValuesError` compiles the package's own `values` in the same context it validates in.
**Rationale**: a `Source` becomes context-free data, which is what the two consumers already hold (a file path, a Kubernetes object's raw bytes); no positions are lost because the filename travels with the bytes; no workspace values file imports anything (checked 2026-09-08), and the cue/load path keeps that possibility open anyway.

### `schema.Cache` owns a private context

**Context**: `Cache.Get(ctx)` builds the schema into whatever context the caller passes, which was the Kernel's. The cli publish gate reads the schema value and compiles catalog packages in the same context to unify them.
**Decision**: `Get()` creates its context on first use inside the `sync.Once`, passes it to `Loader.Load(ctx)` and keeps it for the cache's lifetime; the `Loader` interface is unchanged. A caller that needs to compile against the schema uses `val.Context()`. `opm/internal/schematest.NewCache` follows.
**Rationale**: the memo is the one long-lived evaluation state the Kernel legitimately owns; giving it its own context makes it independent of every verb and of the Kernel's lifetime.

### Verbs read data from their inputs, pinned by a cross-Kernel test

**Context**: with per-verb contexts, an instance synthesized from a module acquired in another context must not touch the module's `Package`.
**Decision**: no code change is needed (`internal/synth` reads `Metadata` and `Source`; `Render` reads `Source` and `Metadata.Name`), but a test acquires a module with one Kernel, synthesizes with a second, acquires a platform with a third and renders with a fourth, asserting the same objects as the single-Kernel run. A second test runs acquire and synth from several goroutines on one Kernel under `-race`.
**Rationale**: the property is what makes the contract true, so it is pinned rather than assumed.

### The concurrency contract inverts and is recorded in ADR-007

**Decision**: `Kernel` is documented as safe for concurrent use; the "one Kernel per goroutine" example becomes "one Kernel per process"; `adr/007-shares-nothing-verbs.md` states the rule for every verb, cites the measured retention, and amends ADR-005's scope statement (which speaks only of renders) without superseding it.

## Risks / Trade-offs

- [Each live artifact pins a runtime] → memory follows what a consumer holds; a consumer that caches a `*platform.Platform` pays its evaluation once, which is the intended trade. The spike measures twenty dropped acquisitions on one Kernel to confirm the heap returns to baseline.
- [A consumer unifies two artifact values from different calls] → none does (both consumers read through `LookupPath` and `Decode`); the doc states the rule and the cross-Kernel test pins the supported shape.
- [A file-backed source that imports a module package] → still compiled through cue/load at the file's directory, so nothing changes; no such file exists in the workspace today.
- [Test migration: 41 `CueContext()` uses in 11 files] → mechanical: `cuecontext.New()` for hand-built values, `schemaVal.Context()` where a test unifies with the schema.

## Migration Plan

1. Library PR: spike first (numbers into the PR description), then the tasks in order; `task check` and the race pass green; consumer builds against a temporary `replace` prove the listed edits.
2. Release-please cuts the alpha that closes the wave (slices 2, 4, 5 and this).
3. cli PR: three sites. Operator PR: the smoke check, delete `kernelMu`/`AcquireKernel` and its three call sites.
4. Rollback: consumers pin the previous alpha; nothing persisted changes shape.

## Open Questions

None.
