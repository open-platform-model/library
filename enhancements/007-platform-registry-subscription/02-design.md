# Design — Platform Registry Subscription

## Design Goals

- One platform can host multiple SemVer builds of the same catalog at once. The matcher pairs each consumer Module against transformers whose stamped FQN matches the consumer's primitive FQN, version-for-version.
- The version axis of the FQN is exact (SemVer). Two builds of the same primitive at different SemVers occupy distinct keys in `#composedTransformers` and never collide; same-SemVer rebuilds with divergent content surface as a unification failure rather than silent drift.
- Platform teams express version policy declaratively (`range` + `allow` + `deny`). Out-of-policy consumer pins surface as a structured "FQN not on platform" diagnostic, not a render-time error or a schema mismatch downstream.
- Catalog modules are plain CUE packages. Catalog authoring is decoupled from `#Module` (the consumer artifact). Authors write `#Resource` / `#Trait` / `#Blueprint` / `#ComponentTransformer` at the top of their package; the catalog's identity (`Version`, `ModulePath`) is a single per-package constant.
- Catalog identity is *burned in at publish time*: the OCI artifact contains concrete `metadata.version` on every primitive, sourced from the SemVer the publish task tagged. Kernel never mutates pulled CUE to inject a version; drift between OCI tag and catalog content is impossible by construction.
- The kernel's match step always unifies the consumer primitive value with the transformer's `requiredResources[FQN]` / `requiredTraits[FQN]` entry before pairing. FQN match is necessary but not sufficient.
- Missing FQNs are hard failures, not warnings. The kernel reports one structured diagnostic per missing FQN (component name, FQN, available alternatives at adjacent SemVers).
- Optional primitives stay optional. A consumer-declared trait on the optional axis of a matched transformer is allowed; a required-axis miss fails the match.

## Non-Goals

- `#ctx` / `#PlatformContext` shape — owned by [004](../004-module-context/). This design keeps `#Platform`'s shape compatible with 004's later addition.
- `#Claim` / `#ModuleTransformer` — owned by [005](../005-claims/). The path-based registry resolves Claim primitives the same way it resolves Resource/Trait/Transformer primitives once 005 lands.
- Platform capabilities — owned by [006](../006-platform-capabilities/).
- Renderer / `#transform` execution model — unchanged from 001/003.
- Replacing `cuelang.org/go/mod` with a custom OCI client. CUE's module proxy / OCI fetch is the substrate; this design wires the kernel into it via a `Registry` field that maps to `CUE_REGISTRY`.
- Signing and verification of catalog artifacts. Whatever guarantees `CUE_REGISTRY` provides are inherited.
- A discovery UX (`opm catalog list`) — separate enhancement.
- Backwards-compatibility for v1alpha2 platform fixtures. The repository's only consumer of the current shape is `library/modules/opm_platform/`, which this enhancement rewrites in lockstep with core.

## High-Level Approach

Three changes, designed together:

**1. `#registry` becomes path-keyed.** Each entry names a CUE module path, an enable flag, and a filter. A subscription stands for "every build of this catalog that the filter selects." The CUE-level `#Platform` value is a *spec*; the kernel materializes it.

**2. Catalog modules drop `#Module.#defines`.** A catalog is a plain CUE package that exports `#Resource`, `#Trait`, `#Blueprint`, `#ComponentTransformer` definitions at top level. A single root-package constant carries the catalog's identity:

```cue
Catalog: {
  Version:    string | *"0.0.0-dev"   // overwritten at publish
  ModulePath: "opmodel.dev/catalogs/opm"
}
```

Every primitive sources its `metadata.version` and `metadata.modulePath` from this constant. The publish task overwrites `Catalog.Version` with the concrete SemVer in a temp build dir before running `cue mod publish`. The OCI artifact is fully concrete; the kernel never injects a version at load time.

**3. Kernel grows a `Materialize` step.** Given a platform spec, `kernel.Materialize` walks `#registry`, resolves each filter against the OCI registry (via `cuelang.org/go/mod`), pulls every selected build into local cache, loads each package, and indexes top-level `#ComponentTransformer` values by their stamped FQN into a synthetic `#composedTransformers` map plus a `#matchers.{resources,traits}` reverse index. Match runs against the materialized platform, with FQN lookup followed by always-on unification before predicate evaluation.

The CUE side is small. The Go side carries the multi-version assembly. Authors who don't touch the kernel see (a) a tighter catalog authoring story and (b) a more honest platform definition.

## Schema / API Surface

The full schema deltas live in `04-schema.md` once that document lands. Headline shapes:

```cue
// core/types.cue
#FQNType: string & =~"^[a-z0-9.-]+(/[a-z0-9.-]+)*/[a-z0-9]([a-z0-9-]*[a-z0-9])?@\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
// MAJOR-only `@v[0-9]+$` retired. `#MajorVersionType` no longer used in primitive metadata.
```

```cue
// core/resource.cue, core/trait.cue, core/blueprint.cue, core/transformer.cue
metadata: {
  modulePath!: #ModulePathType
  version!:    #VersionType    // was #MajorVersionType
  name!:       #NameType
  fqn:         #FQNType & "\(modulePath)/\(name)@\(version)"
}
```

```cue
// core/module.cue — #defines REMOVED.
#Module: {
  metadata: { ... }            // unchanged
  #components: [Id=string]: #Component & { ... }
  #config: _
  debugValues: _
  // #defines? — REMOVED.
}
```

```cue
// core/platform.cue
#Platform: {
  kind: "Platform"
  metadata: { ... }
  type!: string

  #registry: [Id=#NameType]: #Subscription

  // #knownResources / #knownTraits — REMOVED.

  // #composedTransformers and #matchers become typed slots filled by
  // kernel.Materialize, not CUE comprehensions:
  #composedTransformers?: #TransformerMap
  #matchers?: {
    resources: [#FQNType]: [...#ComponentTransformer]
    traits:    [#FQNType]: [...#ComponentTransformer]
  }
}

