# Design Package: `#ctx` — Module Runtime Context

| Field       | Value            |
| ----------- | ---------------- |
| **Status**  | Draft (validated by experiment 001) |
| **Created** | 2026-04-30       |
| **Authors** | OPM Contributors |

## Summary

> **Implementation status (2026-05-13).** None of this enhancement is in v1alpha2 yet. `apis/core/v1alpha2/` carries no `#ModuleContext`, `#PlatformContext`, `#EnvironmentContext`, `#RuntimeContext`, `#ComponentNames`, `#Environment`, or `#ContextBuilder`; `#Module` has no `#ctx` field; `#Component` has no `metadata.resourceName` or `#names`. A related but distinct surface — `#TransformerContext` in `apis/core/v1alpha2/transformer.cue:99-198` — exists for transformer-internal use; unification with `#ctx` is deferred (D15). The Step-1 config unification (`let unifiedModule = #module & {#config: values}` at `apis/core/v1alpha2/module_release.cue:38`) is the only piece of the builder flow that has landed.

Defines `#ctx` as the runtime-context channel injected into every `#Module` at release time. `#ctx` carries the deployment identity (release name, namespace, UUID; module name, version, FQN, UUID), the cluster environment (cluster domain, optional route domain), the per-component computed names (`resourceName`, DNS variants), and an open `platform` extension struct that platform teams use to publish per-platform facts (storage classes, backup backends, TLS issuers, gateways, app domains).

`#ctx` is **not** authored by module developers. It is computed by `#ContextBuilder` and unified into the module by `#ModuleRelease` during evaluation. Components reference it inside their specs (e.g. `#ctx.runtime.route.domain`, `#ctx.runtime.components.foo.dns.fqdn`, `#ctx.platform.appDomain`) without any operator input.

The schema is structured as a layered hierarchy: `#PlatformContext` (Layer 1, supplied by `#Platform`), `#EnvironmentContext` (Layer 2, supplied by `#Environment`), and release identity (Layer 3, from `#ModuleRelease.metadata`) merge into `#ModuleContext` — the value that `#Module.#ctx` resolves to.

This enhancement lands `#ctx` as a standalone schema so that 003 (Platform construct) and 005 (Module schema) can both reference a single context-system source. 003 owns Platform composition; 005 owns the Module shape; 004 owns the context schemas, `#Environment` minimum-context-node form, and `#ContextBuilder` only.

## Documents

1. [01-problem.md](01-problem.md) — Module blindness to deployment context; values vs context confusion; per-platform facts have no schema-level home
2. [02-design.md](02-design.md) — `#ctx` two-layer (`runtime` + `platform`) shape; layered hierarchy; `#ContextBuilder` flow; integration with `#Platform`, `#Environment`, `#ModuleRelease`
3. [03-schema.md](03-schema.md) — CUE definitions for `#ModuleContext`, `#PlatformContext`, `#EnvironmentContext`, `#RuntimeContext`, `#ComponentNames`, `#Environment`, `#ContextBuilder`
4. [04-decisions.md](04-decisions.md) — Decision log

## Scope

### In scope

- `#ctx` definition field on `#Module` (referenced from 005).
- `#ModuleContext`, `#PlatformContext`, `#EnvironmentContext`, `#RuntimeContext`, `#ComponentNames` schemas.
- `#ContextBuilder` helper that assembles the final `#ModuleContext` from layered inputs.
- `#Environment` construct in its minimum form (metadata + `#ctx: #EnvironmentContext` + `#platform` reference). The construct exists primarily as the Layer 2 context node and as the deployment target that `#ModuleRelease.#env` points at.
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

## Cross-References

| Document | Purpose |
| -------- | ------- |
| `CONSTITUTION.md` (repo root) | Core design principles |
| `enhancements/003-platform-construct/` | Sibling — `#Platform.#ctx` references `#PlatformContext` defined here |
| `enhancements/005-claims/` | Sibling — `#Module.#ctx` references `#ModuleContext` defined here |
| `experiments/001-module-context/` | Self-contained CUE sandbox that builds and exercises every schema in `03-schema.md`. 11 test files (40 concrete assertions). Surfaced D33–D35 corrections to the schema. Run: `cue vet -c -t test ./...` from the experiment dir. |
| `apis/core/v1alpha2/module.cue` | Gains `#ctx: #ModuleContext` field |
| `apis/core/v1alpha2/component.cue` | Gains optional `metadata.resourceName` override and a `#names: #ComponentNames` definition field |
| `apis/core/v1alpha2/module_release.cue` | Modified to invoke `#ContextBuilder` and unify both `ctx` and per-component `#names` injections into the module |

## Applicability Checklist

- [x] `03-schema.md` — New CUE definitions for `#ModuleContext`, `#PlatformContext`, `#EnvironmentContext`, `#RuntimeContext`, `#ComponentNames`, `#Environment`, `#ContextBuilder`
- [ ] `NN-pipeline-changes.md` — Go pipeline modifications (deferred — content-hash injection, etc.)
- [ ] `NN-module-integration.md` — Module-author migration of `#config`-borne URLs/identities into `#ctx` references (deferred to a follow-up)
- [ ] `NN-context-flow.md` — Visual flow diagram (folded into 02-design.md while thin)
