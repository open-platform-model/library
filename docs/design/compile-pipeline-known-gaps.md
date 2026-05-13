# Compile Pipeline — Known Gaps (RESOLVED)

Both findings below are fixed. This document is kept for the historical record — the analysis, the reproduction steps, and the fix discussion remain useful context for future work on the matcher / executor.

Two correctness gaps in the Match → Compile path, surfaced by the on-disk integration fixture (`testdata/modules/web_app` + `modules/opm_platform`) and the inspection harness `cmd/flow-inspect`. Both block end-to-end deployment via the canonical opm transformers; one of them (Finding 2) breaks every render call regardless of fixture, the other (Finding 1) breaks any component that triggers more than one transformer at the same required-resource FQN.

This document captures the observed behaviour, the source-line evidence, and the fix surface so the remediation work has a single anchor instead of re-deriving the analysis from scratch.

Sources cited throughout:

- `opm/compile/match.go` — runtime matcher
- `opm/compile/execute.go` — render dispatch
- `apis/core/v1alpha2/platform.cue` — `#PlatformBase` schema (computed views, `#matchers`, `_predicateSignature`, `_invalid`)
- `modules/opm/transformers/*.cue` — opm `#ComponentTransformer` bodies
- `cmd/flow-inspect/main.go` — manual inspection harness
- `opm/kernel/flow_integration_test.go` — passing integration test that pins the working subset of behaviour

---

## Finding 1 — Matcher reports `candidateAmbiguous` when distinct-predicate transformers share a required resource FQN (RESOLVED)

**Status:** Fixed. The matcher no longer treats multi-candidate lookups as a failure mode. `opm/compile/match.go`'s `lookupCandidate` was replaced by `lookupCandidates` returning `[]string`; the resource and trait demand walks iterate the slice and pair every transformer whose predicate is satisfied. `MatchPlan.Ambiguous`, `CompileResult.Ambiguous`, `PlanResult.Ambiguous`, and the `candidateAmbiguous` enum value were removed. The schema's `_predicateSignature`, `_invalid`, and `_noMultiFulfiller` projections were deleted from `apis/core/v1alpha2/platform.cue`, and `#PlatformBase`/strict `#Platform` collapsed into a single `#Platform` definition. Genuine collisions (two transformer definitions claiming the same transformer FQN with divergent bodies) are caught upstream by CUE map unification on `#composedTransformers`. The two negative fixtures (`multi_fulfiller_fixture.cue`, `multi_fulfiller_traits_fixture.cue`) were deleted; the positive fixtures (`predicate_distinct_labels_fixture.cue`, `platform_matchers_fixture.cue`, `trait_matchers_fixture.cue`, `enabled_false_suppresses_fixture.cue`) had their `_invalid` / `_noMultiFulfiller` expectations dropped. The `opm/helper/platform` package lost `*MultiFulfillerError` and `classifyMultiFulfiller`; `Compose` now returns raw CUE diagnostics for genuine collisions. Test `compile.TestMatch_TwoTransformersPairBoth` (formerly `TestMatch_AmbiguousFQN`) pins the new behaviour: two transformers requiring the same resource FQN, both satisfying their predicate, both surface as `MatchedPairs`. The historical analysis below is preserved for context; the framing "distinct predicates ⇒ disjoint match domains" was wrong (distinct signatures can both be satisfied by the same component), but the user's clarification — that the schema's `_invalid` check is solving the wrong problem and CUE map unification handles the genuine collision — closed the issue cleanly.

### Symptom

A component carrying both a Container resource and an Expose trait does not pair with `deployment-transformer`. The matcher tags `opmodel.dev/modules/opm/resources/container@v1` as ambiguous and skips the lookup; only `service-transformer` survives, paired via the `expose@v1` *trait* lookup. End-to-end consequence: the Service renders but the Deployment does not — a stateless workload with a Service is indistinguishable, from the matcher's perspective, from "Deployment is ambiguous, drop it".

