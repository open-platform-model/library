# Transformer `output`-local hidden fields don't resolve after Materialize + FillPath (RESOLVED)

**Status:** RESOLVED structurally by `federate-materialize-transformers` (§14); the earlier kernel
fix (§13, the open `Composed` escape hatch) is superseded. The defect is a **CUE Go-API
closedness/structure-sharing bug**, NOT a CUE *evaluator* bug and NOT our schema. It was triggered by
exactly one line — `materialize.go`'s `p.Package.FillPath(schema.ComposedTransformers, composed)`,
which `FillPath`s the composed map into the **closed, separately-`BuildInstance`'d `c.#Platform`**.
Federation removes that line entirely: the composed map and matcher index are exposed as native
first-class fields (`MaterializedPlatform.Transformers` / `Matchers`) built in the owner context and
**never filled onto the closed platform**, so the closed twin is never built and there is no surface
from which reading a `#transform` corrupts (ADR-003). `Composed` and `Package` are removed. Catalog
workaround (§7) is now defence-in-depth. See §11 (pure-CUE control: not CUE), §12 (root cause),
§13 (the superseded interim fix), and **§14 (the structural resolution)**.
**Severity:** High — breaks *every* container workload render (Deployment / StatefulSet / DaemonSet /
Job / CronJob) until the catalog is patched.
**First observed:** 2026-06-13, rendering `testdata/modules/web_app` on `modules/opm_platform` via the
operator and via `TestFlow_WebApp_OnOpmPlatform`.

This document is a handoff for a later agent to fix the underlying behaviour (or to convert it into a
documented, enforced authoring constraint). The symptom was worked around in `catalog_opm`; the kernel
still mis-evaluates the pattern.

> **2026-06-14 investigation update.** Reproduced deterministically through the real kernel path and
> narrowed substantially. Headlines:
>
> - **It is NOT a CUE bug.** A pure-CUE control (no Go) running the *same* CUE version (v0.17.0-alpha.1)
>   on the *exact* real transformer + finalized component + context renders correctly and `cue vet -c`
>   passes. The Go kernel, fed the identical inputs, fails. The defect is in the library's Go
>   `cue.Value` construction/relocation, not in CUE. **This overturns the earlier "inherent to CUE
>   evaluation" reading of §10.3** — the Go `Unify` reproduced it only because the value was already
>   corrupted upstream in the materialized-platform construction. (§11)
> - The §6 version map was wrong (buggy = v0.5.0–v0.5.4, fixed = **v0.5.5**+, not v0.5.7). (§10.2)
> - A **generic, correctness-preserving kernel workaround** exists (re-fill output-local hidden fields
>   with their direct-lookup values; §10.6). A simpler "rebuild #transform before filling" does *not*
>   work (§10.6).

---

## 1. TL;DR

A `#ComponentTransformer.#transform` may declare hidden helper fields (`_foo`). If such a helper is
declared **lexically inside the `output` struct** (e.g. `output.spec.template.spec._convertedSidecars`)
and it references **other hidden fields declared higher up** (at `#transform` scope) or catalog
definitions, then after the kernel:

1. materializes the platform (`Materialize` → `#composedTransformers` assembled via `FillPath`), and
2. fills `#component` / `#context` into the transform (`executePair` → `FillPath`),

…that `output`-local helper evaluates to an **incomplete value** when consumed *in-expression*
(`list.Concat`, `for` comprehension, JSON marshal). A hidden helper declared at **`#transform` scope**
does **not** have this problem.

Maddening detail: a direct `value.LookupPath(".../output/.../_convertedSidecars")` from Go
**resolves it to a concrete value** (and `Validate(cue.Concrete(true))` returns `nil`). Only the
*in-CUE* reference to it (from a sibling field in the same `output`) sees the incomplete value.

## 2. Symptom

Rendering a stateless workload fails when the compiled Deployment is marshalled to JSON:

```
building inventory entries: converting resource Deployment/default/web-app-web to unstructured:
  marshal json: cue: marshal error:
  platform.#composedTransformers."opmodel.dev/catalogs/opm/transformers/deployment-transformer@0.5.2"
    .#transform.output.spec.template.spec.containers:
    error in call to list.Concat: non-concrete value _
```

The offending field is built like this (catalog `deployment_transformer.cue`, pre-fix):

