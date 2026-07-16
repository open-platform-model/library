# CUE closedness regression: `v0.17.0-alpha.2`+ rejects valid optional nested-struct fields

Status: Investigation / upstream bug report (2026-06-16). **Re-verified against
the official `v0.17.0` release on 2026-07-04 — the regression is NOT fixed; it
shipped in the final release.** Affects the OPM CUE toolchain choice. **Not** an
OPM schema or catalog defect — the same sources validate cleanly on `v0.16.0`
and `v0.17.0-alpha.1`.

## Summary

Starting with `cuelang.org/go v0.17.0-alpha.2`, the CUE evaluator reports
spurious `field not allowed` (closedness) errors for **optional nested struct
fields that are explicitly declared in the closed schema**. The same modules
validate cleanly on `v0.16.0` and `v0.17.0-alpha.1`. The regression was
introduced between `alpha.1` and `alpha.2` and **persists through `alpha.3`,
`rc.1`, and the official `v0.17.0` release** (confirmed 2026-07-04). It is
reproducible both via the `cue` CLI and via the Go kernel at module-build time.

This matters because the OPM kernel (`library`, `opm-operator`) pins
`cuelang.org/go v0.17.0-alpha.1`. A bump to `alpha.2`/`alpha.3` would make the Go
evaluator reject real catalog-backed modules at render time — exactly what the
CLI does today.

## Version matrix

Measured with isolated binaries (`GOBIN=… go install cuelang.org/go/cmd/cue@<ver>`):

| `cue` version      | catalog module (`cue vet modules/web_app`) | `#Module` metadata self-cycle† |
| ------------------ | ------------------------------------------ | ------------------------------ |
| `v0.16.0`          | **clean**                                  | FAIL (`field not allowed`)     |
| `v0.17.0-alpha.1`  | **clean**                                  | FAIL (`field not allowed`)     |
| `v0.17.0-alpha.2`  | **FAIL** (`field not allowed`)             | FAIL (`field not allowed`)     |
| `v0.17.0-alpha.3`  | **FAIL** (`field not allowed`)             | FAIL (`field not allowed`)     |
| `v0.17.0-rc.1`     | **FAIL** (`field not allowed`)             | —                              |
| `v0.17.0` (final)  | **FAIL** (`field not allowed`)             | —                              |

**2026-07-04 re-run** used the current module deps (`opmodel.dev/catalogs/opm@v1.0.0-alpha`,
`opmodel.dev/core@v1.0.0-alpha.1`; the earlier `@v0.5.2` catalog is gone from the
registry). The trigger construct survived the catalog `v0.5.2 → v1.0.0-alpha`
bump — the left-column result is unchanged from the original investigation, and
`alpha.1` remains the newest clean toolchain.

† The metadata-self-cycle column is a **separate, real** OPM schema bug (a
`#Module.metadata` self-reference), fixed independently and orthogonal to this
toolchain regression — it fails on every version. It is included only to show the
columns measure different things: the **left** column is the toolchain
regression (changes between alpha.1 and alpha.2); the right does not.

## Reproduction

Prereqs: a local OCI registry at `localhost:5000` holding `opmodel.dev/core@v1`
and `opmodel.dev/catalogs/opm@v0.5.2` (any version with the workload blueprints),
and a module that uses the stateless-workload blueprint with `scaling` +
`updateStrategy` set — e.g. `modules/web_app` in this workspace.

```bash
export CUE_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'

# clean on alpha.1
GOBIN=/tmp/a1 go install cuelang.org/go/cmd/cue@v0.17.0-alpha.1
( cd modules/web_app && /tmp/a1/cue vet ./... )      # exit 0, no output

# fails on alpha.2 / alpha.3
GOBIN=/tmp/a2 go install cuelang.org/go/cmd/cue@v0.17.0-alpha.2
( cd modules/web_app && /tmp/a2/cue vet ./... )
```

Output on `alpha.2`/`alpha.3`:

```
#components.web.spec.statelessWorkload.scaling: field not allowed:
    .../opmodel.dev/catalogs/opm@v0.5.2/blueprints/workload/stateless_workload.cue:12:2
    ./components.cue:29:5
#components.web.spec.statelessWorkload.updateStrategy: field not allowed:
    .../opmodel.dev/catalogs/opm@v0.5.2/blueprints/workload/stateless_workload.cue:14:2
    ./components.cue:31:5
```

`stateless_workload.cue:12` / `:14` are the **declarations** the value is being
rejected against:

```cue
#StatelessWorkloadSchema: {
    container:       res.#ContainerSchema
    scaling?:        tr.#ScalingSchema     // line 12 — optional, present in the schema
    updateStrategy?: tr.#UpdateStrategySchema  // line 14
    ...
}
```

`scaling` and `updateStrategy` are declared optional fields of the closed schema,
so setting them is valid — and `alpha.1` agrees. `alpha.2`+ rejects them as "field
not allowed."

## The triggering construct

The failing path combines several closed-struct features. In the OPM catalog
(`opmodel.dev/catalogs/opm`):

1. Core `#Component` closes its `spec` over a **comprehension** that merges every
   embedded resource/trait/blueprint's `spec` (`opmodel.dev/core` `component.cue`):
   ```cue
   _allFields: {
       for _, resource in #resources { if resource.spec != _|_ {resource.spec} }
       for _, trait    in #traits    { if trait.spec    != _|_ {trait.spec} }
       for _, blueprint in #blueprints { if blueprint.spec != _|_ {blueprint.spec} }
   }
   spec: close({_allFields})
   ```
2. The `#StatelessWorkload` component embeds the scaling/update-strategy **traits**
   (each a closed `c.#Trait` contributing `spec.scaling` / `spec.updateStrategy`),
   a **blueprint** (`#StatelessWorkloadBlueprint`) contributing
   `spec.statelessWorkload: #StatelessWorkloadSchema`, and a **propagation** block
   (`stateless_workload.cue:60-78`):
   ```cue
   spec: {
       statelessWorkload: #StatelessWorkloadSchema
       if spec.statelessWorkload.scaling != _|_ {
           scaling: spec.statelessWorkload.scaling   // line 63 — cross-references the nested optional
       }
       ...
   }
   ```

The combination — `close()` over a cross-package comprehension, plus a
self-referential propagation copying an optional nested struct (`#ScalingSchema`,
a closed cross-package definition) to a sibling field — is what alpha.2+ mis-closes.

### Refined mechanism (2026-07-04, tested on `v0.17.0`)

A two-stage bisection (first over a copied module cache, then over a scratchpad
module with the blueprint definitions vendored **locally** next to the module —
which still reproduces, so the blueprint layer needs no cross-package boundary)
converged on one rule:

> **The trigger is an `if <path>.<field> != _|_` comprehension guard, inside the
> component's spec, whose condition path traverses a nested non-scalar field.**
> The field named in the guard condition is the one reported "field not
> allowed" — regardless of how (or whether) its value is then copied.

The decisive matrix (each row an isolated edit of the vendored blueprint,
verified applied, `cue vet` with `v0.17.0`):

| Propagation shape | `v0.17.0` |
| --- | --- |
| `if spec.sw.scaling != _|_ { scaling: … }` (catalog as written) | **FAIL** on `scaling` |
| same, guard condition only, body copies nothing | FAIL |
| same, copy into a hidden `_scaling` / via `let` / spread / embed | FAIL |
| `scaling` made **required**, guard kept | FAIL |
| `container` (required struct, always clean before) moved **under a guard** | **FAIL on `container`** |
| `sidecarContainers` (list) guarded, valid value set | FAIL |
| synthetic `foo?: {bar: int}` / `{bar?: int}` / `{bar: int \| *1}` guarded | FAIL (all shapes) |
| **`scaling` copied unconditionally, no guard** | **clean** |
| `restartPolicy` (scalar enum) guarded — catalog as written | clean |
| field set by user but never referenced (`securityContext`) | clean |
| no propagation block at all | clean |

