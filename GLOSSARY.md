# OPM Library

The OPM kernel: a Go library that acquires Modules, Platforms and Catalogs, builds Module Instances and
renders them against a Platform. Every frontend (`cli`, `opm-controller`) embeds it. These terms win over
generic architecture words such as module, interface, seam or adapter.

## Language

### Kernel and tiers

**Kernel**:
The single struct, in `opm/kernel/`, through which a frontend calls every acquire, synthesize, validate and render verb. It holds a schema cache and no build context.
_Avoid_: Engine, runtime, core library, SDK

**Frontend**:
A program that embeds the Kernel and owns presentation, I/O policy and cluster actions, such as the `cli` or the `opm-controller`.
_Avoid_: Consumer app, client, runtime

**Kernel tier**:
The `opm/` packages every frontend must honour: `kernel`, `module`, `platform`, `catalog`, `schema` and `errors`.
_Avoid_: Core, public API layer

**Kubernetes tier**:
The `opm/k8s/` packages beside the Kernel that hold the Kubernetes decisions OPM makes. A Kubernetes frontend must honour them; the Kernel never imports them.
_Avoid_: Kubernetes adapter, k8s helpers

**Helper tier**:
The opt-in `opm/helper/` packages. A frontend may skip them, and nothing outside the tier may import them.
_Avoid_: Utils, convenience layer

**Verb**:
A public Kernel method that does one operation: acquire, synthesize, validate or render. It shares nothing with other verbs.
_Avoid_: Command, endpoint, API call

### Artifacts

**Artifact**:
One of the four kinds the Kernel holds: Module, Module Instance, Platform and Catalog. Each has the shape Metadata plus Package, plus a Source when the Kernel staged it.
_Avoid_: Resource, object, definition

**Module**:
The `#Module` artifact (`*module.Module`): a published or on-disk CUE module that declares components and a `#config` schema.
_Avoid_: Package, app, chart

**Module Instance**:
The `#ModuleInstance` artifact (`*module.Instance`): a Module with concrete values filled in, ready to render.
_Avoid_: Release, deployment, install

**Platform**:
The `#Platform` artifact (`*platform.Platform`): a CUE module that imports its Catalogs and is the only platform input of Render.
_Avoid_: Provider, environment, cluster

**Catalog**:
The `#Catalog` artifact (`*catalog.Catalog`): an acquired kind that the Kernel reads and derives from but never renders and never judges beyond its shape.
_Avoid_: Registry, library of resources

**Component**:
A unit inside a Module Instance, read through `Instance.Components`, that the matcher pairs with Transformers.
_Avoid_: Service, workload, part

**Source**:
The staged tree an artifact was loaded or built from, in on-disk mode or in overlay mode (files as bytes). It lets a later build import the artifact as a package.
_Avoid_: Files, checkout, bundle

**Overlay**:
The in-memory byte form of a Source tree, keyed under a root. A render writes no staging file because it uses overlays.
_Avoid_: Temp dir, virtual filesystem

### Rendering

**Render**:
The single verb that builds one CUE module importing a Module Instance and a Platform, then decodes verdicts and output. Matching and execution happen in CUE, not Go.
_Avoid_: Compile, materialize, build, deploy

**Transformer**:
A Catalog entry (`#ComponentTransformer`) that matches a Component and produces output objects from it. The Platform composes the Transformers of its Catalogs.
_Avoid_: Provider, plugin, generator

**Contract**:
A keyed demand that a Component makes and a Catalog's Transformers fulfil. The Platform's `#contracts` inventory records which keys are defined, required and provided.
_Avoid_: Capability, interface, requirement

**Provider-fulfilled contract**:
A Contract whose demand carries `fulfilment: "provider"`, so the Platform must carry exactly one provider for it.
_Avoid_: Singleton transformer, platform service

**Registry entry**:
One `#registry` line of a Platform that embeds a Catalog by import. Two majors of one Catalog are two entries.
_Avoid_: Catalog install, plugin registration

**Verdict row**:
Plain data (no `Error` method) that the render build decided, such as an unresolved demand, an unmatched component or a contract collision. Rows travel on the render diagnostics.
_Avoid_: Error, warning, finding

**Gate cause**:
A pointer-receiver error that the fail-closed gate builds from verdict rows. It carries the rows unchanged and wraps nothing.
_Avoid_: Render error, failure

**Fail-closed gate**:
The Kernel check after the render build that turns refusing verdicts into a `*RenderError`, in a fixed cause order.
_Avoid_: Validation step, guard

**Core floor**:
The oldest core release a Platform may pin and still render, set by the presence of `providedBy` in `#contracts`. An older pin is refused.
_Avoid_: Minimum version, compat check

**Catalog skew**:
The case where an instance Module requires a newer OPM-namespace Catalog build than the Platform carries. A `SkewPolicy` chooses to warn or refuse.
_Avoid_: Version drift, mismatch

**Compiled**:
One rendered object from Render, with instance, component and Transformer provenance and no platform-native fields.
_Avoid_: Manifest, output, resource

### Schema

**Core schema**:
The `opmodel.dev/core@v2` CUE module that defines the artifact shapes, resolved at runtime through `CUE_REGISTRY` and cached per Kernel. The default loader pins an exact release.
_Avoid_: Spec, API, types

**Schema cache**:
The per-Kernel `*schema.Cache` that memoizes the loaded core schema.
_Avoid_: Module cache, CUE cache

### Kubernetes tier

**Inventory**:
The record of the objects a Module Instance owns, as a list of entries. A frontend stores it; the library owns no wire shape for it.
_Avoid_: State, status, manifest list

**Stale set**:
The previously applied objects that the current render no longer produces; a prune removes them. It ignores component and API version when it compares identity.
_Avoid_: Orphans, diff

**Ownership verdict**:
A pure per-object answer from `opm/k8s/ownership` on whether apply or delete may proceed, with a reason both frontends word the same.
_Avoid_: Permission, policy check

**Adopt annotation**:
The annotation a user sets on a live object, with an instance UUID as value, to let that instance apply over an object it does not own.
_Avoid_: Takeover flag, claim label

**Deletion plan**:
The ordered list of inventory entries to delete, advanced one action per call with caller-owned state. Kernel plans, caller runs.
_Avoid_: Uninstall script, teardown workflow
