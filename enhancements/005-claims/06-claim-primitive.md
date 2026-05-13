# `#Claim` — The Ecosystem-Supplied Need Primitive

This file is the topical narrative for the `#Claim` primitive: what it is, how it relates to `#Resource`, where instances can live, how identity travels across module boundaries, how fulfilment is registered, and how resolution data flows back via `#status`.

Decisions referenced live in `10-decisions.md` under the `CL-` prefix; open questions in `11-open-questions.md`.

For the `#Module` slot list overall, see [`04-module-shape.md`](04-module-shape.md).
For the `#defines` publication channel that surfaces Claim type definitions, see [`05-defines-channel.md`](05-defines-channel.md).
For the transformers that fulfil Claims, see [`07-claim-fulfilment.md`](07-claim-fulfilment.md) (`#ModuleTransformer` + status writeback) and [003/05-component-transformer-and-matcher.md](../003-platform-construct/05-component-transformer-and-matcher.md) (`#ComponentTransformer` schema + matcher algorithm).

## Litmus Question

> **What ecosystem-supplied thing must be fulfilled?**

Sharpened from the original "What must exist?" overlap with `#Resource`. The sharpening (CL-D12) maps to a clean axis:

| Primitive | Source | Rendered by |
|---|---|---|
| `#Resource` | **catalog-fixed** — adding a new Resource type requires a catalog PR plus a Transformer | catalog-shipped Transformer per Resource type |
| `#Claim` | **ecosystem-extended** — anyone (catalog or vendor) can publish a `#Claim` type in any CUE package | any registered transformer whose `requiredClaims` includes the Claim's FQN |

A `#Resource` describes a thing in known catalog vocabulary so a known Transformer can render it. A `#Claim` declares intent that the ecosystem will satisfy at deploy time. **CL-D2** keeps both primitives distinct.

## Schema

```cue
// apis/core/v1alpha2/claim.cue
package v1alpha2

#Claim: {
    apiVersion!: string                            // open — set by concrete claim
    kind:        "Claim"

    metadata: {
        modulePath!: #ModulePathType                // "opmodel.dev/opm/v1alpha2/claims/data"
        version!:    #MajorVersionType              // "v1"
        name!:       #NameType                      // "managed-database"
        #definitionName: (#KebabToPascal & {"in": name}).out

        fqn: #FQNType & "\(modulePath)/\(name)@\(version)"

        description?: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    // MUST be an OpenAPIv3 compatible schema.
    // The field name is the camelCase of metadata.name (kebab-case names
    // are converted: "managed-database" => "managedDatabase").
    #spec!: ((#KebabToCamel & {"in": metadata.name}).out): _

    // Open shape — concrete Claim definitions pin their own #status schema.
    // Populated by the fulfilling transformer at deploy time.
    // Module authors read via #claims.<id>.#status.<field>.
    #status?: _
}

#ClaimMap: [string]: _
```

Lives in the flat `v1alpha2` package alongside `#Resource`, `#Trait`, `#Blueprint`. Helper types (`#ModulePathType`, `#MajorVersionType`, `#NameType`, `#FQNType`, `#KebabToPascal`, `#KebabToCamel`, `#LabelsAnnotationsType`) all live in the same package — no `t.` prefix needed.

## Identity

A `#Claim` carries `apiVersion` plus `metadata.{modulePath, name, version, fqn}`. There is no string `type` field (**CL-D4**). Matching is structural at the CUE level (consumer's `#claims.X` and transformer's `requiredClaims` reference the same `#Claim` definition) and metadata-driven at deploy time (the platform reads `apiVersion` + `fqn` to route requests).

Two layers of identity:

- **CUE-level (authoring time):** A Module's `#claims` references a `#Claim` definition (e.g. `data.#ManagedDatabaseClaim`). A transformer's `requiredClaims` references the same definition's FQN. CUE unification makes the request value structurally identical to the Claim definition.
- **Metadata-level (deploy time):** Each Claim instance carries `apiVersion` + `metadata.fqn`. The platform reads these and matches `#claims` requests to transformers whose `requiredClaims` map keys match.

