# Module-Scope Transformer + Status Writeback

| Field       | Value      |
| ----------- | ---------- |
| **Status**  | Draft      |
| **Created** | 2026-04-30 |
| **Revised** | 2026-05-02 — split: `#ComponentTransformer` schema + matcher algorithm moved to [003/05-component-transformer-and-matcher.md](../003-platform-construct/05-component-transformer-and-matcher.md). This doc now covers only the 005 extension. |
| **Targets** | `apis/core/v1alpha2/transformer.cue` |

## Why this doc exists

[003 D17](../003-platform-construct/04-decisions.md) introduces `#ComponentTransformer` — fires once per matching `#Component`, match keys against a single component, no Claim awareness. That covers Resources/Traits dispatch.

005 extends transformer rendering in three ways:

1. **Module-scope renders.** Some commodity claims live at module level (one-per-Module: DNS hostname, workload identity, mesh tenant, backup orchestration — see CL-D10). A transformer that fulfils one of these fires once per Module, not once per Component. `#ComponentTransformer`'s match keys (which read against a single `#Component`) cannot express that.
2. **Dual-scope renders.** Some module-level claims need data from per-component traits (e.g. K8up `#BackupClaim` orchestration reads each component's `#BackupTrait`). The transformer fires once per Module but iterates the matched Module's components in its body.
3. **Claim fulfilment + status writeback.** Whichever transformer fulfils a Claim writes that Claim's `#status` (CL-D15 / CL-D16) so consumers can read fulfiller-agnostic resolution data.

This doc introduces `#ModuleTransformer` (a sibling primitive to 003's `#ComponentTransformer`), widens `#TransformerMap` and `#ComponentTransformer` for Claims, and pins the `#resolution` channel.

## Runtime guarantee

The runtime guarantee is the same one 003 establishes (D18): every `#transform` invocation receives a fully concrete `#ModuleRelease`. `#ModuleTransformer` bodies index into `#moduleRelease.#components` directly when dual-scope work is needed — no pre-filtered map.

## Design constraints recap

- **CL-D2**: `#Resource` and `#Claim` stay separate primitives.
- **CL-D10**: same `#Claim` primitive, two scopes (component or module).
- **CL-D11**: `#Resource` / `#Trait` stay component-only — never module-level.
- **DEF-D1 / DEF-D2 / DEF-D3**: transformers ship through `#defines.transformers` keyed by FQN.
- **003 D17**: `#ComponentTransformer` is the sole transformer primitive at the 003 layer.
- **TR-D5 (this doc)**: introduce `#ModuleTransformer` as a sibling primitive; widen `#TransformerMap` to `#ComponentTransformer | #ModuleTransformer`; add `requiredClaims` / `optionalClaims` to `#ComponentTransformer`. `requiresComponents` on `#ModuleTransformer` carries the dual-scope pre-fire gate.

## Schema — `#ComponentTransformer` widening

005 adds Claim match keys to 003's `#ComponentTransformer`:

```cue
// apis/core/v1alpha2/transformer.cue (extension over 003)

#ComponentTransformer: {
    // ... (003's apiVersion / kind / metadata / requiredLabels / requiredResources /
    //      requiredTraits / readsContext / producesKinds / #transform unchanged) ...

    // New in 005 — match keys for component-level Claim fulfilment.
    requiredClaims?: [FQN=string]: _
    optionalClaims?: [FQN=string]: _
}
```

A `#ComponentTransformer` whose `requiredClaims` includes a Claim FQN is the fulfiller for any matching component-level `#claims.<id>` instance with that FQN.

## Schema — `#ModuleTransformer`

Fires **once per `#Module`** that satisfies the match. Match keys read against `#Module` top level. The render body receives the `#ModuleRelease` and walks `#moduleRelease.#components` itself when dual-scope work is needed.

```cue
#ModuleTransformer: {
    apiVersion: "opmodel.dev/core/v1alpha2"
    kind:       "ModuleTransformer"

    metadata: {
        modulePath!: #ModulePathType
        version!:    #MajorVersionType
        name!:       #NameType
        #definitionName: (#KebabToPascal & {"in": name}).out
        fqn: #FQNType & "\(modulePath)/\(name)@\(version)"
        description!: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    // Match keys — read against #Module top level.
    // Only Claims and module labels — Resources/Traits are component-only by CL-D11.
    requiredLabels?: #LabelsAnnotationsType
    optionalLabels?: #LabelsAnnotationsType
    requiredClaims?: [FQN=string]: _   // module-level Claims
    optionalClaims?: [FQN=string]: _

    // Pre-fire gate for dual-scope renders (TR-D7).
    // Declares "I expect at least one component carrying X to exist."
    // The matcher checks this before firing; the body still iterates
    // #moduleRelease.#components itself to do the actual work.
    requiresComponents?: {
        resources?: [FQN=string]: _
        traits?:    [FQN=string]: _
        claims?:    [FQN=string]: _
    }

    readsContext?:  [...string]
    producesKinds?: [...string]

    #transform: {
        #moduleRelease: _              // fully concrete #ModuleRelease (003 D18)
        #context:       #TransformerContext

        output: {...}
    }
}
```

## Publication slot — union widening

005 widens 003's `#TransformerMap` to the union:

```cue
#TransformerMap: [#FQNType]: #ComponentTransformer | #ModuleTransformer
```

`#Module.#defines.transformers` is keyed by FQN with the same FQN-binding constraint as the other `#defines` sub-maps:

```cue
transformers?: [FQN=string]: (#ComponentTransformer | #ModuleTransformer) & {
    metadata: fqn: FQN
}
```

The matcher (003's algorithm — see [003/05-component-transformer-and-matcher.md](../003-platform-construct/05-component-transformer-and-matcher.md)) gains a `kind` dispatch when this widening lands: `#ComponentTransformer` fans out per matching component (003's `matchAndRender`); `#ModuleTransformer` checks `satisfiesModule` once per Module, then optionally `anyComponentMatches` for the `requiresComponents` gate.

### `satisfiesModule`

```text
function satisfiesModule(moduleRelease, t) -> bool:
    for (k, v) in t.requiredLabels or {}:
        if moduleRelease.metadata.labels.get(k) != v: return False
    for fqn in t.requiredClaims or {}:
        if not anyClaimWithFQN(moduleRelease.#claims or {}, fqn): return False
    return True
```

### `anyComponentMatches` (gate for `requiresComponents`)

```text
function anyComponentMatches(moduleRelease, rc) -> bool:
    for cmp in moduleRelease.#components.values():
        ok = True
        for fqn in rc.resources or {}:
            if fqn not in fqnsOf(cmp.#resources): ok = False; break
        if not ok: continue
        for fqn in rc.traits or {}:
            if fqn not in fqnsOf(cmp.#traits or {}): ok = False; break
        if not ok: continue
        for fqn in rc.claims or {}:
            if not anyClaimWithFQN(cmp.#claims or {}, fqn): ok = False; break
        if ok: return True
    return False
```

A `#ModuleTransformer` whose `requiresComponents` finds zero matches **does not fire** and the platform reports an unfulfilled dual-scope render. This turns a misconfiguration into a deploy-time error rather than a vacuous output.

`anyClaimWithFQN(claims, fqn)` returns true iff some entry in `claims` has `metadata.fqn == fqn`. Matching is FQN-equality (CL-D4 / DEF-D2) — the spec is the payload, not the match key.

## Status writeback to `#Claim` instances

`#Claim` carries an open `#status?` field (CL-D15). The transformer that fulfils a Claim writes that Claim's `#status` as part of its render. The split lifecycle:

1. **Match.** Matcher walks `#composedTransformers`. A transformer whose `requiredClaims` contains a Claim FQN is the fulfiller for that Claim instance.
2. **Render.** Transformer body runs — `#transform.output` produces the provider-specific resource(s); a sibling `#transform.#resolution` carries per-claim status data.
3. **Inject.** The Go pipeline reads `#resolution` and injects values via `FillPath` into the matched `#Claim` instance's `#status`. Same Strategy B precedent as 004 D12 hash injection.
4. **Consume.** Downstream transformers / component bodies that read `#claims.<id>.#status.<field>` see the populated values. The matcher topologically orders fulfillers before consumers — see [`12-pipeline-changes.md`](12-pipeline-changes.md).

### Schema — `#resolution`

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

A `#PublicEndpointTransformer` that fulfils `net.#PublicEndpointClaim` would emit:

```cue
#resolution: (claimIdForFqn): {
    url:  "https://\(_claim.#spec.publicEndpoint.hostname).\(#context.runtime.route.domain)"
    fqdn: "\(_claim.#spec.publicEndpoint.hostname).\(#context.runtime.route.domain)"
}
```

— and the runtime injects that map into `#claims.<id>.#status` before downstream code reads it.

### Cross-runtime portability

`#status` is the cross-runtime portability surface (CL-D15). The same `#Module` deployed against different fulfilling transformers receives target-appropriate resolution data without changing the consumer's code path. A `#PublicEndpointClaim` resolved by a k8s Gateway transformer or by a compose Traefik-label transformer both write `#status.url`; the consumer reads the same field on both runtimes. This is what makes 003 D9 (per-runtime transformer Modules) work — each runtime registers its own fulfilling transformer for the same Claim FQN, and the consumer Module is unchanged.

### Side-effect-only claims

Claims fulfilled purely by side-effect (e.g. backup orchestration — see Example 7 in `08-examples.md`) may omit `#resolution` entirely. Their `#status` stays empty by design; consumers do not read resolution data because there is none. The schema does not require `#resolution` for any fulfiller — the field is optional, mirroring `#Claim.#status?`.

## Worked module-scope example — Hostname

```cue
#HostnameTransformer: transformer.#ModuleTransformer & {
    metadata: { ... }

    requiredClaims: (platform.#HostnameClaim.metadata.fqn): _
    // No requiresComponents — pure module-scope render.

    readsContext:  ["dns.zones"]
    producesKinds: ["externaldns.k8s.io/v1.DNSEndpoint"]

    #transform: {
        #moduleRelease: _
        #context:       #TransformerContext

        // Look up the claim instance that matched.
        let claim = #moduleRelease.#claims[
            for k, v in #moduleRelease.#claims
            if v.metadata.fqn == platform.#HostnameClaim.metadata.fqn { k }
        ][0]

        output: {
            apiVersion: "externaldns.k8s.io/v1"
            kind:       "DNSEndpoint"
            spec: { hostname: claim.#spec.hostname.fqdn, ... }
        }
    }
}
```

Fires once per Module carrying a `#HostnameClaim`. No per-component fan-out.

## Worked dual-scope example — K8up backup

The K8up `#BackupScheduleTransformer` (Example 7 in `08-examples.md`):

```cue
#BackupScheduleTransformer: transformer.#ModuleTransformer & {
    metadata: {
        modulePath:  "opmodel.dev/k8up/v1alpha2/transformers"
        version:     "v1"
        name:        "backup-schedule-transformer"
        description: "Renders #BackupClaim + per-component #BackupTrait → K8up Backend + Schedule CRs"
    }

    // Module-level orchestration claim.
    requiredClaims: (backup.#BackupClaim.metadata.fqn): _

    // Pre-fire gate — refuse to fire if no component carries the trait.
    requiresComponents: traits: (backup.#BackupTrait.metadata.fqn): _

    readsContext:  ["backup.backends"]
    producesKinds: ["k8up.io/v1.Backend", "k8up.io/v1.Schedule"]

    #transform: {
        #moduleRelease: _
        #context:       #TransformerContext

        // Body walks components itself, filtering inline for the trait.
        let _bearers = {
            for name, cmp in #moduleRelease.#components
            if cmp.#traits != _|_
            if cmp.#traits[backup.#BackupTrait.metadata.fqn] != _|_ {
                (name): cmp
            }
        }

        let _claim = #moduleRelease.#claims[
            for k, v in #moduleRelease.#claims
            if v.metadata.fqn == backup.#BackupClaim.metadata.fqn { k }
        ][0]

        output: {
            backend: { ... uses _claim.#spec.backup.backend + #context.platform.backup.backends ... }
            schedule: {
                apiVersion: "k8up.io/v1"
                kind:       "Schedule"
                spec: {
                    schedule: _claim.#spec.backup.schedule
                    podSelector: matchExpressions: [{
                        key: "app.kubernetes.io/name"
                        operator: "In"
                        values: [for name, _ in _bearers { name }]
                    }]
                    // ... retention from _claim.#spec.backup.retention ...
                }
            }
        }
    }
}
```

Match flow on the Strix media Module:

1. `kind == "ModuleTransformer"` → check `satisfiesModule`.
2. `requiredClaims` check passes — `nightly` claim has the right FQN.
3. `requiresComponents.traits` gate — `anyComponentMatches` returns true (`app` and `db` both carry `#BackupTrait`).
4. `#transform` fires once. Body's `_bearers` comprehension picks up `app` and `db`. Output emits one Backend and one Schedule referencing both.

If the Module satisfied the claim but no component carried `#BackupTrait`, step 3 returns false and the transformer is skipped — the platform reports an unfulfilled dual-scope render at deploy time.

## What changed from the earlier scope-bucket design

| Old (superseded) | New (003 + 005 split) |
|---|---|
| Single `#Transformer` with `componentMatch` / `moduleMatch` buckets | Two primitives: `#ComponentTransformer` (003) + `#ModuleTransformer` (005) |
| Derived `_scope` field | Removed — type identity carries the scope |
| `#transform.#components: [string]: _` (singleton / multi / empty) | `#component` (singular, on `#ComponentTransformer`) or absent (on `#ModuleTransformer`) |
| Pre-filtered map of components for dual-scope | Body walks `#moduleRelease.#components` itself; `requiresComponents` is a pre-fire gate, not a filter |
| `#defines.transformers: [FQN]: #Transformer` | `#defines.transformers: [FQN]: #ComponentTransformer \| #ModuleTransformer` (003 introduces the keyed shape; 005 widens the value type) |

The runtime guarantee (always-concrete `#ModuleRelease`, 003 D18) is what makes the simpler shape work: bodies that need cross-component data iterate `#moduleRelease.#components` rather than receiving a pre-filtered map.

## Migration impact (modules/opm)

When the well-known catalog is rebuilt under v1alpha2 (003's migration table covers component-scope transformers; 005 adds the module-scope and dual-scope rows):

| Source | Change |
|---|---|
| New module-scope transformers (Hostname, ExternalDNS, etc.) | Author as `#ModuleTransformer` |
| K8up backup, cert-manager, Gateway-API routing | Author as `#ModuleTransformer` with `requiresComponents` |

`cue vet` will flag any transformer that does not match either `#ComponentTransformer` or `#ModuleTransformer`.

## Open questions

### TR-Q4 — `requiresComponents` granularity (was Q17)

`requiresComponents` is a single conjunction (resources AND traits AND claims). A transformer that wants "components carrying `#BackupTrait` *or* components with a backup-tagged volume" cannot express that. Disjunctive gates may eventually want their own shape; deferred until a real case appears.

**Revisit trigger**: first transformer that needs a disjunctive gate.

### CL-Q7 — Status writeback ordering (was Q18)

A `#ComponentTransformer` (or `#ModuleTransformer`) that fulfils a Claim writes `#status` via `#resolution`. A second transformer — or a component body — reads the same Claim's `#status` to wire connection data. The matcher must dispatch fulfillers before consumers; the dispatch order is the topological sort of an FQN-graph derived from `requiredClaims` (write edges) and `#claims.<id>.#status.<field>` reads (read edges).

Open sub-questions:

- **Cycle detection.** Two transformers that each write a Claim the other reads form a cycle. Should the matcher detect this at platform-evaluation time (CUE-time) or at deploy-time (Go-pipeline)?
- **Missing fulfiller.** If a consumer reads `#status.X` but no transformer writes it, what is the deploy-time signal? `_|_` from CUE? An explicit unmatched-claim error from the matcher?
- **Multi-fulfiller.** Closed by **003 D13** — multi-fulfiller is forbidden at the `#matchers` layer; platform evaluation fails when two transformers' `requiredClaims` overlap on the same FQN. Status-writeback uniqueness is therefore guaranteed by construction; no per-instance selection logic needed in the writeback phase.

**Revisit trigger**: pipeline implementation, or first author hitting a cycle/missing-fulfiller case. Pipeline contracts live in [`12-pipeline-changes.md`](12-pipeline-changes.md).

## Decisions referenced

(Live in `10-decisions.md` under the TR- and CL- prefixes.)

- **TR-D5** (was D28): two transformer primitives. 003 D17 owns the `#ComponentTransformer` half; this enhancement introduces `#ModuleTransformer` and widens `#TransformerMap` to the union. `#defines.transformers` accepts the union.
- **TR-D6** (was D29): runtime always passes a fully concrete `#ModuleRelease` to `#transform`. **Lifted to 003 D18** as part of the 003 / 005 untangle (2026-05-02) since the guarantee applies to both transformer kinds equally.
- **TR-D7** (was D30): `#ModuleTransformer.requiresComponents` is a pre-fire gate, not a filter. Body iterates `#moduleRelease.#components` itself.
- **CL-D15** (was D31): `#Claim` gains `#status?`. Transformer-written resolution surface; concrete claims pin a `#status` schema (or omit it for side-effect-only fulfilment).
- **CL-D16** (was D32): `#status` injection follows 004 D12 (Strategy B / Go pipeline). Matcher topologically orders fulfillers before consumers; `#resolution` is the channel sketched above.
