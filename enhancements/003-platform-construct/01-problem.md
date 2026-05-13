# Problem Statement — `#Platform` Construct

## Current State

An earlier `#Platform` design used three composition fields:

- `#providers: [...provider.#Provider]` — ordered list of providers whose `#transformers` maps unify into `#composedTransformers`.
- `#ctx: ctx.#PlatformContext` — platform-level context defaults.
- `#composedTransformers`, `#provider`, `#declaredResources`, `#declaredTraits` — computed from `#providers`.

`#Provider` is the unit of transformer registration. Each `#Provider` value lives in its own CUE package (`opmodel.dev/opm/v1alpha2/providers/kubernetes`, `opmodel.dev/k8up/v1alpha2/providers/kubernetes`, etc.) and exports a `#transformers` map keyed by transformer FQN.

Enhancement 005 introduces a flat `#Module` shape with a publication slot, `#defines`, that bundles type definitions (`resources`, `traits`, `claims`) and rendering extensions (`transformers`) keyed by FQN. A Module shipping `#defines.transformers` is structurally equivalent to a `#Provider` for transformer-registration purposes.

## Gap / Pain

There are now two concepts answering the same question: *"how does a platform extension contribute transformers?"*

- `#Provider` answers: by exporting a `#transformers` map and being listed in `#Platform.#providers`.
- `#Module.#defines.transformers` answers: by listing transformers under the Module's publication slot.

This duplication has three concrete consequences:

1. **Static composition.** `#Platform.#providers` is an authored list. Every change requires editing the platform CUE file. There is no schema-level home for runtime-discovered extensions (operators installed on the cluster, vendor modules pulled from a registry, modules registered via a CRD watch).
2. **Two ingress paths for the same data.** A K8up backup operator currently ships a `#Provider`. Under 005 it ships a `#Module` with `#defines.transformers` *plus* `#components`, `#defines.{resources,traits,claims}`. Forcing platforms to consume both `#Provider` (for transformers) and `#Module` (for everything else) doubles the surface and forces operators to either ship both or accept that one ingress goes underused.
3. **Catalog discovery is split.** A platform that wants to answer "what Resources / Traits / Claims are available here?" must walk both `#providers[*]` (for transformer-declared FQNs only) and a separate registry of Modules (for Claim definitions, which transformers do not reference).

## Concrete Example

Consider a platform admin assembling a Kubernetes platform with the OPM core, K8up backup, and a Postgres operator. Under the earlier design:

```cue
#Platform: core.#Platform & {
    metadata: name: "kind-opm-dev"
    type: "kubernetes"
    #providers: [opm.#Provider, k8up.#Provider, pgop.#Provider]
    #ctx: { ... }
}
```

Under 005, the Postgres operator is naturally a `#Module`:

```cue
pgopModule: module.#Module & {
    metadata: { ..., name: "postgres-operator" }
    #components: { controller: ..., crds: ... }
    #defines: transformers: {
        (mdt.metadata.fqn): mdt   // ManagedDatabaseTransformer; requiredClaims: ManagedDatabaseClaim.fqn
    }
}
```

To plug this into the earlier `#Platform`, the admin must either:

- Wrap the Module's transformers back into a synthetic `#Provider` value just to satisfy `#providers: [...]`, ignoring `#components` / `#defines.{resources,traits,claims}` entirely; or
- Maintain two parallel registrations — a Module for runtime deployment, a Provider for transformer composition.

Both paths discard information. The Module's `#defines.{resources,traits,claims}` (catalog publication) has no platform-level home in the earlier design.

## Why Existing Workarounds Fail

- **Synthesise a Provider from a Module's `#defines.transformers` at the platform CUE level.** Works mechanically but loses the catalog-publication pathway entirely. The platform still cannot answer "what Claim types are available here?" because Claim type definitions are not referenced by transformer match keys (only `requiredClaims` is) and live outside `#Provider`.
- **Keep `#Provider` and `#Module` as parallel concepts indefinitely.** Forces every ecosystem participant (operator vendor, capability module author) to ship both forms or accept that platform tooling will only see half their contribution. Doubles the migration burden of 005 and contradicts 005 CL-D2's "ecosystem-extended" vision.
- **Discover modules at runtime via a separate side-channel (CRD watch, registry query).** Possible, but with no schema-level slot the discovered values have nowhere to land in CUE. The platform's computed views (`#composedTransformers`, `#declaredResources`, etc.) cannot be unified across static and runtime-discovered sources.

A proper solution requires `#Platform` to accept `#Module` values directly as the composition unit, with a schema slot that is dynamic enough to receive both author-static and runtime-injected entries.
