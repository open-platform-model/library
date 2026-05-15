# Problem Statement — `#ctx` Module Runtime Context

## Current State

`#Module` exposes `#config` as the user-supplied values contract. Authors put everything that varies per release into `#config`: replicas, image tags, hostnames, namespaces, secret references. The render pipeline unifies `#ModuleRelease.values` into `#config` at deploy time and components read from `#config.*`.

Two kinds of fact have no schema-level home today: the **per-component computed names** (the `{release}-{component}` base name and its `name.ns.svc.cluster.local` FQDN forms used for service references) and the module's own **release/module identity** (release name, namespace, UUID; module name, version, FQN, UUID). Neither is user-supplied — both are derivable from the release and the module — yet there is nowhere a component can read them. They land wherever the module author or transformer can fit them:

- Re-implemented in every transformer — each transformer that emits a Service / Ingress / Route / Certificate independently computes `{release}-{component}` and the FQDN variants.
- Duplicated across modules — every module that constructs a service FQDN re-implements the `name.ns.svc.cluster.local` template by hand.
- Inlined into transformers via Go-side `FillPath` (the existing `#TransformerContext`) — works for transformer-internal code but is invisible to the module's components at definition time.

## Gap / Pain

### Per-component computed names are scattered

Every transformer that emits a named Kubernetes resource computes `{release}-{component}` and FQDN variants like `{release}-{component}.{namespace}.svc.cluster.local` on its own. The same template is re-implemented in each transformer and again in any module that wants to reference the result. The naming convention is convention-only: if it changes, every transformer and every module breaks separately. There is no single computed `#ComponentNames` struct that a `metadata.resourceName` override could propagate through uniformly.

### Modules cannot see their own identity

A component cannot reference its release name, its namespace, or its module FQN at definition time. `#config` is the wrong home — it is operator-supplied and OpenAPIv3-constrained. `#TransformerContext` carries some of this identity, but it is transformer-internal: a component body cannot read it. So a component that wants to construct, say, a self-referential DNS name for a sibling component has no derivable input to build it from.

### Values vs context confusion

`#config` is supposed to be the user-supplied contract — what an *application* needs from an *operator*. Release/module identity and computed resource names are not user-supplied; they are runtime-supplied. Forcing them through `#config` blurs ownership: the operator ends up filling fields they should not own, and the module author cannot tell which `#config` fields are real configuration versus identity leakage. `#config` also carries an OpenAPIv3 constraint (`#config` must be a valid schema); CUE-computed identity values and per-component DNS FQDNs do not fit that constraint cleanly.

## Concrete Example

A module has a `router` component and several `server-*` components. The `router` needs each server's in-cluster Service address, and each server needs its own FQDN for a self-registration env var.

Today the FQDN is fully derivable — from the release name, the component name, the namespace, and the cluster domain — but none of those are exposed to the module author. So the module hardcodes the template:

```cue
// the module re-implements the naming convention by hand
"server-alpha": {
    spec: container: env: SELF: {
        name:  "SELF"
        value: "release-server-alpha.media.svc.cluster.local"   // hand-built, brittle
    }
}
```

Meanwhile the Service transformer computes the *same* name independently as `{release}-{component}`. The two derivations are not linked. If a component sets a `metadata.resourceName` override, the transformer picks it up but the hand-built string in the module body does not — they silently drift. Every module that points one component at another repeats this, and every transformer that emits a named resource repeats the base-name computation.

## Why Existing Workarounds Fail

- **Hardcode the conventions in every transformer.** Done today; produces drift the moment a module wants to deviate (e.g. via a `resourceName` override). An override only works if it propagates uniformly through every name and DNS variant — which requires a single computed `#ComponentNames` struct, not scattered transformer-side computation.
- **Push everything to transformer-internal `#TransformerContext`.** Works for transformer code that reads `#TransformerContext` directly, but module components cannot see it. A component referencing its own or a sibling's computed name cannot express it at definition time.
- **Stuff identity and names into `#config` defaults.** Breaks the values-vs-context distinction — operators end up re-supplying values the catalog already knows — and `#config`'s OpenAPIv3 constraint clashes with CUE-computed values.

A proper solution requires a structured runtime-context channel parallel to `#config`, owned by the catalog, computed from release identity, module identity, and the component set, and unified into the Module before its components are evaluated. Platform- and environment-supplied context — route domains, cluster-domain overrides, platform capabilities — is a separate concern, handled by enhancement 006 (Platform Capabilities); see D36.
