## Context

See `proposal.md` for motivation. What shapes the approach:

- `compile.FinalizeValue` (`opm/compile/finalize.go`) is `v.Syntax(cue.Final())` rebuilt with `BuildExpr`. Its only remaining callers are `Kernel.Finalize` (`opm/kernel/phases.go:147`) and three `opm/compile` test sites that compute a `dataComponents` value to pass to `Execute`, where it is ignored (`module.go:153`).
- `compile.Module.Execute(ctx, inst, schemaComponents, dataComponents, plan)` is public. Callers: `Kernel.compileModuleInstance` (`opm/kernel/compile.go:50`) and the `opm/compile` tests. `cli` and `opm-operator` reach rendering only through `Kernel.Compile` / `Kernel.Plan`.
- `.claude/skills/security-audit/SKILL.md` names finalization as a guard in seven places (lines 10, 49, 103, 107, 115, 122, 137, 179, 256, 259) while line 142 already states the post-step-2 position. The two positions contradict each other today.
- Release mechanics: `MIGRATIONS.md` buckets by impact; a breaking change is a `type!:` commit with a `Migration: <slug>` trailer, and the release gate refuses a breaking commit without a matching `## Unreleased — Breaking` entry. The line is `v1.0.0-alpha.14`, so release-please absorbs the break as an alpha increment.

## Goals / Non-Goals

**Goals:**

- No exported identifier named `Finalize*` remains under `opm/`; `Execute` takes one components value.
- The security-audit skill describes the actual boundary: unification enforces constraints, fill paths are fixed, closedness is guarded by tests.
- The `kernel-runtime` spec and the three pipeline diagrams stop mentioning a finalize step.

**Non-Goals:**

- Any change to what `executePair` fills or how (steps 2 and 3 own that; rendered output is untouched).
- Renaming `schemaComponents` in `Kernel.compileModuleInstance` beyond what the signature change forces.
- Rewriting `docs/design/transformer-output-hidden-field-scope-bug.md`: a dated investigation that measured the finalized value; it stays as a record.
- Touching `MIGRATIONS.md`'s already-graduated entries that mention `FinalizeValue` historically.

## Decisions

### D1: Delete, do not deprecate further

`finalize.go` is removed, `Kernel.Finalize` is removed, the `dataComponents` parameter is removed. No shim, no alias.

**Why:** the deprecation window already happened (step 2 marked the parameter deprecated and moved the strip off the path; step 3 landed on top). There is no external caller to protect, and the line is pre-release. Keeping a shim would keep the strip's mechanism in the codebase, which is the thing 0019 D1 removes.

**Alternative:** keep `FinalizeValue` as a public utility with no kernel role. Rejected: the security-audit skill and the design docs would keep pointing at it as if it were a boundary, and D1's direction is removal.

### D2: `Execute` becomes `Execute(ctx, inst, components, plan)`

Parameter renamed from `schemaComponents` to `components`; doc comment says it is the instance's evaluated components value, the one `Match` reads. `Kernel.compileModuleInstance` passes `schemaComponents` once and drops its "deprecated, ignored" comment. The `opm/compile` test helpers (`runExecute`, the inline site near `compile_test.go:916`, `TestExecute_NilGuards`) drop the `FinalizeValue` call. `TestExecute_DataComponentsIgnored` is deleted: the behaviour it pinned is now structural.

### D3: Spec delta is REMOVED plus ADDED for the utility requirement, MODIFIED for the phase methods

`openspec validate` refuses a MODIFIED block that drops a scenario, so "Utility Methods on Kernel" is REMOVED (reason and migration recorded) and a "No Utility Methods on Kernel" requirement is ADDED with a negative scenario pinning the absence of any finalization method (`DetectAPIVersion`, the block's other method, was already gone with the `opm/apiversion` package; the old text was stale). "Phase-Explicit Methods on Kernel" is MODIFIED in full with the stale "+ Finalize" wording removed.

### D4: The security-audit skill states the post-0019 boundary once, consistently

Rewrite the finalization bullets to: the value a transformer receives is filled at fixed schema paths (`#moduleInstance`, `#component`, `#context.*`); user data cannot redefine constraints because the filled value keeps its constraints and CUE unification refuses a conflicting field (the pure-CUE control's `field not allowed`); audit the fill sites, the closedness guards (`opm/materialize/composed_open_test.go`, `TestMatch_ClosedDefinitionRetained`, the cueregression canary pair) and the parity harness, which is the tripwire against a Go-side transformation being reintroduced. The dimension list, threat table, severity examples and positive observations that name finalization are reworded to the same effect; the "Key files" line drops `finalize.go`.

### D5: The absence is pinned by a test

A small test in `opm/kernel` asserts, via `reflect`, that `*Kernel` has no method named `Finalize`, and a compile-time reference check in `opm/compile` is unnecessary (the package would not build). The reflect check is cheap and turns a future re-addition into a deliberate act rather than drift; it backs the spec's "No finalization method on the Kernel" scenario.

## Risks / Trade-offs

- [A downstream caller of `Execute` exists outside the grep'd repos] → the only known embedders are `cli` and `opm-operator`, both grep-clean; the `MIGRATIONS.md` recipe covers anyone else (drop the fourth argument).
- [Release gate refuses the breaking commit] → the `## Unreleased — Breaking` entry lands in the same change, keyed by the change slug the commit's `Migration:` trailer names.
- [The alpha line graduates before this lands, turning the break into a `/v2` module path] → not the case at `v1.0.0-alpha.14`; the ordering constraint is recorded in 0019 06-operational and this is the slice that discharges it for the library.
- [Security-audit rewrite weakens the audit] → the rewrite replaces a claim about a removed mechanism with the stronger, measured one (unification closedness); the audit gains the parity harness as an explicit check.

## Migration Plan

Code and docs in one change. Delivery: a `refactor!(compile): ...` (or `feat!`) commit carrying `Migration: library-finalize-removal`; release-please cuts the next alpha. `cli` and `opm-operator` re-pin on their normal dependency-bump cadence; no code change. Rollback is a revert of the slice; nothing on the render path moves, so a revert changes no output.

## Open Questions

None.
