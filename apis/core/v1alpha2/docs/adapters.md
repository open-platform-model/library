# OPM Adapters

Adapters are the translation layer between the OPM application model and a concrete target runtime. They describe what a target supports and how Components render into target-specific resources.

Adapters consume [Constructs](constructs.md) and [Primitives](primitives.md), but live outside the composition hierarchy — they do not appear inside a Module's `#components` or as part of a Module's portable definition. They are wired into the runtime (CLI, operator, compile pipeline) at deploy time.

See [Definition Types](definition-types.md) for the full taxonomy.

---

## ComponentTransformer

A **ComponentTransformer** converts an OPM Component into a single platform-specific resource (e.g., a Kubernetes Deployment, Service, or PersistentVolumeClaim). Each ComponentTransformer produces exactly one output resource — a Component that needs multiple platform resources is matched by multiple ComponentTransformers.

ComponentTransformers use a multi-dimensional matching system: required labels, required Resources, and required Traits must **all** be present on a Component for the transformer to match. The compile pipeline computes matches across all ComponentTransformers in the active registry, then invokes each matched transformer's `#transform` to produce its output.

`#ComponentTransformer` is the sole transformer primitive at the Component layer. Module-scope transformation (planned) is handled by a separate `#ModuleTransformer` adapter; both are members of `#TransformerMap`.

### What ComponentTransformer Infers

- "This converts a Component into **one platform-specific resource**"
- "This matches Components by **labels and definition FQNs**"
- "This produces a **single, concrete** runtime resource"

### ComponentTransformer Structure

```cue
#ComponentTransformer: {
    apiVersion: #ApiVersion
    kind:       "ComponentTransformer"

    metadata: {
        name!:        #NameType         // e.g., "deployment-transformer"
        modulePath!:  #ModulePathType   // e.g., "opmodel.dev/opm/transformers/kubernetes"
        version!:     #MajorVersionType
        fqn:          #FQNType          // computed
        description!: string
        labels?:      #LabelsAnnotationsType   // categorization, not matching
        annotations?: #LabelsAnnotationsType
    }

    // Matching criteria — ALL must be satisfied.
    requiredLabels?:    #LabelsAnnotationsType  // Component must carry these
    requiredResources:  [FQN=string]: #Resource // Component must include these
    requiredTraits:     [FQN=string]: #Trait    // Component must include these

    // Optional definitions — used if present, defaults applied otherwise.
    optionalLabels?:    #LabelsAnnotationsType
    optionalResources:  [FQN=string]: #Resource
    optionalTraits:     [FQN=string]: #Trait

    // The transform function — takes a Component, produces ONE platform resource.
    #transform: {
        #component: _                   // matched Component
        #context:   #TransformerContext // metadata, labels, annotations, runtime identity
        output:     {...}               // single provider-specific resource
    }
}

// TransformerMap is the union surface for all transformer adapters
// (today: #ComponentTransformer; future: #ModuleTransformer).
#TransformerMap: [#FQNType]: #ComponentTransformer
```

### TransformerContext

Every `#transform` body receives a `#TransformerContext` that carries:

- `#moduleReleaseMetadata` — name, namespace, fqn, version, uuid, labels, annotations of the release.
- `#componentMetadata` — name, labels, annotations of the matched Component.
- `#runtimeName` — identity of the runtime executing the transform (e.g., `"opm-cli"` for the CLI, `"opm-controller"` for the operator). Stamped onto every rendered resource as `app.kubernetes.io/managed-by`.
- Computed `labels` / `annotations` — final maps merged from module, component, and controller layers, ready to apply to the output.

Runtimes must populate `#runtimeName`; CUE evaluation fails if it is missing.

### Matching Flow

```text
Component
├── metadata.labels:   {"core.opmodel.dev/workload-type": "stateless", ...}
├── #resources:        {"...container@v1": ...}
└── #traits:           {"...scaling@v1": ..., "...expose@v1": ...}

                              ▼ pipeline checks each ComponentTransformer:

DeploymentTransformer (#ComponentTransformer)
├── requiredLabels:    {"core.opmodel.dev/workload-type": "stateless"}  ✓
├── requiredResources: {"...container@v1": ...}                          ✓
└── requiredTraits:    {}                                                ✓
                                                              → MATCH (emits Deployment)

ServiceTransformer (#ComponentTransformer)
├── requiredLabels:    {}                                                ✓
├── requiredResources: {}                                                ✓
└── requiredTraits:    {"...expose@v1": ...}                             ✓
                                                              → MATCH (emits Service)
```

A single Component may match multiple ComponentTransformers — each contributes a different runtime resource.

**CUE schema**: [`../transformer.cue`](../transformer.cue)

---

## Platform

> **Planned** — not present in `core/v1alpha2` yet.

A **Platform** models a deployment target as a single, composable construct. It carries the target's identity (`metadata`, `type`), platform-level context, and a registry of capabilities — registered Modules, the union of their Resources/Traits, the composed ComponentTransformer set, and a reverse matcher index used by the compile pipeline.

`#Platform` retires the older `#Provider` shape: instead of a static `#providers` list, the matcher consumes the Platform's computed `#composedTransformers` and `#matchers` projections directly. A companion `#PlatformMatch` walks a consumer Module's FQN demand against `#matchers` and surfaces `matched` / `unmatched` / `ambiguous` projections per deploy.

Platform integration lands incrementally via the kernel-redesign slices (see [`library/enhancements/001-kernel-redesign-around-platform/`](../../../../enhancements/001-kernel-redesign-around-platform/)) and the catalog enhancement (see [`catalog/enhancements/014-platform-construct/`](../../../../../catalog/enhancements/014-platform-construct/)). This document will be expanded once the construct ships in the kernel.
