## Context

Catalog 014 redefines the matching contract. The old model — a flat `#Provider` carrying a transformer map — gives way to a richer `#Platform` whose `#registry` of registered `#Module` values produces, by CUE comprehension, a `#composedTransformers` map (FQN → transformer) and a `#matchers.{resources, traits}` reverse index (primitive FQN → list of fulfilling transformers). Multi-fulfiller is forbidden at the platform level (catalog D13): if two transformers claim the same primitive FQN, platform evaluation fails before the kernel sees it.

The kernel matcher's job becomes simpler:

1. Walk the consumer Module's `#components`. For each component, collect the union of `#resources` and `#traits` FQNs it references.
2. For each demanded FQN, look it up in `Platform.#matchers.resources[FQN]` (or `.traits[FQN]`). If present, the lookup yields a one-element transformer list (multi-fulfiller forbidden at platform layer).
3. Pair the component with the located transformer. Surface unmatched FQNs (no fulfiller) and the legacy "ambiguous" diagnostic (which should always be empty given platform-layer enforcement; retained as a safety net).

Per umbrella decision (Q1), the matcher implements this walk in Go rather than instantiating catalog 014's `#PlatformMatch` CUE construct. The Go implementation mirrors `#PlatformMatch`'s semantics. Reasoning: keeps the existing `pkg/render/match.go` testing approach; avoids a CUE evaluation per match; allows the matcher to surface diagnostics in Go-native shape.

This is the only slice that changes runtime behavior. By the time it lands:

- Slice 01: `Kernel` struct exists.
- Slice 02: Module / Release have uniform shape; binding paths abstract field access.
- Slice 06: `MatchInput`, `MatchPlan`, `Compile`, etc. are phase methods.
- Slice 08: `Platform` type, loader, binding paths exist.
- `add-multi-apiversion-support`: binding registry exists.

## Goals / Non-Goals

**Goals:**
- Matcher consumes `*Platform` exclusively. `*Provider` removed from kernel inputs and from `pkg/`.
- Match algorithm walks consumer Module's `#components` for Resource/Trait FQN demand.
- Match algorithm looks up demand in `Platform.#matchers` via the binding paths.
- `*MatchPlan` API surface preserved (callers see the same structure shape, slightly different internals).
- Execute phase resolves transformers by FQN from `Platform.#composedTransformers`.
- All test fixtures rewrite from Provider to Platform; net behavior unchanged for fixtures with single-fulfiller setups.

**Non-Goals:**
- Implementing `#PlatformMatch` walking in CUE. Per umbrella Q1, matching is in Go.
- Claim demand or `#ModuleTransformer` execution. Deferred per umbrella scope (catalog 015 not yet stabilized).
- The `requiredLabels` matching. Catalog 014 still supports labels in `#ComponentTransformer`; the matcher walks them. But slice 09 keeps the existing labels-matching code path; it's an internal-only change of how transformers are located, not how match keys are evaluated.
- Multi-fulfiller resolution. Per catalog D13, multi-fulfiller is forbidden at the platform layer; the kernel honors this and returns ambiguous diagnostics if the platform somehow produced multi-candidate lists (defensive, should be empty).

## Decisions

**Match algorithm sketch:**

```
for each component in consumerModule.#components:
   demand.resources ∪= component.#resources keys (FQNs)
   demand.traits    ∪= component.#traits    keys (FQNs)

for each FQN in demand.resources:
   candidates := Platform.#matchers.resources[FQN]
   if candidates empty:        unmatched.resources += FQN
   elif len(candidates) > 1:   ambiguous.resources += FQN  (defensive)
   else:                       matched += (component, candidates[0])

(repeat for traits)
```

**Execute phase locates transformers by FQN.** Given a matched pair `(component, transformerFQN)`, execute looks up `Platform.#composedTransformers[transformerFQN]` to obtain the transformer's `#transform` body. FillPath component + #context, evaluate, decode `output`, emit `[]*core.Rendered`.

**`render.Module` runtime helper takes `*Platform`.** Old: `NewModule(p *provider.Provider, runtimeName string) *Module`. New: `NewModule(plat *platform.Platform, runtimeName string) *Module`. Behavior preserved otherwise.

**`*provider.Provider` package deleted.** Reason: no remaining consumer in the kernel after this slice. Downstream consumers migrate to `*platform.Platform`. Provided that slice 10 ships `pkg/helper/platform/Compose`, the migration is a one-line constructor change.

**`requiredLabels` continues to use the existing label-matching code.** That code is at `pkg/render/match.go` already; relocating it during slice 09 is unnecessary. Reason: keeps the slice focused on Provider→Platform substitution.

**Test fixtures rewrite to Platform.** Existing fixtures construct `Provider` literals or load `provider.cue` files. They become `Platform` literals or `platform.cue` files with a `#registry` containing the previously-implicit Module. Behavior is preserved for the single-fulfiller case (which is every existing fixture).

**Defensive ambiguous handling.** Even though catalog D13 forbids multi-fulfiller at the platform layer, the matcher tests `len(candidates) > 1` and surfaces an ambiguous diagnostic if it somehow happens (e.g. a misconstructed Platform fed directly to the kernel). Reason: defensive correctness; the cost is one length check per FQN.

**Binding-driven path access only.** The matcher never hardcodes `"#matchers"` or `"#composedTransformers"`. All access goes through `binding.Paths().*`. Reason: multi-version support; v1alpha1 (if backported) might use different paths.

## Risks / Trade-offs

**Risk — large blast radius.** This slice touches `pkg/render/match.go`, `pkg/render/execute.go`, `pkg/render/module.go`, deletes `pkg/provider/`, deletes provider loader. Mitigation: every prior slice landed first; this slice is purely the substitution. Diff is mechanical for the most part.

**Risk — fixture migration drift.** Test fixtures are central to the kernel's correctness story. If a fixture's behavior changes during the migration, hard to tell whether it's a regression or expected. Mitigation: snapshot every fixture's pre-slice-09 output; assert post-slice-09 output is byte-equal; investigate every diff.

**Risk — downstream consumer migration is not one-line.** Unlike slice 02 (path lookup substitution), slice 09 changes what artifact a consumer constructs. `cli` previously loaded a Provider; now it loads a Platform (and registers Modules into it). Mitigation: slice 10 ships the composition helper to make this one line. Until slice 10 lands, downstream consumers use a verbose direct-CUE composition pattern.

**Risk — existing `requiredLabels` semantics regression.** Mitigation: tests cover label-matching scenarios; the code path is preserved.

**Trade-off — match in Go, not CUE.** Per umbrella Q1. Cost: matching behavior must be reimplemented in Go to mirror CUE 014. Benefit: faster (no CUE evaluation per match), simpler integration with Go-native error types and testing.

**Trade-off — no Claim support.** Deferred. The matcher is structurally ready to accept claim demand; adding it later is a sibling pass at slice-9-equivalent depth, not a re-architecture.