`apiVersion` is **open** on the `#Claim` base — concrete Claims (e.g. `#ManagedDatabaseClaim`, `#VectorIndexClaim`) set their own (**CL-D7**). A vendor's `#VectorIndexClaim` carries `apiVersion: "vendor.com/vectordb/v1alpha2"`; the OPM core catalog's `#ManagedDatabaseClaim` carries `apiVersion: "opmodel.dev/opm/v1alpha2"`. The base does not pin one.

This differs from `#Resource` and `#Directive` which today pin `apiVersion` to OPM core; whether to open them similarly is **CL-Q4** (deferred).

## Demand and Resolution — `#spec` and `#status`

`#Claim` carries two complementary fields, mirroring the Kubernetes `spec` / `status` convention:

- **`#spec!`** (required) — the **demand** the consumer Module makes. A `#ManagedDatabaseClaim` request specifies engine, version, sizeGB. The shape is defined by the concrete Claim type.
- **`#status?`** (optional, open on the base; pinned by concrete Claims) — the **resolution** the fulfilling transformer writes back at deploy time. A `#ManagedDatabaseClaim.#status` carries `host`, `port`, `passwordRef.{secretName, key}`. Module authors read from it to wire env vars.

`#status` is the cross-runtime portability surface (**CL-D15**). The same `#Module` deployed against different fulfilling transformers receives target-appropriate resolution data without changing the consumer's code path. A `#PublicEndpointClaim` resolved by a k8s Gateway transformer or by a compose Traefik-label transformer both write `#status.url`; the consumer reads the same field on both runtimes.

### Resolution lifecycle

1. **Match.** The matcher walks `#composedTransformers`. A transformer whose `requiredClaims` contains a Claim FQN is the fulfiller for that Claim instance.
2. **Render.** Transformer body runs — `#transform.output` produces the provider-specific resource(s); a sibling `#transform.#resolution` carries per-claim status data (sketched in `07-claim-fulfilment.md`).
3. **Inject.** The Go pipeline reads `#resolution` and injects values via `FillPath` into the matched `#Claim` instance's `#status`. Same Strategy B precedent as 004 D12 hash injection (**CL-D16**).
4. **Consume.** Downstream transformers / component bodies that read `#claims.<id>.#status.<field>` see the populated values. The matcher topologically orders fulfillers before consumers.

### Side-effect-only fulfilment

Some Claims fulfil purely by side-effect — the transformer creates resources but the consumer never reads connection data. `#BackupClaim` is the canonical example: the K8up transformer creates Schedule + Backend CRs; the consumer's components don't reference the schedule by name. For these, `#status` may stay empty — concrete Claim definitions need not pin a `#status` schema. Empty `#status` is a valid by-design outcome.

Compare: `#ManagedDatabaseClaim.#status` is non-trivial because consumers wire `DATABASE_HOST` etc.; `#BackupClaim.#status` is empty because nothing in the consumer reads it.

## Placement Rules

`#Claim` lives at two scopes (**CL-D10**):

| Placement | Use case | Examples |
|---|---|---|
| **Component-level** — `#components.X.#claims.Y` | Per-component data-plane needs | DB, queue, cache, secret |
| **Module-level** — `#claims.Y` | Per-module platform-relationship needs | DNS hostname, workload identity, observability backend, mesh tenant, backup orchestration |

Same primitive, two scopes; placement determines semantics. A Module may need a shared DB (module-level) *and* each component may have its own cache (component-level) — naming distinguishes them.

`#Resource` stays component-level only (**CL-D11**). Shared resources should be modeled as their own `#Component`. Allowing module-level Resources collapses the component-as-unit-of-composition boundary.

| Primitive | Component-level | Module-level (`#claims`) | `#defines` (type publication) |
|---|---|---|---|
| `#Resource` | yes | no | yes |
| `#Trait` | yes | no | yes |
| `#Blueprint` | no (composes Components) | no | **no** (DEF-D6 — CUE-import sugar, no platform-level consumer) |
| `#Claim` | yes | yes | yes |
| `#Transformer` | no | no | yes (`#defines.transformers`) |

## Triplet (or Quartet) Pattern

