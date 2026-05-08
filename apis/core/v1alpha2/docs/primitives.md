# OPM Primitives

Primitives are schema contracts — independently authored building blocks that share the same shape: `metadata` (carrying `name`, `modulePath`, `version`, computed `fqn`) plus a `spec` (OpenAPIv3-compatible schema, namespaced under the definition's camelCase name). They are composed into [Constructs](constructs.md).

A Primitive:

- Defines a reusable `spec` schema
- Is independently authored and versioned
- Is composed into Components or other Primitives (Blueprints)
- Can be reused across multiple Modules

See [Definition Types](definition-types.md) for the full taxonomy.

---

## Resource

A **Resource** represents a fundamental, deployable entity that must exist in the runtime environment. Resources are the "nouns" of OPM — they answer the question "what is being deployed?" A Resource is standalone and has its own lifecycle; it can exist independently without requiring other definitions to make sense. Examples include Container (a running process), Volume (persistent storage), ConfigMap (configuration data), and Secret (sensitive configuration).

Resources are separate from Traits because they represent **existence** rather than behavior. A Component must have at least one Resource because without something that exists, there is nothing to modify (Trait) or compose (Blueprint).

### What Resource Infers

- "This thing **must exist** in the environment"
- "This is the **root** of something deployable"
- "Without this, there is nothing to modify or govern"

### When to Create a Resource

Ask yourself:

- Does this thing need to exist in the runtime for the application to function?
- Can it exist on its own, without depending on another primitive?
- Does it have its own lifecycle (create, update, delete)?

**Examples**: Container, Volume, ConfigMap, Secret, Database, Queue

### Resource Structure

```cue
#Resource: {
    apiVersion: #ApiVersion
    kind:       "Resource"

    metadata: {
        name!:        #NameType         // e.g., "container"
        modulePath!:  #ModulePathType   // e.g., "opmodel.dev/opm/resources/workload"
        version!:     #MajorVersionType // e.g., "v1"
        fqn:          #FQNType          // computed: "{modulePath}/{name}@{version}"
        description?: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    // OpenAPIv3-compatible schema. The exposed field name is the camelCase
    // form of metadata.name (e.g., name "container" -> spec.container).
    spec!: (strings.ToCamel(metadata.#definitionName)): _
}
```

### Resource Example

```cue
#ContainerResource: core.#Resource & {
    metadata: {
        name:        "container"
        modulePath:  "opmodel.dev/opm/resources/workload"
        version:     "v1"
        description: "A container definition for workloads"
        labels: {
            "core.opmodel.dev/category":      "workload"
            "core.opmodel.dev/workload-type": "stateless"
        }
    }

    spec: container: {
        image!:           string
        command?:         [...string]
        args?:            [...string]
        env?:             [...{name: string, value: string}]
        ports?:           [...{containerPort: int, protocol?: string}]
        imagePullPolicy?: "Always" | "IfNotPresent" | "Never"
    }
}
```

The Resource's labels propagate up to any Component that includes it — ComponentTransformers use these inherited labels for matching.

**CUE schema**: [`../resource.cue`](../resource.cue)

---

## Trait

A **Trait** represents a behavioral characteristic or configuration modifier that attaches to a Resource. Traits are the "adjectives" of OPM — they answer the question "how does this thing behave?" or "how is this thing configured?" A Trait cannot exist in isolation; it requires a Resource to make sense. Examples include Scaling (instance count and autoscaling), HealthCheck (liveness probing), Expose (network reachability), and RestartPolicy (failure response).

Traits describe **modification** rather than existence. They express **preference** — a Trait says "I want this behavior."

### What Trait Infers

- "This **modifies** how something operates"
- "This **requires** a Resource to make sense"
- "This describes **behavior** or **configuration**"

### When to Create a Trait

Ask yourself:

- Does this modify how something else operates?
- Is this a preference/configuration rather than a mandate?
- Can it only make sense when attached to a Resource?

**Examples**: Scaling, HealthCheck, Expose, RestartPolicy, UpdateStrategy

### Trait Structure

```cue
#Trait: {
    apiVersion: #ApiVersion
    kind:       "Trait"

    metadata: {
        name!:        #NameType
        modulePath!:  #ModulePathType
        version!:     #MajorVersionType
        fqn:          #FQNType
        description?: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    // Resources this Trait can be applied to (full references).
    appliesTo!: [...#Resource]

    spec!: (strings.ToCamel(metadata.#definitionName)): _
}
```

### Key Difference from Resource

Traits carry an `appliesTo` list that declares which Resources they can modify:

```text
Trait → appliesTo → Resource
```

### Trait Example

```cue
#ScalingTrait: core.#Trait & {
    metadata: {
        name:        "scaling"
        modulePath:  "opmodel.dev/opm/traits/workload"
        version:     "v1"
        description: "Scaling behavior for a workload"
    }

    appliesTo: [#ContainerResource]

    spec: scaling: {
        count: int & >=1 & <=1000 | *1
        auto?: #AutoscalingSpec
    }
}
```

**CUE schema**: [`../trait.cue`](../trait.cue)

---

## Blueprint

A **Blueprint** represents a reusable pattern that composes Resources and Traits into a higher-level abstraction. Blueprints are the "templates" of OPM — they answer the question "what is the standardized pattern?" A Blueprint simplifies complex configurations by grouping related definitions under a single schema, hiding the complexity of individual primitives from the end user.

Blueprints are used to define standardized workload types (e.g., "StatelessWorkload", "SimpleDatabase") composed from specific Resources (Container, Volume) and Traits (Scaling, Expose).

### What Blueprint Infers

- "This is a **composition** of Resources and Traits"
- "This is a **reusable pattern**"
- "This **simplifies** configuration"

### When to Create a Blueprint

Ask yourself:

- Do you find yourself repeatedly defining the same set of Resources and Traits?
- Do you want to standardize a specific architectural pattern?
- Do you want to expose a simplified schema while managing underlying complexity?

**Examples**: StatelessWorkload, StatefulWorkload, CronJob, SimpleDatabase

### Blueprint Structure

```cue
#Blueprint: {
    apiVersion: #ApiVersion
    kind:       "Blueprint"

    metadata: {
        name!:        #NameType
        modulePath!:  #ModulePathType
        version!:     #MajorVersionType
        fqn:          #FQNType
        description?: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    composedResources!: [...#Resource]   // required composition
    composedTraits?:    [...#Trait]      // optional composition

    spec!: (strings.ToCamel(metadata.#definitionName)): _
}
```

### Blueprint Example

```cue
#StatelessWorkloadBlueprint: core.#Blueprint & {
    metadata: {
        name:        "stateless-workload"
        modulePath:  "opmodel.dev/opm/blueprints/workload"
        version:     "v1"
        description: "A stateless workload pattern"
    }

    composedResources: [#ContainerResource]
    composedTraits:    [#ScalingTrait, #ExposeTrait]

    spec: statelessWorkload: {
        image!:  string
        scaling: { count: int | *1 }
        port?:   int
    }
}
```

**CUE schema**: [`../blueprint.cue`](../blueprint.cue)
