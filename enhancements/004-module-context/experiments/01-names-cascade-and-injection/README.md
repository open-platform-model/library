# 01-names-cascade-and-injection

## Hypothesis

The `#ContextBuilder` specified in [004/03-schema.md §`#ContextBuilder`](../../03-schema.md#contextbuilder) produces `#ComponentNames` such that the four `dns.*` variants cascade from `resourceName`, the `metadata.resourceName` override propagates through the cascade, `_clusterDomain` self-defaults to `"cluster.local"` (no override path in post-slim 004), and after `#ModuleRelease` unifies the injections back, every component's `#names` is value-equal to its `#ctx.runtime.components` entry.

| Case | Scenario | Expected |
| ---- | -------- | -------- |
| 01-default-cascade | one component, no `metadata.resourceName` | `resourceName = "{release}-{component}"`; all four `dns.*` derive from it |
| 02-resource-name-override | `metadata.resourceName: "router"` | every `dns.*` variant uses `"router"` as the base name |
| 03-cluster-domain-default | no override fixture; `_clusterDomain` self-defaults | `dns.fqdn` ends `".svc.cluster.local"` — no override surface exists in 004 |
| 04-names-lockstep | two components (one default, one override); full `#ModuleRelease` flow | `release.components.<k>.#names == release.ctx.runtime.components.<k>` |

## Setup

Copied / trimmed from `apis/core/v1alpha2/`:

- `schemas/types.cue` — `#NameType`, `#ModuleFQNType`, `#VersionType`, `#UUIDType` only.
- `schemas/component.cue` — trimmed `#Component` carrying just `metadata`, `metadata.resourceName?`, `#names: #ComponentNames`, and an optional `spec?: _` slot.
- `schemas/module.cue` — trimmed `#Module` carrying just `metadata`, `#components`, `#config`, `#ctx`.
- `schemas/module_release.cue` — trimmed `#ModuleRelease` implementing the 3-step flow.

Authored fresh per [004/03-schema.md](../../03-schema.md):

- `schemas/context.cue` — `#ModuleContext`, `#RuntimeContext`, `#ComponentNames`.
- `schemas/context_builder.cue` — `#ContextBuilder` (identity-only).

Each fixture lives in its own package under `cases/NN-name/main.cue`. Cases 01–03 invoke `#ContextBuilder` directly with synthetic inputs (focused on the cascade); case 04 routes through `#ModuleRelease` so the schema-application path is exercised end-to-end.

Cases assert via `_assertX: <literal> & <computed>` unification — any cascade or lock-step regression turns into a `conflicting values` CUE error pointing at the offending field.

## Run

```bash
cd enhancements/004-module-context/experiments/01-names-cascade-and-injection
./run.sh
```

`cue vet -c ./cases/<name>/...` per case. All four cases must pass.

## Outcome (2026-05-15)

All four cases produce the expected results.

```text
=== 01-default-cascade (expect pass) ===
[result] pass

=== 02-resource-name-override (expect pass) ===
[result] pass

=== 03-cluster-domain-default (expect pass) ===
[result] pass

=== 04-names-lockstep (expect pass) ===
[result] pass
```

### Findings

- **D10 confirmed.** With no `metadata.resourceName`, `resourceName` defaults to `"{release}-{component}"` and every `dns.*` variant (`local`, `namespaced`, `svc`, `fqdn`) derives without surprise.
- **D13 confirmed.** `metadata.resourceName` flows into `#ComponentNames.resourceName` via `#ContextBuilder`'s `if comp.metadata.resourceName != _|_ { resourceName: ... }` block, and CUE unification replaces the default. The cascade through `dns.*` carries the override with no additional wiring.
- **D36 / cluster-domain self-default confirmed.** `_clusterDomain` resolves to `"cluster.local"` purely from the `string | *"cluster.local"` default; no `#platform` / `#environment` input is needed, and no override surface exists in 004. Case 03 demonstrates this by absence — no override fixture is even possible.
- **D32 lock-step confirmed end-to-end.** After `#ModuleRelease` unifies `_builderOut.injections` back into `#module.#components`, `release.components.<k>.#names & release.ctx.runtime.components.<k>` produces no conflict. The single `_componentNames` let-binding in `#ContextBuilder` is the source of truth for both surfaces; threading is correct.
- **Stub footnote.** `#Component.spec` had to be made optional (`spec?: _`) in the trimmed stub for `cue vet -c` to accept the fixture. Production `#Component.spec` is `close({_allFields})`, which is concrete via the merge of resource/trait/blueprint specs. The trimmed stub doesn't carry those, so a `_` slot is incomplete under `-c`. This is an experiment-only artifact, not a real 004 concern.

### Notes for 004 docs

- `04-decisions.md` D10, D13, D32, D36 — confirmed under CUE v0.16.1; can be promoted from "expected behavior" to "validated".
- `02-design.md` §`#ContextBuilder` — the single-source-of-truth `_componentNames` let-binding is now empirically verified; the design note about `#Component.#names == #ctx.runtime.components[<key>]` (D32) holds.

## Status

Concluded — 2026-05-15. May be deleted once 004 lands in `apis/core/v1alpha2/`.
