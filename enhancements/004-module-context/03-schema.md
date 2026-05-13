# Schema — `#ctx` Module Runtime Context

## Summary

Three new files in `core/v1alpha2` (flat package, no subdirectories):

- `core/v1alpha2/context.cue` — `#ModuleContext`, `#PlatformContext`, `#EnvironmentContext`, `#RuntimeContext`, `#ComponentNames`
- `core/v1alpha2/environment.cue` — `#Environment`
- `core/v1alpha2/context_builder.cue` — `#ContextBuilder`

Two modifications:

- `core/v1alpha2/component.cue` — adds optional `metadata.resourceName`
- `core/v1alpha2/module_release.cue` — invokes `#ContextBuilder` and unifies the result

`#Module` (005) gains `#ctx: #ModuleContext` as a definition field; that change is recorded in 005's schema, not here.

## `#ModuleContext`

The top-level value of `#Module.#ctx`. Two named layers.

```cue
// apis/core/v1alpha2/context.cue
package v1alpha2

// #ModuleContext: the value injected into #Module.#ctx by #ModuleRelease.
// Module authors read it inside #components; never assign it directly.
#ModuleContext: {
    runtime:  #RuntimeContext
    platform: { ... }   // open struct, platform-team-owned
}
```

## `#RuntimeContext`

Catalog-owned, schema-validated. All fields are required to be concrete when the module is rendered.

```cue
#RuntimeContext: {
    // Release identity — mirrors ModuleRelease.metadata.
    release: {
        name!:      #NameType
        namespace!: string
        uuid!:      #UUIDType
    }

    // Module identity — mirrors Module.metadata.
    module: {
        name!:    #NameType
        version!: #VersionType
        fqn!:     #ModuleFQNType
        uuid!:    #UUIDType
    }

    // Cluster environment.
    cluster: {
        // DNS search domain for Kubernetes Services.
        // Defaults to "cluster.local"; overridable via #Platform.#ctx and #Environment.#ctx.
        domain: *"cluster.local" | string
    }

    // Ingress/route environment — absent when no route domain is configured.
    route?: {
        domain: string
    }

    // Per-component computed names. One entry per component key in #Module.#components.
    components: [compName=string]: #ComponentNames & {
        _releaseName:   release.name
        _namespace:     release.namespace
        _clusterDomain: cluster.domain
        _compName:      compName
    }
}
```

## `#ComponentNames`

Per-component computed names. All four DNS variants cascade automatically from `resourceName`. When a `#Component` sets `metadata.resourceName`, `#ContextBuilder` passes the override here and CUE unification replaces the default; all `dns` variants propagate without further change.

```cue
#ComponentNames: {
    _releaseName:   string
    _namespace:     string
    _clusterDomain: string
    _compName:      string

    // Base Kubernetes resource name for all resources produced by this component.
    // Defaults to "{release}-{component}". Overridden when the component
    // sets metadata.resourceName — #ContextBuilder passes the override here.
    resourceName: string | *"\(_releaseName)-\(_compName)"

    dns: {
        local:      resourceName                                                        // resourceName
        namespaced: "\(resourceName).\(_namespace)"                                      // resourceName.namespace
        svc:        "\(resourceName).\(_namespace).svc"                                  // resourceName.namespace.svc
        fqdn:       "\(resourceName).\(_namespace).svc.\(_clusterDomain)"                // resourceName.namespace.svc.clusterDomain
    }
}
```

DNS variant uses:

| Variant | Example | Use case |
| ------- | ------- | -------- |
| `local` | `jellyfin-jellyfin` | Same-namespace reference, short form |
| `namespaced` | `jellyfin-jellyfin.media` | Same-namespace, explicit namespace |
| `svc` | `jellyfin-jellyfin.media.svc` | Cross-namespace |
| `fqdn` | `jellyfin-jellyfin.media.svc.cluster.local` | Fully qualified, cross-cluster |

## `#PlatformContext`

