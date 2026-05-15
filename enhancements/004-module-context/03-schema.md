# Schema — `#ctx` Module Runtime Context

## Summary

Two new files in `core/v1alpha2` (flat package, no subdirectories):

- `core/v1alpha2/context.cue` — `#ModuleContext`, `#RuntimeContext`, `#ComponentNames`
- `core/v1alpha2/context_builder.cue` — `#ContextBuilder`

Two modifications:

- `core/v1alpha2/component.cue` — adds optional `metadata.resourceName` and a `#names: #ComponentNames` definition field
- `core/v1alpha2/module_release.cue` — invokes `#ContextBuilder` and unifies the result

`#Module` (005) gains `#ctx: #ModuleContext` as a definition field; that change is recorded in 005's schema, not here.

`#Environment`, `#PlatformContext`, and `#EnvironmentContext` were in earlier drafts of 004; they are extracted to enhancement 006 (Platform Capabilities). See D36.

## `#ModuleContext`

The top-level value of `#Module.#ctx`. One layer.

```cue
// apis/core/v1alpha2/context.cue
package v1alpha2

// #ModuleContext: the value injected into #Module.#ctx by #ModuleRelease.
// Module authors read it inside #components; never assign it directly.
//
// 004 defines the `runtime` layer. Enhancement 006 (Platform Capabilities)
// does not extend this — module-supplied capability values land in
// #Module.#consumes itself (006 D7/D8), not in a second #ctx layer.
#ModuleContext: {
    runtime: #RuntimeContext
}
```

## `#RuntimeContext`

Catalog-owned, schema-validated. All fields are derived from release identity, module identity, and the component set — no platform or environment inputs. All fields are required to be concrete when the module is rendered.

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

    // Per-component computed names. One entry per component key in #Module.#components.
    components: [compName=string]: #ComponentNames & {
        _releaseName: release.name
        _namespace:   release.namespace
        _compName:    compName
    }
}
```

## `#ComponentNames`

Per-component computed names. All four DNS variants cascade automatically from `resourceName`. When a `#Component` sets `metadata.resourceName`, `#ContextBuilder` passes the override here and CUE unification replaces the default; all `dns` variants propagate without further change.

`dns.fqdn` embeds a cluster domain. 004 self-defaults `_clusterDomain` to `"cluster.local"` — there is no override path in 004. Letting a platform or environment set a non-default cluster domain returns with `#Environment` in enhancement 006 (this supersedes the earlier-draft D8, which located the default at `#PlatformContext.runtime.cluster.domain`).

```cue
#ComponentNames: {
    _releaseName:   string
    _namespace:     string
    _compName:      string
    // Self-defaulted; no external input in 004. The override path returns
    // with #Environment / #Platform in enhancement 006.
    _clusterDomain: string | *"cluster.local"

    // Base Kubernetes resource name for all resources produced by this component.
    // Defaults to "{release}-{component}". Overridden when the component
    // sets metadata.resourceName — #ContextBuilder passes the override here.
    resourceName: string | *"\(_releaseName)-\(_compName)"

    dns: {
        local:      resourceName                                         // resourceName
        namespaced: "\(resourceName).\(_namespace)"                       // resourceName.namespace
        svc:        "\(resourceName).\(_namespace).svc"                   // resourceName.namespace.svc
        fqdn:       "\(resourceName).\(_namespace).svc.\(_clusterDomain)" // resourceName.namespace.svc.clusterDomain
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

// #ContextBuilder: assembles #ModuleContext from release identity, module
// identity, and the component map, plus the per-component #names injections.
// Invoked inline by #ModuleRelease via a let binding.
#ContextBuilder: {
    // Inputs
    #release:    { name: #NameType, namespace: string, uuid: #UUIDType }
    #module:     { name: #NameType, version: #VersionType, fqn: string, uuid: #UUIDType }
    #components: [string]: _   // component key map; values inspected for metadata.resourceName

    // Computed once, reused by both `ctx.runtime.components` and the
    // per-component `injections.<name>.#names` field below. Single source
    // of truth for resourceName / DNS variants. `_clusterDomain` is left
    // unset here — #ComponentNames self-defaults it to "cluster.local".
    let _componentNames = {
        for compName, comp in #components {
            (compName): {
                _releaseName: #release.name
                _namespace:   #release.namespace
                _compName:    compName
                if comp.metadata.resourceName != _|_ {
                    resourceName: comp.metadata.resourceName
                }
            }
        }
    }

    // Outputs.
    //
    // `ctx` is the module-level #ctx value.
    // `injections` is a per-component map that #ModuleRelease unifies
    // back into #module.#components so each component sees its own
    // #ComponentNames as #names.
    out: {
        ctx: #ModuleContext & {
            runtime: #RuntimeContext & {
                release:    #release
                module:     #module
                components: _componentNames
            }
        }

        injections: {
            for compName, _ in #components {
                (compName): #names: _componentNames[compName]
            }
        }
    }
}
```

The earlier-draft cluster-domain resolution (a conditional struct guarded by `_|_` checks against `#environment.#ctx.runtime.cluster` — D33) is gone: with no `#platform` / `#environment` inputs there is nothing to resolve. `#ComponentNames` self-defaults `_clusterDomain`.

