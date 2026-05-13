# Decisions

Numbered design decisions, each with rationale and alternatives considered. Decisions are referenced from individual slice `design.md` files by ID (`D1` through `D10`).

## D1 — Kernel keeps validation; helpers do source-positioned validation on top

**Decision.** The kernel always validates the unified values value against `Module.Package`'s `#config` schema (Tier 2). A helper package (`opm/helper/values`, slice 05) optionally validates each individual source layer with source-positioned diagnostics (Tier 1) before unification. Tier 1 is the user-facing diagnostic surface; Tier 2 is a correctness safety net.

**Rationale.** Validation is part of correctness — bad input must be rejected before the kernel proceeds. Skipping validation must not be silently easy. Validation is also tied to the schema version the kernel dispatches on; putting it in a helper would force the helper to do version dispatch too. Layering, by contrast, is policy — different frontends layer values differently, and baking one merge order into the kernel forces every consumer to fight it.

**Alternatives considered.**

- *Validation entirely in the helper.* Rejected: kernel could not guarantee correctness, and the helper would inherit version-dispatch responsibility.
- *No Tier-1 validation; kernel produces all diagnostics.* Rejected: kernel cannot attribute errors to specific source layers (CLI flag vs. ConfigMap vs. CR overlay) without per-layer context, which is a frontend concern.

## D2 — Kernel struct with phase-explicit methods, not free functions

**Decision.** Public surface is a `Kernel` struct constructed via `kernel.New(...)`. Methods: `Validate`, `Match`, `Plan`, `Compile`, plus utility methods (`DetectAPIVersion`, `Finalize`, `CueContext`). Free functions in current packages collapse into methods or move to internal helpers.

**Rationale.** Cross-cutting dependencies (logger, tracer, clock, schema-binding registry lookup) accrete over time. With free functions, every new dependency adds a parameter to every entry-point signature, breaking every consumer. With a struct, dependencies are fields set once at construction. A struct also gives downstream consumers a single mental anchor — "the Kernel" — instead of a loose collection of packages.

**Alternatives considered.**

- *Go interface type (`type Kernel interface{...}`).* Rejected: there is exactly one implementation; a Go `interface` is over-engineering until a second concrete kernel exists.
- *Keep free functions; pass dependencies via `context.Context`.* Rejected: context is for cancellation and request-scoped data, not for kernel-lifetime configuration.

## D3 — Hybrid types: typed Metadata + raw `cue.Value`; CUE is source of truth

**Decision.** Each artifact type carries a decoded `Metadata` struct alongside a raw `Package cue.Value`. When the two disagree, `Package` wins. Typed metadata is an ergonomic projection (used for log fields, UI display, routing); CUE is authoritative.

**Rationale.** The kernel needs cheap access to fields like `metadata.name`, `metadata.namespace`, `metadata.labels` for almost every internal operation; re-traversing CUE every time is wasteful. But OPM's design property is that all schema, data, and information flow are clearly written in CUE — the typed projection must not become a parallel source of truth.

**Alternatives considered.**

- *cue.Value-only types.* Rejected: forces every internal access to go through `LookupPath` + decode, hurting both readability and performance.
- *Fully typed structs (decode everything).* Rejected: ties the Go API to schema layout; adding fields in CUE requires Go changes.

## D4 — Kernel detects `apiVersion` from input; closed binding registry

**Decision.** The kernel inspects each input artifact's `apiVersion` field, looks up the corresponding binding from a closed registry (populated by `init()` in each `opm/api/<version>` package), and dispatches all path lookups and decoders through that binding. Frontends do not declare or pass version information.

**Rationale.** Centralizes version policy. The kernel is the only thing that knows the full set of supported versions. Frontends pass artifacts; the kernel routes. Adding a new version is a new sibling package under `opm/api/`; no kernel edits.

**Alternatives considered.**

- *Caller declares version explicitly.* Rejected: every frontend would have to detect and pass the version, distributing the policy.
- *Open registry — external consumers can register custom bindings.* Rejected: defeats the point of a single reference runtime; consumers would diverge.

## D5 — Two-tier validation pattern (formalizes D1)

**Decision.** Tier 1 (helper, source-positioned) and Tier 2 (kernel, unified-value correctness) are separate, well-named layers. Tier 2 always runs; Tier 1 is opt-in but recommended.

**Rationale.** Captures D1 as a named pattern so slice 05 has a clear contract to implement and frontends have a clear path: "always call helper Tier-1; kernel handles the rest."

## D6 — `#ModuleDebug` is retired; `Module.debugValues` is the only debug surface

**Decision.** Top-level `#ModuleDebug` is removed from the v1alpha2 schema (already in flight via catalog enhancements). The kernel never accepts a debug artifact as input. `Module.debugValues` is a field within the Module's CUE package; the frontend reads it and decides whether to layer it into the values stack via `helper/values`.