Concrete Claims follow the catalog triplet pattern (**CL-D6**): `#X` (schema) + `#XDefaults` (defaults) + `#XClaim` (`#Claim` wrapper). When the Claim has resolution data (CL-D15), the triplet extends to a quartet with `#XStatus` (status shape).

```cue
// modules/opm/claims/data/managed_database.cue
package data

import (
    core "opmodel.dev/core/v1alpha2"
)

// Schema — what the consumer requests.
#ManagedDatabase: {
    engine!:  "postgres" | "mysql" | "mongodb"
    version!: string
    sizeGB!:  int & >0
    highAvailability?: bool | *false
}

// Defaults — opinionated starting values.
#ManagedDatabaseDefaults: {
    engine: "postgres"
    sizeGB: 10
    highAvailability: false
}

// Status — what the fulfilling transformer writes back.
#ManagedDatabaseStatus: {
    host!: string
    port!: int & >0 & <=65535
    passwordRef!: {
        secretName!: string
        key!:        string
    }
    sslMode?: "disable" | "require" | "verify-full"
}

// Wrapper — ties schema + status to the #Claim primitive and pins identity.
#ManagedDatabaseClaim: core.#Claim & {
    apiVersion: "opmodel.dev/opm/v1alpha2"
    metadata: {
        modulePath:  "opmodel.dev/opm/v1alpha2/claims/data"
        version:     "v1"
        name:        "managed-database"
        description: "Well-known commodity contract for a managed relational database."
    }
    #spec:   managedDatabase: #ManagedDatabase
    #status: #ManagedDatabaseStatus
}
```

A consumer reads:

```cue
#components: web: #resources: container: #spec: container: env: [
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
```

The same module deploys against pg-operator, Aiven, RDS — each fulfiller writes its own values to `#status`. The consumer code is unchanged.

The triplet pattern stays consistent with `#Container` / `#ContainerDefaults` / `#ContainerResource` (and now `#ContainerStatus` where applicable). Authors moving between Resources and Claims see the same shape.

Specialty Claims follow the same shape, different package — `vendor.com/vectordb/v1alpha2/claims/vector_index.cue` ships `#VectorIndex` / `#VectorIndexDefaults` / `#VectorIndexStatus` / `#VectorIndexClaim`. CL-Q2 captures the open question of what naming convention prevents collisions when two vendors ship a `vector-index` Claim.

## Capability Fulfilment — Transformer `requiredClaims`

A Module fulfilling a Claim ships a transformer (in `#defines.transformers`) whose `requiredClaims` map contains the Claim's FQN. The transformer that does the rendering **is** the registration. There is no separate `#Api` wrapper primitive (**CL-D14**).

Two transformer kinds (TR-D5):

- **`#ComponentTransformer.requiredClaims: <FQN>`** — fulfils a component-level claim. Fires once per matching component.
- **`#ModuleTransformer.requiredClaims: <FQN>`** — fulfils a module-level claim. Fires once per matching module.

```cue
#ManagedDatabaseTransformer: transformer.#ComponentTransformer & {
    metadata: { ... }
    requiredClaims: (data.#ManagedDatabaseClaim.metadata.fqn): _
    #transform: _   // emits Postgres CRD instance from claim spec; #resolution populates #status
}

postgresOperator: core.#Module & {
    metadata: {...}
    #components: {...}    // controller + CRDs + RBAC
    #defines: transformers: {
        (#ManagedDatabaseTransformer.metadata.fqn): #ManagedDatabaseTransformer
    }
}
```

CRDs continue to ship as `#CRDsResource` inside `#components` (**CL-D8** operative half — the `#Api` framing clause is moot after CL-D14).

### How `#Claim` plays three roles