The opm catalog deliberately overlaps `requiredResources["…/container@v1"]` across six transformers (deployment, statefulset, daemonset, job, cronjob, service — see the inspector's `#matchers.resources` dump in *Reproduction* below), so this is not a hypothetical fixture quirk. It fires whenever a real component attaches a workload-type plus the Expose trait.

### Source

`opm/compile/match.go:177-227` — `lookupCandidate` filters the bucket by `candidateSatisfied` (per-transformer predicate evaluation) and then collapses by `len(survivors)`:

```go
switch len(survivors) {
case 0:
    return "", candidateMissing
case 1:
    return survivors[0], candidateFound
default:
    return "", candidateAmbiguous
}
```

Any bucket where two or more transformers' predicates pass becomes `candidateAmbiguous` and the demanding component never paired with any of them via this FQN. The matcher continues — other FQN lookups (e.g. the trait demand walk at `match.go:117-129`) can still yield matches — but the ambiguity surface forfeits all candidates instead of resolving them.

The schema layer at `apis/core/v1alpha2/platform.cue:142-160` already computes a `_predicateSignature` for every transformer (`labelPart;traitPart`, deterministic and string-comparable) and `platform.cue:167-182` uses it to flag FQNs where *two transformers share the same signature* — those are the genuine collisions. Distinct signatures (deployment's `core.opmodel.dev/workload-type=stateless;` vs. service's `;opmodel.dev/modules/opm/traits/network/expose@v1`) are explicitly accepted as legal — see the inline comment at `platform.cue:162-166`:

> Shared FQNs across candidates with *different* predicates are fine (e.g. all workload transformers require container@v1 but each is gated by a unique workload-type label, so no Component can match more than one).

The runtime matcher in `match.go:210-217` does not consult that signature. It collapses by candidate count alone, so the schema's "distinct predicates ⇒ legal" invariant is lost.

### Reproduction

The platform's reverse index (taken verbatim from `flow-inspect` Stage 2, `#matchers.resources` section):

```
- opmodel.dev/modules/opm/resources/container@v1
    → opmodel.dev/modules/opm/transformers/deployment-transformer@v1
    → opmodel.dev/modules/opm/transformers/statefulset-transformer@v1
    → opmodel.dev/modules/opm/transformers/daemonset-transformer@v1
    → opmodel.dev/modules/opm/transformers/job-transformer@v1
    → opmodel.dev/modules/opm/transformers/cronjob-transformer@v1
    → opmodel.dev/modules/opm/transformers/service-transformer@v1
```

Add the Expose trait back to the `web` component in `testdata/modules/web_app/components.cue` (i.e. attach `tr_network.#ExposeTrait`, populate `spec.expose`, and re-publish if needed). Re-run `task cue:test:flow:inspect`. Predicate evaluation for the now-stateless-and-exposed component:

| Transformer       | requiredLabels                                  | requiredResources    | requiredTraits        | Satisfied? |
| ----------------- | ----------------------------------------------- | -------------------- | --------------------- | ---------- |
| deployment        | `core.opmodel.dev/workload-type=stateless`      | `container@v1`       | —                     | ✅         |
| statefulset       | `core.opmodel.dev/workload-type=stateful`       | `container@v1`       | —                     | ❌ label   |
| daemonset         | `core.opmodel.dev/workload-type=daemon`         | `container@v1`       | —                     | ❌ label   |
| job               | `core.opmodel.dev/workload-type=task`           | `container@v1`       | `job-config@v1`       | ❌ label   |
| cronjob           | `core.opmodel.dev/workload-type=scheduled-task` | `container@v1`       | `cron-job-config@v1`  | ❌ label   |
| service           | —                                               | `container@v1`       | `expose@v1`           | ✅         |

`survivors = [deployment, service]` ⇒ length 2 ⇒ `candidateAmbiguous`. Result in `MatchPlan`: `Ambiguous: ["…/container@v1"]`, `MatchedPairs: [(web, service-transformer)]` — service slipped through because the trait-demand walk at `match.go:118-129` looks up `expose@v1` in a single-candidate bucket, but deployment is dropped.

The current fixture removes the Expose trait specifically to dodge this; see the comment in `testdata/modules/web_app/components.cue:19-25`.

### Fix direction

`lookupCandidate` should compute the predicate signature of each survivor (the same way `apis/core/v1alpha2/platform.cue:142-160` constructs `labelPart;traitPart`) and treat survivors as ambiguous only when *two share a signature*. Distinct-signature survivors are by definition non-overlapping match domains — whichever predicate the component satisfied uniquely is the answer.

A second-best alternative is for the runtime to read `#matchers._invalid` directly: the schema already pre-computes the genuine collisions, and the matcher could trust that and pair the unique component-side survivor without recomputing the signature on every lookup. This couples the matcher to a CUE-side projection rather than recomputing in Go, but it skips the duplicated logic.

Either path needs a regression fixture pinning the dual-transformer case (web component matches both deployment and service simultaneously) so the trade-off does not regress the `platform_matchers_projects_single_candidate` invariant in `apis/core/v1alpha2/testdata/`.

---

## Finding 2 — `executePair` shreds single-resource transformer outputs into per-field `Compiled` items (RESOLVED)

**Status:** Fixed in two iterations.

The first pass treated every `#transform.output` as a single resource, citing the schema text at `apis/core/v1alpha2/transformer.cue:74,81` ("IMPORTANT: output must be a single resource"). That fixed the singleton transformers (Deployment, Service, …) but broke the five plural transformers in the opm catalog that legitimately emit N resources — `configmap_transformer.cue`, `secret_transformer.cue`, `pvc_transformer.cue`, `crd_transformer.cue`, `role_transformer.cue`. Each formerly used a map-keyed-by-name output (`output: { "<name>": k8s.#Foo & {...} }`) which under the singleton-only renderer collapsed into one `Compiled` whose value was a struct of K8s resources, indistinguishable to the apply layer.

The second pass introduced kind-based dispatch in `opm/compile/execute.go`:
- `cue.StructKind` → one `Compiled` per (component, transformer) pair; `Compiled.Value` is the whole struct.
- `cue.ListKind` → one `Compiled` per list item; each item is one resource.

The renderer never inspects fields inside the value; apply-layer code is responsible for interpreting the resource shape. Schema text at `transformer.cue:81` was updated to `output: {...} | [...{...}]`. The five plural transformers were converted from map output to list-comprehension output; map keys (formerly the K8s resource name) now live inside each resource as `metadata.name`, removing duplication. A `config` component carrying two `#ConfigMaps` entries was added to `testdata/modules/web_app/components.cue` to pin the list-dispatch path in the integration test (`flow_integration_test.go` asserts `seenTransformers["configmap-transformer"] == 2` for the 2-entry fixture).

`flow-inspect` Stage 4 confirms the expected output: web's deployment + service render as one Compiled each (struct outputs), and config's two ConfigMaps render as two Compiled items from a single transformer pair (list output). The historical analysis below is preserved for context but reflects the first-pass framing only.

### Symptom

Every opm transformer in `modules/opm/transformers/` emits a single Kubernetes resource as the `output` field of `#transform`. The compile phase produces one `*core.Compiled` per *field* of that struct rather than one per resource. A Deployment becomes four `Compiled` entries: the `apiVersion` string `"apps/v1"`, the `kind` string `"Deployment"`, the `metadata` substruct, and the `spec` substruct. None of those is a deployable manifest — apply-time reassembly is impossible because `Compiled` carries no field-name back-reference.

### Source

The transformer side (`modules/opm/transformers/deployment_transformer.cue:107-119`):

```cue
output: k8sappsv1.#Deployment & {
    apiVersion: "apps/v1"
    kind:       "Deployment"
    metadata: { … }
    spec: { … }
}
```

`output` is a struct whose root *is* the Deployment — no list, no named-key wrapper. Every transformer in `modules/opm/transformers/` follows this shape: `service_transformer.cue:71`, `statefulset`, `daemonset`, `job`, `cronjob`, `configmap_transformer.cue`, etc.

The dispatch side (`opm/compile/execute.go:136-157`):

```go
// Decode the output into rendered items. Two supported forms:
//   1. List of items   — cue.ListKind
//   2. Map of items    — cue.StructKind keyed by stable id
//
// Singleton outputs MUST wrap themselves in a one-element list. The
// pipeline does not auto-detect singletons because the heuristic
// ("apiVersion + kind at root") is k8s-shape-specific and would
// misclassify outputs of other platforms (compose service, nomad job,
// terraform resource, ...).
switch outputVal.Kind() {
case cue.ListKind:
    res, err := collectCompiledList(outputVal, releaseName, compName, tfFQN)
    return res, warnings, err
case cue.StructKind:
    res, err := collectCompiledMap(outputVal, releaseName, compName, tfFQN)
    return res, warnings, err
…
}
```

`collectCompiledMap` (`execute.go:179-195`) iterates `outputVal.Fields()` and emits one `Compiled{Value: iter.Value(), …}` per field. For a Deployment that means four `Compiled` items: `"apps/v1"`, `"Deployment"`, `{ name, namespace, labels }`, `{ replicas, selector, template }`.

The contract `execute.go:140-144` documents — "Singleton outputs MUST wrap themselves in a one-element list" — is violated by every opm transformer in the catalog. Neither the contract nor the renderer enforces it: there is no validation hook that rejects a struct whose root carries `apiVersion`+`kind`, and there is no test that pins the expected per-pair Compiled shape.

### Reproduction

`task cue:test:flow:inspect STAGES=plan` against the current `web_app` fixture. Stage 4 prints (verbatim, abridged):

```
--- CompileResult.Compiled (4 items) ---

--- Compiled[0]: component=web transformer=…/deployment-transformer@v1 ---
  "apps/v1"

--- Compiled[1]: component=web transformer=…/deployment-transformer@v1 ---
  "Deployment"

--- Compiled[2]: component=web transformer=…/deployment-transformer@v1 ---
  { labels: {…}, name: "web-app-demo-web", namespace: "default" }

--- Compiled[3]: component=web transformer=…/deployment-transformer@v1 ---
  { replicas: 2, selector: {…}, template: {…} }
```

Four `Compiled` per transformer call, one per field of the Deployment. A consumer that calls `kubectl apply -f` on each value individually deploys nothing — the apiVersion string is not a manifest, the metadata struct is not a manifest, and so on.

The integration test at `opm/kernel/flow_integration_test.go` passes because its assertions check `Compiled` *provenance* (Component + Transformer fields are populated) rather than `Compiled.Value` shape. The test author flagged this gap explicitly — see the comment at `flow_integration_test.go:217-221`:

> the renderer's exact output shape is governed by the transformers under test (see execute.go's StructKind branch).

