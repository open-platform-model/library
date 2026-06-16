# CUE closedness regression: `v0.17.0-alpha.2`+ rejects valid optional nested-struct fields

Status: Investigation / upstream bug report (2026-06-16). Affects the OPM CUE
toolchain choice. **Not** an OPM schema or catalog defect — the same sources
validate cleanly on `v0.16.0` and `v0.17.0-alpha.1`.

## Summary

Starting with `cuelang.org/go v0.17.0-alpha.2`, the CUE evaluator reports
spurious `field not allowed` (closedness) errors for **optional nested struct
fields that are explicitly declared in the closed schema**. The same modules
validate cleanly on `v0.16.0` and `v0.17.0-alpha.1`. The regression was
introduced between `alpha.1` and `alpha.2` and persists in `alpha.3`.

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

† The metadata-self-cycle column is a **separate, real** OPM schema bug (a
`#Module.metadata` self-reference), fixed independently and orthogonal to this
toolchain regression — it fails on every version. It is included only to show the
columns measure different things: the **left** column is the toolchain
regression (changes between alpha.1 and alpha.2); the right does not.

## Reproduction

Prereqs: a local OCI registry at `localhost:5000` holding `opmodel.dev/core@v0`
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

### Minimal in-package reproductions do NOT trigger it

Several reduced, single-package models of the above (a `close({...})` with the
same propagation `scaling: spec.statelessWorkload.scaling`, including a
comprehension over a `#traits` map) evaluate **cleanly on alpha.3**. The trigger
therefore appears to require the real multi-layer, cross-package closed-type
embedding (core `#Component` → catalog `#Trait`/`#Blueprint` → `#ScalingSchema`),
not just the surface shape. This narrows the regression to closedness propagation
across module/definition boundaries rather than a single local pattern.

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
   `go.mod` until this is resolved upstream.
2. **Before any bump past alpha.1**, re-run the matrix above (and the operator's
   `task dev:test:local` render integration tests) against the candidate version.
3. **Report upstream** to `cuelang.org/go` with the reproduction above; offer to
   bisect alpha.1→alpha.2 if maintainers want a narrower commit range.
4. For local CLI work in this workspace, use a `cue` CLI at `v0.17.0-alpha.1`
   (matches the kernel) or `v0.16.0`; do not trust closedness errors from
   `alpha.2`/`alpha.3`.

## References

- `modules/web_app/` — the module that surfaces it (clean on alpha.1).
- `opmodel.dev/catalogs/opm` `blueprints/workload/stateless_workload.cue`,
  `traits/scaling.cue` — the closed schemas being mis-rejected.
- `opmodel.dev/core` `component.cue` — `spec: close({_allFields})`.
- `library/docs/design/repro-hidden-field/` — a sibling CUE closedness
  investigation (different root cause: Go `FillPath` into a closed value).
