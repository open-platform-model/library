# OPM Constructs

Constructs are framework types that organize, compose, deploy, and verify the application model. They consume [Primitives](primitives.md) but don't define schemas for composition themselves.

See [Definition Types](definition-types.md) for the full taxonomy.

---

## Composition

### Component

A **Component** composes Primitives (Resources, Traits, Blueprints) into a single unit with a unified `spec`. Components are the building blocks of a Module — each Component represents at least one deployable piece of the application.

Resources, Traits, and Blueprints each define independent `spec` schemas (each namespaced under its own camelCase definition name). A Component merges all of them via CUE unification into a single flat `spec`. Conflicting field definitions between attached primitives are caught at definition time — not at deployment.

#### What Component Infers

- "This is one **deployable unit** within the application"
- "This **composes** independently authored primitives into a single spec"
- "This is what **ComponentTransformers** target to produce platform-specific output"

#### Component Structure

```cue
#Component: {
    apiVersion: #ApiVersion
    kind:       "Component"

    metadata: {
        name!:        #NameType
        labels?:      #LabelsAnnotationsType  // unified from attached primitives
        annotations?: #LabelsAnnotationsType  // unified from attached primitives
    }

    #resources:  #ResourceMap     // required — at least one
    #traits?:    #TraitMap        // optional behavioral modifiers
    #blueprints?: #BlueprintMap   // optional reusable patterns

    // _allFields merges every attached primitive's spec by unification.
    _allFields: { ... }

    // Closed unified spec. Must be made concrete by the consumer.
    spec: close({
        _allFields
    })
}
```

#### Key Relationships

```text
Resources ──┐
Traits ─────┤──unify──▶ Component.spec
Blueprints ─┘

Component.metadata.labels ◀── inherited from attached primitives
ComponentTransformer ── matches ──▶ Component (via labels + definition FQNs)
```

Labels flow upward: when a Resource or Trait declares labels (e.g., `"core.opmodel.dev/workload-type": "stateless"`), they propagate onto any Component that includes that primitive. ComponentTransformers then match Components by these inherited labels plus the FQNs of their attached primitives.

**CUE schema**: [`../component.cue`](../component.cue)

---

### Module

A **Module** is the top-level application definition. It groups Components and a config schema (`#config`) into a portable, versionable unit that a Module Author publishes.

Modules enforce a clear separation between the configuration **contract** (`#config` — the constraints consumers must satisfy) and concrete values supplied at deploy time by [`#ModuleRelease`](#modulerelease). The `debugValues` field carries optional, in-module example values used by build/validation tooling.

#### What Module Infers

- "This is the **complete application definition**"
- "This is **versioned and publishable** to a registry"
- "This defines the **configuration contract** consumers must satisfy"

#### Module Structure

```cue
#Module: {
    apiVersion: #ApiVersion
    kind:       "Module"

    metadata: {
        name!:        #NameType        // e.g., "my-app"
        modulePath:   #ModulePathType  // e.g., "example.com/modules"
        version:      #VersionType     // SemVer 2.0
        fqn:          #ModuleFQNType   // computed: "{modulePath}/{name}:{version}"
        uuid:         #UUIDType        // UUIDv5 of fqn under OPMNamespace
        description?: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    // The application's deployable pieces (developer-defined, required).
    #components: [Id=string]: #Component

    // Configuration contract — constraints (and optionally defaults).
    // MUST be OpenAPIv3 compliant — no CUE templating (no for/if).
    #config: _

    // Bundled example values used by build/validation tooling.
    debugValues: _
}
```

#### Key Relationships

```text
Module Author ──defines──▶ Module (#components, #config, debugValues)
End-user ──deploys──▶ ModuleRelease (concrete values into #config)
```

#### Module Example

```cue
basicModule: core.#Module & {
    metadata: {
        name:       "basic-module"
        modulePath: "example.com/modules"
        version:    "0.1.0"
    }

    #components: {
        web: components.basicComponent & {
            spec: {
                replicas: #config.web.replicas
                container: image: #config.web.image
            }
        }
    }

    #config: {
        web: {
            replicas: int
            image:    string
        }
    }

    debugValues: {
        web: {
            replicas: 1
            image:    "nginx:1.20.0"
        }
    }
}
```

**CUE schema**: [`../module.cue`](../module.cue)

---

## Deployment

### ModuleRelease

A **ModuleRelease** is the concrete deployment instance of a Module. It binds a Module to a target namespace with concrete values that satisfy the module's `#config` schema.

The separation between Module and ModuleRelease is fundamental to OPM's delivery flow. The Module Author publishes a portable definition; the End-user (or deployment system) creates a ModuleRelease that supplies environment-specific values. CUE ensures the provided values satisfy the `#config` contract at definition time.

Internally, the release unifies the referenced Module with `{#config: values}`, so the release's concrete values flow through `#config` into Component specs. This is what makes `release.components` contain fully-resolved Components rather than templates with unresolved config references.

`#AutoSecrets` walks the resolved config and discovers any `#Secret` instances. When secrets are present, an additional `opm-secrets` Component is automatically appended to the release's `components`.

#### What ModuleRelease Infers

- "This Module is **being deployed** to this namespace with these values"
- "All configuration is **concrete and validated** against the module's contract"
- "This is the **input to the compile pipeline**"

#### ModuleRelease Structure

```cue
#ModuleRelease: {
    apiVersion: #ApiVersion
    kind:       "ModuleRelease"

    metadata: {
        name!:        #NameType        // release name
        namespace!:   string           // target environment
        uuid:         #UUIDType        // UUIDv5 of (module.uuid:name:namespace)
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    #module!:        #Module
    #moduleMetadata: #module.metadata

    // Components resolved with concrete values; auto-secrets folded in.
    components: { ... }

    // Concrete values satisfying #module.#config.
    values: _
}
```

#### ModuleRelease Example

```cue
productionRelease: core.#ModuleRelease & {
    metadata: {
        name:      "basic-module-prod"
        namespace: "production"
    }
    #module: basicModule
    values: {
        web: {
            replicas: 3
            image:    "nginx:1.21.6"
        }
    }
}
```

**CUE schema**: [`../module_release.cue`](../module_release.cue)
