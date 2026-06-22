# Release CR: imported published module fails to load on `core@v0.4.0` (FIXED — released as `core@v0.5.0`)

**Status:** RESOLVED and PUBLISHED. Fixed by `core` commit `68e4520`
(*feat(module): make #Module identity author-supplied (fix self-cycle
re-admission)*), shipped in the `core@v0.5.0` release (2026-06-16). Verified
2026-06-17 against the pinned toolchain (`cuelang.org/go v0.17.0-alpha.1`).
Consumers on `core@v0` ≥ `v0.5.0` get the fix automatically; anything still
pinned to `v0.4.0` (e.g. the `opm-kind-demo` registry) hits the old failure.
**Affects:** `Release` CRs whose `release.cue` imports a published `#Module`
(`#module: <registry import>`). Does **not** affect `ModuleRelease`, and does
**not** affect `Release` packages that author the module inline.

This note records the empirical resolution of the issue documented in
[`opm-operator/docs/design/release-vs-modulerelease-render-divergence.md`](../../../opm-operator/docs/design/release-vs-modulerelease-render-divergence.md)
(its recommended durable fix — Option 2, fix the closedness in `core@v0` —
shipped in `core@v0.5.0`). It is a sibling to the toolchain regression in
[`cue-closedness-regression-alpha2.md`](./cue-closedness-regression-alpha2.md),
whose footnote already flagged the `#Module.metadata` self-reference as "a
separate, real OPM schema bug … fixed independently." This is that fix,
verified end-to-end.

## Summary

The operator (and the CLI) can author a `Release` two ways:

- **inline** — `release.cue` declares the whole `#Module` body itself
  (`#module: { core.#Module, metadata: …, #components: … }`). One `#Module`
  instance in one evaluation; nothing is re-unified. **Works.**
- **imported** — `release.cue` imports a *published* module and binds it
  (`#module: webapp`). This re-unifies an already-closed, published `#Module`
  into `#ModuleRelease.#module!: #Module`. On `core@v0.4.0` this **failed**:

  ```
  #module.metadata.modulePath: invalid interpolation: field not allowed
  #module.metadata.version:    invalid interpolation: field not allowed
  ```

The imported shape is the ergonomic one (reuse a registry module; values in the
CR), so "Release works" is only half true on the published schema — the inline
shape works, the imported shape did not.

## Reproduction (pinned toolchain)

The toolchain matters: a `cue` CLI at `v0.17.0-alpha.2`+ adds a *separate*
spurious closedness error (see `cue-closedness-regression-alpha2.md`), which
masks this one. Reproduce on `v0.17.0-alpha.1` — the version the kernel pins —
so the result is unambiguous.

```bash
GOBIN=/tmp/cue-a1 go install cuelang.org/go/cmd/cue@v0.17.0-alpha.1
export CUE_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'
```

A scratch module pinning `opmodel.dev/core@v0` + a published module
(`opmodel.dev/modules/web_app@v0.0.1`), with `release.cue`:

```cue
import (
    core   "opmodel.dev/core@v0"
    webapp "opmodel.dev/modules/web_app@v0"
)
imported: core.#ModuleRelease & {
    metadata: { name: "web-app", namespace: "web-app" }
    #module: webapp
    values: { image: {repository: "nginx", tag: "1.27", digest: ""}, replicas: 1, port: 80, serviceType: "ClusterIP" }
}
```

| core version pinned  | `cue eval ./imported.cue`                                   |
| -------------------- | ----------------------------------------------------------- |
| `v0.4.0`             | **FAIL** — `#module.metadata.modulePath: field not allowed` |
| `v0.5.0` (the fix)   | **OK** — `metadata.name → "web-app"`, `#module.metadata.fqn → "opmodel.dev/modules/web-app:0.1.0"`, `imported.components` materialize concretely |

(During investigation the `v0.5.0` row was reproduced by publishing the
equivalent `core/src` to a local registry as a throwaway tag; the released
`core@v0.5.0` is byte-equivalent for `#Module.metadata`.)

## Root cause

`core@v0.4.0`'s `#Module.metadata` declared its identity fields
**self-referentially** (`modulePath: metadata.modulePath`,
`version: metadata.version`), with `fqn` interpolating them. Binding an
already-closed published `#Module` into `#ModuleRelease.#module!: #Module`
re-admits the imported module's concrete `metadata` against a freshly
re-evaluated `#Module` whose self-references resolve to bottom — so the
caller-supplied `modulePath` / `version` are rejected as "field not allowed."
This is independent of the module's contents and of the published version, and
it reproduces on `v0.16.0` / `alpha.1` as well as the `alpha.2+` regression —
i.e. it is a genuine schema defect, not the toolchain regression.

### Why `ModuleRelease` was never affected

