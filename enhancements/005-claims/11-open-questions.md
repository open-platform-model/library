# Open Questions — `#Module` Flat Shape with `#Claim` Primitive and `#defines` Channel

Items that remain unresolved after the initial design conversation. Grouped by topic; numbering resets within each topic group. Each entry carries a `(was Q#)` parenthetical mapping to the prior chronological numbering.

Topic groups: **MS** (module shape), **DEF** (defines channel), **CL** (claim primitive), **TR** (transformer redesign). The `07-claim-fulfilment.md` doc surfaces TR-Q1 through TR-Q4 inline; this file is the cross-cutting index.

## Categories

- **Resolved-but-pending-implementation:** Decision is made (see `10-decisions.md`); implementation must answer specifics.
- **Genuinely open:** No design decision yet; needs further design or user input.
- **Deferred:** Out of scope for this enhancement; tracked for follow-up.

---

## Module Shape (MS)

### MS-Q1 — Deferred: Migration plan for existing Modules (was Q6)

Existing Modules use the current `#Module` shape with `#policies`. The flat shape adds `#lifecycles`, `#workflows`, `#claims`, `#defines` as optional and **removes** `#policies` (MS-D4).

- Migration path for additions: existing Modules continue to work unchanged for the new optional fields.
- Migration path for `#policies` removal: any Module that authored `#policies` entries must drop them. Their behaviour reattaches via the policy redesign (enhancement 012) when that lands. In the interim, modules with policy needs are blocked on 012.
- No automated tooling for the removal — it is a one-line drop per Module, surfaced by `task vet` failing.

**Revisit trigger:** When 012 converges and reintroduces a policy/directive home (likely not as `#Module.#policies`).

---

## `#defines` Channel (DEF)

### DEF-Q1 — Genuinely open: Self-service catalog API (was Q3)

If the OPM platform implements both a self-service catalog (browse known Claim types and Resources/Traits) and a deploy-time match resolver (route `#claims` requests to fulfilling transformers), the platform-side API surface for those needs definition:

- What CLI does a platform admin use to list registered Claim types? (`opm catalog claims`?)
- How does a developer browse the self-service catalog? (Web UI? CLI? OPM-platform-specific?)
- How does the matching pipeline expose its decisions? (`opm claim resolve` showing matched fulfilling transformer?)

These are platform-implementation questions, not primitive-design questions, but they constrain what metadata the registered Module values should carry (e.g. whether `#Claim.metadata` should grow an `examples?: _` field for catalog UIs).

**Revisit trigger:** When a platform implementation begins. Likely landed in `opm-operator/` or `cli/`.

---

## `#Claim` Primitive (CL)

### CL-Q1 — Component-level vs Module-level `#Claim` resolution semantics *(CLOSED by CL-D17)*

**Status:** CLOSED 2026-05-02 by **CL-D17** (each Claim instance is fulfilled independently). Module-level `#claims.db` and component-level `#components.X.#claims.db` resolve as two instances ⇒ two `#status` values. Two component-level `#claims.db` on different components likewise resolve independently. No merge / override / share-fulfilment semantics. See `10-decisions.md` CL-D17 for full text.

---

### CL-Q2 — Genuinely open: Specialty Claim type versioning across vendors (was Q2)

Two vendors might independently ship a Claim type with the same `metadata.name` (e.g. both ship `vector-index`) at different `modulePath`s. The `fqn` (`modulePath` + `name` + `version`) disambiguates, but a consumer Module that imports both will face naming collisions in CUE.

- Is there a naming convention recommendation for vendor-published Claims (e.g. always include vendor domain in `name`: `vendor-vector-index`)?
- Or do consumers always alias on import (CUE `import vendora "vendor-a.com/..."`)?
- Should the platform enforce uniqueness at deploy time, or accept multiple `fqn`s with the same `name` from different paths?

**Revisit trigger:** First conflict between two real ecosystem participants.

---

### CL-Q3 — Genuinely open: Validation of `#Claim` request `#spec` against the type definition (was Q4)

A Module's `#claims` reference a Claim type and provide `#spec` values. CUE unification validates the values against the type's schema at authoring time. But when a `#Claim` is serialized and sent to the platform for matching, validation must happen again at deploy time.

- Does the platform re-validate the `#spec` against the embedded `#Claim` definition before matching?
- If a vendor publishes a new version of `#VectorIndexClaim` with a stricter schema, do existing consumer Modules break or stay pinned to the old version they imported?
- Is there a Claim version-compatibility policy (semver-like)?

