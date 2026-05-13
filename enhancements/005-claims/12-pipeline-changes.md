# Pipeline Changes — `#Module` flat shape with `#Claim` and `#defines`

| Field       | Value      |
| ----------- | ---------- |
| **Status**  | Draft      |
| **Created** | 2026-05-02 |
| **Targets** | OPM Go render pipeline (in `cli/` and/or `opm-operator/`) |

## Why this doc exists

Several decisions in `10-decisions.md` and open questions in `11-open-questions.md` resolve cleanly at the schema level but require Go-pipeline behavior to round out. The schema reserves the channel; the pipeline owns the algorithm. This doc captures the contracts the pipeline must implement so that the CUE schema and the Go renderer stay in lockstep.

Items captured here:

- **`#resolution`** — the writeback channel sketched in `07-claim-fulfilment.md` and referenced by **CL-D16**. Schema is convention-level; pipeline pins the field name + injection ordering.
- **CL-Q3** — `#Claim.#spec` / `#status` validation policy at deploy time and across version evolutions. CUE handles authoring-time validation; the pipeline owns deploy-time + version-skew handling.
- **CL-Q7** — Topological-sort ordering for `#status` writeback. Multi-fulfiller half closed by **003 D13**; cycle detection + missing-fulfiller halves remain pipeline concerns.

Out of scope (separate enhancements):

- The matcher's component / module dispatch loop itself (specified in [003/05-component-transformer-and-matcher.md](../003-platform-construct/05-component-transformer-and-matcher.md) for the component-scope half; `#ModuleTransformer` dispatch + `requiresComponents` gate in `07-claim-fulfilment.md`).
- `#PolicyTransformer` interaction (TR-Q1) — gated on enhancement 012.

---

## `#resolution` channel

### Schema sketch (recap from `07-claim-fulfilment.md`)

```cue
#ComponentTransformer: {
    ...
    #transform: {
        #moduleRelease: _
        #component:     _
        #context:       #TransformerContext

        output: {...}                          // provider-specific render

        // Per-claim status data the runtime writes back into the matched
        // #Claim instance's #status. Keyed by the consumer Module's claim id
        // (e.g. "db" if the module declared #claims.db: ...). The matcher
        // resolves claim ids by FQN-equality between requiredClaims and the
        // candidate component's #claims.
        #resolution?: [claimId=string]: _
    }
}

#ModuleTransformer: {
    ...
    #transform: {
        #moduleRelease: _
        #context:       #TransformerContext

        output: {...}

        #resolution?: [claimId=string]: _    // resolves against #moduleRelease.#claims
    }
}
```

### Pipeline contract

When the matcher invokes a transformer that fulfils a Claim:

1. **Identify matched instance(s).** For `#ComponentTransformer`: scan the fired component's `#claims` for entries whose `metadata.fqn` matches a key in `requiredClaims`. For `#ModuleTransformer`: scan `#moduleRelease.#claims` (module-level) similarly. Independent fulfilment per instance per **CL-D17**.
2. **Run the render body.** `#transform.output` produces the provider-specific rendered resources; `#transform.#resolution` (optional) carries per-instance status data keyed by the consumer's claim id.
3. **Inject via `FillPath`.** For each entry in `#resolution`, the pipeline calls `FillPath` to write `<value>` into `#claims.<claimId>.#status` on the matched instance — module-level instance for `#ModuleTransformer`, component-level instance for `#ComponentTransformer`. Same Strategy B mechanism as 004 D12 hash injection.
4. **Re-evaluate downstream consumers.** Any transformer or component body that reads `#claims.<id>.#status.<field>` re-evaluates after the write phase.

### Matcher resolves claim ids by FQN-equality

The transformer declares `requiredClaims` keyed by FQN. The consumer Module's `#claims` declares instances keyed by author-chosen ids (e.g. `db`, `cache`, `nightly`). The matcher walks the consumer's claim map, finds entries whose `metadata.fqn` matches a `requiredClaims` key, and uses each matched id as the `#resolution` map key.

