# Problem Statement — Platform Registry Subscription

## Current State

[Enhancement 003](../003-platform-construct/) introduced `#Platform.#registry` as a map of `#ModuleRegistration` entries, each embedding a concrete `#Module` value:

```cue
// core/platform.cue today
#Platform: {
  #registry: [Id=#NameType]: #ModuleRegistration
}

#ModuleRegistration: {
  #module!: #Module
  enabled: bool | *true
  presentation?: { ... }
}
```

A platform is wired up by *importing* a Module via CUE and assigning it to `#registry.<id>.#module`. The default Kubernetes fixture at `library/modules/opm_platform/platform.cue` does exactly this:

```cue
import opm_package "opmodel.dev/modules/opm"
#registry: opm: { #module: opm_package, enabled: true }
```

`#Platform` then projects four computed views over `#registry`:

- `#knownResources: [FQN]: #Resource` — union of every `#defines.resources` across enabled entries
- `#knownTraits: [FQN]: #Trait` — same for `#defines.traits`
- `#composedTransformers: [FQN]: #ComponentTransformer` — same for `#defines.transformers`
- `#matchers.{resources,traits}: [FQN]: [...#ComponentTransformer]` — reverse index from primitive FQN to the list of transformers requiring it

Primitive FQNs are MAJOR-only: `path/name@v1` (`#FQNType` regex pins `@v[0-9]+$`). Each catalog Module is itself a `#Module` value with its primitives published through `#defines.{resources,traits,transformers}` (a map keyed by `#FQNType` with `#Resource` / `#Trait` / `#ComponentTransformer` values). The kernel's Go matcher (`opm/compile/match.go`) walks the consumer Module's component-level resource/trait FQNs, looks each up in `#matchers`, and pairs the survivors via predicate evaluation.

## Gap / Pain

The static-Module shape closes three doors that a real platform needs open:

1. **One version per platform.** CUE imports pin a single MAJOR via `cue.mod/module.cue`, and the assigned `#module:` value is one concrete Module. The platform can host exactly one build of each primitive at a time. End-users whose Modules pin different patches of the same catalog cannot share a platform — every primitive-version drift forces a separate platform definition.

2. **MAJOR-only FQNs hide minor/patch drift.** `container@v1` covers every `1.x.x` build. Schema additions in `1.4.0` are invisible to the matcher: if the platform was built against `1.0.0` and the consumer pins `1.4.0`, the FQN matches but the schemas may diverge silently. The Go matcher has no signal to refuse the pairing.

3. **`#Module.#defines` conflates two roles.** Today `#Module` is simultaneously the consumer artifact (deployed via `#ModuleRelease`) and the catalog publication channel (`#defines` exposes primitives to platforms). Catalog authors and application authors share a type that doesn't fit either job cleanly: catalog modules carry empty `#components` / `#config` / `debugValues`; consumer modules carry empty `#defines`. The 1-to-1 of (Module value imported via CUE) to (platform-side identity) is what locks the platform to a single Module-version.

A subtler corollary: the platform's `#registry` is a fixed compile-time artifact. There is no path-of-truth for "which versions of this catalog does the platform accept?" — the answer is "exactly the one the platform's CUE imports happen to pin." Platform teams have no lever to express version policy beyond editing imports and republishing the platform CUE.

## Concrete Example

Suppose a platform team operates the `k8s-prod` platform and supports the `opmodel.dev/modules/opm` catalog. Two application teams use it:

- **App A** depends on `opmodel.dev/modules/opm@v1.0.4` — pinned at that build because their charts were authored before `1.1.0` shipped.
- **App B** depends on `opmodel.dev/modules/opm@v1.4.0` — pinned at that build because they need a `scaling` trait field added in `1.4.0`.

Today's platform CUE pins exactly one:

```cue
import opm_package "opmodel.dev/modules/opm"  // resolves to one tag, say 1.4.0
#registry: opm: { #module: opm_package }
```

App A's Module Release goes through the matcher. Its components declare `container@v1` (FQN), and the platform has a transformer keyed on `container@v1`. The FQN matches. The platform's `1.4.0` Container schema requires a field that App A's `1.0.4` Container value doesn't supply — but neither the FQN check nor the predicate check sees this. The render proceeds with the platform's `1.4.0` view of the resource and the consumer's `1.0.4` value, and the K8s object that comes out is either silently wrong (field defaulted to platform-1.4.0 behavior) or fails at apply time with a diagnostic that doesn't trace back to the version mismatch.

If the platform team flips the import to `1.0.4` to fix App A, App B breaks the same way.

The escape hatch — stand up two platforms, `k8s-prod-old` and `k8s-prod-new` — duplicates every other piece of platform policy (labels, type, future `#ctx`) and defeats the point of having one platform serving the cluster.

## Why Existing Workarounds Fail

**Pinning everyone to the same catalog version.** Forces lockstep upgrades across every application that targets the platform. The catalog is supposed to be a library; libraries don't impose lockstep on consumers.

**Splitting the platform per supported version.** Multiplies platform definitions by the cardinality of versions the cluster needs to support. Every cross-cutting platform change (a new trait, a label change, a new `#ctx` field) propagates to N forks.

**Bumping MAJOR for every schema change.** Makes MAJOR FQNs useful for matching again, at the cost of marking every patch-grade catalog change as a breaking version bump. Catalog evolves like 1.0.0 → 2.0.0 → 3.0.0 inside a week. Loses SemVer's semantic value.

**Manual schema-version assertions in transformer predicates.** A transformer's `requiredLabels` could carry a `catalog.version=1.4.0` label that consumer Modules also stamp. Catches mismatch, but requires every author to opt in and surfaces version drift as a generic predicate failure rather than as a structured diagnostic.

None of these address the underlying shape: a platform should be able to *subscribe* to a catalog (a range of its versions), and the matcher should refuse pairings where the consumer's primitive FQN — including its SemVer — isn't part of the subscribed set, *and* refuse pairings where the FQN matches but the primitive schemas diverge.
