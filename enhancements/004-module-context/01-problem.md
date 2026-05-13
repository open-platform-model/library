# Problem Statement — `#ctx` Module Runtime Context

## Current State

`#Module` exposes `#config` as the user-supplied values contract. Authors put everything that varies per release into `#config`: replicas, image tags, hostnames, namespaces, published URLs, secret references, storage class names. The render pipeline unifies `#ModuleRelease.values` into `#config` at deploy time and components read from `#config.*`.

Cluster-environment facts (cluster DNS domain, ingress route domain, namespace defaults), per-platform extensions (default storage class, backup backends, TLS issuers, app domains), and per-component computed names (the `name-namespace.svc.cluster.local` FQDN forms for service references) currently have no schema-level home. They land wherever the module author can fit them:

- Hardcoded into `#config` defaults — e.g. `publishedServerUrl?: string` in Jellyfin, requiring the operator to manually re-supply a value derivable from the environment.
- Inlined into individual transformers via Go-side `FillPath` (the existing `#TransformerContext`) — works for transformer-internal use but invisible to the module's components at definition time.
- Duplicated across modules — every module that needs `cluster.local` writes the literal string, and every module that constructs a service FQDN re-implements the `name.ns.svc.cluster.local` template.

## Gap / Pain

### Module blindness to deployment context

A component that needs to construct an HTTP URL for its public endpoint (`https://jellyfin.example.com`) cannot derive that URL from anything available at module-definition time. The route domain (`example.com`) is environment-specific. The release name (`jellyfin`) and namespace (`media`) are release-specific. Today the operator must compute the URL externally and supply it through `#config.publishedServerUrl`. The module's components are downstream of `#config` and cannot synthesise the value themselves.

### Values vs context confusion

`#config` is supposed to be the user-supplied contract — what an *application* needs from an *operator*. Cluster facts, environment domains, and computed resource names are not user-supplied; they are runtime-supplied. Forcing them through `#config` blurs ownership: the operator ends up filling fields they should not own, and the module author cannot tell which `#config` fields are real configuration versus environment leakage.

`#config` also carries an OpenAPIv3 constraint (`debugValues` is concrete, `#config` must be a valid schema). Computed cluster domains and per-component DNS FQDNs do not fit that constraint cleanly; they are CUE-native computed values, not OpenAPI parameters.

### Per-component computed names are scattered

Every transformer that emits a Service / Ingress / Route / Certificate independently computes names like `{release}-{component}` and FQDN variants like `{release}-{component}.{namespace}.svc.cluster.local`. The same template is re-implemented in each transformer and in any module that wants to reference the result (e.g., a Jellyfin client URL pointing at the Jellyfin Service). The naming convention is convention-only; if it changes, every transformer + every module breaks separately.

### No place for per-platform extensions

Operational commodities (backup, TLS, routing — see the example in [005/08-examples.md](../005-claims/08-examples.md)) need platform-team-supplied configuration: which backup backends are available, which cert-manager issuers exist, which Gateway listeners are configured. Today these facts have to live somewhere the catalog has not modeled: ad-hoc `#config` extensions that every module author has to anticipate, or external configuration files outside CUE entirely.

## Concrete Example

The Jellyfin module ships with a `publishedServerUrl` field in `#config`:

```cue
// modules/jellyfin/module.cue
#config: {
    publishedServerUrl?: string
}

// modules/jellyfin/components.cue
if #config.publishedServerUrl != _|_ {
    JELLYFIN_PublishedServerUrl: {
        name:  "JELLYFIN_PublishedServerUrl"
        value: #config.publishedServerUrl
    }
}
```

The operator must compute and supply this value manually, even though it is fully derivable from the environment's route domain and the release identity. Three places conspire to make this awkward: (1) the field exists in `#config` even though it is not application configuration; (2) the operator has to do the computation that the catalog could do; (3) every module needing a similar URL re-invents the same `https://{release}.{routeDomain}` template.

A module that needs to point one component at another's Service via FQDN faces the same pattern: the FQDN is fully derivable from the release, the component, the namespace, and the cluster domain — none of which the catalog exposes to the module author at definition time.

## Why Existing Workarounds Fail

- **Stuff cluster facts into `#config` defaults.** Breaks the values-vs-context distinction. Operators end up re-supplying values the platform already knows. `#config`'s OpenAPI constraint clashes with CUE-computed values.
- **Push everything to transformer-internal `#TransformerContext`.** Works for transformer code that reads `#TransformerContext` directly, but module components cannot see it. Cross-module references (component A's service URL referenced inside component A's own env vars) become impossible to express at definition time.
- **Hardcode the conventions in every transformer.** Done today; produces drift the moment a module wants to deviate (e.g. via a `resourceName` override). A `metadata.resourceName` override only works if it propagates uniformly through every name and DNS variant — which requires a single computed `#ComponentNames` struct, not scattered transformer-side computation.
- **Live with operator-supplied values.** Forces the operator to know things they should not need to know (cluster domain, FQDN templates, hash content) and prevents cross-module composition (one module cannot reliably reference another's computed name).

A proper solution requires a structured runtime-context channel parallel to `#config`, owned by the catalog (for OPM-known fields), open at the top for platform-team extensions, computed from the layered Platform → Environment → Release inputs, and unified into the Module before its components are evaluated.