The shape `#Platform.#ctx` resolves to. Sets cluster-level defaults plus platform-team extensions.

```cue
#PlatformContext: {
    runtime: {
        cluster: {
            domain: *"cluster.local" | string
        }
        route?: {
            domain: string
        }
    }
    // Platform-team extensions — open struct for platform-specific fields.
    // Not schema-validated by OPM; conventions left to platform teams.
    // Examples: defaultStorageClass, appDomain, backup.backends, tls.issuers, routing.gateways.
    platform: { ... }
}
```

## `#EnvironmentContext`

The shape `#Environment.#ctx` resolves to. Sets the namespace default and per-environment overrides.

```cue
#EnvironmentContext: {
    runtime: {
        release: {
            // Default namespace for releases in this environment.
            // Individual ModuleReleases can override via metadata.namespace.
            namespace: string
        }
        cluster?: {
            // Override platform's cluster domain if this environment
            // targets a cluster with a non-default domain.
            domain: string
        }
        route?: {
            // Environment-specific route domain.
            // Typically varies per environment: "dev.example.com" vs "example.com".
            domain: string
        }
    }
    // Inherits platform extensions; can add env-specific extensions.
    platform: { ... }
}
```

## `#Environment`

```cue
// apis/core/v1alpha2/environment.cue
package v1alpha2

// #Environment: the deployment-target binding. Carries:
//   - a reference to the target #Platform (capabilities + composed transformers)
//   - the environment's own #ctx contribution (Layer 2 of the hierarchy)
// #ModuleRelease.#env points at an #Environment value.
#Environment: {
    apiVersion: "opmodel.dev/core/v1alpha2"
    kind:       "Environment"

    metadata: {
        name!:        #NameType
        description?: string
    }

    // Target platform — determines available capabilities and providers.
    // Multiple environments can reference the same platform.
    #platform!: #Platform

    // Environment-level context contributions (Layer 2 in the hierarchy).
    // Overrides platform-level #ctx defaults for this environment.
    #ctx: #EnvironmentContext
}
```

`#Environment`'s sole structural concern in this enhancement is the `#ctx` contribution. The `#platform` reference is the binding that lets `#ContextBuilder` pull Layer 1 from the platform without `#ModuleRelease` needing to know about it directly.

### Field ownership

| Field | Required | Who sets it | Purpose |
| --- | --- | --- | --- |
| `metadata.name` | Yes | Environment author | Human-readable environment identifier |
| `#platform` | Yes | Environment author | Reference to a `#Platform` value |
| `#ctx.runtime.release.namespace` | Yes | Environment author | Default namespace for all releases in this environment |
| `#ctx.runtime.cluster.domain` | No | Environment author | Override platform's cluster domain (rare) |
| `#ctx.runtime.route.domain` | No | Environment author | Environment-specific ingress/route domain |
| `#ctx.platform` | No | Environment author | Additional platform extensions for this environment |

## `#Component.metadata.resourceName`

```cue
// apis/core/v1alpha2/component.cue
#Component: {
    ...
    metadata: {
        name!: #NameType

        // Override the Kubernetes resource base name for this component.
        // When absent, resourceName defaults to "{release}-{component}".
        // All DNS variants in #ctx.runtime.components cascade from this value.
        resourceName?: #NameType

        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }
    ...
}
```

`#ContextBuilder` reads this field per component when computing `#ctx.runtime.components`. CUE unification replaces the default in `#ComponentNames.resourceName`; all DNS variants update through the cascade.

## `#Component.#names`

```cue
// apis/core/v1alpha2/component.cue
#Component: {
    ...
    // Per-component computed names, injected by #ContextBuilder so a component
    // can read its own resourceName / DNS variants without typing its map key.
    // Equal to #ctx.runtime.components[<this component's key>].
    #names: #ComponentNames

    ...
}
```

`#names` is a CUE definition field (excluded from export, same as `#ctx`). Module authors read `#names.resourceName`, `#names.dns.svc`, `#names.dns.fqdn` from inside the component body without referring back to the map key. Cross-component reads still go through `#ctx.runtime.components[<otherKey>]` by design — pulling another component's name should require naming that component.

