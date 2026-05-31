## Context

The compile pipeline borrows its `*cue.Context` from the **materialized platform** rather than from the caller Kernel. Two non-test sites do this:

```
Kernel.Compile
  └─ compileModuleRelease(ctx, rel, mp, runtimeName)          // kernel/compile.go — free func
       ├─ compile.FinalizeValue( mp.Package.Context() , …)    // (1) ctx from PLATFORM  ← replace
       ├─ compile.Match(schemaComponents, mp, releaseName)    //     builds nothing; unify receiver = component (caller) ✓
       └─ compile.NewModule(mp, runtimeName).Execute(…)
            └─ cueCtx := r.platform.Package.Context()          // (2) ctx from PLATFORM  ← replace
               executeTransforms(cueCtx, mp.Package, …)
                 └─ transformVal.FillPath(#component, dataComp) //     receiver = platform value (shared) — keep
                    schema.BuildTransformerContext(cueCtx, …)   //     builds #context.* with the sourced ctx
```

Today this is harmless: a single-Kernel caller materializes the platform in its own context, so `mp.Package.Context()` *is* the caller's context and renders are correct. But it funnels every render through the platform's one context (serializing concurrency) and relies on `cue.Value.Context()`, which CUE v0.17 deprecates.

The `opm-operator` rewrite needs to render many `ModuleRelease`s concurrently against one `*MaterializedPlatform` materialized **once** per Platform-CR generation. The spike (`opm-operator/KERNEL-CONCURRENCY-SPIKE-RESULTS.md`; `adr/002`) proved the per-goroutine-Kernel + one-shared-read-only-platform model is race-clean and ~2.5× faster — **on CUE v0.17**, where cross-context `FillPath`/`Unify` is legal. On the current v0.16.1 pin, cross-Kernel rendering panics (`values are not from the same runtime`).

Constraint: this change touches only `opm/compile` + `opm/kernel` Go code, makes **no** CUE-version bump, and must stay behavior-preserving on v0.16.1 (existing suite green). Library Principle VIII (small batches) scopes it to the context-threading refactor alone.

## Goals / Non-Goals

**Goals:**

- The compile pipeline builds every value it constructs in the **caller Kernel's** `*cue.Context` (`k.CueContext()`), consuming the materialized platform as read-only input — removing both platform-derived `Value.Context()` sites.
- Behavior is identical for today's single-Kernel callers; the existing compile + kernel suites pass unchanged on the v0.16.1 pin.
- The pipeline is positioned so that, once v0.17 is pinned, per-goroutine Kernels can render concurrently against one shared platform with no further pipeline restructuring.

**Non-Goals:**

- Pinning CUE → v0.17 (alpha); the production `go.mod` stays at v0.16.1.
- Asserting cross-goroutine / cross-Kernel sharing safety in specs (the **Goroutine Safety Contract** stays unchanged — it is still false on v0.16). Deferred to the v0.17 change.
- The permanent concurrent `-race` regression test (needs a v0.17-compatible in-memory catalog; published `catalogs/opm@v0.4.0` does not parse under v0.17).
- A full sweep of `Value.Context()` across `materialize` / `schema`.
- Any change to the `Kernel` public API or to downstream `cli` / `opm-operator`.

## Decisions

### D1: The build context comes from the caller Kernel, not the platform

`compileModuleRelease` sources `k.cueCtx` and threads it into `FinalizeValue` and the Execute path. The materialized platform's `Package` is read-only input — the `FillPath` argument / cross-read source — never the owner of the build context.

- *Why:* on v0.16 single-Kernel callers `k.cueCtx == mp.Package.Context()`, so this is behavior-identical and the suite stays green; it removes the deprecated `Value.Context()`; post-v0.17 it lets each goroutine's Kernel build in its own context while cross-reading one shared platform — the proven ~2.5× model.
- *Alternative:* keep sourcing from the platform and rely on v0.17 alone → leaves the deprecated call and keeps every render funnelled through the platform's single context (serializing). Rejected.

### D2: Thread the context into Execute via the `Module` constructor

`Module` gains a `cueCtx *cue.Context` field; `compile.NewModule(cueCtx, mp, runtimeName)` takes it as the leading parameter; `Execute` reads `r.cueCtx`. `Execute`'s own signature is unchanged (`executeTransforms` already accepts a `cueCtx` param — it now gets the caller's).