Example: a `#ManagedDatabaseTransformer` with `requiredClaims: ("opmodel.dev/.../managed-database@v1"): _` matches a consumer's `#claims.db: data.#ManagedDatabaseClaim` entry — the transformer writes `#resolution: db: { host: ..., port: ..., passwordRef: ... }`, and the pipeline injects into `#claims.db.#status`.

### Side-effect-only fulfilment

Claims with no resolution data (e.g. `#BackupClaim` per `08-examples.md` Example 7) may omit `#resolution` entirely. The pipeline simply runs the transformer body, takes `output`, and skips the writeback step. `#status` stays empty by design.

### Failure modes

- **Missing claim id in `#resolution`** when the transformer's `requiredClaims` matched: pipeline warns; consumer reads `#status.X` produce `_|_` and surface as render-time errors against the consumer body. Acceptable v1 behavior.
- **Extra claim id in `#resolution`** that doesn't match any consumer instance: pipeline drops silently. Transformer authors should keep `#resolution` keys in sync with what their `requiredClaims` declares.
- **Cycle / missing-fulfiller**: see "Topological-sort ordering" below.

---

## Topological-sort ordering for `#status` writeback (CL-Q7)

### What CL-D16 specifies

The matcher dispatches fulfillers before consumers. `requiredClaims` declares both a *match* and a *write* edge; `#claims.<id>.#status.<field>` reads inside another transformer's body or a component spec declare the *read* edge. The dependency graph is the topological-sort input.

### What the pipeline must implement

1. **Build the dependency graph** before render dispatch. Nodes = transformer invocations (one per matched instance). Edges = `(writer transformer for FQN X) → (reader transformer / component body that reads #status from a Claim with FQN X)`.
2. **Topologically sort** the nodes. Detect cycles during sort.
3. **Dispatch in topological order**, performing the `#resolution` injection between each transformer and its dependent reader.

### Edge cases

#### Cycle detection

Two transformers each writing a Claim the other reads form a cycle. Example: a hypothetical `TLSCertificateTransformer` that reads `#claims.dns.#status.fqdn` to compute the cert SAN, and a hypothetical `DNSTransformer` that reads `#claims.tls.#status.certName` to publish the cert thumbprint as a TXT record. Neither can fire first.

**Pipeline behavior:** detect cycles at graph-construction time, fail the deploy with a structured error naming the participating Claim FQNs. **Decision needed** before first cycle case: does cycle detection happen at platform-eval time (CUE-side, via a follow-up `#Platform` projection) or at deploy time (Go-side, in the renderer)? The latter is cheaper to implement v1; the former gives fail-fast without invoking the runtime. Lean: **deploy time** — CUE-side detection requires materializing the full transformer graph as a CUE construct, which `#PlatformMatch` doesn't currently do.

#### Missing fulfiller

A consumer reads `#status.X` but no transformer writes it. CUE evaluation of the read produces `_|_`.

**Pipeline behavior:** before the matcher fires anything, scan the consumer Module for `#status` reads against Claims that have no fulfiller in `#composedTransformers` (the `#PlatformMatch.unmatched.claims` projection from 003 already surfaces this). Fail the deploy with a structured error naming the missing fulfiller — *before* any partial render runs, to avoid leaving half-applied state.

#### Multi-fulfiller

Closed by **003 D13** — multi-fulfiller is forbidden at the `#matchers` layer. Status-writeback uniqueness is therefore guaranteed by construction; the pipeline never sees two transformers competing to write the same Claim's `#status`.

### What the pipeline does NOT need to handle

- **Per-FQN writer selection.** D13 forbids multi-fulfiller; selection is unambiguous.
- **Conflict resolution between two `#resolution` for the same instance.** Cannot happen — only one fulfiller per instance per CL-D17 + 003 D13.
- **Cross-Module status reads.** Bundle-level / cross-Module references are out of scope for this enhancement (004 OQ2). Pipeline operates within a single `#ModuleRelease` boundary.

---

## Validation of `#Claim` `#spec` and `#status` (CL-Q3)