See D32 for rationale.

## `#ContextBuilder`

```cue
// apis/core/v1alpha2/context_builder.cue
package v1alpha2

// #ContextBuilder: assembles #ModuleContext from layered inputs and the
// per-component #names injections.
// Invoked inline by #ModuleRelease via a let binding.
#ContextBuilder: {
    // Inputs
    #release:     { name: #NameType, namespace: string, uuid: #UUIDType }
    #module:      { name: #NameType, version: #VersionType, fqn: string, uuid: #UUIDType }
    #components:  [string]: _   // component key map; values inspected for metadata.resourceName
    #platform:    #Platform
    #environment: #Environment

    // Resolve cluster domain: environment override beats platform default.
    // #EnvironmentContext.runtime.cluster is optional, so guard with a
    // conditional struct rather than a `*` default disjunction. The
    // disjunction form (`*#env...cluster.domain | #platform...cluster.domain`)
    // fails when the env omits cluster — CUE reports
    // `cannot reference optional field: cluster` instead of falling through
    // to the second disjunct. (See experiments/001-module-context Finding 1.)
    let _resolved = {
        domain: string
        if #environment.#ctx.runtime.cluster != _|_ {
            domain: #environment.#ctx.runtime.cluster.domain
        }
        if #environment.#ctx.runtime.cluster == _|_ {
            domain: #platform.#ctx.runtime.cluster.domain
        }
    }
    let _resolvedClusterDomain = _resolved.domain

    // Computed once, reused by both `ctx.runtime.components` and the
    // per-component `injections.<name>.#names` field below. Single source
    // of truth for resourceName / DNS variants.
    let _componentNames = {
        for compName, comp in #components {
            (compName): {
                _releaseName:   #release.name
                _namespace:     #release.namespace
                _clusterDomain: _resolvedClusterDomain
                _compName:      compName
                if comp.metadata.resourceName != _|_ {
                    resourceName: comp.metadata.resourceName
                }
            }
        }
    }

    // Outputs.
    //
    // `ctx` is the module-level #ctx value (existing surface).
    // `injections` is a per-component map that #ModuleRelease unifies
    // back into #module.#components so each component sees its own
    // #ComponentNames as #names.
    out: {
        ctx: #ModuleContext & {
            runtime: #RuntimeContext & {
                release: #release
                module:  #module
                cluster: domain: _resolvedClusterDomain

                if #environment.#ctx.runtime.route != _|_ {
                    route: #environment.#ctx.runtime.route
                }

                components: _componentNames
            }

            // Merge platform extensions from both layers.
            platform: #platform.#ctx.platform & #environment.#ctx.platform
        }

        injections: {
            for compName, _ in #components {
                (compName): #names: _componentNames[compName]
            }
        }
    }
}
```

## `#ModuleRelease` integration

`#config: values` must be unified into the module **before** the builder reads `#components`. Modules can build components dynamically from `#config` (mc_java_fleet's `for _srvName, _c in #config.servers { "server-\(_srvName)": ... }`); reading `#module.#components` against the bare `#Module` returns the static comprehension wrapper without those dynamic entries, the builder produces an empty `#ctx.runtime.components`, and the dynamic components never receive a `#names` injection. (See experiments/001-module-context Finding 2.)

```cue
// apis/core/v1alpha2/module_release.cue
#ModuleRelease: {
    apiVersion: "opmodel.dev/core/v1alpha2"
    kind:       "ModuleRelease"

    metadata: {
        name!:      #NameType
        namespace!: string
        uuid!:      #UUIDType
        ...
    }

    // Target environment — carries platform reference and Layer 2 #ctx.
    #env: #Environment

    #module: #Module
    values:  _

    // Step 1 — unify values into #config so dynamic #components materialise
    // (modules may build components via `for ... in #config.<…>` comprehensions).
    let _withConfig = #module & { #config: values }
    let _moduleMetadata = _withConfig.metadata

    // Step 2 — feed the post-config component map to the builder.
    let _builderOut = (#ContextBuilder & {
        #release: {
            name:      metadata.name
            namespace: metadata.namespace
            uuid:      metadata.uuid
        }
        #module: {
            name:    _moduleMetadata.name
            version: _moduleMetadata.version
            fqn:     _moduleMetadata.fqn
            uuid:    _moduleMetadata.uuid
        }
        #components:  _withConfig.#components
        #platform:    #env.#platform
        #environment: #env
    }).out

    // Step 3 — unify the builder's outputs back into the (config-resolved) module.
    let unifiedModule = _withConfig & {
        #ctx:        _builderOut.ctx
        #components: _builderOut.injections
    }

    components: {
        for name, comp in unifiedModule.#components { (name): comp }
    }
}
```

By the time `components` are extracted, `#ctx` is fully resolved and every component has its own `#names` field set to its `#ComponentNames` entry. The render pipeline iterates components without further CUE-side context wiring.