Dimensions ruled OUT (each varied independently, no effect): optional vs
required; hidden vs public destination; copy mechanism; schema definition
identity (a duplicate local schema still fails); trait embeds present or absent;
`#StatelessWorkloadSchema` opened with `...`; core `spec: close(…)` removed.
Scalar/enum-valued fields are exempt from the guard rule; struct- and
list-valued fields are affected. `spec.scaling` and
`spec.statelessWorkload.scaling` share the same definition
(`traits.#ScalingSchema`) — no schema mismatch is involved.

Why only `scaling`/`updateStrategy` surface in the real catalog: they are the
only fields that are simultaneously (a) non-scalar, (b) referenced in a guard
condition, and (c) set by the module under test. `container` is propagated
unconditionally; `restartPolicy` is a scalar; `sidecarContainers`/
`initContainers` are guarded and *would* fail, but no module sets them;
`securityContext` is never referenced.

### Working workarounds (validated on both `alpha.1` and `v0.17.0`)

Two catalog-side rewrites of the propagation block clear the error while
preserving semantics (flattened `spec.scaling` etc. export byte-identically to
the pristine blueprint on `alpha.1`):

1. **Hoist the guards out of the spec block** (general — works for structs and
   lists):

   ```cue
   spec: {
       statelessWorkload: #StatelessWorkloadSchema
       container: spec.statelessWorkload.container
   }
   if spec.statelessWorkload.scaling != _|_ {
       spec: scaling: spec.statelessWorkload.scaling
   }
   // … same for updateStrategy, sidecarContainers, initContainers
   ```

2. **Guard on a scalar leaf instead of the struct** (needs a required or
   defaulted leaf in the nested schema):

   ```cue
   if spec.statelessWorkload.scaling.count != _|_ {
       scaling: spec.statelessWorkload.scaling
   }
   ```

**Adopted 2026-07-04:** workaround 1 (hoisted guards) is implemented across all
five workload blueprints in `catalog_opm` (`src/blueprints/workload/*.cue`),
with the authoring rule documented in the catalog repo at
`docs/cue-guard-closedness-workaround.md` and a pitfall note in its `CLAUDE.md`.
Validated: catalog vet + the `web_app` repro module clean on both `alpha.1` and
`v0.17.0`, flattened spec output byte-identical. The library-pin recommendation
stands until a fixed catalog version is published and the operator/CLI/kernel
matrix is re-run against it.

### Minimal in-package reproductions do NOT trigger it

Several reduced, single-package models of the above (a `close({...})` with the
same propagation `scaling: spec.statelessWorkload.scaling`, including a
comprehension over a `#traits` map) evaluate **cleanly on alpha.3**. The trigger
therefore appears to require the real multi-layer, cross-package closed-type
embedding (core `#Component` → catalog `#Trait`/`#Blueprint` → `#ScalingSchema`),
not just the surface shape. This narrows the regression to closedness propagation
across module/definition boundaries rather than a single local pattern.

## Go-kernel confirmation (2026-07-04)

The CLI matrix above is the evaluator regression seen through the `cue` binary.
Because the kernel embeds the *same* evaluator as a Go dependency, the failure
also reproduces at module-build time in the library. Confirmed by a throwaway
`cuelang.org/go v0.17.0-alpha.1 → v0.17.0` bump of `library/go.mod` (reverted
after measurement):

| library pin        | `go test ./opm/kernel -run TestIntegration_Live_ValidateRealConfig` |
| ------------------ | ------------------------------------------------------------------- |
| `v0.17.0-alpha.1`  | **PASS**                                                            |
| `v0.17.0` (final)  | **FAIL** — `building module package from …/testdata/modules/web_app: #components.web.spec.statelessWorkload.scaling: field not allowed (and 1 more errors)` |

`opm/materialize` package tests pass on both pins — that path fills already-built
values rather than re-validating the closed component spec, so it does not
re-trigger the closedness check. The regression bites specifically where the Go
kernel **builds/validates** a catalog-backed `#Module` (`Kernel.Validate` /
`ValidateRealConfig`), i.e. the exact render-time path a pin bump would break.
(Two other kernel tests fail independently of the CUE version: the pre-existing
catalog FQN version-skew in `TestFlow_WebApp_OnOpmPlatform`.)