### Authoring-time validation (CUE)

CUE unification enforces `#spec` and `#status` shape against the concrete Claim definition. A consumer module that writes `#claims.db: data.#ManagedDatabaseClaim & {#spec: managedDatabase: engine: "snowflake"}` fails at module-eval time because `engine` is constrained to `"postgres" | "mysql" | "mongodb"`. No pipeline involvement.

### Deploy-time validation

When a `#Module` is serialized for matching (e.g. shipped through a `ModuleRelease` CR to `opm-operator`), the platform may unify it again against the registered Claim definitions in `#Platform.#knownClaims` to catch:

- **Spec values that pass authoring-time CUE but violate the platform's registered Claim definition** — e.g. consumer imported v1.0 of `#ManagedDatabaseClaim` but the platform has v1.1 registered with stricter constraints.
- **Status writebacks** from transformers that don't match the registered Claim's `#status` schema.

### Pipeline contract

1. **Before render dispatch**, walk the consumer's `#claims` (module-level + per-component) and unify each entry against the corresponding Claim definition from `#Platform.#knownClaims` (lookup by `metadata.fqn`). Failure ⇒ structured error in `ModuleRelease.status.conditions`.
2. **After each transformer fires**, before injecting `#resolution`, unify the writeback values against the Claim's `#status` schema. Failure ⇒ structured error naming the offending writer + field.

### Version evolution

A vendor that ships v1.1 of `#VectorIndexClaim` with additional `#status` fields must guarantee old consumer reads still resolve. Two cases:

- **Additive `#status` change** (new field): old consumers don't read the new field, no break.
- **Breaking `#status` change** (removed / renamed field): old consumers reading the old field name produce `_|_` at deploy time. Pipeline reports unmatched-status-field error. Vendor versioning policy is the lever.

The same logic applies to `#spec` evolution. A vendor introducing a stricter `#spec` field (e.g. `engine: "postgres" | "mysql"` → `engine: "postgres"`) breaks existing consumers; the platform reports the validation failure at deploy time and the consumer must pin an older Claim version or update.

### Out of scope

- A semver-style version-compatibility policy at the Claim level (e.g. "minor versions are additive only"). Recommended convention but not enforced by the schema. **Revisit:** first version evolution by a real ecosystem participant (CL-Q3 retains this trigger).
- A Claim-version-pinning mechanism on `#Module.#claims`. Today the consumer pins by importing a specific CUE package version; no schema-level pin needed.

---

## Implementation order

When the Go pipeline is built (in `cli/` or `opm-operator/`):

1. **Status-writeback channel** — pin `#resolution` field name in `#ComponentTransformer.#transform` / `#ModuleTransformer.#transform` (CUE schema), implement the `FillPath` injection step in the renderer.
2. **Missing-fulfiller pre-scan** — implement the unmatched-claim error before render dispatch (consumes `#PlatformMatch.unmatched.claims`).
3. **Topological dispatch** — implement the dependency graph + topological sort + cycle detection.
4. **Deploy-time `#spec` / `#status` validation** — implement the platform-side unification step.
5. **Status-injection-then-rerun loop** — handle the case where a downstream transformer reads `#status` written by an upstream transformer.

Each step is independently testable. A spec-only delivery (steps 1, 2, 4) gives a working render for non-cross-claim-reading Modules; steps 3 + 5 unlock the dependency-chain case.

---

## Open Decisions

These are pipeline-implementation decisions deferred until first use case:

- **Cycle detection phase** — CUE-side at platform-eval, or Go-side at deploy? Lean: deploy-side (cheaper v1).
- **Missing-fulfiller error format** — single aggregated error vs. per-claim errors. Lean: per-claim, surfaced in `ModuleRelease.status.conditions` for visibility.
- **`#resolution` shape** — hidden field `#resolution` (current sketch) vs. export-visible `resolution` vs. nested under `#transform.output` as a sibling `#status`. Lean: `#resolution` (hidden) — keeps the CUE-time `#transform` body shape symmetrical with the existing `output` field and excludes status writes from rendered manifests.
