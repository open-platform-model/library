# Design — `#ctx` Module Runtime Context

## Design Goals

- `#ctx` is a single runtime-context channel on `#Module`, parallel to `#config` but owned by the catalog — never by the operator.
- Single-layer shape: `runtime` (OPM-owned, schema-validated, fully populated when components evaluate). The platform-extension layer is enhancement 006 (see D36).
- Every field in `runtime` is derived from release identity, module identity, and the component set — no platform or environment inputs. Module authors never write to `#ctx`; they only read it.
- All per-component name variants (`resourceName`, `dns.local`, `dns.namespaced`, `dns.svc`, `dns.fqdn`) cascade from a single base, so a `metadata.resourceName` override propagates everywhere automatically.
- Each component sees its own `#ComponentNames` entry as `#names`, injected by `#ContextBuilder`. Components read `#names.dns.fqdn` from inside their own body without retyping their map key. Cross-component reads still go through `#ctx.runtime.components[<otherKey>]`.
- Computation lives in CUE via a `#ContextBuilder` helper. The catalog is independently testable as a CUE value; no Go-side wiring is required.
- `#config` and `#ctx` stay strictly separate. `#config` is what the operator supplies; `#ctx` is what the runtime computes.

## Non-Goals

- `#Platform` composition (`#registry`, computed views over registered Modules) — owned by 003.
- `#Module`'s top-level slot list (`#components`, `#claims`, `#defines`, etc.) — owned by 005.
- `#TransformerContext` migration / unification with `#ctx` — deferred. They overlap (release name, namespace, component name, label computation) but are computed independently for now; a follow-up enhancement will resolve the relationship.
- Bundle-level shared context (cross-module references — module A reads module B's computed names) — deferred.
- Content hashes for immutable ConfigMaps and Secrets. The hash slot was deliberately removed from this enhancement (see D31); transformers continue to compute and bake hashes on their own until a concrete need surfaces a module-readable hash channel.
- Runtime connection details (kubeContext, kubeConfig). These belong to a separate runtime-config mechanism and are not part of `#ctx`.
- The `#Environment` construct, `#PlatformContext` / `#EnvironmentContext`, the layered Platform → Environment hierarchy, the cluster-domain override, and the `route` domain. All of these were in earlier drafts of 004; they are extracted to enhancement 006 (Platform Capabilities). 006 does **not** reintroduce `#Environment` — per-platform variation uses CUE unification of `#Platform` values (006 OQ6). See D36.

## High-Level Approach

`#ctx` is a CUE definition field on `#Module`:

```cue
#Module: {
    ...
    #ctx: ctx.#ModuleContext   // computed and injected by #ModuleRelease
    ...
}
```

The value of `#ctx` has one layer:

```text
#ctx
└── runtime         OPM-owned, schema-validated, always fully populated
    ├── release     { name, namespace, uuid }
    ├── module      { name, version, fqn, uuid }
    └── components  [name]: #ComponentNames
```

`runtime` carries every fact the catalog can derive from the release and the module alone. The catalog guarantees these fields are present when components evaluate. Module authors write `#ctx.runtime.components.foo.dns.fqdn` knowing it will resolve.

The platform-extension layer — earlier drafts' `#ctx.platform` open struct — and the `#Environment` construct that fed the layered hierarchy are extracted to enhancement 006 (Platform Capabilities), which replaces the open struct with a typed, FQN-identified capability model (`#Capability`, `#Module.#consumes`, `#Platform.#provides`). 006 does **not** reintroduce `#Environment` — per-platform variation uses CUE unification of `#Platform` values (006 OQ6). Module bodies read matched capability values straight from `#consumes` (no separate `#ctx.capabilities` layer). See D36.

### Vocabulary stance

`#ctx.runtime` uses Kubernetes vocabulary as the canonical substrate. `release.namespace` and the `dns.{local,namespaced,svc,fqdn}` quartet are k8s-shaped fields treated as the universal contract every runtime presents. The choice is deliberate: k8s is the most expressive deploy substrate the project targets today; building a runtime-agnostic abstraction before a second concrete runtime exists tends to least-common-denominator outcomes (see D29). Non-Kubernetes runtimes (compose, nomad, …) interpret the same field names by mapping to local concepts — see "Non-Kubernetes Runtime Semantics" below. Cross-runtime portability for ecosystem-supplied resolutions (URLs, peer addresses, connection strings) flows through Claim `#status` (005 CL-D15), not through `runtime` field abstractions.

### Per-component computed names

For every component in `#Module.#components`, `#ContextBuilder` adds an entry to `#ctx.runtime.components` keyed by the component's name. The entry's shape is `#ComponentNames`:

```cue
#ComponentNames: {
    // Base Kubernetes resource name. Defaults to "{release}-{component}".
    // Overridden when the Component sets metadata.resourceName.
    resourceName: string

    dns: {
        local:      string   // resourceName
        namespaced: string   // resourceName.namespace
        svc:        string   // resourceName.namespace.svc
        fqdn:       string   // resourceName.namespace.svc.<clusterDomain>
    }
}
```

All four DNS forms cascade from `resourceName` — overriding the base name automatically propagates. `metadata.resourceName` on `#Component` is the single point of override; `#ContextBuilder` reads it and unifies it into the per-component entry. Authors never have to override the DNS forms separately.

`dns.fqdn` embeds a cluster domain. 004 self-defaults that domain to `"cluster.local"` inside `#ComponentNames` (a hidden `_clusterDomain` field); there is no override path in 004. Letting a platform or environment set a non-default cluster domain returns with `#Environment` in enhancement 006.

The same per-component entry is also injected back into the component itself as `#names`, so the component body can read `#names.resourceName` and `#names.dns.*` directly:

```cue
"router": {
    spec: container: env: {
        SELF_FQDN: { name: "SELF_FQDN", value: #names.dns.fqdn }
    }
}

for _srvName, _c in #config.servers {
    "server-\(_srvName)": {
        spec: container: env: {
            SELF: { name: "SELF", value: #names.dns.fqdn }
        }
    }
}
```

`#names` is exactly `#ctx.runtime.components[<this component's key>]`. The two surfaces are kept in lock-step by `#ContextBuilder` (same `_componentNames` let binding feeds both). See D32 for rationale.

### Where `#ctx` is computed and injected

`#ModuleRelease` invokes `#ContextBuilder` inline via `let` bindings, then unifies the result into the module along with `values`. **Order matters**: `#config: values` must be unified into the module *before* the builder reads `#components`, because modules can build components dynamically from `#config` (e.g. mc_java_fleet's `for _srvName, _c in #config.servers { "server-\(_srvName)": ... }`). Reading `#module.#components` against the bare `#Module` returns the static comprehension wrapper without those dynamic entries; the builder would then produce an empty `#ctx.runtime.components` and the dynamic components would never get a `#names` injection. (Validated experimentally — see [`experiments/001-module-context/`](../../experiments/001-module-context/) Finding 2; note the experiment predates the 004 slim — see D36.)

```cue
#ModuleRelease: {
    metadata: { name, namespace, uuid, ... }
    #module: module.#Module
    values:  _

    // Step 1 — unify values into #config so dynamic #components materialise.
    let _withConfig = #module & { #config: values }

    // Step 2 — feed the post-config component map to the builder.
    let _builderOut = (helpers.#ContextBuilder & {
        #release:    { name: metadata.name, namespace: metadata.namespace, uuid: metadata.uuid }
        #module:     { name: #moduleMetadata.name, version: ..., fqn: ..., uuid: ... }
        #components: _withConfig.#components
    }).out

    // Step 3 — unify the builder's outputs back into the module.
    let unifiedModule = _withConfig & {
        #ctx:        _builderOut.ctx
        #components: _builderOut.injections
    }

    components: { for name, comp in unifiedModule.#components { (name): comp } }
}
```

By the time `components` are extracted, `#ctx` is fully resolved. The render pipeline iterates components without further context wiring on the CUE side.

### Authoring-time lexical scope for `#ctx` and `#names`

`#ctx` and `#names` are declared on `#Module` and `#Component` respectively. Inside a module's own package files (the normal authoring path), references like `#ctx.runtime.components.router.dns.fqdn` and `#names.dns.fqdn` resolve via package-level lexical scope — the field exists on the enclosing definition and is in scope for every component body.

When inlining a `#Module & {...}` (or `#Component & {...}`) **literal** outside its own package — typically in tests, examples, or doc snippets — CUE's lexical scope does *not* reach into the type definition to find `#ctx` / `#names`. The literal must declare the field at its own level (`#ctx: _` on the module literal, `#names: _` on each inlined component) to bring the identifier into scope; the concrete value still arrives via `#ContextBuilder` unification at release time.

This is a CUE evaluation rule, not a schema bug — but it surprises authors who try to inline a module literal once and reuse the snippet. Real catalog modules, which live in their own packages, never need the workaround.

## How `#ctx` differs from `#config`

| | `#config` | `#ctx` |
|---|---|---|
| Who supplies values | Operator (via `ModuleRelease.values`) | Runtime (via `#ContextBuilder` from release + module + component identity) |
| Content | Application configuration | Deployment identity + per-component computed names |
| Schema constraint | OpenAPIv3-compatible (no CUE templating) | CUE-native (computed fields, let bindings) |
| Module author writes it | No (it's the schema; values come from operator) | No (computed by `#ContextBuilder`) |
| Module author reads it | Yes, via `#config.fieldName` | Yes, via `#ctx.runtime.<…>` |

Both fields are CUE definition fields (`#`-prefixed) so they are excluded from `cue export` output. Both are abstract at module-definition time and become concrete only after `#ModuleRelease` unification.

## Integration with `#Module`

- **005 (Module)** introduces `#ctx: ctx.#ModuleContext` as a definition field on `#Module`, parallel to `#config`. Module authors reference `#ctx.runtime` inside `#components`.
- **006 (Platform Capabilities)** extends `#ContextBuilder` with `#platform` + `#consumes` inputs and a matching step, and adds a kernel-populated `#platform: #Platform` field on `#ModuleRelease`. It does **not** modify `#ModuleContext` — `#ctx` stays identity-only; matched capability values land in `#Module.#consumes` itself, not in a `#ctx.capabilities` layer.
- **`#Component.metadata.resourceName`** (introduced here, used by `#ComponentNames`) is the single override point for resource names. All DNS variants cascade automatically.

## Information flow (visual)

```text
  #ModuleRelease
    metadata.name, namespace, uuid
    #module → #Module
    values → #config

       │
       ▼
  #ContextBuilder
    INPUTS:  #release, #module, #components
    COMPUTE: release identity + module identity + per-component #ComponentNames
    OUTPUT:  #ModuleContext  +  per-component #names injections

       │
       ▼
  unifiedModule = #module & {
                    #config:     values
                    #ctx:        <computed.ctx>
                    #components: <computed.injections>   // adds #names per component
                  }

       │
       ▼
  components: extracted with #ctx fully resolved and
              each component's own #names already set → render pipeline
```

There are no layered Platform / Environment inputs: 004's `#ContextBuilder` takes only the release identity, the module identity, and the component map. The layered hierarchy returns with `#Environment` in enhancement 006.

## Non-Kubernetes Runtime Semantics

`#ctx.runtime` uses Kubernetes vocabulary as the canonical substrate. Non-k8s runtimes (compose, nomad, future targets) interpret each field by mapping to local concepts. The same module body reads `#ctx.runtime.components.<x>.dns.svc` and gets a string; on Kubernetes the string resolves via kube-dns Service discovery, on Docker Compose the same string is a network alias on the compose service. The string doesn't need to be runtime-shaped to work — it just needs to be a stable identifier the runtime can route on.

### Field mapping

| `#ctx.runtime` field | Kubernetes meaning | Docker Compose mapping | HashiCorp Nomad mapping |
| --- | --- | --- | --- |
| `release.name` | Release identifier | Compose project name component | Nomad job name component |
| `release.namespace` | Kubernetes namespace | Compose project name (often `release.name`) | Nomad namespace |
| `release.uuid` | Identity label | Identity label | Identity label |
| `components.<x>.resourceName` | Kubernetes resource basename | Compose service name | Nomad task / group name |
| `components.<x>.dns.local` | Same-namespace short-form | Network alias (primary) | Service registration short-form |
| `components.<x>.dns.namespaced` | `name.namespace` short-form | Network alias (secondary) | `task.namespace` consul form |
| `components.<x>.dns.svc` | `name.namespace.svc` form | Network alias (tertiary) | `task.namespace.service.consul` form |
| `components.<x>.dns.fqdn` | Fully qualified `name.namespace.svc.<clusterDomain>` | Network alias (full form) | Fully qualified consul form |

Compose accepts arbitrary network aliases per service; the four `dns.*` forms can all be aliases on the same service. Nomad relies on Consul service registration for the same naming surface.

### Why k8s-canonical instead of a target split

An earlier design considered splitting `#ctx.runtime` into `runtime.universal` + `runtime.kubernetes` / `runtime.compose` / `runtime.nomad` subtrees. The split would have made portability honest at the cost of every module body needing target-specific reads. With k8s-canonical + claim-based portability via 005 CL-D15 (`#status`), the split is unnecessary: the runtime fields stay legible across targets, and *cross-runtime* resolutions (public URLs, peer addresses, DB connection strings) flow through the rich `#status` channel. See D30.

## Before / After

The 01-problem.md scenario: a module with a `router` and several `server-*` components, where each component needs its own in-cluster FQDN.

### Before — hand-built FQDN, drifts from the transformer

```cue
"server-alpha": {
    spec: container: env: SELF: {
        name:  "SELF"
        // hand-built; the Service transformer computes the same name independently.
        value: "release-server-alpha.media.svc.cluster.local"
    }
}
```

### After — read `#names.dns.fqdn`

```cue
"server-alpha": {
    spec: container: env: SELF: {
        name:  "SELF"
        value: #names.dns.fqdn
    }
}
```

A cross-component reference works the same way through `#ctx.runtime.components`:

```cue
"router": {
    spec: container: env: UPSTREAM: {
        name:  "UPSTREAM"
        value: #ctx.runtime.components["server-alpha"].dns.fqdn
    }
}
```

`#ComponentNames` is the single computed source. If `server-alpha` sets `metadata.resourceName`, the override cascades through every `dns.*` variant — and through both the transformer-side name and the module-side read — so they cannot drift.

## File Layout

```text
apis/core/v1alpha2/
├── context.cue                  // #ModuleContext, #RuntimeContext, #ComponentNames
└── context_builder.cue          // #ContextBuilder
```

Files live in the flat `v1alpha2` package; no subdirectories.

`#Module` (005) references `#ModuleContext` through its `#ctx` field type. Enhancement 006 adds `capability.cue` (`#Capability`), extends `context_builder.cue`, and modifies `module.cue` / `platform.cue` / `module_release.cue` — `context.cue` is untouched by 006.