```cue
#transform: {
    #component: _
    #context:   c.#TransformerContext

    _mainContainer: (#ToK8sContainer & {"in": _container, ...}).out   // declared at #transform scope

    _sidecarContainers: [...] | *[]
    if #component.spec.sidecarContainers != _|_ { _sidecarContainers: #component.spec.sidecarContainers }

    output: k8sappsv1.#Deployment & {
        spec: { template: { spec: {
            _convertedSidecars: (#ToK8sContainers & {"in": _sidecarContainers, ...}).out  // declared INSIDE output
            containers: list.Concat([[_mainContainer], _convertedSidecars])
        }}}
    }
}
```

`containers` references `_mainContainer` (transform scope, fine) and `_convertedSidecars`
(output-local, broken).

## 3. Blast radius

All five workload transformers in `opmodel.dev/catalogs/opm` use the identical
`_convertedSidecars`-inside-`output` + `list.Concat` pattern: `deployment`, `statefulset`, `daemonset`,
`job`, `cronjob`. Every container workload render hits it. ConfigMap-only / non-container transformers
are unaffected (no `_convertedSidecars`).

## 4. Decisive evidence

Temporary instrumentation in `opm/compile/execute.go` (`executePair`, gated on `OPM_DEBUG_COMPILE`,
since reverted — re-add per §6) dumped, for the real kernel render of the deployment pair:

| field | scope | `IsConcrete()` | `Validate(Concrete)` |
| --- | --- | --- | --- |
| `_mainContainer` | `#transform` | `true` | `nil` |
| `_sidecarContainers` | `#transform` | `true` | `nil` |
| `_convertedSidecars` | **inside `output`** | `true` | `nil` |
| `output…spec.containers` | inside `output` | **`false`** | `list.Concat: non-concrete value _` |

So **every input the expression depends on validates concrete via a direct `LookupPath`**, yet the
expression that references them is non-concrete.

Two corroborating experiments:

1. **Swap the consuming op.** Replacing `list.Concat([[_mainContainer], _convertedSidecars])` with a
   comprehension `[_mainContainer, for c in _convertedSidecars {c}]` changed the error to
   `cannot range over _convertedSidecars (incomplete type _)`. → The consuming builtin/operation is not
   at fault; the *value* `_convertedSidecars` is incomplete in-expression. (This is why "is it a
   `list.Concat` bug?" is a red herring — it is not.)

2. **Move the declaration.** Relocating `_convertedSidecars` from `output.spec.template.spec` up to
   `#transform` scope (next to `_mainContainer`), with `list.Concat` unchanged, makes
   `containers` evaluate `concrete=true`. This is the whole fix.

## 5. Where it happens (code pointers / suspects)

The value travels through several `FillPath` / cross-context steps. The hidden-field reference survives
for `#transform`-scope helpers but not `output`-local ones — pin down which step drops it.

- `opm/materialize/index.go:35` `indexCatalogs` — each transformer is `LookupPath`'d out of its catalog
  build (`b.Value.LookupPath(schema.Transformers)`, line 39) and `FillPath`'d into a **fresh**
  `octx.CompileString("{}")` map (lines 99–101). Cross-context move #1.
- `opm/materialize/materialize.go:102` — the composed map is `FillPath`'d onto a copy of the platform
  `Package` at `#composedTransformers`. Cross-context move #2. (Doc comment claims "same `*cue.Context`
  throughout" and "FillPath is non-mutating" — verify that holds for nested hidden fields.)
- `opm/compile/execute.go:81-84` `executePair` — `transformVal = platformVal.LookupPath(#composedTransformers/<fqn>/#transform)`.
- `opm/compile/execute.go:106` — `unified := transformVal.FillPath(schema.Component, dataComp)` (fills
  `#component`). FillPath #3.
- `opm/compile/execute.go:117` — `unified.FillPath(schema.Context, ctxVal)` (fills `#context`). FillPath #4.
- `opm/compile/execute.go:123` — `outputVal := unified.LookupPath(schema.Output)`; the Compiled value
  is `outputVal` verbatim (StructKind) — its lazy `containers` expression is marshalled later by the
  consumer, which is where the operator sees the failure.
- `opm/compile/finalize.go:12` `FinalizeValue` — `v.Syntax(cue.Final())` → `cueCtx.BuildExpr`. Applied
  to *components* (not transformers), but the same Syntax→rebuild technique is a candidate mental model
  for what's detaching references; worth checking whether anything finalizes the transformer/output.
- `dataComp` is the **finalized, constraint-free** component (`execute.go:95`). Filling a finalized
  data value into a transform that still carries definitions/hidden-fields is the exact composition
  that misbehaves.

Schema paths: `opm/schema/paths.go` — `ComposedTransformers`, `Transform`, `Component`, `Output`.

## 6. Reproduction

Prereqs: local OCI registry at `localhost:5000` with `opmodel.dev/core@v1` and a catalog that still has
the bug (i.e. `_convertedSidecars` declared inside `output`). **Correction (see §10.2):** the buggy
placement is `@v0.5.0`–`@v0.5.4`; the workaround landed at **`@v0.5.5`** (not v0.5.7). The clean
buggy-but-matching repro version is **`@v0.5.2`** — `0.5.0`/`0.5.1` additionally hit the matcher miss
documented in §10.5 (and `opm-operator/hack/kind-opm-dev-test/FINDINGS.md` §4). The single-version
recipe in §10.1 supersedes the multi-step instrumentation below.

A clean repro path:

1. Re-introduce the bug in a scratch catalog build: move `_convertedSidecars` back inside
   `output.spec.template.spec` in `deployment_transformer.cue`, publish as a throwaway version (e.g.
   `cd catalog_opm && CUE_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works' task publish VERSION=v0.5.99`).
2. Pin `testdata/modules/web_app/cue.mod/module.cue` dep `opmodel.dev/catalogs/opm@v0` to that version.
3. Re-add instrumentation to `opm/compile/execute.go` `executePair`, just after `outputVal` is fetched:

```go
import ("os"; "cuelang.org/go/cue/format")   // add to the import block

// ...after: outputVal := unified.LookupPath(schema.Output); err check...
if os.Getenv("OPM_DEBUG_COMPILE") != "" {
    dump := func(label string, v cue.Value) {
        fmt.Printf("  %s: exists=%v concrete=%v validate=%v\n",
            label, v.Exists(), v.IsConcrete(), v.Validate(cue.Concrete(true)))
        if n := v.Syntax(cue.Final(), cue.Hidden(true)); n != nil {
            if b, e := format.Node(n); e == nil { fmt.Printf("    %s\n", b) }
        }
    }
    hid := func(name string) cue.Value {
        return unified.LookupPath(cue.MakePath(cue.Hid(name, "opmodel.dev/catalogs/opm/transformers")))
    }
    fmt.Printf("\n[OPM_DEBUG] %s / %s\n", compName, tfFQN)
    dump("_mainContainer", hid("_mainContainer"))
    dump("_sidecarContainers", hid("_sidecarContainers"))
    spec := unified.LookupPath(cue.ParsePath("output.spec.template.spec"))
    dump("_convertedSidecars(local)",
        spec.LookupPath(cue.MakePath(cue.Hid("_convertedSidecars", "opmodel.dev/catalogs/opm/transformers"))))
    dump("containers", outputVal.LookupPath(cue.ParsePath("spec.template.spec.containers")))
}
```

1. Run the flow test against the local registry:

```bash
cd library
export CUE_REGISTRY='opmodel.dev=localhost:5000+insecure,testing.opmodel.dev=localhost:5000+insecure,registry.cue.works'
OPM_DEBUG_COMPILE=1 OPM_FLOW_TEST_FORCE=1 go test ./opm/kernel/... -run 'TestFlow_WebApp_OnOpmPlatform/Compile' -v
```

You'll see `_convertedSidecars(local): concrete=true` but `containers: concrete=false … list.Concat:
non-concrete value _`. Move the field to `#transform` scope and `containers` flips to `concrete=true`.

`cmd/flow-inspect -stages plan` also dumps the unevaluated `containers: list.Concat(...)` expression
with all the `let`-binding context (handy for reading the resolved reference graph).

## 7. Catalog-side workaround already shipped

In `catalog_opm` (branch `fix/container-non-concrete-marshal`, local tag `…/catalogs/opm@v0.5.7`), all
five workload transformers were changed to declare `_convertedSidecars` at `#transform` scope instead of
inside `output`. `containers: list.Concat([[_mainContainer], _convertedSidecars])` is unchanged.
Validated: flow test passes; operator renders web-app `@v1.0.10` → Deployment 2/2 + Service + HTTPRoute
- 2 ConfigMaps.

This unblocks rendering but does **not** fix the kernel. Any future transformer author who declares a
referencing hidden field inside `output` will reintroduce the bug with no compile-time warning.

## 8. What a real fix looks like (for the next agent)

Pick one (or both):

- **Kernel fix.** Find which `FillPath`/cross-context step (§5) drops the in-expression resolution of
  `output`-local hidden fields, and make the materialized + filled transform evaluate them the same way
  a fresh `LookupPath` does. Likely areas: `materialize/index.go` (rebuilding the composed map in a new
  context) or `compile/execute.go` (the `#component`/`#context` fills). A minimal Go reproduction
  outside the catalog would help: construct a tiny `#transform`-shaped value with an `output`-local
  hidden field referencing an outer hidden field, run it through `Materialize`-like `FillPath` steps,
  and assert the inner field is consumable in-expression.
- **Authoring constraint.** If the kernel behaviour is deemed inherent to CUE's evaluation model,
  document and enforce: *"compute transformer helper values at `#transform` scope; never declare a
  referencing hidden field inside `output`."* Enforcement options: a `cue vet` lint, a catalog CI check,
  or a kernel-side validation that rejects transformers with hidden fields under `output`.

### Acceptance criteria

- A transformer with a referencing hidden field declared inside `output` renders to a concrete,
  marshallable resource through the full `Materialize → Match → Compile → ToUnstructured` path; **or**
- the kernel/CI rejects such a transformer with a clear, actionable diagnostic (not a downstream
  `list.Concat: non-concrete value _` at JSON-marshal time).

## 9. Cross-references

- `opm-operator/hack/kind-opm-dev-test/FINDINGS.md` §3a — operator-side discovery + end-to-end validation.
- `docs/design/compile-pipeline-known-gaps.md` — prior compile-pipeline findings (matcher / executor),
  same fixture pair (`testdata/modules/web_app` + `modules/opm_platform`).
- `cmd/flow-inspect` — inspection harness.
- `opm/kernel/flow_integration_test.go` — `TestFlow_WebApp_OnOpmPlatform`.
- `docs/design/repro-hidden-field/` — pure-CUE control proving the bug is in Go, not CUE (§11).

## 10. 2026-06-14 investigation update (deterministic repro + root-cause narrowing)

### 10.1 Deterministic reproduction (single catalog version)

The bug reproduces through the real kernel path with a **single** catalog version selected — the
multi-version composed map is *not* required. Recipe used:

1. Pin `testdata/modules/web_app` dep `opmodel.dev/catalogs/opm@v0` to a buggy version (`v0.5.2`).
2. Inject a subscription filter so Materialize selects exactly that version (the on-disk fixture has
   **no** filter, which makes Materialize pull all published versions — see §10.5):
   `#registry."opmodel.dev/catalogs/opm".filter.range: ">=0.5.2 <0.5.3"`.
3. Compile the release and `MarshalJSON()` every `*core.Compiled.Value` (what the operator's
   `ToUnstructured` does). Only `deployment-transformer@0.5.2` fails:

   ```
   deployment-transformer@0.5.2: cue: marshal error:
     …#transform.output.spec.template.spec.containers:
     error in call to list.Concat: non-concrete value _
   ```

   `configmap` / `http-route` / `service` transformers (no output-local hidden field) marshal fine.

### 10.2 The §6 version map was wrong

Verified by reading each published transformer out of the warm CUE cache
(`~/.cache/cue/mod/extract/opmodel.dev/catalogs/opm@vX/transformers/deployment_transformer.cue`):

| versions | `_convertedSidecars` placement | renders? |
| --- | --- | --- |
| `v0.5.0`–`v0.5.4` | **inside** `output.spec.template.spec` | **fails** |
| `v0.5.5`+ | at `#transform` scope (the workaround) | OK |

So the fix landed at **v0.5.5**, not v0.5.7, and the clean buggy-but-matching repro version is
**v0.5.2** (v0.5.0/v0.5.1 additionally hit the §10.5 matcher miss). `v0.5.5` uses a `for`-comprehension
spelling; `v0.5.6`/`v0.5.7` use `list.Concat` — both at `#transform` scope, both fine.

### 10.3 The two-copy behaviour (mechanism) — but see §11 for the real root cause

Instrumenting `executePair` confirmed the two-copy behaviour from §4 exactly:

| field | direct `LookupPath` | in-expression (from `containers`) |
| --- | --- | --- |
| `_convertedSidecars` (output-local) | `[]`, concrete | bare `_` (incomplete) |

The same path resolves to two different values.

> **CORRECTION (see §11).** This section originally concluded "the bug is inherent to how CUE
> evaluates this structure" because a Go `value.Unify({#component,#context})` reproduced it just like
> `FillPath`. That conclusion was **wrong**. The Go value being unified onto was *already corrupted*
> by the upstream materialized-platform construction; both `Unify` and `FillPath` then faithfully
> propagate the corrupted value. §11's pure-CUE control (which performs the same unification in CUE,
> not Go, on a freshly-built value) renders correctly — proving the corruption is introduced by the
> Go `cue.Value` plumbing, not by CUE's evaluator. The defect is in the library, upstream of
> `executePair`.

