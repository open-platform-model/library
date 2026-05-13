# Examples — `#Module` Flat Shape with `#Claim` Primitive and `#defines` Channel

Seven worked examples illustrate the new shape:

1. **Application Module** — a web app with a component-level claim
2. **Application Module with module-level claim** — a multi-component app with a platform-relationship claim
3. **Operator Module** — a Postgres operator that fulfils the well-known `ManagedDatabase` claim by shipping a transformer with `requiredClaims` in `#defines.transformers`
4. **Specialty vendor + consumer** — a vendor publishes `VectorIndex`; a consumer module uses it
5. **Claim-only Module** — a vendor publishes a `#Claim` type via `#defines.claims` without fulfilling it (fulfilment ships in a separate operator Module)
6. **Publication-only Module** — the OPM core catalog packaged as a single Module with `#defines` only
7. **Operational commodity (backup)** — the dual-scope pattern: a Module-level `#Claim` for orchestration plus a per-component `#Trait` for local data; a `#ModuleTransformer` with `requiredClaims` and a `requiresComponents.traits` pre-fire gate, rendering K8up CRs. TLS certificates and Gateway-API routing follow the same shape (summary at the end of Example 7).

## Example 1 — Application Module with component-level claim

A stateless web application that needs a Postgres database. The `#Claim` lives inside the component because the dependency is a per-component data-plane need.

```cue
package web_app

import (
    core "opmodel.dev/core/v1alpha2"
    container "opmodel.dev/opm/v1alpha2/resources/workload"
    data "opmodel.dev/opm/v1alpha2/claims/data"
)

webApp: core.#Module & {
    metadata: {
        modulePath: "example.com/apps"
        name:       "web-app"
        version:    "0.1.0"
    }

    #config: {
        replicas: int | *2
        image:    string | *"example.com/web:1.0"
    }

    #components: {
        web: {
            #resources: {
                container: container.#ContainerResource & {
                    #spec: container: {
                        image: #config.image

                        // Reads from #status — populated by the fulfilling
                        // transformer at deploy time (see 03-schema.md
                        // "Resolution channel — #status").
                        env: [
                            {name: "DATABASE_HOST", value: #claims.db.#status.host},
                            {name: "DATABASE_PORT", value: "\(#claims.db.#status.port)"},
                            {
                                name: "DATABASE_PASSWORD"
                                valueFrom: secretKeyRef: {
                                    name: #claims.db.#status.passwordRef.secretName
                                    key:  #claims.db.#status.passwordRef.key
                                }
                            },
                        ]
                    }
                }
            }
            #claims: {
                db: data.#ManagedDatabaseClaim & {
                    #spec: managedDatabase: {
                        engine:  "postgres"
                        version: "16"
                        sizeGB:  20
                    }
                }
            }
        }
    }
}
```

## Example 2 — Application Module with module-level claim

A multi-component app that needs a public DNS name and a workload identity shared across components. These are platform-relationship needs, not per-component data-plane needs, so they live at module level.

```cue
package payments_app

import (
    core "opmodel.dev/core/v1alpha2"
    platform "opmodel.dev/opm/v1alpha2/claims/platform"
)

paymentsApp: core.#Module & {
    metadata: {
        modulePath: "example.com/apps"
        name:       "payments"
        version:    "1.2.0"
    }

    #components: {
        api:    {...}
        worker: {...}
    }

    #claims: {
        // Module-level: DNS hostname for the entire module
        hostname: platform.#HostnameClaim & {
            #spec: hostname: { fqdn: "payments.example.com" }
        }
        // Module-level: shared workload identity for all components
        identity: platform.#WorkloadIdentityClaim & {
            #spec: workloadIdentity: { name: "payments-prod", roles: ["pubsub-publisher"] }
        }
    }
}
```

## Example 3 — Operator Module that fulfils `ManagedDatabase`

A vendor ships a Postgres operator. The Module deploys the controller and CRDs as components, declares an install lifecycle, and publishes the transformer that fulfils the well-known `ManagedDatabase` commodity contract by mapping any `ManagedDatabaseClaim` request to a Postgres CRD instance. The `ManagedDatabaseClaim` is component-level (per-component data-plane need), so the transformer is a `#ComponentTransformer` and its `requiredClaims` field is the registration.