**Update (CL-D15, 2026-05-01):** `#status` shape (CL-D15) carries the same lifecycle question. Concrete Claims pin both `#spec` and `#status` schemas; a vendor that ships a new minor version with additional `#status` fields must guarantee old consumer reads still resolve. The validation policy for `#spec` and `#status` should be the same — answering one answers the other.

**Revisit trigger:** First Claim version evolution by a real ecosystem participant.

---

### CL-Q4 — Genuinely open: Should `#Resource` and `#Directive` apiVersion become open like `#Claim`? (was Q7)

CL-D7 leaves `apiVersion` open on `#Claim` so vendor specialties can set their own. `#Resource` and `#Directive` today pin `apiVersion` to the OPM core (see `core/v1alpha2/resource.cue` and the corresponding directive file once 012 lands).

- Should we make `#Resource` and `#Directive` apiVersion open for consistency?
- Would that allow vendors to ship Resources outside the catalog?
- Or is the catalog-fixed nature of Resources (CL-D2) the reason to keep apiVersion pinned?

**Lean direction:** Keep `#Resource` apiVersion pinned (catalog-fixed semantics). `#Directive` is more debatable — directives are operational verbs that vendors might extend, similar to Claims. Worth a separate enhancement.

**Revisit trigger:** First vendor-published Directive request.

---

### CL-Q5 — Deferred: Examples of well-known commodity Claims (was Q9)

This enhancement establishes the pattern. The catalog must populate well-known commodity Claims to make the pattern useful. Initial candidates:

- `#ManagedDatabaseClaim` (data) — relational DBs
- `#MessageBusClaim` (data) — pub/sub queues
- `#ObjectStoreClaim` (data) — S3-compatible buckets
- `#HostnameClaim` (platform) — public DNS names
- `#WorkloadIdentityClaim` (platform) — module-level identity
- `#ImageRegistryClaim` (platform) — image registry endpoints
- `#TLSCertificateClaim` (network) — TLS certs
- `#MeshTenantClaim` (network) — service mesh membership

Each needs its own Claim definition triplet (`#X`, `#XDefaults`, `#XClaim`) — and a `#XStatus` schema where the Claim has resolution data (CL-D15) — in the appropriate `claims/` subpackage.

**Revisit trigger:** During or after initial implementation. Each commodity may warrant its own enhancement for schema design.

---

### CL-Q6 — Deferred: Relationship to enhancement 012's noun grammar (was Q11)

Enhancement 012 explores cross-component noun grammar (shared networks, identities, storage pools). This enhancement provides a noun answer at module/component scope (`#Claim` for needs). But 012's "shared mesh tenant across multiple Modules" use case may still need a higher-level construct.

- Does `#Claim` at module level fully cover 012's noun grammar, or is there still a gap at the bundle / environment level?
- Does 012 need to converge before this enhancement lands, or can they ship independently?

**Lean direction:** Independent. `#Claim` covers per-Module nouns. 012 may still need work for cross-Module shared nouns (a single mesh tenant joined by multiple Modules).

**Revisit trigger:** When 012 reaches design phase.

---

### CL-Q8 — Multiple module-level Claims of the same FQN *(CLOSED by CL-D18)*

**Status:** CLOSED 2026-05-02 by **CL-D18** (duplicate module-level Claim FQN is rejected at `#Module` schema time). The check is a CUE constraint on `#Module.#claims` — two entries with the same `metadata.fqn` produce `_|_` at module-evaluation time, before the matcher runs. See `10-decisions.md` CL-D18 for full text + the schema constraint.

---

### CL-Q7 — Genuinely open: Status writeback ordering (was Q18, surfaced in 07-claim-fulfilment.md)

A `#ComponentTransformer` (or `#ModuleTransformer`) that fulfils a Claim writes `#status` via the `#resolution` channel sketched in `07-claim-fulfilment.md`. A second transformer — or a component body — reads the same Claim's `#status` to wire connection data. The matcher must dispatch fulfillers before consumers; the dispatch order is the topological sort of an FQN-graph derived from `requiredClaims` (write edges) and `#claims.<id>.#status.<field>` reads (read edges).

Open sub-questions:

- **Cycle detection.** Two transformers that each write a Claim the other reads form a cycle. Should the matcher detect this at platform-evaluation time (CUE-time) or at deploy-time (Go-pipeline)?
- **Missing fulfiller.** If a consumer reads `#status.X` but no transformer writes it, what is the deploy-time signal? `_|_` from CUE? An explicit unmatched-claim error from the matcher?
- **Multi-fulfiller.** Closed by **003 D13** — forbidden at the `#matchers` layer; platform evaluation fails when two transformers' `requiredClaims` overlap on the same FQN. Status-writeback uniqueness is guaranteed by construction.