### 10.4 No synthetic minimal repro triggers it

A standalone Go harness (CompileString → materialize-style round-trip → fill `#component` → marshal)
was built up to faithfully mimic the catalog pattern and **never reproduced** the bug, even with all
of: helper-projection indirection `(#H & {"in": x}).out`, the `X="in"` alias binding, the
`[...] | *[]` default disjunction, a conditional that sources the list from `#component`, a closed
`output` definition, **package-scoped** hidden fields (`package transformers`), the
lookup-into-fresh-`{}`-map composed-transformer round-trip, and a `FinalizeValue`-finalised data
component. Every combination marshalled cleanly.

Conclusion (refined by §11): the synthetic Go harness never reproduced because it never reconstructs
the **real materialized-platform value** that the kernel builds — that construction is what corrupts the
value, and §11 shows the same inputs are fine in pure CUE. So this is not a "minimal CUE repro" hunt;
it is a "which Go construction step corrupts the value" hunt. §10.7 records how far that was narrowed.

### 10.5 Side finding: the flow test currently matches nothing

`TestFlow_WebApp_OnOpmPlatform` is red in the current checkout for an **unrelated** reason: the
on-disk `modules/opm_platform` subscription has no `filter`, so Materialize pulls **all 8** published
catalog versions (v0.5.0–v0.5.7) into one composed map + reverse index, and Match then returns
**zero** pairs (`2 component(s) have no matching transformer: [config web]`). Adding a single-version
`filter.range` restores matching. This is a matcher-vs-multi-version-index gap (cf.
`compile-pipeline-known-gaps.md`), separate from this bug but it blocks the §6 repro path until either
the fixture gets a filter or the matcher is fixed.