`ModuleRelease` never re-unifies a published module in CUE. The controller
calls `synth.Release` (`library/opm/helper/synth/release.go`), which sidesteps
the closedness with a Go-side workaround: it builds a scope value with a
non-hidden `userModule` field, fills the module there, and compiles
`#module: userModule` via `cue.Scope` — the module enters as a *value*, not a
re-emitted source fragment, so there is no second closed copy to re-admit. The
`Release` load path (`helper/loader/file.LoadReleasePackage`) has no equivalent
escape hatch: it evaluates the author's `release.cue` verbatim, which performs
exactly the `#module: <import>` unification the workaround exists to avoid.

## Resolution

`core@v0.5.0`'s `#Module.metadata` declares the identity fields as plain
required fields (`modulePath!: #ModulePathType`, `version!: #VersionType`)
rather than self-references. Verified above: re-unifying a published module
against this schema admits the concrete `metadata` and renders components
concretely. This is the operator doc's Option 2 (durable fix in `core@v0`), and
it also makes `synth.Release`'s `userModule`-scope workaround droppable for any
consumer whose floor is ≥ `v0.5.0`.

## Remaining work

Status as of 2026-06-17. Items 2, 3, and the doc-refresh below are owned by the
active library OpenSpec change **`simplify-render-single-build`** (do not open
new changes for them — they would duplicate/collide with in-flight work). That
change applies ADR-003 (the no-cross-build-FillPath invariant, via its
single-build tactic for the render path) to converge both render paths and
delete the workarounds; its task 0.1 already records
`core@v0.5.0` as the met precondition.

1. **Bump stale consumers off `v0.4.0`.** — **DONE.** `opm-kind-demo`'s
   `web_app/cue.mod/module.cue` pins `opmodel.dev/core@v0` → `v0.5.0`; no live
   `v0.4.0` pin remains in the demo.
2. **Drop the `synth.Release` workaround** — **TRACKED** in
   `simplify-render-single-build` (tasks 3.1–3.3: rewrite `Release` onto a
   single-build virtual package, delete `buildReleaseScope` / the `userModule`
   `cue.Scope`, and delete the `FillPath(#config)` Go pre-merge). The fix goes
   beyond "let `ModuleRelease` re-unify directly" — it merges both paths onto one
   mechanism. The workaround + its now-stale `v0.4.0` comment in
   `opm/helper/synth/release.go` remain until that change lands.
3. **Add `LoadReleasePackage` coverage** — **TRACKED** in
   `simplify-render-single-build` (task 4.7: direct `LoadReleasePackage` import
   test; 4.4: imported-module integration test with a `v0.4.0` negative control;
   4.6: retire the `core/v1alpha1` fixture; 4.8: registrytest core-version
   override; 4.9: registry/`-short` gating). The old `test/fixtures/releases/hello/`
   has since been removed, so the v1alpha1 fixture is no longer present (it was
   deleted, not yet replaced with the `v0` imported-module fixture above).
4. **Correct stale claims** — **DONE.** `opm-kind-demo/web_app/release.cue` +
   `web_app/README.md` now state the import works on `core@v0` ≥ `v0.5.0` and
   describe the `v0.4.0` failure as historical. The operator render-divergence
   doc (`opm-operator/docs/design/release-vs-modulerelease-render-divergence.md`)
   was updated 2026-06-17 to a split status: closedness failure RESOLVED in
   `core@v0.5.0`; render-path divergence OPEN, tracked in
   `simplify-render-single-build`. The doc-refresh of *this* file is itself
   tracked as task 5.2 of that change.

## References

- `core/src/module.cue:11-19` — the (now non-self-referential) `#Module.metadata`.
- `core/src/module_release.cue` — `#module!: #Module & {#ctx: …}`.
- `library/adr/003-no-cross-build-fillpath-into-closed-values.md` — ADR-003, the no-cross-build-FillPath invariant; `simplify-render-single-build` applies its single-build tactic to the render path.
- `library/opm/helper/synth/release.go` — `synth.Release` now constructs the release via single-build CUE evaluation of a synthesized package that **imports** the module; the `userModule`-scope workaround and the Go `#config` pre-merge were deleted by `simplify-render-single-build`.
- `library/opm/helper/loader/file/build.go` — the shared `buildAndShapeGate` both `synth.Release` (overlay source) and `LoadReleasePackage` (on-disk source) now run.
- `library/opm/helper/loader/file/release.go` — `LoadReleasePackage` (re-pointed at the shared build step).
- `opm-operator/docs/design/release-vs-modulerelease-render-divergence.md` — full two-path analysis.
- `library/docs/design/cue-closedness-regression-alpha2.md` — the separate `alpha.2+` toolchain regression that masks this on a newer CLI.
