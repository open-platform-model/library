## Context

The kernel's job is to deterministically transform `(Module + ModuleRelease + Platform + Values)` into `[]Rendered`. Today the values input is `[]cue.Value` and the kernel unifies internally. That bakes one merge order — `merged := values[0]; for _, v := range values[1:] { merged = merged.Unify(v) }` — into the kernel.

Different frontends layer values differently:

- **CLI**: defaults → `-f file1.cue` → `-f file2.cue` → `-f overlay.cue`. Order is command-line-driven.
- **Operator**: defaults → ConfigMap → Secret → ModuleRelease CR overlay. Order is fixed but the sources are Kubernetes objects, not files.
- **Crossplane fn**: composition input is one structured object; layering happens upstream of the function.

Forcing each frontend through the kernel's slice merge means each one fights the kernel's order. Source-attribution (which layer caused this validation error?) is also lost — the kernel sees only `[]cue.Value`, not `[]LabeledLayer`.

This slice narrows the kernel's contract: the kernel takes one `values cue.Value`, already unified. Layering is a frontend / helper concern. Slice 05 (`introduce-tiered-validation`) introduces `pkg/helper/values/` to standardize the layering implementation across frontends and add source-positioned Tier-1 validation.

## Goals / Non-Goals

**Goals:**
- Kernel signature accepts a single `cue.Value` of values, not a slice.
- Kernel still validates the unified value against `#config` (Tier-2 correctness safety net per umbrella D1, D5).
- Provide a temporary `UnifyAndValidate` helper for downstream consumers to migrate incrementally.

**Non-Goals:**
- Defining the layering convention. That is slice 05.
- Removing Tier-2 validation. The kernel never trusts that Tier-1 ran; it always validates.
- Position-rich diagnostics. Tier-2 produces correctness errors; Tier-1 (slice 05) produces source-attributed errors.

## Decisions

**Single signature change, not a deprecation pair.** Change `validate.Config` and `module.ParseModuleRelease` outright (MAJOR bump). Reason: the slice signature was always a leaky abstraction; keeping a deprecated `[]cue.Value` form alongside the new `cue.Value` form doubles the surface and confuses migrations.

**Temporary `validate.UnifyAndValidate` helper.** A single function that takes `[]cue.Value`, unifies via the same loop the kernel used to do, then calls the new single-value `Config`. Marked `// Deprecated: use pkg/helper/values for layering and call validate.Config with the unified result` from day one. Removed when slice 05 ships.

**Empty values: zero-value `cue.Value` vs. nil.** The new signature accepts `cue.Value`. An "absent" values is the zero value (`cue.Value{}`); the validator already handles this case (returns success when no values to check). Document explicitly.

**Tier-2 validation runs unconditionally.** Even when called from Tier-1-aware helpers that already validated, the kernel re-validates. This is the correctness safety net. Cost is negligible compared to render itself.

**Internal helper for the merge loop.** Move the previous `for _, v := range values[1:] { merged = merged.Unify(v) }` loop into `validate.UnifyAndValidate` so the kernel internals do not retain a slice merge path that could confuse future readers.

## Risks / Trade-offs

**Risk — downstream consumer breakage.** `cli` and `opm-operator` pass `[]cue.Value` today. Mitigation: ship `UnifyAndValidate` as a one-line migration ("change `validate.Config(s, vs, ...)` to `validate.Config(s, validate.UnifyAndValidate(vs), ...)`"). Loud CHANGELOG entry and migration recipe.

**Risk — source-attribution regression.** Today, `validate.Config` sometimes can attribute errors to one source by walking each `cue.Value` in the slice individually. Once unified, that attribution is lost (CUE positions point to source files, not to "layer N"). Mitigation: this is the entire point of slice 05's helper — it gets per-source validation BEFORE unification. For this slice in isolation, downstream consumers may briefly see worse error messages until slice 05 lands and they migrate.

**Risk — `Tier 1` becomes optional and skippable.** Frontends in a hurry skip helper validation entirely. Mitigation: doc and CHANGELOG strongly recommend the helper. The kernel's Tier-2 net catches actual correctness; only diagnostic quality suffers if Tier-1 is skipped.

**Trade-off — temporary helper is dead code on day one for new consumers.** A new frontend writing against this kernel never has reason to call `UnifyAndValidate`. That's fine — it serves migrating consumers only and goes away with slice 05.