### Fix direction

Two viable fixes; they are not equivalent.

**Option A — wrap on the transformer side.** Every transformer in `modules/opm/transformers/` rewrites `output: k8sDef & {…}` to `output: [k8sDef & {…}]` (or to a named-map form `output: deployment: k8sDef & {…}`). `execute.go` is unchanged. The pipeline becomes correct immediately. Cost: 12 transformer files edited and the catalog re-published. A schema-level constraint on `#ComponentTransformer.#transform.output` (require `[…]` or `[name=string]: {…}`) prevents the regression from re-introducing itself. Best long-term shape; matches the documented contract at `execute.go:140-144`.

**Option B — auto-wrap on the dispatch side.** `executePair` adds a third branch: when `outputVal.Kind() == cue.StructKind` and the value carries `apiVersion` + `kind` at the root, treat it as a single resource and emit one Compiled. The contract comment at `execute.go:141-144` rejects this exact heuristic ("would misclassify outputs of other platforms (compose service, nomad job, terraform resource, ...)"), so adopting it requires either accepting that misclassification cost or extending the heuristic into a binding-supplied predicate (`api.Binding.IsSingletonOutput(cue.Value) bool`). Cost: one branch in `executePair` plus a per-binding hook; zero transformer-side churn. Trade-off: the "what counts as a single resource" decision migrates from the schema into Go, distributed across binding implementations.