```cue
package postgres_operator

import (
    core "opmodel.dev/core/v1alpha2"
    transformer "opmodel.dev/core/v1alpha2:transformer"
    container "opmodel.dev/opm/v1alpha2/resources/workload"
    crd "opmodel.dev/opm/v1alpha2/resources/extension"
    rbac "opmodel.dev/opm/v1alpha2/resources/security"
    data "opmodel.dev/opm/v1alpha2/claims/data"
    pgcrd "vendor.com/postgres-operator/v1alpha2/crd"
)

// Transformer ships with the operator Module. Renders any
// ManagedDatabaseClaim request into a Postgres CRD instance.
// requiredClaims on a #ComponentTransformer registers fulfilment for
// the component-level #ManagedDatabaseClaim.
#ManagedDatabaseTransformer: transformer.#ComponentTransformer & {
    metadata: {
        modulePath:  "vendor.com/operators"
        name:        "managed-database-transformer"
        version:     "v1"
        description: "Renders ManagedDatabaseClaim → Postgres CRD instance"
    }
    requiredClaims: (data.#ManagedDatabaseClaim.metadata.fqn): data.#ManagedDatabaseClaim
    // Fires once per matching component; #transform.#component is the matched component.
    #transform: _   // emits pgcrd.#Postgres from claim spec
}

postgresOperator: core.#Module & {
    metadata: {
        modulePath: "vendor.com/operators"
        name:       "postgres"
        version:    "0.5.0"
        description: "Postgres operator — fulfils ManagedDatabase via CRD reconciliation"
    }

    #config: {
        operatorImage: string | *"vendor.com/pg-operator:0.5.0"
        watchAllNamespaces: bool | *true
    }

    #components: {
        controller: {
            #resources: {
                deployment: container.#ContainerResource & {
                    #spec: container: { image: #config.operatorImage }
                }
                serviceAccount: rbac.#ServiceAccountResource & {...}
                role:           rbac.#RoleResource & {...}
            }
        }
        crds: {
            #resources: {
                postgres: crd.#CRDsResource & {
                    #spec: crds: postgres: {
                        group:    "postgres.vendor.com"
                        kind:     "Postgres"
                        // ... full CRD spec
                    }
                }
            }
        }
    }

    #lifecycles: {
        install: {...}   // controller readiness → CRDs registered → ready
    }

    // Rendering extension — publication form. The transformer's
    // requiredClaims declares which Claim type this Module fulfils.
    #defines: transformers: {
        (#ManagedDatabaseTransformer.metadata.fqn): #ManagedDatabaseTransformer
    }
}
```

When this Module is registered with a `#Platform` (see enhancement 003):

- Its `#components` install the operator (controller + CRDs).
- Its `#defines.transformers` extends the platform's render pipeline.
- Consumer Modules' `#components.X.#claims: db: data.#ManagedDatabaseClaim & {…}` requests match this transformer via the shared FQN in `#ComponentTransformer.requiredClaims`, and are rendered to Postgres CRD instances at deploy time.

## Example 4 — Specialty vendor + consumer

A vendor publishes a specialty `VectorIndex` Claim type in their own CUE package. A consumer Module imports it and claims an instance.

### Vendor publishes (catalog package)

```cue
// vendor.com/vectordb/v1alpha2/claims/vector_index.cue
package vectordb

import (
    core "opmodel.dev/core/v1alpha2"
)

#VectorIndex: {
    dimensions!: int & >0
    metric!:     "cosine" | "euclidean" | "dot"
}

#VectorIndexDefaults: {
    metric: "cosine"
}

// #status shape — written by the vendor's #VectorIndexTransformer.
#VectorIndexStatus: {
    endpoint!:  string   // e.g. "https://vectordb.media.svc:443"
    indexName!: string   // server-side index identifier
    apiKeyRef?: {
        secretName!: string
        key!:        string
    }
}

#VectorIndexClaim: core.#Claim & {
    apiVersion: "vendor.com/vectordb/v1alpha2"
    metadata: {
        modulePath:  "vendor.com/vectordb/v1alpha2/claims"
        version:     "v1"
        name:        "vector-index"
        description: "Vendor-specialty contract for a vector index service."
    }
    #spec:   vectorIndex: #VectorIndex
    #status: #VectorIndexStatus
}
```

### Vendor's operator Module fulfils