**Rationale.** Debug overlays are policy. Operator: never in production. CLI: always when `--debug` is set. XR fn: depends on the composition. Centralizing in the kernel forces one policy; making it a Module field + helper layer lets each frontend express its own policy. Also aligns with enhancement 005's eight-slot Module shape.

**Alternatives considered.**

- *Keep `#ModuleDebug` as a kernel input with an enable flag.* Rejected: the kernel becomes aware of "debug mode," which is a frontend concern.

## D7 — Method granularity: Option C (Validate, Match, Plan, Compile + utilities)

**Decision.** Expose four phase methods (`Validate`, `Match`, `Plan`, `Compile`) plus utilities (`DetectAPIVersion`, `Finalize`, `CueContext`). Each phase method takes a small input struct and returns a phase-specific result.

**Rationale.** All three frontends need different depths of the pipeline. Operator wants `Validate` for admission and `Match` for status reporting before committing to render. CLI maps phases to subcommands. XR fn calls `Compile` only. A single `Compile` method with a `StopAfter` option (alternative B) muddles return types; separate methods are clearer.

**Alternatives considered.**

- *Minimal: `Compile` only.* Rejected: forces operator and CLI to either run the full pipeline or implement their own phase splits.
- *Phase methods with `StopAfter` flag.* Rejected: return type becomes a sum of all phase outputs; harder to reason about.

## D8 — Kernel owns `cue.Context`; never appears in public method signatures

**Decision.** The `Kernel` struct constructs and holds a `cue.Context` for its lifetime. Public methods take `(ctx context.Context, input X)` — never `*cue.Context`. Helpers either are methods on `*Kernel` or take `*Kernel` as a parameter. A `k.CueContext()` accessor exists for advanced cases.

**Rationale.** Hides CUE plumbing from the public API. A future kernel that swaps the evaluator (CUE-as-WASM, a different IR) does not break the surface. Lifecycle is clear — context lives exactly as long as the kernel. Helpers naturally share the kernel's context, so values built by helpers are always compatible with kernel operations.

**Trade-off.** `Kernel` is not goroutine-safe across compile calls. Operator workers construct one Kernel per goroutine. Construction is cheap; this is fine.

**Alternatives considered.**

- *Caller passes `*cue.Context` per call.* Rejected: leaks plumbing into every frontend.
- *Kernel constructs a fresh `cue.Context` per `Compile` call.* Rejected: incompatible with helper-built values, since CUE values are context-bound.

## D9 — Layout A: `opm/kernel/` subpackage; helpers nested under `opm/helper/`

**Decision.** The repo will be renamed from `library` to `kernel`. Within the repo, the `Kernel` struct lives at `opm/kernel/`. Optional helpers live under `opm/helper/{loader,values,platform,embed}/`. Other internal packages (`opm/core`, `opm/errors`, `opm/apiversion`, `opm/api/<v>`, `opm/module`, `opm/render`, `opm/validate`) keep their flat layout.

**Rationale.** Symmetric with the existing flat layout (no disruption from a layout-flattening or package-promoting refactor on top of everything else this enhancement does). The `opm/helper/` subdirectory makes the "optional, opinionated" boundary visible in the directory tree. Once the repo is renamed to `kernel`, the import path `github.com/open-platform-model/kernel/opm/kernel` is verbose but accurate; aliasing on import (`import kernel "github.com/open-platform-model/kernel/opm/kernel"`) is straightforward.

**Alternatives considered.**

- *Layout B — `Kernel` struct at module root, no `opm/kernel/` subpackage.* Rejected for now: would require flattening `opm/` simultaneously with the kernel redesign. Defer; could be a follow-up enhancement once the design has settled.

## D10 — Uniform `(APIVersion, Metadata, Package)` shape across all OPM artifacts

**Decision.** Every artifact type accepted by the kernel — `Module`, `ModuleRelease`, `Platform`, and any future artifact — has the same Go shape:

```go
type X struct {
    ApiVersion apiversion.Version
    Metadata   *XMetadata
    Package    cue.Value
}
```

**Rationale.** One contract for every artifact. Multi-version dispatch becomes a single rule (peek `Package` for `apiVersion`, look up binding, navigate via binding paths). New artifact types (e.g. a future `#Bundle` or `#Workflow` artifact) follow the same shape — no new contract to learn. Decoupled from CUE schema layout: the kernel never says "the spec lives at field X" in Go; the binding's paths say it.

**Trade-off.** `Values` is *not* an artifact and stays as a bare `cue.Value` — values are user data, not OPM artifacts. They have no `apiVersion` or `metadata`. Don't force them into this shape.

**Alternatives considered.**

- *Per-type custom shapes (current state).* Rejected: every new artifact type adds a new contract. Multi-version dispatch is per-type.
- *cue.Value-only types (drop `Metadata`).* Considered and folded into D3 — kept the typed projection for ergonomics.