Both options need a Compile-side regression fixture that pins the expected `Compiled` shape (one per K8s manifest, with `apiVersion + kind` reachable at `Compiled.Value`'s root). The current `opm/kernel/flow_integration_test.go` only checks provenance.

Option A is the cleaner fix because the catalog of opm transformers is small (12 files) and the contract was explicit before the implementation drifted. Option B preserves drift while adding plumbing that future binding authors must reproduce.

---

## Cross-references

- `cmd/flow-inspect/main.go` — manual inspection harness; Stage 2's `#matchers.resources` dump materialises Finding 1's bucket layout, Stage 4's `CompileResult.Compiled` dump materialises Finding 2's per-field Compiled shape.
- `opm/kernel/flow_integration_test.go` — passing integration test; the working subset (single-transformer flow with provenance-only Compiled assertions) is exactly the slice that dodges both findings.
- `testdata/modules/web_app/components.cue:19-25` — comment recording why Expose was removed from the fixture (Finding 1 workaround).
- `apis/core/v1alpha2/testdata/predicate_distinct_labels_fixture.cue` — schema-side fixture proving `_invalid` accepts distinct-predicate buckets; the runtime matcher needs to honour the same invariant.
- `apis/core/v1alpha2/testdata/multi_fulfiller_fixture.cue` — schema enforcement of D13 (multi-fulfiller forbidden when *predicates match*); Finding 1's fix should preserve this guard while accepting the distinct-predicate case.