### 10.6 Fix experiments

Run against the real `deployment-transformer@0.5.2` repro:

| approach | result |
| --- | --- |
| `unified.Syntax(cue.Final())` → `BuildExpr` rebuild before extracting `output` | ❌ serialises the already-broken `_` |
| Fill `#component`/`#context` at **full paths** in the platform value (no `LookupPath` detach) | ❌ still fails |
| `Unify` conjunct instead of `FillPath` | ❌ still fails (originally read as "proves §10.3"; actually just propagates the already-corrupted value — see §11) |
| Rebuild `#transform` via `Syntax(Final)`→`BuildExpr` **before** filling `#component` | ❌ still fails (Syntax captures the already-broken reference) |
| Upgrade `cuelang.org/go` v0.17.0-alpha.1 → alpha.3 | ⚠️ blocked: alpha.3 tightens closedness and breaks the fixtures (`#components.web.spec.statelessWorkload.scaling: field not allowed`) before this code path is reached — a separate coordinated upgrade |
| **Re-fill each output-local hidden field with its direct-lookup value** | ✅ **works** |

The working fix (`FIX6` in the throwaway instrumentation): after filling `#component`/`#context`,
walk `output` for every concrete hidden field, and `FillPath` each back at its own path with the value
a direct `LookupPath` returns (which is correct — see §10.3 table). Because the re-filled value is the
*correct* one, this is correctness-preserving, not just marshal-silencing: verified by adding a real
sidecar (`tr.#SidecarContainers` + a `fluent/fluent-bit` entry) to the web fixture and confirming the
rendered Deployment's `containers` list contains **both** `web` and the `log-shipper` sidecar.
Transformers with zero output-local hidden fields are untouched (0 re-fills).

Sketch (not yet committed; needs hardening + an integration test gated like the flow test):

```go
// after the two FillPath calls in executePair, before LookupPath(schema.Output):
out := unified.LookupPath(schema.Output)
var paths []cue.Path
collectHiddenPaths(out, schema.Output, &paths) // recursive Fields(cue.Hidden(true)), collect sel.PkgPath()!=""
for _, p := range paths {
    if v := unified.LookupPath(p); v.Exists() && v.IsConcrete() {
        unified = unified.FillPath(p, v)
    }
}
```

### 10.7 Recommendation

Two viable, non-exclusive paths — pick per appetite for kernel surface area:

1. **Ship the §10.6 re-fill as a kernel workaround.** Pros: defends every front-end (operator, CLI)
   and lets future transformer authors declare output-local helpers without tripping the bug; the
   catalog workaround (§7) then becomes belt-and-suspenders. Cons: a per-render walk + re-fill over
   `output`; should carry a `// TODO` referencing this doc and be deleted once the upstream Go
   plumbing bug (§11) is fixed. Needs an integration test (the §10.1 recipe).
2. **Enforce the authoring constraint instead.** Add a kernel- or CI-side validation that rejects a
   transformer declaring a hidden field under `output` with an actionable diagnostic, satisfying the
   §8 acceptance criteria's second branch. Cheaper, no per-render cost, but doesn't "just work" for
   authors.
3. **Fix the actual Go bug (best).** §11 proves the corruption is introduced in the library's
   materialized-platform construction, *before* `executePair`. Fixing that removes the need for both
   workarounds. §11.3 records how far the construction was bisected and where to look next.

## 11. 2026-06-14 — pure-CUE control: the bug is in our Go code, not CUE

This is the decisive experiment requested in the handoff: remove Go from the equation, express the
kernel's "glue" (fill `#component`/`#context` into a transformer's `#transform`) in **pure CUE**, and
render the buggy `deployment-transformer@0.5.2`. Harness preserved under
`docs/design/repro-hidden-field/` (`repro.cue`, `realcomp.cue`, README with the run recipe).

### 11.1 Setup

A throwaway CUE module depending on `opmodel.dev/catalogs/opm@v0.5.2` (the buggy catalog, hidden field
*inside* `output`), `opmodel.dev/core@v1`, `cue.dev/x/k8s.io@v0`. The glue is plain unification:

```cue
import tf "opmodel.dev/catalogs/opm/transformers"

_applied: tf.#DeploymentTransformer.#transform & {
    #component: { /* … */ }
    #context:   { #moduleInstanceMetadata: {…}, #componentMetadata: {…}, #runtimeName: "opm-test" }
}
containers: _applied.output.spec.template.spec.containers
```

Critically, the CLI was pinned to **the exact CUE version the Go library uses** —
`v0.17.0-alpha.1` (`go install cuelang.org/go/cmd/cue@v0.17.0-alpha.1`) — so version is controlled for.

### 11.2 Result — renders cleanly

| harness variant (alpha.1 CLI) | `containers` |
| --- | --- |
| minimal `#component` | ✅ concrete `[{nginx web}]` |
| same, via a `#composedTransformers`-style map indirection (`_composed[fqn].#transform & …`) | ✅ concrete |
| **exact real finalized `web` component** (dumped from the kernel via `OPM_DUMP_CUE`) | ✅ concrete |
| whole module `cue vet -c` | ✅ passes (fully concrete) |

Same CUE version, same transformer, same component, same context as the failing Go render — **pure CUE
produces a correct, concrete Deployment**. The Go kernel produces `list.Concat: non-concrete value _`.
∴ **the defect is in the library's Go `cue.Value` construction, not in CUE.**

### 11.3 Where in the Go code (bisection so far)

A Go test (`opm/materialize`, since removed) rebuilt the kernel's steps one at a time against the same
loaded catalog and filled `#component`/`#context` the same way the executor does. Each step in
isolation stayed **concrete**:

- A — transformer referenced directly from the loaded catalog value
- B — after the materialize FillPath-into-fresh-`{}`-map relocation
- C — full `indexCatalogs` output
- D — after `FillPath(#composedTransformers, composed)` onto a bare `{}`
- E — D + a `FinalizeValue`-finalised component
- (the exact real finalized component, as data, also stayed concrete)