```cue
// vendor.com/vectordb/v1alpha2/operator/operator.cue
package operator

import (
    core "opmodel.dev/core/v1alpha2"
    transformer "opmodel.dev/core/v1alpha2:transformer"
    vectordb "vendor.com/vectordb/v1alpha2/claims"
)

#VectorIndexTransformer: transformer.#ComponentTransformer & {
    metadata: {
        modulePath: "vendor.com/vectordb/v1alpha2"
        name:       "vector-index-transformer"
        version:    "v1"
        description: "Renders VectorIndexClaim → vendor CRD"
    }
    requiredClaims: (vectordb.#VectorIndexClaim.metadata.fqn): _
    #transform: _
}

vectordbOperator: core.#Module & {
    metadata: {
        modulePath: "vendor.com/vectordb/v1alpha2"
        name:       "operator"
        version:    "0.1.0"
    }
    #components: {...}
    #defines: transformers: {
        (#VectorIndexTransformer.metadata.fqn): #VectorIndexTransformer
    }
}
```

### Consumer claims

```cue
package ml_app

import (
    core "opmodel.dev/core/v1alpha2"
    container "opmodel.dev/opm/v1alpha2/resources/workload"
    vectordb "vendor.com/vectordb/v1alpha2/claims"
)

mlApp: core.#Module & {
    metadata: {modulePath: "example.com/apps", name: "ml-app", version: "0.1.0"}
    #components: {
        inference: {
            #resources: {
                container: container.#ContainerResource & {
                    #spec: container: env: [
                        {name: "VECTOR_ENDPOINT",  value: #claims.vec.#status.endpoint},
                        {name: "VECTOR_INDEX",     value: #claims.vec.#status.indexName},
                    ]
                }
            }
            #claims: {
                vec: vectordb.#VectorIndexClaim & {
                    #spec: vectorIndex: { dimensions: 1536, metric: "cosine" }
                }
            }
        }
    }
}
```

The consumer imports the vendor's CUE package directly. CUE unification matches the consumer's `#claims.vec` to the vendor operator's transformer because both reference the same `#VectorIndexClaim` definition (same `metadata.fqn`) — the consumer's request value carries the FQN, and the `#ComponentTransformer` declares that FQN in `requiredClaims`.

## Example 5 — Claim-only Module

A platform team or vendor publishes a `#Claim` *type definition* without fulfilling it. Fulfilment ships separately in any operator Module whose `#ComponentTransformer.requiredClaims` (component-level Claims) or `#ModuleTransformer.requiredClaims` (module-level Claims) includes the Claim's FQN. This pattern is how new commodity contracts enter the ecosystem before any implementation exists.

```cue
package image_registry_claim

import (
    core "opmodel.dev/core/v1alpha2"
)

#ImageRegistry: {
    url!:         string
    credentials?: _
}

// #status shape — written by whichever Module's transformer eventually
// fulfils ImageRegistryClaim. Until that fulfiller is registered, consumer
// claims of this type stay unmatched.
#ImageRegistryStatus: {
    endpoint!: string                // resolved registry URL
    pullSecretRef?: {
        secretName!: string
        namespace!:  string
    }
}

#ImageRegistryClaim: core.#Claim & {
    apiVersion: "example.com/platform/v1alpha2"
    metadata: {
        modulePath:  "example.com/platform/v1alpha2/claims"
        version:     "v1"
        name:        "image-registry"
        description: "Self-service image registry contract"
    }
    #spec:   imageRegistry: #ImageRegistry
    #status: #ImageRegistryStatus
}

imageRegistryClaim: core.#Module & {
    metadata: {
        modulePath:  "example.com/platform/v1alpha2"
        name:        "image-registry-claim"
        version:     "0.1.0"
        description: "Publishes the ImageRegistryClaim type. Fulfilment ships separately."
    }

    // No #components, no #config, no #claims.
    // Pure type publication.
    #defines: claims: {
        (#ImageRegistryClaim.metadata.fqn): #ImageRegistryClaim
    }
}
```

Once registered with a `#Platform` (003), the Claim type appears in `#Platform.#knownClaims` and is discoverable by any consumer Module that wants to issue requests. Consumer requests stay unmatched until a fulfilling operator Module is also registered.

## Example 6 — Publication-only Module: OPM core catalog

