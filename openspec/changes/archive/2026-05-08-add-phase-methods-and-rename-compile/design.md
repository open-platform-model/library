## Context

Slice 01 (`add-kernel-struct`) gave the kernel a single anchor type. Slice 06 puts the public phase verbs on it. Three frontends need different depths of the pipeline:

- **CLI** maps to subcommands: `vet` → `Validate`, `match` → `Match`, `plan` → `Plan`, `apply` → `Compile`.
- **Operator** uses `Validate` for admission webhook, `Match` for status reporting before render, `Compile` for reconcile, `Plan` for diff/preview.
- **Crossplane fn** calls `Compile` only.

The rename from "Render" to "Compile" reflects what the kernel actually does — lower a declarative OPM model into platform-neutral resource values, the way a compiler lowers source to target IR. "Render" carries graphics-flavored connotations (HTML, Helm) that misframe the operation. The umbrella's decision (D7) selected `Compile` over alternatives (`Synthesize`, `Materialize`, `Realize`).

This slice depends on slice 01 (Kernel struct exists). It does NOT yet require slice 08 (Platform construct) — phase methods initially work against the current `*provider.Provider` shape; slice 08 / 09 substitute `*Platform` later.

## Goals / Non-Goals

**Goals:**
- Four first-class phase methods on `*Kernel` matching umbrella D7.
- End-to-end rename of the `Render`/`Process` flow to `Compile`.
- Deprecation aliases for old names; no breaking change for downstream.
- Small input structs per phase (each phase takes only what it needs).
- Each method's output type is appropriate to its phase: `Validate` returns `error`; `Match` returns `*MatchPlan`; `Plan` returns `*PlanResult`; `Compile` returns `*CompileResult`.

**Non-Goals:**
- Replacing `*provider.Provider` with `*Platform`. That is slice 08 + 09. Phase methods accept `*provider.Provider` in this slice; the substitution happens transparently when slice 08 ships.
- Implementing the rewritten match algorithm. That is slice 09. Match in this slice still uses the existing `opm/render/match.go`.
- Deleting deprecated names. Defer to a future MAJOR.

## Decisions

**Method signatures use small input structs, not positional arguments.** Reason: input shape will change (slice 08 swaps `Provider` for `Platform`). A struct lets us add fields without breaking call sites; positional args would force every caller to update on each input change.

**`Plan` and `Compile` are distinct method names with distinct return types.** Reason: callers asking for a plan want a different result shape than callers asking for a compile (Plan has a "would-be-rendered" preview but no `Rendered` slice; Compile has the full `Rendered` slice). One method with a `Mode` enum was considered and rejected — it complicates return-type discrimination.

**`Plan` semantics are: run the full Compile pipeline and discard the rendered slice — return component summaries, unmatched, ambiguous, and warnings only.** Reason: pinning Plan and Compile to one execution path means any error a real Compile would surface (transformer evaluation, finalization) also surfaces at Plan time, so a green Plan is a strong signal that Compile will succeed. The umbrella's OQ2 asked whether Plan should be cheaper ("stop after match"); the resolution chose the single-pipeline form because no frontend currently needs the speed difference. If one later does, Plan can grow a stop-after-match flag without changing its return type.

**`CompileResult` carries `Rendered`, `Components`, `Warnings`, `Unmatched`, `Ambiguous`.** Reason: matches the umbrella's `RenderResult` sketch. `Resolution` (for Claims) is added later when Claim support lands; not in this slice.

**`*ModuleResult` becomes a type alias to `*CompileResult`.** Reason: zero-cost migration for downstream callers that reference the old name. Avoids a MAJOR.

**`render.NewModule` and `render.Module` keep their names.** Reason: those describe a per-render execution context (the runtime helper that holds provider + transformers + runtime name), not the pipeline verb. Renaming them confuses with the new `Compile`/`Module` distinction. Revisit after slice 09.

**`DetectAPIVersion` and `Finalize` are kernel methods, not free functions.** Reason: both need access to the kernel's `cue.Context`. Keeping them as methods avoids the cue.Context-leaks-to-callers anti-pattern.

**Method receiver is `*Kernel`, not `Kernel`.** Reason: methods may eventually mutate internal caches (e.g. binding lookup memoization). Pointer receiver future-proofs.

## Risks / Trade-offs

**Risk — surface area doubles temporarily.** Both `Process*` and `Compile*` exist; both `*ModuleResult` and `*CompileResult` exist (as alias). Mitigation: deprecation comments steer to the new names; godoc shows "preferred" vs "alias" clearly.

**Risk — `Plan` semantics drift.** As more sophisticated dry-run features are requested ("show me what would change vs. last apply"), `Plan` may outgrow this slice's narrow contract. Mitigation: keep slice 06's `Plan` minimal; capture future Plan work in a follow-up enhancement, not in this slice.

**Risk — input structs accrete fields.** Each subsequent slice adds a field. Mitigation: input structs ship with explicit godoc on each field; structs grow additively; old call sites that omitted new fields just take the zero value.

**Trade-off — `Plan` is no cheaper than `Compile`.** Plan delegates to Compile and only drops the `Rendered` slice; the full match → finalize → execute path runs both times. The alternative (a stop-after-match shortcut) was rejected because no current frontend needs the speedup and a single-pipeline guarantee is more valuable: a green Plan implies a green Compile.

**Trade-off — rename is broad.** `Render` appears in many filenames, doc comments, test names, and CHANGELOG entries. Mechanical churn but small per-site change.