| Placement | Role |
|---|---|
| `#defines.claims["fqn"]: #Claim` | Module **publishes** this Claim type (catalog vocabulary) |
| `#claims.id: #Claim & {#spec: ...}` (component or module level) | Module **requests** an instance (demand) |
| `#defines.transformers["fqn"]: #ComponentTransformer & {requiredClaims: …}` (component-level) or `#ModuleTransformer & {requiredClaims: …}` (module-level) | Module **fulfils** this Claim type (supply, via the transformer's match keys) |

A single Module may do all three. The OPM core Module typically only does the first; an operator Module typically does the last (and ships any new Claim types it introduces under `#defines.claims`).

## Why `#Claim` Doesn't Have A `#ClaimType` Primitive

Considered: a separate `#ClaimType` (defines schema) + `#Claim` (request) split, gRPC-like. Rejected (**CL-D3**).

CUE's package system already provides type identity. A catalog-published `#Claim` definition (e.g. `#ManagedDatabaseClaim`) is simultaneously the type and the request shape — instances unify with the definition. Adding `#ClaimType` would add a primitive without proportional benefit.

## Why `#Api` Was Removed

The supply-side wrapper paired with `#Claim` as the demand-side was originally named `#Offer` then renamed to `#Api` (CL-D1). CL-D14 removed it entirely.

Rationale: with transformers gaining `requiredClaims` as a match key (TR-D5), the same information is carried by the transformer that does the rendering. No wrapper layer needed. Removing `#Api` collapsed three placements of `#Claim` (publish via `#defines.claims`, request via `#claims`, fulfil via `#apis`) into two (publish, request), with fulfilment expressed by the transformer pipeline.

The supersession chain through CL-D14 is preserved in `10-decisions.md` for the audit trail. CL-D1 (replace `#Offer` with `#Api`), CL-D5 (`#Api` is 1:1 with `#Claim`), CL-D9 (`#Api` deploy-time semantics platform's choice), CL-D13 (`#Api` apiVersion pinned) all remain as historical records of the prior shape.

The framing clause of CL-D8 ("CRDs deploy via `#CRDsResource`, **not** via `#Api`") loses the `#Api` half but the operative claim — CRDs ship via `#CRDsResource` — stands.

## Decisions Referenced

Live decisions:

- **CL-D2** — Keep both `#Resource` and `#Claim`.
- **CL-D3** — No separate `#ClaimType` primitive.
- **CL-D4** — `#Claim` carries `apiVersion` + path metadata.
- **CL-D6** — Triplet (or quartet, with `#status`) pattern.
- **CL-D7** — `apiVersion` open on `#Claim` base.
- **CL-D8** — CRDs deploy via `#CRDsResource` (operative half).
- **CL-D10** — Both component-level and module-level placement.
- **CL-D11** — `#Resource` stays component-level only.
- **CL-D12** — Sharpened litmus.
- **CL-D14** — Remove `#Api` primitive (supersedes CL-D1, CL-D5, CL-D9, CL-D13, framing clause of CL-D8).
- **CL-D15** — `#Claim` gains `#status?`.
- **CL-D16** — `#status` written by matching transformer; injection via Strategy B.

Historical (superseded by CL-D14, retained in `10-decisions.md`):

- CL-D1 (replace `#Offer` with `#Api`)
- CL-D5 (`#Api` is 1:1 with `#Claim`)
- CL-D9 (`#Api` deploy-time semantics)
- CL-D13 (`#Api` apiVersion pinned)

Cross-topic:

- **TR-D5** — Two transformer primitives (`#ComponentTransformer`, `#ModuleTransformer`) — the kinds that ship `requiredClaims`.
- **DEF-D1** — `#defines` slot. Where Claim type definitions and fulfilling transformers are published.

Full text in [`10-decisions.md`](10-decisions.md).

## Open Questions

- **CL-Q1** — Component-level vs module-level resolution semantics (lean: independent fulfilments, two-instance two-`#status`).
- **CL-Q2** — Specialty Claim type versioning across vendors.
- **CL-Q3** — `#spec` / `#status` validation policy at deploy time and across version evolutions.
- **CL-Q4** — `#Resource` / `#Directive` apiVersion openness (consistency with CL-D7).
- **CL-Q5** — Well-known commodity Claims to populate.
- **CL-Q6** — Relationship to enhancement 012's noun grammar.
- **CL-Q7** — Status writeback ordering (cycle detection, missing fulfiller, multi-fulfiller).

Full list in [`11-open-questions.md`](11-open-questions.md).