The OPM core catalog (`opmodel.dev/opm/v1alpha2`) ships every well-known `#Resource` and `#Trait` definition plus the Kubernetes-provider transformers. (Blueprint definitions ship in their own CUE packages and are imported directly by consumers — they are not aggregated in `#defines`; see DEF-D6.) Under the new shape, the entire catalog is packaged as a single Module whose only filled slot is `#defines`.

```cue
package opm_kubernetes_core

import (
    core "opmodel.dev/core/v1alpha2"

    // Resource definitions
    container "opmodel.dev/opm/v1alpha2/resources/workload"
    config    "opmodel.dev/opm/v1alpha2/resources/config"
    storage   "opmodel.dev/opm/v1alpha2/resources/storage"
    security  "opmodel.dev/opm/v1alpha2/resources/security"
    extension "opmodel.dev/opm/v1alpha2/resources/extension"

    // Trait definitions
    network   "opmodel.dev/opm/v1alpha2/traits/network"
    workload  "opmodel.dev/opm/v1alpha2/traits/workload"
    sectraits "opmodel.dev/opm/v1alpha2/traits/security"

    // Transformers
    deploy    "opmodel.dev/opm/v1alpha2/providers/kubernetes/transformers/deployment"
    svc       "opmodel.dev/opm/v1alpha2/providers/kubernetes/transformers/service"
    // ... rest
)

// Blueprint definitions (e.g. wlbp.#StatelessWorkloadBlueprint, databp.#SimpleDatabaseBlueprint)
// ship in their own CUE packages and are imported directly by consumers — they are not
// aggregated in #defines (see DEF-D6).

opmKubernetesCore: core.#Module & {
    metadata: {
        modulePath:  "opmodel.dev/opm/v1alpha2"
        name:        "opm-kubernetes-core"
        version:     "0.1.0"
        description: "OPM core Kubernetes catalog: resources, traits, transformers"
    }

    // No #components — pure publication+rendering Module.
    // No #config — nothing to configure.

    #defines: {
        resources: {
            (container.#ContainerResource.metadata.fqn):    container.#ContainerResource
            (config.#ConfigMapsResource.metadata.fqn):       config.#ConfigMapsResource
            (config.#SecretsResource.metadata.fqn):          config.#SecretsResource
            (storage.#VolumesResource.metadata.fqn):         storage.#VolumesResource
            (security.#RoleResource.metadata.fqn):           security.#RoleResource
            (security.#ServiceAccountResource.metadata.fqn): security.#ServiceAccountResource
            (extension.#CRDsResource.metadata.fqn):          extension.#CRDsResource
        }
        traits: {
            (network.#ExposeTrait.metadata.fqn):             network.#ExposeTrait
            (network.#HttpRouteTrait.metadata.fqn):          network.#HttpRouteTrait
            (workload.#ScalingTrait.metadata.fqn):           workload.#ScalingTrait
            (workload.#SizingTrait.metadata.fqn):            workload.#SizingTrait
            (sectraits.#WorkloadIdentityTrait.metadata.fqn): sectraits.#WorkloadIdentityTrait
            // ...
        }
        transformers: {
            (deploy.#DeploymentTransformer.metadata.fqn): deploy.#DeploymentTransformer
            (svc.#ServiceTransformer.metadata.fqn):       svc.#ServiceTransformer
            // ... rest
        }
    }
}
```

Imports the existing CUE packages verbatim — no rewriting of definitions. The `#defines` map is the registration index that lets a `#Platform` (enhancement 003) discover what this Module publishes without scanning source.

## Example 7 — Operational commodity (backup)

This example demonstrates the **dual-scope pattern**: a `#ModuleTransformer` that consumes both a Module-level `#Claim` (cross-component orchestration: schedule, backend, retention, restore choreography) and per-component `#Trait` data (local data: which volumes, app-specific quiescing hooks). The render body walks `#moduleRelease.#components` to find the trait-bearing components; `requiresComponents` is the pre-fire gate that ensures at least one such component exists. This replaces the prior `#Directive` + `#PolicyTransformer` machinery under the `#Claim`-based model.

The example targets backup via K8up. TLS certificates (cert-manager) and Gateway-API routing follow the same shape — see the summary at the end.

### Component-local `#BackupTrait`

Carries only what the component knows about its own data: which volumes to back up, app-specific quiescing hooks. No schedule, backend, or retention.

