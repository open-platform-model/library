# CUE closedness regression: `v0.17.0-alpha.2`+ rejects valid optional nested-struct fields

Status: Investigation / upstream bug report (2026-06-16). **Re-verified against
`v0.17.1` on 2026-07-16 — still NOT fixed.** `v0.17.1` closed and fixed
[cue-lang/cue#4423](https://github.com/cue-lang/cue/issues/4423) *as filed*, but
that is a different symptom; OPM's symptom survives unchanged and is not yet
reported upstream (see [Root cause + upstream status](#root-cause--upstream-status)).
**Not** an OPM schema or catalog defect — the same sources validate cleanly on
`v0.16.0` and `v0.17.0-alpha.1`.

**The toolchain has since moved to `v0.17.1` (2026-07-16), i.e. onto an affected
version.** Rendering is green only because the catalog carries the hoisted-guard
workaround. That workaround is now load-bearing rather than precautionary — do
not retire it. `opm/internal/cueregression/closedness_test.go` is the live
canary pair that will fail when upstream finally fixes this.

## Summary

Starting with `cuelang.org/go v0.17.0-alpha.2`, the CUE evaluator reports
spurious `field not allowed` (closedness) errors for **optional nested struct
fields that are explicitly declared in the closed schema**. The same modules
validate cleanly on `v0.16.0` and `v0.17.0-alpha.1`. The regression was
introduced between `alpha.1` and `alpha.2` and **persists through `alpha.3`,
`rc.1`, the official `v0.17.0` release, and `v0.17.1`** (confirmed 2026-07-16).
It is reproducible both via the `cue` CLI and via the Go kernel at module-build
time.

This used to matter because the kernel pinned the last clean toolchain
(`v0.17.0-alpha.1`). Since 2026-07-16 the kernel pins `v0.17.1`, an affected
version, so the *only* thing preventing the evaluator from rejecting real
catalog-backed modules at render time is the hoisted-guard form in
`catalog_opm/src/blueprints/workload/*.cue`. Reverting those guards breaks
rendering immediately.

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
| `v0.17.1`          | **FAIL** (`field not allowed`)             | —                              |

**2026-07-04 re-run** used the current module deps (`opmodel.dev/catalogs/opm@v1.0.0-alpha`,
`opmodel.dev/core@v1.0.0-alpha.1`; the earlier `@v0.5.2` catalog is gone from the
registry). The trigger construct survived the catalog `v0.5.2 → v1.0.0-alpha`
bump — the left-column result is unchanged from the original investigation.

**2026-07-16 re-run (`v0.17.1`)** required rebuilding the trigger, because the
catalog now ships the hoisted-guard workaround and therefore no longer carries
it. The pre-workaround blueprint was restored from `catalog_opm` commit `c0cd3ec`
and wired into the `web_app` fixture via `cue.mod/local-module.cue`. Result:
`alpha.1` clean, `v0.17.0` FAIL, `v0.17.1` FAIL — `alpha.1` remains the newest
clean toolchain even though the project has deliberately moved past it.

Two measurement traps cost real time here; heed them on any re-run:

- **The CUE module cache is keyed by `module@version`, not by registry.**
  Publishing different content under the same `module@version` to a different
  registry resolves the *stale cached copy* instead, silently. Use a fresh
  `CUE_CACHE_DIR` per measurement (the cache lives at `~/.cache/cue`, not
  `~/.cache/cuelang`), or replace the dep with `cue.mod/local-module.cue`, which
  bypasses the cache entirely.
- **`language.version` does not gate this bug** — varied independently on both the
  main module and the dependency, with no effect. Only the tool/SDK version
  matters. (This retroactively invalidates the premise of `core` commit
  `40daf05`, which pinned `language.version` to `alpha.1` believing it would
  avoid the bug; that pin was inert and has since been reverted.)

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

### Minimal in-package reproduction — SUPERSEDED 2026-07-16: one exists

This section previously concluded that reduced single-package models evaluate
cleanly and that the trigger "requires the real multi-layer, cross-package closed-type
embedding … not just the surface shape". **That conclusion was wrong** — it
reflected reductions that dropped a load-bearing element, not an inherent need
for cross-package structure.

A single-file, dependency-free reproduction now lives at
`docs/design/repro-cue-closedness/` (ready to file upstream) and is asserted —
as an embedded string, hermetically — by
`opm/internal/cueregression/closedness_test.go`. No core, no catalog, no
registry, no `close()`:

```cue
#Inner: {b?: {n: int}}
#Base: {#parts: {...}, out: {for _, p in #parts {p}}}
#Derived: #Base & {
	#parts: only: a: #Inner
	out: {
		a: #Inner
		if out.a.b != _|_ {a: {}}   // CONDITION traverses a struct-typed field of closed #Inner
	}
}
x: {#Derived, out: a: b: n: 2}     // v0.17.0/v0.17.1: "x.out.a.b: field not allowed"
```

Six elements are individually load-bearing — removing any one makes it clean on
every version. This is what earlier reductions kept losing:

1. `b` must be **struct-typed** (a scalar `b: int` is exempt).
2. `#Inner` must be a **definition** (closedness must originate somewhere).
3. `#Base` must build `out` via a **field comprehension**.
4. `#Derived` must extend it via **unification** (`#Base & {…}`), not inline.
5. The **condition** is the trigger, not the body (the body never mentions `b`).
6. A **concrete usage** is required; definitions alone do not trigger it.

Notably `close()` is **not** required — the real catalog's `spec: close({_allFields})`
is incidental — and neither is optionality (`b: {n: int}` reproduces too).

Element 1 independently explains the field-level pattern seen in the real
catalog: the error names `scaling` and `updateStrategy` (structs) but never
`restartPolicy` (a scalar enum). Two independent lines of evidence agreeing is
the strongest signal the rule is right.

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

**That canary is now dead (2026-07-16).** It resolves the *published* catalog,
and the published catalog now carries the hoisted-guard workaround — so the
trigger it depended on is gone. Measured: it PASSES on `v0.17.0`, the known-bad
version. It can no longer distinguish a fixed CUE from a broken one and must not
be used as a bump gate; it green-lit `v0.17.1` while proving nothing.

Its replacement is `opm/internal/cueregression/closedness_test.go`, which
evaluates the trigger shape directly from an embedded string (no registry, no
catalog, no filesystem) and asserts the bug is still present, plus a hoisted-form
twin asserting the workaround shape stays clean. The original single-canary form
(`opm/kernel/cue_closedness_regression_test.go`, since superseded) was verified
to discriminate: PASS on `v0.17.1` (bug present), FAIL on `v0.17.0-alpha.1` (bug
absent).

`opm/materialize` package tests pass on both pins — that path fills already-built
values rather than re-validating the closed component spec, so it does not
re-trigger the closedness check. The regression bites specifically where the Go
kernel **builds/validates** a catalog-backed `#Module` (`Kernel.Validate` /
`ValidateRealConfig`), i.e. the exact render-time path a pin bump would break.
(The `TestFlow_WebApp_OnOpmPlatform` catalog FQN version-skew referenced here was
resolved by the 2026-07-16 fixture migration to `catalogs/opm@v1` (#38); the full
suite is green on both pins as of that change.)

## Root cause + upstream status (2026-07-04)

**Upstream issue: [cue-lang/cue#4423](https://github.com/cue-lang/cue/issues/4423)**
("Field not allowed regression in v0.17.0", opened 2026-07-02). **Closed COMPLETED
2026-07-10 and fixed in `v0.17.1` — but only for the symptom as filed, which is
not ours.** Verified 2026-07-16:

| reproduction | `v0.16.1` | `alpha.1` | `v0.17.0` | `v0.17.1` |
| --- | --- | --- | --- | --- |
| upstream's, as filed (`adding field … not allowed as field set was already referenced`) | clean | clean | FAIL | **fixed** |
| OPM's (`… field not allowed`) | clean | clean | FAIL | **still FAILS** |

Different error strings, different code paths — two distinct bugs that were
conflated because they bisect to the same commit and both arrived in alpha.2.
The `v0.17.1` release notes naming #4423 therefore do **not** mean OPM is clear.
**OPM's symptom has not been reported upstream.** The single-file reproduction in
`docs/design/repro-cue-closedness/` is ready to file as-is.
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

- **Today (since 2026-07-16):** none at runtime, but for a different reason than
  before. The kernel (`library`/`opm-operator`) now pins `cuelang.org/go v0.17.1`
  — an **affected** version. Rendering is green solely because the catalog no
  longer contains the trigger: all five workload blueprints use the hoisted-guard
  form. The safety margin is the workaround, not the toolchain.
- **If the workaround is reverted:** the evaluator immediately rejects every
  catalog-backed module using the workload blueprints with `scaling`/
  `updateStrategy` set — real rendering breaks, not just the CLI. This is why
  `catalog_opm`'s authoring rule is permanent and why the canary exists.
- **Historically (pre-2026-07-16):** the kernel pinned the clean `alpha.1`, so
  only a developer running a newer `cue` CLI locally saw the false errors — and
  could misdiagnose a module as broken, which happened during the
  [release-vs-modulerelease render-divergence](../../../opm-operator/docs/design/release-vs-modulerelease-render-divergence.md)
  investigation.

## Recommendations

Superseded 2026-07-16. Recommendation 1 previously read "Stay on
`cuelang.org/go v0.17.0-alpha.1` … do not bump"; the project has deliberately
moved to `v0.17.1` on the strength of the workaround, accepting that the bug is
still live. Current guidance:

1. **Never revert the hoisted-guard form** in `catalog_opm/src/blueprints/workload/*.cue`.
   It is the only thing keeping the toolchain's live bug off OPM's rendering
   path. Treat `catalog_opm/docs/cue-guard-closedness-workaround.md` as a
   permanent authoring rule, not a temporary patch.
2. **The bump gate is `opm/internal/cueregression/closedness_test.go`**, not
   `TestIntegration_Live_ValidateRealConfig` (dead — see above). When the
   trigger-form canary starts failing, upstream has fixed OPM's symptom: at that
   point re-run this matrix, then retire the authoring rule, the reproducer, and
   the canary together. The hoisted-form twin failing instead means the
   workaround shape itself broke — do not adopt that CUE version.
3. **Report OPM's symptom upstream.** #4423 is closed, so this will not be picked
   up as part of it. `docs/design/repro-cue-closedness/` is a single-file,
   dependency-free reproduction ready to file.
4. For local CLI work, a `cue` CLI at `v0.17.1` matches the kernel. Closedness
   errors from `alpha.2` onward remain untrustworthy on the *pre-workaround*
   shape — if a module trips `field not allowed` on a nested struct field the
   schema clearly declares, suspect this bug before suspecting the module, and
   check whether an in-`spec` guard crept in.
5. When measuring across CUE versions, use a fresh `CUE_CACHE_DIR` and prefer
   `cue.mod/local-module.cue` for dep replacement — see the version-matrix
   section for why the cache will otherwise lie to you.

## References

- `modules/web_app/` — the module that surfaces it (clean on alpha.1).
- `opmodel.dev/catalogs/opm` `blueprints/workload/stateless_workload.cue`,
  `traits/scaling.cue` — the closed schemas being mis-rejected.
- `opmodel.dev/core` `component.cue` — `spec: close({_allFields})`.
- `library/docs/design/repro-hidden-field/` — a sibling CUE closedness
  investigation (different root cause: Go `FillPath` into a closed value).
