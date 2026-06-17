## Why

`Materialize` resolves a `#Platform`'s `#registry` subscriptions into a sealed `*MaterializedPlatform`. It does this by pulling each selected catalog build as its own CUE evaluation, indexing the transformers into a `#composedTransformers` map, and then `FillPath`-ing that map onto the platform value:

```
opm/materialize/materialize.go:  filled := p.Package.FillPath(schema.ComposedTransformers, composed)
```

`p.Package` is a **closed** `c.#Platform` value built independently of the pulled catalogs. Filling the composed map into it triggers exactly the ADR-003 failure mode: it corrupts lazy in-expression resolution of output-local hidden fields inside transformers (documented in `docs/design/transformer-output-hidden-field-scope-bug.md` §12). The current fix does not remove the seam — it **works around** it by exposing a second, *open* value:

```
opm/materialize/types.go:  MaterializedPlatform.Composed   // open map the executor must read instead of Package
```

with a standing WARNING that the executor MUST read transforms from `Composed`, never from the closed `Package`. That is a live tripwire: any future code that reads `#transform` off `Package` silently corrupts output, and `cue vet` will not catch it. The same cross-build closedness tension that produced two render-path workarounds (addressed in `simplify-render-single-build`) is present here in a third form.

This change rewrites materialize to **compose within a single CUE evaluation** (ADR-003), so the composed transformer map and the platform are one closed value with one set of definition identities — removing the need for the `Composed` escape hatch and the read-from-`Composed` tripwire.

## What Changes

- Rewrite the composition step in `opm/materialize` so the platform and the selected catalog builds are assembled in **one build** (synthesize a virtual package that imports the platform source and the selected catalog versions, and let CUE build `#composedTransformers` / `#matchers`), rather than `FillPath`-ing an independently-built composed map onto a closed `#Platform`.
- **Remove the `Composed` escape hatch** from `MaterializedPlatform` (or formally deprecate it) once the executor can read `#transform` off the single-build `Package` without corruption. Update `opm/compile/execute.go` and `opm/compile/module.go` to drop the `Composed` read path and the WARNING.
- Preserve all observable materialize behavior: subscription resolution, SemVer range/allow/deny filtering, stable-version default selection, transformer indexing, `MaterializedPlatform` lookups (`#composedTransformers`, `#matchers.{resources,traits}`), resolved-version-per-path diagnostics, `MaterializeError` shape, idempotency/non-mutation, opt-in cache, and concurrent read-only safety.
- Add a regression test that renders a transformer with output-local hidden fields directly off the materialized `Package` (the exact case the `Composed` hatch exists for) and asserts concrete output — proving the seam is gone, not relocated.

## Capabilities

### Modified Capabilities

- `platform-materialization`: the `MaterializedPlatform` is produced by single-build composition (ADR-003) rather than `FillPath` of a separately-built composed map onto a closed platform. Output shape, filtering, diagnostics, idempotency, caching, and concurrent-read-only semantics are preserved; the `Composed` field and the "executor must read transforms from `Composed`, not `Package`" constraint are removed because `Package` is now corruption-free.

## Impact

- **Affected packages**: `opm/materialize` (composition rewrite, `MaterializedPlatform` shape), `opm/compile` (`execute.go`, `module.go` drop the `Composed` read path), `opm/schema` (paths unchanged, verify). Tests across materialize + flow.
- **SemVer**: removing `MaterializedPlatform.Composed` is a **breaking change to a public `opm/` struct → MAJOR** (or a deprecation window if `Composed` is retained as an alias of `Package` for one release). Internal consumers (`opm/compile`) update in lockstep; external consumers are unlikely but MUST be flagged in `MIGRATIONS.md`. The behavioral contract (materialize output) is preserved.
- **Risk / size**: larger than the render change. Materialize is inherently "gather from N registry pulls"; expressing that as one CUE build is a real redesign, not a mechanical refactor. This change is **spike-first** (see design) — the spike decides whether single-build composition is achievable as designed or needs a fallback. Scope is deliberately limited to the composition mechanism; subscription/filter/cache logic is reused as-is.
- **Sequencing**: independent of `simplify-render-single-build` (different packages, different seam) but shares ADR-003. Recommended to land the render change first (smaller, proves the pattern) and carry the lessons here.
- **Out of scope**: any change to subscription/filter semantics or the cache interface; the v0.17 concurrency contract (preserved, not modified); render-path construction (sibling change).