```cue
// modules/opm/operations/backup/trait.cue
package backup

import (
    core "opmodel.dev/core/v1alpha2"
)

#BackupTrait: core.#Trait & {
    metadata: {
        modulePath:  "opmodel.dev/opm/v1alpha2/operations/backup"
        version:     "v1"
        name:        "backup"
        description: "Declares this component participates in backup and what of its data to include"
    }

    #spec: backup: {
        targets!: [...{
            volume?: string    // references a #VolumesResource entry on this component
            path?:   string
            pvc?:    string
        }] & list.MinItems(1)

        include?: [...string]
        exclude?: [...string]

        // App-specific quiescing — knows the component's internals.
        preBackup?:  [...#BackupHook]
        postBackup?: [...#BackupHook]
    }
}

#BackupHook: {
    name!:           string
    command!:        [...string] & list.MinItems(1)
    container?:      string
    onError?:        *"fail" | "continue"
    timeoutSeconds?: int & >=1 | *300
}
```

### Module-level `#BackupClaim` (replaces `#BackupPolicy` directive)

Carries the cross-component orchestration. Expressed as a `#Claim` type instead of a `#Directive`. The `restore` sub-field is a declarative procedure read by the CLI at restore time.

```cue
// modules/opm/operations/backup/claim.cue
package backup

import (
    core "opmodel.dev/core/v1alpha2"
)

#Backup: {
    schedule!:  string                     // cron expression, validated by the backend
    backend!:   string                     // resolves against #ctx.platform.backup.backends
    retention?: {
        keepLast?:    int & >=0
        keepHourly?:  int & >=0
        keepDaily?:   int & >=0
        keepWeekly?:  int & >=0
        keepMonthly?: int & >=0
        keepYearly?:  int & >=0
    }
    tags?: [string]: string

    // Declarative restore procedure, read by `opm release restore`.
    // Snapshot selection is imperative (CLI argument) and is not authored here.
    restore?: {
        healthChecks?: [compName=string]: {
            path!:           string
            port!:           int & >=1 & <=65535
            timeoutSeconds?: int & >=1 | *300
            expectStatus?:   int | *200
        }
        preRestore?:  [...#RestoreStep]
        postRestore?: [...#RestoreStep]
        inPlace?: { requiresScaleDown?: bool | *true }
        disasterRecovery?: { managedByOPM?: bool | *false }
    }
}

#BackupDefaults: {
    retention: { keepDaily: 7, keepWeekly: 4 }
}

#BackupClaim: core.#Claim & {
    apiVersion: "opmodel.dev/opm/v1alpha2"
    metadata: {
        modulePath:  "opmodel.dev/opm/v1alpha2/operations/backup"
        version:     "v1"
        name:        "backup"
        description: "Schedule, destination, retention, and restore workflow for the components in this Module"
    }
    #spec: backup: #Backup
}

#RestoreStep: {
    component!:      string
    action!:         "scale-down" | "scale-up" | "delete-pods" | "wait-health" | "exec"
    args?:           [...string]
    timeoutSeconds?: int & >=1
}
```

### K8up `#BackupScheduleTransformer` — dual-scope

A `#ModuleTransformer`. `requiredClaims` matches the module-level `#BackupClaim` (cross-component orchestration). `requiresComponents.traits` is a pre-fire gate that refuses to fire unless at least one component carries `#BackupTrait` (local data facts). The render body itself walks `#moduleRelease.#components` to pick up the trait-bearing components — the gate just stops the transformer firing on a Module that has the claim but no eligible components.

```cue
// modules/k8up/transformers/backup.cue
package transformers

import (
    transformer "opmodel.dev/core/v1alpha2:transformer"
    backup "opmodel.dev/opm/v1alpha2/operations/backup"
)

#BackupScheduleTransformer: transformer.#ModuleTransformer & {
    metadata: {
        modulePath:  "opmodel.dev/k8up/v1alpha2/transformers"
        version:     "v1"
        name:        "backup-schedule-transformer"
        description: "Renders #BackupClaim + per-component #BackupTrait → K8up Backend + Schedule CRs"
    }

    // Module-level orchestration claim (lives at #Module.#claims).
    requiredClaims: (backup.#BackupClaim.metadata.fqn): _

    // Pre-fire gate — refuse to fire if no component carries #BackupTrait.
    // Body iterates #moduleRelease.#components itself to find them.
    requiresComponents: traits: (backup.#BackupTrait.metadata.fqn): _

    readsContext:  ["backup.backends"]
    producesKinds: ["k8up.io/v1.Backend", "k8up.io/v1.Schedule"]

    // Fires once per Module. Body walks #moduleRelease.#components.
    #transform: _   // emits one Backend per (namespace, backend) and one Schedule covering trait-bearing components
}
```

