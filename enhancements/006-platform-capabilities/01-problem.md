# Problem Statement — Platform Capabilities

## Current State

Enhancement 004 defines `#ctx` as the runtime-context channel on `#Module`. After the 004 slim (004 D36), `#ctx` carries one layer: `#ctx.runtime` — release identity, module identity, and per-component computed names. Every field is derived from the release and the module alone; nothing in `#ctx.runtime` needs a platform or an environment to supply a value. 004's slim also removed the `#Environment` construct and the layered Platform → Environment hierarchy entirely. 006 does **not** reintroduce `#Environment`; per-platform variation is handled via plain CUE unification of `#Platform` values (see [02-design.md](02-design.md), and OQ6 in [04-decisions.md](04-decisions.md)).

Earlier 004 drafts carried a second layer, `#ctx.platform`: an unconstrained open struct (`{ ... }`) that platform teams populated with per-platform facts — default storage classes, backup backends, TLS issuers, Gateway listeners, app domains. They also kept `cluster.domain` and `route.domain` as `#ctx.runtime` fields fed by a layered Platform → Environment hierarchy. 004 D4 made the platform struct a flat open struct deliberately ("conventions emerge organically"); 004 D28 kept it unconstrained even after operational commodities started populating well-known sub-keys (`backup.backends`, `tls.issuers`, `routing.gateways`). 004 D36 removed all of it — the open struct, `cluster`, `route`, `#Environment`, and the layering — to be re-designed here.

Meanwhile the catalog already has a mature pattern for FQN-identified, schema-bearing primitives: `#Resource` (`apis/core/v1alpha2/resource.cue`) and `#Trait` (`trait.cue`). Each has a `metadata.fqn` computed from `modulePath`/`name`/`version`, an OpenAPIv3-compatible `spec`, and is published, indexed (`#Platform.#knownResources`, `#knownTraits`), and matched (`#Platform.#matchers` — a reverse index from primitive FQN to the transformers that render it, `platform.cue:97-134`). Collisions surface as CUE bottoms because the maps are FQN-keyed.

## Gap / Pain

### The platform layer has no contract

`#ctx.platform` was `{ ... }`. A module reading `#ctx.platform.appDomain` had no schema to program against — the field might be a string, a struct, or absent, and nothing caught the mismatch until a render-time failure or, worse, a silently malformed manifest. A platform team adding `#ctx.platform.backup.backends.s3` had no schema to conform to. The "interface" between a platform's provisions and a module's expectations was convention only, written down nowhere CUE could check.

### No discovery, no collision handling

Because the layer was a flat open struct, there was no way to ask "what does this platform provide?" — no computed view analogous to `#Platform.#knownResources`. Two platform teams choosing the same key (`#ctx.platform.domain`) with different shapes would unify into a CUE bottom with no diagnostic pointing at the collision; two teams choosing *different* keys for the same concept (`appDomain` vs `publicDomain`) would never even be noticed. 004 OQ3 flagged this and deferred it; 004 D36 resolves it by extracting the layer here.

### A second, weaker extension mechanism

The catalog already solved "FQN-identified, schema-bearing, collision-detecting, discoverable extension point" — that is exactly what `#Resource` and `#Trait` are. `#ctx.platform` was a *second* extension mechanism, weaker on every axis, invented for the one case (platform-supplied context) the existing pattern had simply never been pointed at.

### The module cannot state what it needs

A module that reads `#ctx.platform.appDomain` has a dependency on its target platform, but nowhere to declare it. The dependency is discovered the hard way — the module renders against a platform that does not supply `appDomain`, and the failure surfaces deep in evaluation, far from the declaration site. There is no equivalent of a transformer's `requiredResources`: a place where the module says "I require capability X" and gets a clear, early error if the target platform cannot satisfy it.

## Concrete Example

Take the Jellyfin module: it needs a public app domain to build a published URL, and it needs the cluster's default storage class for its media volume. Both are platform-supplied facts — neither is derivable from the release or the module, so neither has a home in 004's identity-only `#ctx.runtime`.

Before — the open-struct world 004 D36 removed:

```cue
// platform side — no schema, no identity, no discovery
#ctx: platform: {
    appDomain:           "apps.example.com"
    defaultStorageClass: "fast-ssd"
}

// module side — no declared dependency, no schema check
JELLYFIN_AppHost: {
    name:  "JELLYFIN_AppHost"
    value: "jellyfin.\(#ctx.platform.appDomain)"   // typo'd key? silent _|_ at the use site
}
```

Nothing records that the Jellyfin module *requires* `appDomain`. Nothing constrains `appDomain` to be a hostname-shaped string. Nothing lets a tool list the platform's provisions. If a second platform calls the field `publicDomain`, the Jellyfin module silently breaks against it — and the break surfaces at render time, not when the release is bound to the wrong platform.

## Why Existing Workarounds Fail

- **Keep the open struct, add conventions.** 004 D4's bet. Conventions are not checkable; the "interface" stays unwritten; discovery and collision handling never arrive. 004 OQ3 is the standing admission that this does not scale.
- **Define a monolithic `#StandardPlatformExtensions` schema.** 004 D28's rejected alternative. It couples every commodity's expected shape into one core-catalog schema; every platform must validate against the union; unrelated package versions must sync. Rejected for good reason — but the rejection left the problem unsolved.
- **Live with render-time failures.** The status quo for the open struct. Pushes every contract violation to the latest possible moment and gives the module author no way to declare the dependency up front.

The catalog already has the right shape for this — FQN-identified, independently versioned, schema-bearing primitives with a matcher. `#Capability` applies that shape to platform-supplied context: per-capability schemas (no monolith), a `#provides` / `#consumes` contract (declared dependencies, early errors), and FQN-keyed maps (structural collision handling and discovery).
