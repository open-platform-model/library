# Design Package: `#ctx` — Module Runtime Context

## Summary

Defines `#ctx` as the runtime-context channel injected into every `#Module` at release time. `#ctx` carries deployment identity (release name, namespace, UUID; module name, version, FQN, UUID) and the per-component computed names (`resourceName`, DNS variants). Every field is derived from the release, the module, and the component set — `#ctx` takes no platform or environment inputs.

`#ctx` is **not** authored by module developers. It is computed by `#ContextBuilder` and unified into the module by `#ModuleRelease` during evaluation. Components reference it inside their specs (e.g. `#ctx.runtime.components.foo.dns.fqdn`) without any operator input.

`#ContextBuilder` takes the release identity, the module identity, and the component map, and produces `#ModuleContext` — the value `#Module.#ctx` resolves to. There are no layered inputs: 004 is self-contained.

This enhancement lands `#ctx` as a standalone schema so that 003 (Platform construct) and 005 (Module schema) can both reference a single context-system source.

Two surfaces from earlier drafts of 004 have been moved out. The `#ctx.platform` extension layer and the `#Environment` construct — together with `#PlatformContext` / `#EnvironmentContext`, the layered Platform → Environment hierarchy, the cluster-domain override, and the `route` domain — are all enhancement 006 (Platform Capabilities), which adds a typed `#Capability` model and a kernel-populated `#platform: #Platform` field on `#ModuleRelease`. 006 does **not** reintroduce `#Environment` — per-platform variation uses CUE unification of `#Platform` values (006 OQ6). 004 covers identity-only `#ctx.runtime`. See [04-decisions.md](04-decisions.md) D36.

## Documents

1. [01-problem.md](01-problem.md) — Per-component computed names are scattered across transformers; modules have no schema-level home for release/module identity
2. [02-design.md](02-design.md) — `#ctx` single-layer (`runtime`) shape; `#ComponentNames` cascade; `#ContextBuilder` flow; integration with `#Module` and `#ModuleRelease`
3. [03-schema.md](03-schema.md) — CUE definitions for `#ModuleContext`, `#RuntimeContext`, `#ComponentNames`, `#ContextBuilder`
4. [04-decisions.md](04-decisions.md) — Decision log

## Scope

### In scope

- `#ctx` definition field on `#Module` (referenced from 005).
- `#ModuleContext`, `#RuntimeContext`, `#ComponentNames` schemas.
- `#ContextBuilder` helper that assembles `#ModuleContext` from release identity, module identity, and the component map.
- `#ModuleRelease` integration sketch (how the builder is invoked).
- `#Component.metadata.resourceName` override and the cascade through `#ComponentNames`.
- `#Component.#names: #ComponentNames` per-component injection so a component reads its own resourceName / DNS variants without retyping its map key (D32).

### Out of scope

- `#Platform` composition (`#registry`, computed views) — owned by 003.
- `#Module` schema (slots, `#defines`, `#claims`) — owned by 005.
- `#TransformerContext` and how `#ctx` relates to it — deferred to a follow-up.
- Bundle-level context (cross-module references) — deferred (see Open Questions).
- Content hashes for immutable ConfigMaps/Secrets — removed from this enhancement (see D31); revisit when a concrete module-readable use case surfaces.
- Runtime-fill mechanism for `#registry` (003's territory).
- `#Environment` construct, `#PlatformContext` / `#EnvironmentContext`, the layered Platform → Environment hierarchy, the cluster-domain override, and the `route` domain — all extracted to enhancement 006 (Platform Capabilities); see D36.
- `#ctx.platform` extension layer (the open struct, and its typed replacement) — enhancement 006; see D36.

## Cross-References

| Document | Purpose |
| -------- | ------- |
| `CONSTITUTION.md` (repo root) | Core design principles |
| `experiments/` | In-tree experiments validating the post-slim design. See [experiments/README.md](experiments/README.md) for the index; the per-experiment READMEs carry the detail. |
| `catalog/experiments/001-module-context/` *(sibling repo)* | Self-contained CUE sandbox built against the **pre-slim** 004 design. Finding 1 (cluster-domain resolution / D33) and the layered Platform → Environment fixtures are superseded by D36. Findings 2–3 (config-first ordering D34, lexical scope D35) are re-grounded in the in-tree `experiments/02-release-flow-ordering/`. Retained as historical provenance. |
| `apis/core/v1alpha2/module.cue` | Gains `#ctx: #ModuleContext` field |
| `apis/core/v1alpha2/component.cue` | Gains optional `metadata.resourceName` override and a `#names: #ComponentNames` definition field |
| `apis/core/v1alpha2/module_release.cue` | Modified to invoke `#ContextBuilder` and unify both `ctx` and per-component `#names` injections into the module |

## Applicability Checklist

- [x] `03-schema.md` — New CUE definitions for `#ModuleContext`, `#RuntimeContext`, `#ComponentNames`, `#ContextBuilder`
- [ ] `NN-pipeline-changes.md` — Go pipeline modifications (deferred — content-hash injection, etc.)
- [ ] `NN-module-integration.md` — Module-author migration of `#config`-borne identities into `#ctx` references (deferred to a follow-up)
- [ ] `NN-context-flow.md` — Visual flow diagram (folded into 02-design.md while thin)
- [x] `experiments/` — Two experiments concluded 2026-05-15. [01-names-cascade-and-injection](experiments/01-names-cascade-and-injection/README.md) validates the `#ComponentNames` cascade, the `metadata.resourceName` override, the `_clusterDomain` self-default, and the D32 lock-step. [02-release-flow-ordering](experiments/02-release-flow-ordering/README.md) validates D11 identity propagation, the D34 config-first ordering (including the silent-failure counter-fixture), and the D35 inline-literal scope requirement.