## `#ModuleRelease` integration

`#config: values` must be unified into the module **before** the builder reads `#components`. Modules can build components dynamically from `#config` (mc_java_fleet's `for _srvName, _c in #config.servers { "server-\(_srvName)": ... }`); reading `#module.#components` against the bare `#Module` returns the static comprehension wrapper without those dynamic entries, the builder produces an empty `#ctx.runtime.components`, and the dynamic components never receive a `#names` injection. (See experiments/001-module-context Finding 2; the experiment predates the 004 slim — see D36.)

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
        #components: _withConfig.#components
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

`#ModuleRelease` gains a `#env: #Environment` field in enhancement 006, which re-adds the `#platform` / `#environment` inputs to `#ContextBuilder` for capability matching.

## Field documentation

### `#ModuleContext`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `runtime` | `#RuntimeContext` | yes | OPM-owned, schema-validated layer |

Enhancement 006 does not extend `#ModuleContext` — module-supplied capability values live in `#Module.#consumes`, not in `#ctx`. See 006 D7/D8.

### `#RuntimeContext`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `release.name` | `#NameType` | yes | Release name from `ModuleRelease.metadata.name` |
| `release.namespace` | `string` | yes | Release namespace from `ModuleRelease.metadata.namespace` |
| `release.uuid` | `#UUIDType` | yes | Release UUID |
| `module.name` | `#NameType` | yes | Module name |
| `module.version` | `#VersionType` | yes | Module SemVer version |
| `module.fqn` | `#ModuleFQNType` | yes | Module FQN |
| `module.uuid` | `#UUIDType` | yes | Module UUID |
| `components` | `[name]: #ComponentNames` | yes | One entry per `#Module.#components` key |

### `#ComponentNames`

| Field | Type | Description |
|-------|------|-------------|
| `resourceName` | `string` | Base K8s name; defaults to `"{release}-{component}"`; overridden by `#Component.metadata.resourceName` |
| `dns.local` | `string` | Same-namespace short form (= `resourceName`) |
| `dns.namespaced` | `string` | `resourceName.namespace` |
| `dns.svc` | `string` | `resourceName.namespace.svc` |
| `dns.fqdn` | `string` | `resourceName.namespace.svc.<clusterDomain>`; `_clusterDomain` self-defaults to `"cluster.local"` in 004 |

## File locations

### New files

| Path | Purpose |
|------|---------|
| `apis/core/v1alpha2/context.cue` | `#ModuleContext`, `#RuntimeContext`, `#ComponentNames` |
| `apis/core/v1alpha2/context_builder.cue` | `#ContextBuilder` |

### Modified files

| Path | Change |
|------|--------|
| `apis/core/v1alpha2/component.cue` | Add optional `metadata.resourceName: #NameType`; add `#names: #ComponentNames` definition field |
| `apis/core/v1alpha2/module_release.cue` | Invoke `#ContextBuilder`; unify `_builderOut.ctx` into `#module.#ctx` and `_builderOut.injections` into `#module.#components` alongside `values → #config` |
