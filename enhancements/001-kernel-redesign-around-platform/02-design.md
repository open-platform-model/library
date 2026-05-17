# Design

This document captures the cross-cutting kernel design and how the slices fit together. Each slice has its own `design.md` under `library/openspec/changes/archive/2026-05-08-<slice>/` (planned slices) or under the matching dated archive directory for follow-ups. This file is the umbrella — read it once, then open individual slice designs for implementation specifics.

## Architectural shape

```
   ┌──────────────────────────────────────────────────────────────────┐
   │  Frontend (cli / opm-operator / Crossplane fn)                   │
   │                                                                  │
   │  Owns: source resolution (FS / OCI / K8s objects / gRPC input).  │
   │  Owns: position-rich Tier-1 validation per source.               │
   │  Owns: layering policy for values (defaults → user → env → debug).│
   │  Owns: Platform composition policy (which Modules go in registry).│
   │  Owns: persistence of kernel output (SSA, YAML, XR composed).    │
   └─────────────────────────────────┬────────────────────────────────┘
                                     │
   ┌─────────────────────────────────▼────────────────────────────────┐
   │  opm/helper/  (kernel-shipped, opt-in)                           │
   │                                                                  │
   │   loader/file/   [shipped, slice 07]   Build cue.Value from FS.  │
   │   loader/bytes/  [skeleton, slice 07]  In-memory loading; full   │
   │                                        impl deferred until a    │
   │                                        consumer asks.           │
   │   platform/      [shipped, slice 10]   Compose(shell, modules) →│
   │                                        *Platform with #registry │
   │                                        filled.                  │
   │   synth/         [follow-up]           Synthesise a release CUE │
   │                                        value from typed inputs. │
   │                                        Peer of loader/.         │
   │   embed/         [deferred]            One-call wrapper for the │
   │                                        most common embedding    │
   │                                        patterns.                │
   │                                                                  │
   │  Tier-1 layered validation USED to live here as helper/values    │
   │  (slice 05). It was moved onto the kernel itself by the follow-up │
   │  redesign-config-validation; see D5 amendment.                   │
   └─────────────────────────────────┬────────────────────────────────┘
                                     │ canonical pre-unified inputs
                                     ▼
   ╔══════════════════════════════════════════════════════════════════╗
   ║  KERNEL                                                          ║
   ║                                                                  ║
   ║   type Kernel struct {                                           ║
   ║       cueCtx *cue.Context  // owned, never leaked                ║
   ║       logger *slog.Logger                                        ║
   ║       tracer trace.Tracer                                        ║
   ║       clock  Clock                                               ║
   ║       opts   Options                                             ║
   ║   }                                                              ║
   ║                                                                  ║
   ║   func New(opts ...Option) *Kernel                               ║
   ║                                                                  ║
   ║   k.Validate(ctx, ValidateInput) error                           ║
   ║   k.Match(ctx, MatchInput)       (*MatchPlan, error)             ║
   ║   k.Plan(ctx, PlanInput)         (*PlanResult, error)            ║
   ║   k.Compile(ctx, CompileInput)   (*CompileResult, error)         ║
   ║                                                                  ║
   ║   k.DetectApiVersion(v cue.Value) (apiversion.Version, error)    ║
   ║   k.Finalize(v cue.Value)        (cue.Value, error)              ║
   ║   k.CueContext()                 *cue.Context  // advanced       ║
   ╚══════════════════════════════════════════════════════════════════╝
```

## Uniform artifact shape

Implemented in slice 02, OpenSpec change `unify-artifact-shape`. See `openspec/changes/unify-artifact-shape/` for the full proposal, design, specs, and tasks.

Every OPM artifact accepted by the kernel has the same Go shape:

```go
type Module struct {
    ApiVersion apiversion.Version
    Metadata   *ModuleMetadata
    Package    cue.Value      // the whole loaded CUE — components, defines, config, etc.
}

type ModuleRelease struct {
    ApiVersion apiversion.Version
    Metadata   *ReleaseMetadata
    Package    cue.Value
}

type Platform struct {
    ApiVersion apiversion.Version
    Metadata   *PlatformMetadata
    Package    cue.Value      // contains #registry + computed views
}
```