**Revisit trigger:** Pipeline implementation, or first author hitting a cycle/missing-fulfiller case.

---

## Transformer Redesign (TR)

### TR-Q1 — Genuinely open: Interaction with `#PolicyTransformer` (was Q10)

`#PolicyTransformer` (from the policy redesign, enhancement 012) matches `#Policy` (with rules + directives) against components. `#PolicyTransformer` operates on the verb-flavor commodity surface. This enhancement introduces `#Claim` for the noun-flavor surface.

- Are these two pipelines independent, or do they interact?
- Could a `#Directive` in a Policy reference a `#Claim` (e.g., a backup Directive that targets a `BackupTargetClaim`)?
- Does the rendering pipeline run `#PolicyTransformer`s and `#Claim` matching in sequence, parallel, or interleaved?

**Revisit trigger:** During implementation of either pipeline; likely exposes the interaction.

---

### TR-Q2 — Multi-implementation resolution for transformer `requiredClaims` *(CLOSED by 003 D13)*

**Status:** CLOSED 2026-05-01 by **003 D13** — multi-fulfiller is forbidden at the `#matchers` layer. Two registered transformers whose `requiredClaims` (or `requiredResources` / `requiredTraits`) overlap on the same FQN cause `#Platform` evaluation to fail with `_|_`. Resolution is admin-driven — disable one entry via `#ModuleRegistration.enabled: false` (003 D14), pin a different version, or remove one of the two registrations. A future enhancement may reopen multi-fulfiller with a deliberate selection policy; until then the constraint stays.

Original text retained for the historical record:

> Two registered Modules may each ship a transformer (`#ComponentTransformer` or `#ModuleTransformer`) whose `requiredClaims` includes the same Claim FQN (e.g. one Postgres operator and one Aiven operator, both fulfilling `ManagedDatabase`). The platform's render pipeline must pick one per consumer request. Candidates: admin-selected default fulfiller per Claim FQN, consumer-pinned fulfiller (transformer FQN), or registry priority order. This question replaces the old `#Api` collision question (former Q12 / OQ5 in 003).

---

### TR-Q3 — Does 003's `#provider` synthetic value still work? *(CLOSED — superseded by 003 D12)*

**Status:** CLOSED 2026-05-02. **003 D12** retired `#Provider` and the synthetic `#provider` shim entirely. The matcher consumes `#composedTransformers` + the new `#matchers` reverse index directly. The `#resolution` injection-phase sub-question moves to `12-pipeline-changes.md`.

Original text retained for the historical record:

> 003/02-design.md claims the existing matcher interface is preserved via a synthetic `#provider` wrapping `#composedTransformers`. With two transformer kinds, the matcher dispatches by `kind`. Either the existing `#provider` shape carries enough information, or 003 needs a follow-up amendment.

---

### TR-Q4 — Genuinely open: `requiresComponents` granularity (was Q17, in 07-claim-fulfilment.md)

`requiresComponents` is a single conjunction (resources AND traits AND claims). A transformer that wants "components carrying `#BackupTrait` *or* components with a backup-tagged volume" cannot express that. Disjunctive gates may eventually want their own shape; deferred until a real case appears.

**Revisit trigger:** First transformer that needs a disjunctive gate.

---

## Summary of Genuinely Open Questions

| ID | Topic | Blocking implementation? |
|----|-------|--------------------------|
| CL-Q1 | Component vs module Claim resolution | **CLOSED by CL-D17** (independent fulfilments) |
| CL-Q2 | Specialty Claim versioning across vendors | No (eventual) |
| CL-Q3 | Claim spec / status validation at deploy time | Delegated to Go pipeline — see `12-pipeline-changes.md` |
| CL-Q4 | `#Resource` / `#Directive` apiVersion openness | No (consistency) |
| CL-Q7 | Status writeback ordering | Delegated to Go pipeline — see `12-pipeline-changes.md` (multi-fulfiller half closed by 003 D13) |
| CL-Q8 | Duplicate module-level Claim FQN | **CLOSED by CL-D18** (CUE constraint on `#Module.#claims`) |
| DEF-Q1 | Self-service catalog API | No (platform-impl) |
| TR-Q1 | Interaction with `#PolicyTransformer` | Yes (pipeline) — gated on 012 |
| TR-Q2 | Multi-implementation resolution | **CLOSED by 003 D13** (forbidden at `#matchers`) |
| TR-Q3 | 003 `#provider` synthetic + status injection phase | **CLOSED by 003 D12** (`#provider` retired; status injection tracked in `12-pipeline-changes.md`) |
| TR-Q4 | `requiresComponents` granularity | No (future) |
