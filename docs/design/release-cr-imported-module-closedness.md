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

1. **Bump stale consumers off `v0.4.0`.** The demo registry
   (`opm-kind-demo`, `CORE_VERSION: v0.4.0`) still pins the broken schema, so the
   demo's optional `Release` path can't use the imported shape until it moves to
   `v0.5.0`.
2. **Optionally drop the `synth.Release` workaround** — with a
   non-self-referential `#Module`, `ModuleRelease` can re-unify the acquired
   module directly. Gate on a guaranteed `≥ v0.5.0` floor before removing it.
3. **Add `LoadReleasePackage` coverage** — no test renders a real *imported*
   module through the `Release` load path; `test/fixtures/releases/hello/` is
   still on retired `core/v1alpha1`. Add a `core@v0` (≥ v0.5.0) imported-module
   fixture and a registry-gated integration test so the contract can't silently
   rot again.
4. **Correct stale claims** — `opm-kind-demo/web_app/release.cue` +
   `web_app/README.md` assert the import path "fails to load"; true only against
   `v0.4.0`, fixed in `v0.5.0`. The operator render-divergence doc's status
   should move to "resolved, released as core@v0.5.0."

## References

- `core/src/module.cue:11-19` — the (now non-self-referential) `#Module.metadata`.
- `core/src/module_release.cue` — `#module!: #Module & {#ctx: …}`.
- `library/opm/helper/synth/release.go:111-189` — the `userModule`-scope workaround + comment.
- `library/opm/helper/loader/file/release.go` — `LoadReleasePackage` (no workaround).
- `opm-operator/docs/design/release-vs-modulerelease-render-divergence.md` — full two-path analysis.
- `library/docs/design/cue-closedness-regression-alpha2.md` — the separate `alpha.2+` toolchain regression that masks this on a newer CLI.