## Root cause + upstream status (2026-07-04)

**Upstream issue: [cue-lang/cue#4423](https://github.com/cue-lang/cue/issues/4423)**
("Field not allowed regression in v0.17.0", opened 2026-07-02, `NeedsInvestigation`).
The reporter's repro is the same shape as ours — an `if type == "RollingUpdate"`
guard inside a closed k8s `#Deployment.spec.strategy` — and their `git bisect`
lands on commit `339485ddf008` (2026-05-09):

> `internal/core/adt: dependency-tracking comprehension pushdown` — "Switch the
> comprehension pushdown algorithm from field pushdown — eagerly materializing
> the body's fields on child arcs — to dependency pushdown: pre-create the arcs
> as ArcPending placeholders, run the comp eagerly when its guard fields are
> concrete…"

This is the "comprehension algorithm redesign" headlined in the
v0.17.0-alpha.2/final release notes, and it matches our bisected trigger
exactly: guards are now *deferred* until their referenced fields are concrete,
and when the guard's condition path traverses a field of a closed nested
struct, the pre-created/deferred arc is evidently not registered with the
closedness bookkeeping — so the field is later rejected as not allowed.
`CUE_DEBUG=opendef` (closedness checks disabled) makes the repro pass on
`v0.17.0`, confirming the failure is purely closedness bookkeeping, not
evaluation.

Relation to the evaluator rewrite ("evalv3", the experiment that rethought
closedness and went official): evalv3 has been the default since `v0.13` and a
**locked stable experiment** since `v0.15` — on `v0.17.0`,
`CUE_EXPERIMENT=evalv3=0` fails with `cannot disable stable experiment
"evalv3"`, so there is no old-evaluator escape hatch. But evalv3 itself is not
the direct cause: `alpha.1` runs pure evalv3 and is clean. The regression is
this specific May-2026 comprehension-pushdown commit, part of the evalv3
completion work.

## Impact on OPM

- **Today:** none at runtime. The kernel (`library`/`opm-operator`) pins
  `cuelang.org/go v0.17.0-alpha.1`, which is clean. Only a developer running a
  newer `cue` CLI locally sees the false errors (and may misdiagnose a module as
  broken — this happened during the
  [release-vs-modulerelease render-divergence](../../../opm-operator/docs/design/release-vs-modulerelease-render-divergence.md)
  investigation).
- **On a toolchain bump:** if `cuelang.org/go` is bumped to `alpha.2`+ before this
  is fixed upstream, the Go evaluator will reject every catalog-backed module that
  uses the workload blueprints with `scaling`/`updateStrategy` set — i.e. real
  rendering breaks, not just the CLI.

## Recommendations

1. **Stay on `cuelang.org/go v0.17.0-alpha.1`** in `library`/`opm-operator`
   `go.mod`. The official `v0.17.0` did **not** fix this — do not bump to it.
2. **Before any future bump past alpha.1**, re-run the matrix above (and the
   operator's `task dev:test:local` render integration tests) against the
   candidate version. `TestIntegration_Live_ValidateRealConfig` in
   `opm/kernel` is the fastest Go-level canary.
3. **Report upstream** to `cuelang.org/go` — the reproduction now points at a
   *released* version, so this is a shipped regression, not a pre-release
   wart. Offer to bisect alpha.1→alpha.2 if maintainers want a narrower range.
   Watch for a `v0.17.1`/`v0.18` fix and re-test then.
4. For local CLI work in this workspace, use a `cue` CLI at `v0.17.0-alpha.1`
   (matches the kernel) or `v0.16.0`; do not trust closedness errors from
   `alpha.2` through `v0.17.0`.

## References

- `modules/web_app/` — the module that surfaces it (clean on alpha.1).
- `opmodel.dev/catalogs/opm` `blueprints/workload/stateless_workload.cue`,
  `traits/scaling.cue` — the closed schemas being mis-rejected.
- `opmodel.dev/core` `component.cue` — `spec: close({_allFields})`.
- `library/docs/design/repro-hidden-field/` — a sibling CUE closedness
  investigation (different root cause: Go `FillPath` into a closed value).