`Metadata` is an ergonomic projection — useful for log fields, UI display, and routing. `Package` is the source of truth. When the two disagree, **`Package` wins**: `Metadata` is a cache, not an authority. This protects the "everything in CUE" property from drifting as the Go API evolves.

The kernel reads field paths within `Package` through the version-binding (`opm/api/<version>`). No path string is hardcoded in render or match logic.

## Compile pipeline

```
   CompileInput {
     Module        *Module        // consumer
     ModuleRelease *ModuleRelease
     Platform      *Platform      // pre-composed by frontend
     Values        cue.Value      // singular, pre-unified, helper-validated
     RuntimeName   string
   }
        │
        ▼
   1. Detect & verify APIVersion across all artifacts;
      look up the binding (opm/api/<version>).
        │
        ▼
   2. Tier-2 Validate Values vs Module's #config (via binding paths).
        │
        ▼
   3. Fill validated Values into ModuleRelease.Package at binding's values path.
        │
        ▼
   4. Build the demand walk:
        - Walk Module.#components for required Resource / Trait FQNs.
        - Look up each FQN in Platform.#matchers.{resources,traits}.
        - Surface unmatched / ambiguous; matched produces (component, transformer)
          pairs.
      (Slice 09 implements this in Go, mirroring the CUE shape from
       enhancement 003's #PlatformMatch. Claim demand deferred to a later slice.)
        │
        ▼
   5. For each matched (component, transformer):
        - FillPath component + binding-built #context into transformer.
        - Evaluate; decode `output` (cue.ListKind | cue.StructKind).
        - Emit *core.Compiled with full provenance (release/component/transformer FQN).
        │
        ▼
   6. Finalize and assemble:
        CompileResult {
          Compiled    []*core.Compiled
          Components  []ComponentSummary
          Unmatched   []FQN
          Ambiguous   []FQN
          Warnings    []string
          // Resolution map[string]cue.Value — added when Claims land.
        }
```

## Two-tier validation (post-amendment)

The original design placed Tier 1 in `opm/helper/values`. The follow-up `redesign-config-validation` change moved both tiers onto the kernel — Tier 1 is now `Kernel.ValidateConfigDetailed` with `Source` values and the optional `Partial()` option; Tier 2 is the existing `Kernel.ValidateConfig`. See D5 amendment for the rationale.

```
   Per-source values (defaults, user --values flags, env overlay, debugValues)
                │
                ▼
   ┌─────────────────────────────────────┐
   │  KERNEL — TIER 1                     │
   │                                      │
   │  Kernel.ValidateConfigDetailed       │
   │  Source{Value, Name, Origin}          │
   │  Optional: Partial()                  │
   │                                      │
   │  - Validate each source individually │
   │  - Source-attributed errors via      │
   │    token.Pos.Filename = Origin       │
   │  - Optional but recommended for     │
   │    quality frontend UX.              │
   └────────────────┬─────────────────────┘
                    │ unified cue.Value
                    ▼
   ┌─────────────────────────────────────┐
   │  KERNEL — TIER 2                     │
   │                                      │
   │  Kernel.ValidateConfig               │
   │                                      │
   │  - Re-validates unified value        │
   │  - Errors: "values do not satisfy    │
   │     #config: <CUE error tree>"       │
   │  - Correctness safety net;           │
   │    should not fire in practice if    │
   │    Tier 1 ran.                        │
   │  - Runs only when Values is set;     │
   │    zero cue.Value skips validation.  │
   └─────────────────────────────────────┘
```

The kernel never trusts that Tier 1 ran. Tier 2 still runs when `Values` is supplied. Frontends that skip Tier 1 get correct output, just with worse error messages.

## `cue.Context` ownership

The `Kernel` constructs and owns its `cue.Context`. It is never exposed in method signatures — `Validate`, `Match`, `Plan`, `Compile` take `(ctx context.Context, input X)` only. Helpers either call methods on `*Kernel` or take `*Kernel` as a parameter and reach in for the context (no helper builds its own `cue.Context`).

A `k.CueContext()` accessor exists for advanced cases (programmatic CUE construction in tests). Documented as advanced; most callers never use it.

