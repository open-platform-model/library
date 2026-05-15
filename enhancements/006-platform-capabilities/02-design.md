# Design — Platform Capabilities

## Design Goals

- `#Capability` is an FQN-identified, schema-bearing construct — the `#Resource` / `#Trait` pattern applied to platform-supplied context rather than render targets.
- A module declares what it `#consumes`; a platform declares what it `#provides`; the two are matched by FQN.
- `#Module.#consumes` is **both** declaration and resolved read surface — matched provider specs are unified back into `#consumes`. Mirrors `#Component.#resources` precedent (`component.cue:23-55`).
- The match is a checkable contract: a *required* capability with no provider is a release-time error; a provided value that violates the capability's schema is a CUE bottom.
- Each capability is independently versioned and packaged — there is no monolithic central schema (this is what resolves 004 D28's tension).
- `#ModuleRelease.#platform: #Platform` is **kernel-populated** — end-users authoring a release never write `#platform`; the runtime fills it at apply time. Same precedent as `#TransformerContext.#runtimeName!` (`transformer.cue:121`).
- Matching is CUE-side, inside `#ContextBuilder` — extends 004's slimmed builder with `#platform` and `#consumes` inputs. No Go changes for the core resolution.

## Non-Goals

- `#ctx.runtime` — release/module identity, per-component DNS names. Owned by 004; identity-only after the slim (004 D36).
- A typed `#ctx.capabilities` second layer on `#ModuleContext`. Not introduced — `#consumes` is both declaration and resolved read surface, mirroring `#Component.#resources`. `#ModuleContext` stays single-layer (`{ runtime: #RuntimeContext }`).
- The `#Environment` construct. 004 D36 removed it; 006 does not reintroduce it. Per-platform variation is handled via plain CUE unification of `#Platform` values; formalizing that pattern is OQ6.
- Reintroducing `route` and the cluster-domain override as shipped `#Capability` definitions — deferred (OQ1).
- Publishing capabilities through `#Module.#defines` or surfacing them in `#Platform.#registry` views. v1 capabilities are imported directly (OQ2).
- Component-scoped capabilities — the "Trait flavour", a capability with an `appliesTo` attaching at component rather than module granularity (OQ3).
- Transformers consuming capabilities, and the relationship to `#TransformerContext` (004 OQ1 / D15) (OQ4).
- Bundle-level provision — a `#Module` providing capabilities for sibling modules in a bundle (004 OQ2 / D20; here OQ5).

## High-Level Approach

### `#Capability` — a sibling construct, not a reuse

`#Capability` is a new construct in `apis/core/v1alpha2/capability.cue`, sibling to `#Resource` and `#Trait`. It is **not** a reuse of `#Resource`: a `#Resource` is a render *output* — `#Platform.#matchers` pairs it with transformers that emit Kubernetes. A `#Capability` is a render *input* — the platform supplies it, the module reads it, nothing renders it. Keying context into `#Resource` would put it into the transformer matcher, which would then try to render it. The *pattern* transfers — FQN identity, OpenAPIv3 `spec`, FQN-keyed map unification — the *type* does not.

A `#Capability` definition is the **interface**: an FQN plus a `spec` schema. `route@v1` is `{ domain: string }`; `storage-class@v1` is `{ name: string, ... }`. The definition lives in a package and is imported by both the provider side and the consumer side, so both program against one schema.

### `#Platform` provides; `#Module` consumes

`#Platform.#provides` is an FQN-keyed map of `#Capability` instances with *concrete* specs — the platform's actual provisions. It is the single source of capability values; per-platform variation is handled via CUE unification of `#Platform` values, not via a second provider tier (see OQ6 for the pattern).

`#Module.#consumes` has two FQN-keyed sub-maps, `required` and `optional`, mirroring the transformer's `requiredResources` / `optionalResources` split (`transformer.cue:55-65`). The module imports the same `#Capability` definitions the provider uses and keys them by FQN. `required` entries must be matched or the release fails; `optional` entries are matched when a provider exists and absent otherwise.

### Kernel-populated `#platform` on `#ModuleRelease`

`#ModuleRelease` gains a single `#`-prefixed field, `#platform: #Platform`. End-users authoring a release never write `#platform`; the kernel/CLI/operator unifies `#platform: <chosen>` into the release at apply time. The release artifact stays portable — release files contain no `#platform` line; the binding is a runtime decision, not an authored one.

Direct precedent: `#TransformerContext.#runtimeName!` (`transformer.cue:121`) is exactly this pattern — a required field the runtime fills, end-user never sees it ("Mandatory — CUE evaluation fails if the runtime forgets to fill this"). The kernel-populated field is consistent with how the catalog already wires runtime-supplied state into CUE-side evaluation.

The `#`-prefix excludes `#platform` from `cue export` output; `#Platform` values do not leak into rendered manifests.

### Matching — `#ContextBuilder` gains a step

004's slimmed `#ContextBuilder` takes three inputs (`#release`, `#module`, `#components`) and produces `out.ctx.runtime` plus `out.injections` (per-component `#names`). 006 adds two inputs — `#platform` and `#consumes` — and one output — `out.consumes`, the matched capability map. `#ModuleRelease` invokes the builder inline (one call) and unifies all three outputs back into the module:

- `out.ctx` → `#module.#ctx`
- `out.injections` → `#module.#components` (per-component `#names`)
- `out.consumes` → `#module.#consumes` (matched provider specs unified into the consumed capability entries)

For each FQN in `#consumes.required`, the matcher unifies `#platform.#provides[fqn]` into the consumed `#Capability`. Three outcomes:

- **Provider exists, schema matches**: the capability's `spec` becomes concrete; module reads it via `#consumes.required[fqn].spec.X`.
- **Provider missing**: the consumed `#Capability`'s `spec!` stays incomplete. `cue vet -c` reports `out.consumes.required.<fqn>.spec` as an incomplete value — a release-time error that names the missing capability.
- **Provider exists, schema mismatch**: `provided & consumed` is a CUE bottom at that FQN — a release-time error.

For `#consumes.optional`, the outer comprehension `if` drops the entry entirely when no provider exists, so `#consumes.optional[fqn]` is absent; the module guards with `if #consumes.optional[<fqn>] != _|_`.

### Read surface — `#consumes` itself

Module bodies read straight from `#consumes`:

```cue
// somewhere in #components
JELLYFIN_AppHost: {
    name:  "JELLYFIN_AppHost"
    value: "jellyfin.\(#consumes.required["opmodel.dev/opm/capabilities/routing/route@v1"].spec.domain)"
}
```

`#consumes` is BOTH the declaration (FQN + schema, authored on `#Module`) and the resolved read surface (matched specs unified in by the builder). Same pattern as `#Component.#resources` (`component.cue:23-55`) where a component declares `#resources: { Container: workload.#Container & {spec: ...} }` and reads `#resources.Container.spec.…` from the same place. No separate `#ctx.capabilities` mirror is needed.

`#ModuleContext` (defined in 004) stays single-layer (`{ runtime: #RuntimeContext }`); 006 does not extend it.

## Schema / API Surface

Full CUE in [03-schema.md](03-schema.md). The key shapes:

```cue
// New construct — apis/core/v1alpha2/capability.cue
#Capability: {
    apiVersion: #ApiVersion
    kind:       "Capability"
    metadata: {
        name!:       #NameType
        modulePath!: #ModulePathType
        version!:    #MajorVersionType
        fqn:         #FQNType & "\(modulePath)/\(name)@\(version)"
        description?: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }
    // The interface. OpenAPIv3-compatible. Unwrapped — capabilities are
    // addressed by FQN, never flattened into a component spec (D2).
    spec!: _
}

// #Module gains (touches 005):
#consumes: {
    required: [FQN=#FQNType]: #Capability & {metadata: fqn: FQN}
    optional: [FQN=#FQNType]: #Capability & {metadata: fqn: FQN}
}

// #Platform gains (touches 003):
#provides: [FQN=#FQNType]: #Capability & {metadata: fqn: FQN}

// #ModuleRelease gains (touches 004) — kernel-populated, not authored:
#platform: #Platform   // runtime fills via FillPath at apply time
```

## Before / After

### Before — the open-struct world (004 D36 removed it)

```cue
// platform side — no schema, no identity, no discovery
#ctx: platform: appDomain: "apps.example.com"

// module side — no declared dependency, no schema check
JELLYFIN_AppHost: {
    name:  "JELLYFIN_AppHost"
    value: "jellyfin.\(#ctx.platform.appDomain)"
}
```

### After — `#Capability` + `#provides` + `#consumes`

```cue
// capability definition — the interface, in a shared package
// import routing "opmodel.dev/opm/capabilities/routing"
routing.#Route: #Capability & {
    metadata: {name: "route", modulePath: "opmodel.dev/opm/capabilities/routing", version: "v1"}
    spec: domain: string
}

// platform side — concrete, schema-checked, FQN-identified
#Platform & {
    #provides: "opmodel.dev/opm/capabilities/routing/route@v1": routing.#Route & {
        spec: domain: "apps.example.com"
    }
}

// module side — the dependency is declared
#Module & {
    #consumes: required: "opmodel.dev/opm/capabilities/routing/route@v1": routing.#Route
}

// component body — read the resolved, schema-validated value from #consumes itself
JELLYFIN_AppHost: {
    name:  "JELLYFIN_AppHost"
    value: "jellyfin.\(#consumes.required["opmodel.dev/opm/capabilities/routing/route@v1"].spec.domain)"
}
```

If the platform does not provide `route@v1`, the required `#consumes` entry leaves `spec` incomplete and `cue vet -c` fails at release time, naming the FQN. If the platform provides it with the wrong shape (`spec: domain: 42`), the provided value unified against `routing.#Route` is a CUE bottom. Both contract violations surface early, at the release boundary, not deep in render.

The FQN string key is verbose at the read site — the same friction `#Module.#defines` already has. Import aliasing handles the *type* (`routing.#Route`); a `let` binding handles the *read* (`let _route = #consumes.required["…route@v1"]`). A terser read surface is an open ergonomics question, not a v1 blocker.

## File Layout

```text
apis/core/v1alpha2/
├── capability.cue          // NEW — #Capability, #CapabilityMap
├── context_builder.cue     // MOD — #ContextBuilder gains #platform + #consumes inputs;
│                           //       adds out.consumes (matched specs unified back)
├── platform.cue            // MOD — #Platform gains #provides
├── module.cue              // MOD — #Module gains #consumes
└── module_release.cue      // MOD — #ModuleRelease gains kernel-populated #platform: #Platform;
                            //       passes #platform / #consumes to #ContextBuilder;
                            //       unifies out.consumes into #module.#consumes
```

`apis/core/v1alpha2/context.cue` is **untouched** — `#ModuleContext` stays single-layer; no `#ctx.capabilities`, no `#CapabilitySet`.

## Integration Points

- **004 (Module Context)** owns identity-only `#ctx.runtime`, `#ComponentNames`, the three-input `#ContextBuilder` core, and the three-step `#ModuleRelease` flow. 006 extends `#ContextBuilder` with two new inputs and one new output, and adds the kernel-populated `#platform` field on `#ModuleRelease` (which feeds the builder). 004 D36 records the slim that made room.
- **003 (Platform)** owns `#Platform`. 006 adds a `#provides` map; it does not touch `#registry`, `#knownResources`, or `#matchers`.
- **005 (Claims / Module shape)** owns `#Module`'s slot list. 006 adds `#consumes` alongside `#config`, `#components`, `#defines`. The relationship between a `#Capability` and a Claim `#status` (the cross-runtime resolution surface, 005 CL-D15) is noted but not designed here.