- *Why:* `NewModule` is already on a deprecation arc (`//nolint:SA1019` at `kernel/compile.go:50`), so adding its leading context param touches the smallest exported surface; it mirrors the package's own convention — `compile.FinalizeValue(cueCtx, v)` already takes the context first.
- *Alternatives:* (1) add a `*cue.Context` param to `Execute` → changes a second exported signature for no gain; (2) a parallel `NewModuleWithContext` constructor → expands the public surface against YAGNI (Principle VII). Both rejected.

### D3: `Match` is left unchanged

`unifyIntersection` (`match.go:263`) does `cv.Unify(iter.Value()).Validate(cue.Concrete(false))` where `cv` is the component value (caller-built) and `iter.Value()` is the transformer requirement (platform); the unified value is validated and discarded.

- *Why:* the `Unify` receiver is already the caller's component value, so the result resolves in the caller's context — the orientation the v0.17 model wants — and nothing is built that borrows the platform's context. The only v0.16 failure is the cross-context panic itself, resolved by the v0.17 pin (deferred), not by any code change here.
- *Alternative:* add a context param to `Match` for symmetry → an unused parameter today, violating YAGNI. Rejected. (No `platform-matching` spec delta follows.)

### D4: SemVer — behavior-preserving for Kernel callers, breaking for `compile.NewModule`

Classified as a refactor that preserves `Kernel` behavior while breaking the exported `compile.NewModule` signature (leading `*cue.Context`). Recorded in `MIGRATIONS.md`; release tooling decides the bump.

- *Why:* `compile` has "no public entry; called via Kernel" — the only in-tree caller is `kernel/compile.go`, and downstream binaries consume the unchanged `Kernel` surface.
- *Alternative:* preserve `NewModule` and overload elsewhere → parallel surface, rejected per D2.

### D5: v0.17 pin and concurrency enablement are a separate, gated change

This change lands the v0.16-safe half only. The v0.17 pin, the Goroutine-Safety-Contract / MaterializedPlatform spec rewrites, and the concurrent `-race` test ship later.

- *Why:* v0.17 is alpha and the published catalogs do not parse under it (see Risks); bundling the pin would block this landing on cross-repo preconditions and breach the small-batch gate. The split lets the deprecation fix + parallelism enabler land now.

## Risks / Trade-offs

- **A latent cross-context dependency elsewhere in the pipeline** → mitigated by Task 0.1's `grep` pre-flight asserting exactly the two known sites, plus the unchanged-suite gate.
- **The change is invisible on v0.16** (single-Kernel callers see identical output, so a regression could hide) → mitigated by Task 3.1 asserting a rendered value's context *is* `k.CueContext()` — a direct, observable check of the new contract independent of output equality.
- **`compile.NewModule` signature break** → low blast radius (kernel-only caller); mitigated by the `MIGRATIONS.md` recipe; `Kernel` API untouched.
- **The actual concurrency win is unverified here** (no v0.17 → no parallel test) → accepted: the keystone (`adr/002` §3) already proved the primitive; this change only prepares the pipeline, and the v0.17 follow-up adds the regression test.

## Migration Plan

Single commit group (one focused refactor; no phasing):

1. `compile/module.go` — add `Module.cueCtx`; change `NewModule(cueCtx, mp, runtimeName)`; `Execute` reads `r.cueCtx` instead of `r.platform.Package.Context()`. Update compile tests to pass their context.
2. `kernel/compile.go` — make `compileModuleRelease` a method on `*Kernel`; use `k.cueCtx` for `FinalizeValue` and for `compile.NewModule(k.cueCtx, …)`. Update the `Kernel.Compile` call site in `phases.go`.
3. Add the context-identity assertion; `MIGRATIONS.md` entry; flip `adr/002` to Accepted.

```go
// kernel/compile.go
func (k *Kernel) compileModuleRelease(ctx context.Context, rel *module.Release,
    mp *materialize.MaterializedPlatform, runtimeName string) (*compile.CompileResult, error) {
    dataComponents, err := compile.FinalizeValue(k.cueCtx, schemaComponents)        // (1)
    plan, err := compile.Match(schemaComponents, mp, rel.Metadata.Name)             // unchanged
    return compile.NewModule(k.cueCtx, mp, runtimeName).Execute(ctx, rel, …, plan)  // (2)
}
// compile/module.go — Module gains cueCtx; Execute uses r.cueCtx, not r.platform.Package.Context().
```

Rollback: a single self-contained commit; revert restores the platform-sourced context. There is no data/registry migration and no CUE-version change, so no external state to undo.

## Open Questions

None. Scope, the two target sites, the threading mechanism (D2), the `Match` no-op (D3), and the v0.17 split (D5) are all resolved. The remaining unknowns (stable v0.17, v0.17 catalog re-publish) belong to the deferred follow-up, not this change.