A `Kernel` is **not goroutine-safe across compile calls**. Each goroutine constructs its own kernel. Construction is cheap (a struct + a fresh `cue.Context`). The operator pattern is "one Kernel per worker"; the XR fn pattern is "one Kernel per request"; the CLI pattern is "one Kernel per command."

## Multi-version dispatch (provided by `add-multi-apiversion-support`)

`add-multi-apiversion-support` is an in-flight prerequisite. It introduces:

- `opm/apiversion` — typed version enum + detection from a `cue.Value`.
- `opm/api/<version>` — per-version binding (paths, decoders, context shape).
- A closed registry populated by `init()` in each binding package.

Every slice in this enhancement uses the binding interface. The kernel never hardcodes a path string after that change lands.

## Slice dependency graph (as shipped)

```
   apiversion (prerequisite — shipped 2026-05-08)
        │
        ├──► 01 add-kernel-struct ✅
        │         │
        │         ├──► 06 add-phase-methods-and-rename-compile ✅
        │         │         │
        │         │         └──► 07 reorganize-helpers-under-helper ✅
        │         │
        │         └──► (07 also depends on 01)
        │
        ├──► 02 unify-artifact-shape ✅
        │         │
        │         └──► 08 add-platform-construct ✅
        │                   │
        │                   ├──► 09 rewrite-match-around-platform ✅
        │                   │
        │                   └──► 10 add-platform-composition-helper ✅
        │
        ├──► 03 retire-module-debug ✅   (independent)
        │
        └──► 04 accept-single-values-input ✅
                  │
                  └──► 05 introduce-tiered-validation ✅ (amended)
                            │
                            └──► redesign-config-validation ✅ (follow-up; removed helper/values)

   Independent follow-ups (no graph relation, ordered by ship date):
     2026-05-09 fold-deprecated-functions-into-kernel
     2026-05-09 slim-kernel-inputs (= original slice 11)
     2026-05-10 add-cue-schema-test-harness / -coverage
     2026-05-12 add-release-synth-helper
     2026-05-12 unify-loaders-as-packages
     2026-05-14 add-loader-shape-gates
     2026-05-14 replace-load-platform-file-with-package
```

Slices 03 and 04 landed independently as planned. Slices 01 and 02 are the foundation for everything else. Slice 09 — the matcher rewrite — was the highest-risk slice and landed in the same batch as its prerequisites (08, 10) once `#Platform` was stable.

## Why this slicing

- **Each slice is independently shippable.** No slice leaves the kernel in a broken state. Existing fixtures and tests pass at every commit.
- **Each slice is independently reviewable.** The largest slice (09) touches `opm/render/match.go` and the matching contract; the smallest (03) deletes a single construct. Reviewers can engage at the depth a slice deserves.
- **Cross-slice dependencies are explicit, not implicit.** The graph above lets us land slices in parallel where possible. 03 and 04 do not block 01 or 02.
- **The risky slice is gated.** Slice 09 is the matcher rewrite. By the time it lands, 02 and 08 have already established the artifact shape and the Platform type, so 09 is purely about matching logic — not types.
- **`#Claim` does not appear in any slice.** Until enhancement 005 stabilizes, the kernel matches Resources/Traits only. When 005 lands, a follow-up enhancement (`006-claims-in-kernel` or similar) adds Claim demand walking, `#ModuleTransformer` execution, and `#resolution` writeback.

## Open design questions — resolutions

- **Layering shape.** Moot. `helper/values` was removed; the kernel exposes ordered `[]Source` via `Kernel.ValidateConfigDetailed`. No named-layer abstraction shipped.
- **`helper/embed`.** Deferred indefinitely. YAGNI held — no consumer asked.
- **`PlanResult` vs `CompileResult` reuse.** Distinct `PlanResult` type. See `opm/kernel/results.go`. Fields: `MatchPlan`, `Components`, `Unmatched`, `Warnings`. Note: original sketch listed an `Ambiguous` field; ambiguity is instead surfaced via `MatchPlan.UnhandledTraits` and `MatchPlan.Warnings()`.
- **`Kernel.New` options style.** Functional options (`WithLogger`, `WithTracer`, `WithClock`). See `opm/kernel/kernel.go`.