### The K8up Module ships everything

```cue
// modules/k8up/module.cue
package k8up

import (
    core "opmodel.dev/core/v1alpha2"
    transformers "opmodel.dev/k8up/v1alpha2/transformers"
)

k8upModule: core.#Module & {
    metadata: {
        modulePath: "opmodel.dev/k8up/v1alpha2"
        name:       "k8up"
        version:    "1.0.0"
    }
    #components: {
        controller: { ... }    // K8up operator deployment + RBAC
        crds:       { ... }    // K8up CRDs via #CRDsResource
    }
    #defines: transformers: {
        (transformers.#BackupScheduleTransformer.metadata.fqn):
            transformers.#BackupScheduleTransformer
    }
}
```

### Consumer Module (Strix media)

```cue
package strix_media

import (
    core "opmodel.dev/core/v1alpha2"
    backup "opmodel.dev/opm/v1alpha2/operations/backup"
)

strixMedia: core.#Module & {
    metadata: {
        modulePath:       "opmodel.dev/modules"
        name:             "strix-media"
        version:          "0.1.0"
        defaultNamespace: "media"
    }

    #components: {
        "app": #StatefulWorkload & backup.#BackupTrait & {
            #spec: {
                container: image: "strix:latest"
                volumes: [
                    {name: "config", size: "10Gi"},
                    {name: "cache",  size: "50Gi"},
                ]
                backup: {
                    targets: [{volume: "config"}]    // cache omitted by intent
                    exclude: ["*.log", "*.tmp"]
                }
            }
        }

        "db": #StatefulWorkload & backup.#BackupTrait & {
            #spec: {
                container: image: "postgres:16"
                volumes: [{name: "data", size: "5Gi"}]
                backup: {
                    targets: [{volume: "data"}]
                    preBackup: [{
                        name:    "pg-checkpoint"
                        command: ["psql", "-U", "postgres", "-c", "CHECKPOINT"]
                    }]
                }
            }
        }
    }

    #claims: nightly: backup.#BackupClaim & {
        #spec: backup: {
            schedule:  "0 2 * * *"
            backend:   "offsite-b2"
            retention: { keepDaily: 7, keepWeekly: 4, keepMonthly: 3 }
            tags: { "app": "strix-media" }

            restore: {
                preRestore: [
                    {component: "app", action: "scale-down"},
                    {component: "db",  action: "scale-down"},
                ]
                postRestore: [
                    {component: "db",  action: "scale-up"},
                    {component: "db",  action: "wait-health"},
                    {component: "app", action: "scale-up"},
                    {component: "app", action: "wait-health"},
                ]
                healthChecks: {
                    "app": {path: "/health",  port: 8096}
                    "db":  {path: "/healthz", port: 5432}
                }
                inPlace: requiresScaleDown: true
            }
        }
    }
}
```

The Module satisfies both halves of the transformer's match: it carries `#BackupClaim` at module level (`#ModuleTransformer.requiredClaims`), and components `app` and `db` both carry `#BackupTrait` (`requiresComponents.traits` gate). The transformer fires once per Module; its render body walks `#moduleRelease.#components`, picks the trait-bearing entries (`app`, `db`), and emits the backup infrastructure.

#### `#status` is empty for side-effect-only fulfilment

Note that `#BackupClaim` does not pin a `#status` schema. Backup is a **side-effect orchestration** — the transformer creates a Schedule and a Backend; the consumer Module never references the schedule or backend by name from inside its own components. There is nothing useful to write back into the claim instance. Compare with `#ManagedDatabaseClaim`, where the consumer reads `#status.host` / `#status.passwordRef.*` to wire env vars: there the resolution data is consumed at render time and the `#status` schema is non-trivial. Some claims fulfil purely by side-effect; their `#status` may stay empty by design. (See 03-schema.md "Resolution channel — `#status`".)