## Field documentation

### `#ModuleContext`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `runtime` | `#RuntimeContext` | yes | OPM-owned, schema-validated layer |
| `platform` | `{...}` | yes (open struct) | Platform-team-owned, no catalog constraints |

### `#RuntimeContext`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `release.name` | `#NameType` | yes | Release name from `ModuleRelease.metadata.name` |
| `release.namespace` | `string` | yes | Release namespace (env default + release override) |
| `release.uuid` | `#UUIDType` | yes | Release UUID |
| `module.name` | `#NameType` | yes | Module name |
| `module.version` | `#VersionType` | yes | Module SemVer version |
| `module.fqn` | `#ModuleFQNType` | yes | Module FQN |
| `module.uuid` | `#UUIDType` | yes | Module UUID |
| `cluster.domain` | `string` | yes (default `"cluster.local"`) | Cluster DNS search domain |
| `route.domain` | `string` | no | Ingress/route domain; absent when not configured |
| `components` | `[name]: #ComponentNames` | yes | One entry per `#Module.#components` key |

### `#ComponentNames`

| Field | Type | Description |
|-------|------|-------------|
| `resourceName` | `string` | Base K8s name; defaults to `"{release}-{component}"`; overridden by `#Component.metadata.resourceName` |
| `dns.local` | `string` | Same-namespace short form (= `resourceName`) |
| `dns.namespaced` | `string` | `resourceName.namespace` |
| `dns.svc` | `string` | `resourceName.namespace.svc` |
| `dns.fqdn` | `string` | `resourceName.namespace.svc.<clusterDomain>` |

### `#PlatformContext`, `#EnvironmentContext`

Documented above; both have a `runtime` sub-struct (subset of `#RuntimeContext` fields the layer is allowed to set) and an open `platform` extension struct.

### `#Environment`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `metadata.name` | `#NameType` | yes | Environment identifier |
| `metadata.description` | `string` | no | Human-readable summary |
| `#platform` | `#Platform` | yes | Target platform reference |
| `#ctx` | `#EnvironmentContext` | yes | Layer 2 context contributions |

## File locations

### New files

| Path | Purpose |
|------|---------|
| `apis/core/v1alpha2/context.cue` | `#ModuleContext`, `#PlatformContext`, `#EnvironmentContext`, `#RuntimeContext`, `#ComponentNames` |
| `apis/core/v1alpha2/environment.cue` | `#Environment` |
| `apis/core/v1alpha2/context_builder.cue` | `#ContextBuilder` |

### Modified files

| Path | Change |
|------|--------|
| `apis/core/v1alpha2/component.cue` | Add optional `metadata.resourceName: #NameType`; add `#names: #ComponentNames` definition field |
| `apis/core/v1alpha2/module_release.cue` | Invoke `#ContextBuilder`; unify `_builderOut.ctx` into `#module.#ctx` and `_builderOut.injections` into `#module.#components` alongside `values → #config` |