#Subscription: {
  path!:   #ModulePathType   // e.g. "opmodel.dev/catalogs/opm"
  enable:  bool | *true
  filter?: #SubscriptionFilter
}

#SubscriptionFilter: {
  range?: string   // SemVer constraint, e.g. ">=1.0.0 <2.0.0"
  allow?: [...#VersionType]
  deny?:  [...#VersionType]
}
```

Go surface:

```go
// opm/kernel/kernel.go
type Kernel struct {
  Registry string        // default GHCR; threaded to CUE_REGISTRY for OCI ops
  cueCtx   *cue.Context
  // ... existing fields
}

// opm/platform/platform.go (or new opm/materialize package)
type MaterializedPlatform struct {
  Platform *Platform
  Package  cue.Value            // synthetic value with #composedTransformers + #matchers filled
}

func (k *Kernel) Materialize(p *Platform) (*MaterializedPlatform, error)
```

```go
// opm/compile/match.go — Match's signature accepts a MaterializedPlatform.
func Match(components cue.Value, plat *MaterializedPlatform, b api.Binding) (*MatchPlan, error)
// Inside: for each (component, FQN):
//   1. composed[FQN] → if absent, append MissingFQN error
//   2. unify(consumer primitive, tf.requiredResources[FQN]) — failure → UnifyError
//   3. predicate eval (labels, requiredResources, requiredTraits)
```

## Before / After

### Catalog source layout

```diff
  opmodel.dev/modules/opm/
    cue.mod/module.cue
-   module.cue                            # #Module with #defines.{resources,traits,transformers}
+   version.cue                           # Catalog: { Version, ModulePath }
    resources/container.cue
    traits/scaling.cue
    transformers/deployment_transformer.cue
```

```diff
  // resources/container.cue (excerpt)
  #ContainerResource: c.#Resource & {
    metadata: {
-     modulePath: "opmodel.dev/modules/opm/resources"
-     version:    "v1"
+     modulePath: "\(opm.Catalog.ModulePath)/resources"
+     version:    opm.Catalog.Version
      name:       "container"
    }
    spec: container: #ContainerSchema
  }
```

### Platform definition

```diff
  // library/modules/opm_platform/platform.cue
  package opm_platform

- import (
-   p          "opmodel.dev/core/v1alpha2@v1"
-   opm_package "opmodel.dev/modules/opm"
- )
+ import p "opmodel.dev/core"

  p.#Platform
  metadata: { name: "k8s-default", description: "Default Kubernetes Platform" }
  type: "kubernetes"

  #registry: {
    opm: {
-     #module: opm_package
-     enabled: true
+     path:   "opmodel.dev/modules/opm"
+     enable: true
+     filter: { range: ">=1.0.0 <2.0.0" }
    }
  }
```

### Concrete example from `01-problem.md`

App A pins `1.0.4`, App B pins `1.4.0`. Platform subscription filter: `range: ">=1.0.0 <2.0.0"`. Kernel `Materialize` pulls every published build in the range, say `1.0.4`, `1.1.0`, `1.2.0`, `1.4.0`. Synthetic `#composedTransformers` contains:

```
"opmodel.dev/modules/opm/transformers/deployment-transformer@1.0.4": { ... }
"opmodel.dev/modules/opm/transformers/deployment-transformer@1.1.0": { ... }
"opmodel.dev/modules/opm/transformers/deployment-transformer@1.2.0": { ... }
"opmodel.dev/modules/opm/transformers/deployment-transformer@1.4.0": { ... }
"opmodel.dev/modules/opm/resources/container@1.0.4":     { ... }   (carried inside each TF's requiredResources)
...
```

App A's release: components declare `container@1.0.4`. Matcher looks up `container@1.0.4` in `#matchers.resources` → finds the `deployment-transformer@1.0.4` candidate → unifies App A's container value with the 1.0.4 schema → pairs → renders against the 1.0.4 transformer body.

App B's release: same flow, different keys. Pairs against `deployment-transformer@1.4.0` and its 1.4.0 schema. Render uses 1.4.0 semantics.

App C pins `2.0.0`: filter excludes the major. Matcher errors `container@2.0.0` as missing; release fails at match time with a clear, structured diagnostic.

## Catalog Publish Flow

The catalog repo's Taskfile carries a `publish` task that:

1. Validates the `VERSION` argument against SemVer 2.0.
2. `rsync` the catalog source into `.build/catalog/`.
3. Overwrites `.build/catalog/version.cue` with concrete `Catalog: { Version: <VERSION>, ModulePath: "..." }`.
4. Runs `cue vet` over the build dir to confirm the FQN regex accepts the stamped values and the package is internally consistent.
5. Runs `cue mod publish v<VERSION>` from the build dir — OCI tag equals the stamped version by construction.
6. No source-tree mutation; failure mid-flow leaves the build dir for inspection.

Source-tree `version.cue` ships with `Version: string | *"0.0.0-dev"`. Dev-time `cue vet` from source passes; primitives get `…@0.0.0-dev` FQNs. The publish flow never edits source.

The CUE spike at the bottom of this design (see `03-decisions.md` D-NN reference) confirmed all of these properties end-to-end in `cue 0.16.1`.