### Platform-side context (admin authored, once per environment)

```cue
#Platform & {
    metadata: name: "kind-opm-dev"
    type: "kubernetes"

    #ctx: platform: backup: backends: {
        "offsite-b2": {
            type:              "b2"
            bucket:            "jacero-backups"
            credentialsSecret: "b2-creds"
            encryption:        { repoPasswordSecret: "restic-repo-pw" }
        }
        "local-minio": {
            type:              "s3"
            endpoint:          "http://minio.storage.svc:9000"
            bucket:            "backups"
            credentialsSecret: "minio-creds"
        }
    }

    #registry: {
        "opm-core": { #module: opmCore.#Module }
        "k8up":     { #module: k8upModule }    // registers BackupScheduleTransformer via #defines
        // ...
    }
}
```

The `backup.backends` struct lives in `#ctx.platform` (open struct, platform-team-owned — see enhancement 004 for the full `#ctx` schema). The transformer's `readsContext: ["backup.backends"]` declares it depends on this slice; the render pipeline supplies the resolved struct at execution time.

### Restore at the CLI

Restore does not go through the render pipeline. The CLI reads the declarative `restore` sub-field from the Module's `#BackupClaim`, combines it with an imperative snapshot selector, and drives the workflow:

```text
opm release restore strix-media --snapshot latest
opm release restore strix-media --snapshot <restic-id> --dry-run
opm release restore strix-media --tag "pre-upgrade-2026-04-21"
opm release restore strix-media --components db --snapshot latest
```

Steps the CLI executes:

1. Locate the Claim in the resolved Module's `#claims` covering the requested components.
2. Execute `restore.preRestore` steps in order.
3. Invoke the backup backend's restore mechanism with the selected snapshot.
4. Execute `restore.postRestore` steps in order.
5. Poll `restore.healthChecks[component]` until pass or timeout.

No OPM-side transformer or controller is invoked. The operator (K8up) handles data movement; the CLI orchestrates the choreography declared in the Claim.

### TLS certificates and Gateway-API routing follow the same shape

Both can be expressed under the same dual-scope `#ModuleTransformer` + `requiresComponents` pattern. The structural mapping is:

| Commodity | Component-local primitive | Module-level Claim | Platform context | Transformer kind + keys |
|---|---|---|---|---|
| Backup | `#BackupTrait` (volumes, hooks) | `#BackupClaim` (schedule, backend, retention, restore) | `#ctx.platform.backup.backends.*` | `#ModuleTransformer.requiredClaims` + `requiresComponents.traits` |
| TLS | `#CertificateResource` (hostnames, secret name) | `#CertificateClaim` (issuer ref, renewal, key alg) | `#ctx.platform.tls.issuers.*` | `#ModuleTransformer.requiredClaims` + `requiresComponents.resources` |
| Gateway routing (HTTP/TLS/gRPC/TCP/UDP) | `#HTTPRouteResource` (paths, methods, backend port) — and 4 sibling kinds | `#HTTPRouteClaim` (gateway ref, hostnames, default filters) | `#ctx.platform.routing.gateways.*` | `#ModuleTransformer.requiredClaims` + `requiresComponents.resources` |

The TLS variant uses `requiresComponents.resources` (not `traits`) because a TLS Certificate is a standalone Kubernetes entity with its own lifecycle, not a behaviour modifier on a workload — the standard litmus split. Gateway routes work the same way for the same reason. The dual-scope pattern stays unchanged; only the per-component primitive type swaps in.

## Summary

Same `#Module` type covers all seven shapes. None of the unfilled slots add cognitive overhead — they are simply absent. Component-level and module-level `#claims` work side-by-side. Operators register fulfilment via `#ComponentTransformer.requiredClaims` or `#ModuleTransformer.requiredClaims` inside `#defines.transformers` — no wrapper primitive. Specialty Claims extend the ecosystem without catalog PRs. Publication-only Modules (the OPM core catalog, vendor-published primitive sets, Claim-only publishers) ship type definitions through `#defines` with no runtime footprint. Operational commodities (backup, TLS, routing) use the dual-scope pattern: a `#ModuleTransformer` whose `requiredClaims` matches the orchestration Claim and whose `requiresComponents` gate ensures the necessary component-side facts exist before firing.