Yet the **real** `(*Kernel).Compile` path fails. The remaining un-replicated difference is that the
kernel fills `#composedTransformers` onto a **loaded, closed `c.#Platform`** value whose field is typed
`#composedTransformers?: #TransformerMap` (`[#FQNType]: #ComponentTransformer`). Re-unifying each
composed transformer against the closed `#ComponentTransformer`/`#Platform` schema — carrying closedness
through the materialized twin — is the leading suspect for where the output-local hidden field's
reference scope gets corrupted. Confirming it needs the real loaded `#Platform` (the bare-`{}`
reconstruction in D doesn't carry the schema/closedness), e.g. by bisecting `materialize.Materialize`'s
`p.Package.FillPath(schema.ComposedTransformers, composed)` against the actual platform fixture. That is
the concrete next step for a root-cause fix. **→ Done in §12; the suspect is confirmed.**

## 12. 2026-06-14 — root cause CONFIRMED + fix

### 12.1 The one-variable experiment

A Go test held *everything* constant — same loaded catalog, same `indexCatalogs` composed map, same
full real component, same context, same executor fill sequence — and varied **only** the value the
composed map is filled onto:

| fill `#composedTransformers` onto … | `output…containers` |
| --- | --- |
| `octx.CompileString("{}")` (open, separately built) | ✅ concrete |
| the real **loaded, closed `c.#Platform`** fixture (`IsClosed()==true`) | ❌ `list.Concat: non-concrete value _` |

That is the whole bug. The corrupting operation is **`materialize.go:102`**:

```go
filled := p.Package.FillPath(schema.ComposedTransformers, composed)
```

`p.Package` is the platform fixture built independently via `octx.BuildInstance` — a **closed**
`c.#Platform` (`kind: "Platform"`, closed definition). `FillPath`-ing the composed transformer map
into that closed value corrupts the lazy in-expression resolution of output-local hidden fields inside
the transformers (`_convertedSidecars`), so a sibling reference (`containers: list.Concat([…, _convertedSidecars])`)
sees bare `_` while a direct `LookupPath` of the same field sees the correct `[]`.

### 12.2 Why this is a CUE Go-API bug, not ours and not CUE-the-language

- **Not the schema / not CUE evaluation.** §11 wraps the exact same transformer in the exact same
  closed `c.#TransformerMap` *and* full closed `c.#Platform` in **pure CUE** (one unification pass) and
  it renders concretely. CUE the language handles the closed schema fine.
- **It is the Go API doing incremental `FillPath`/`Unify` into a closed, separately-built value.**
  `bare-{}` (open, also separately built) works; the closed loaded `#Platform` does not. The
  discriminator is **closedness of the fill target**, combined with cross-build incremental filling —
  precisely the hazard the SDK flags via `Value.UnifyAccept(w, accept)` ("like `Unify` but disregards
  closedness rules … used for piecemeal unification", `research/cue/sdk/value.md`). `Unify` instead of
  `FillPath` does **not** help (tested) — it is the closedness, not the method.
- Reproducible upstream candidate: `closedPlatform.FillPath(typedClosedMapField, mapOfDefs)` where a def
  in the map has an `output`-nested hidden field referencing an outer hidden field. (A minimal,
  catalog-free reduction is still worth distilling for a cuelang issue.)

### 12.3 Fix (verified in isolation)

Keep `#composedTransformers` (and, by symmetry, `#matchers`) as **separate open `cue.Value`s** — the
form `indexCatalogs` already produces — and have the executor and matcher read transforms from those,
**never** from inside the closed `mp.Package`. In the one-variable test, looking the transform up from
the open `composed` map (`fix1-separate-composed`) renders **concrete**; filling onto the closed
platform and looking it up there does not.

Concretely:
- `materialize.MaterializedPlatform`: add `Composed cue.Value` and `Matchers cue.Value` fields set to the
  open `indexCatalogs` outputs. (Keep `Package` as-is for `Source`/diagnostics, or stop filling the two
  slots onto it.)
- `compile/execute.go:81` `executePair`: read the `#transform` from `mp.Composed`, not
  `platformVal.LookupPath(schema.ComposedTransformers)`.
- `compile/match.go:98-100` `Match`: read `composed`/`matchers` from `mp.Composed`/`mp.Matchers`.

This is a public-surface change (`MaterializedPlatform` gains fields; the compile/match readers change)
— a small, well-scoped library change, not a workaround. It removes the need for the §10.6 re-fill and
the catalog authoring constraint (§7 becomes defence-in-depth). It should ship with the §10.1
single-version integration test as a regression guard, and a `// TODO` linking the upstream CUE issue.

Alternative if a struct change is undesirable now: the §10.6 re-fill workaround in `executePair`
remains valid and is strictly smaller, at the cost of a per-render walk.

### 12.4 Reproduction artifacts

- Pure-CUE control (proves "not CUE"): `docs/design/repro-hidden-field/{repro,realcomp,closed}.cue` +
  README. `closed.cue` is the §11 closed-schema control.
- The Go one-variable experiment (`bare-{}` vs loaded-closed `#Platform`, plus the `fix1`/`fix2` probes)
  was a throwaway test in `opm/materialize`; re-create from §12.1 if needed (load `modules/opm_platform`
  via `load.Instances`, `indexCatalogs` the v0.5.2 catalog, fill onto each base, compare).

## 13. 2026-06-14 — landed kernel fix

Per §12.3, the executor now sources every `#transform` from the **open** composed map instead of
out of the closed materialized `Package`.

Change set (minimal variant):
- `opm/materialize/types.go` — `MaterializedPlatform` gains `Composed cue.Value` (the open
  `#composedTransformers` map from `indexCatalogs`); `Package`'s doc warns against reading transforms
  from it.
- `opm/materialize/materialize.go` — `Materialize` returns `Composed: composed` (the pre-fill open
  value). The existing `FillPath` onto `Package` stays — harmless, and keeps the matcher / flow-inspect
  reading `Package` unchanged (they read only FQNs/labels, which the closedness bug does not corrupt).
- `opm/compile/module.go` — passes `r.platform.Composed` (was `.Package`) into `executeTransforms`.
- `opm/compile/execute.go` — `executePair` reads `composedVal.LookupPath(fqn).LookupPath(#transform)`
  (drops the `#composedTransformers` prefix; the param is the map itself).

Regression guards:
- `opm/materialize/composed_open_test.go::TestComposed_RendersConcreteWherePackageDoesNot` — with the
  real buggy catalog `@v0.5.2`, asserts a transform read from `Composed` renders a concrete,
  marshallable Deployment while the same transform read from the closed `Package` does not. Registry-
  gated (skips if the catalog pull fails).
- `opm/compile/compile_test.go::TestExecute_ReadsTransformFromComposedNotPackage` — hermetic wiring
  guard: `Package` and `Composed` carry divergent outputs; asserts the executor's output comes from
  `Composed`. No registry.

Not fixed here (separate, pre-existing): `TestFlow_WebApp_OnOpmPlatform` still fails at **Match**
because the on-disk `modules/opm_platform` subscription has no `filter` and pulls all 8 catalog
versions → zero matches (§10.5). That is a matcher-vs-multi-version-index gap, unrelated to this fix.

## 14. RESOLVED — `federate-materialize-transformers` (supersedes §13)

The interim fix (§13) kept the corrupt closed twin alive (`Materialize` still `FillPath`-ed the
composed map onto `Package`) and relied on a comment to keep the executor reading the open `Composed`
map instead. Correctness depended on every future reader honoring that comment — a live tripwire that
`cue vet` cannot catch.

`federate-materialize-transformers` removes the seam **structurally** (ADR-003): the composed map and
the matcher reverse index are exposed as native first-class fields on `MaterializedPlatform` —
`Transformers` and `Matchers` — built in the owner `*cue.Context` by `indexCatalogs`, and **are no
longer `FillPath`-ed onto the closed `c.#Platform`**. The closed twin is never constructed, so there
is no surface from which reading a `#transform` corrupts output-local hidden fields. The
comment-enforced `Composed` workaround and the now-empty-of-materialized-data `Package` field are both
removed; the unfilled closed spec remains reachable as `Source.Package` for `#registry`/metadata.

Why federation rather than the single-build composition ADR-003 first sketched: CUE Minimal Version
Selection admits only one version per `path@major` per build, which is incompatible with OPM's
required multi-version-per-major catalog composition (a platform may subscribe to `catalog@0.5.0` and
`catalog@0.5.1` simultaneously). Federation keeps each selected version under its distinct
version-bearing FQN in one native merged map and preserves all other materialize behavior.

Regression guards (retargeted onto the native surface):
- `opm/materialize/composed_open_test.go::TestTransformers_RenderConcreteWhereClosedPlatformDoesNot`
  — with the real buggy catalog `@v0.5.2`, asserts a transform read from the native composed map
  (what `mp.Transformers` exposes) renders a concrete, marshallable Deployment, while the same
  transform read from a locally-reconstructed closed platform value still does not. Registry-gated.
- `opm/compile/compile_test.go::TestExecute_ReadsTransformFromNativeTransformers` — hermetic wiring
  guard: `Source.Package` and `Transformers` carry divergent outputs; asserts the executor's output
  comes from `Transformers`. No registry.
- `opm/materialize/materialize_test.go::TestMaterialize_DoesNotFillClosedPlatform` — locks the seam
  shut: a successful materialization populates `Transformers` while `Source.Package`'s
  `#composedTransformers` / `#matchers` stay unfilled.

§10.5 follow-up: the matcher now reads the native `Matchers` index instead of the corrupt closed
`Package`. Whether that fully clears the "multi-version subscription → zero pairs" symptom is asserted
by the flow integration test; any residual matcher behavior is tracked separately, not by this doc.
