## Why

`compile.FinalizeValue` and its wrapper `Kernel.Finalize` have been off the render path since `library-component-fill` (0019 step 2), and `compile.Module.Execute` has carried an ignored, deprecated `dataComponents` parameter since then so that slice could stay MINOR. With `library-instance-fill` (step 3) green, 0019 Phase A step 4 removes the strip from the public surface: the kernel exposes exactly the pipeline it runs, and the security-audit skill stops describing a guard that no longer exists.

## What Changes

- **BREAKING** `compile.FinalizeValue` is deleted (`opm/compile/finalize.go` goes). It has no caller in `library` outside tests, none in `cli`, none in `opm-operator` (grep, 2026-08-25).
- **BREAKING** `Kernel.Finalize` is deleted from `opm/kernel/phases.go`, together with `TestKernel_Finalize`.
- **BREAKING** `compile.Module.Execute` drops the deprecated `dataComponents` parameter: `Execute(ctx, inst, components, plan)`. `Kernel.Compile` and the `opm/compile` tests are the only callers.
- The security-audit skill's "finalization before fill" guidance is rewritten: constraint enforcement is CUE unification itself; the audit points are the fixed fill paths and the closedness guards, not a strip step (0019 05-risks, mitigation for the invalidated security claim).
- Pipeline wording "finalize → match → execute → emit" in `CLAUDE.md`, `README.md`, `CONSTITUTION.md` and the `kernel-runtime` spec (the "Validate + Match + Execute + Finalize" scenarios) becomes "match → execute → emit". `docs/design/transformer-output-hidden-field-scope-bug.md` is a dated investigation record and keeps its historical references.
- `MIGRATIONS.md` gains the `## Unreleased — Breaking` entry with the recipe for each removed identifier.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `kernel-runtime`: requirement "Utility Methods on Kernel" loses `Finalize` (REMOVED, its `DetectAPIVersion` half being stale since the `opm/apiversion` package was dropped, and replaced by an ADDED "No Utility Methods on Kernel" requirement, since a MODIFIED block cannot drop a scenario); requirement "Phase-Explicit Methods on Kernel" loses the "+ Finalize" wording from its Plan and Compile scenarios (behaviour already matches; the text is stale since step 2).

## Impact

- **SemVer: MAJOR** (three `opm/` removals). The library is on the `v1.0.0-alpha` line, so the break is absorbed as an alpha increment with no `/v2` module path (0019 06-operational: land Phase A breaks before the line graduates out of alpha). Downstream migration cost: `cli` and `opm-operator` re-pin; neither calls any removed identifier, so no code changes. The Conventional Commit for this slice carries `!` and a `Migration: library-finalize-removal` trailer per `MIGRATIONS.md`'s maintenance rules.
- **Packages:** `opm/compile` (`finalize.go` deleted, `module.go` signature, `compile_test.go` helpers), `opm/kernel` (`phases.go`, `compile.go` call site, `phase_test.go`), `.claude/skills/security-audit/SKILL.md`, `CLAUDE.md`, `README.md`, `CONSTITUTION.md`, `MIGRATIONS.md`.
- **Rendered output:** unchanged. Nothing on the render path is touched; the parity harness is the proof.
- **Enhancement 0019:** implements D1 (the strip leaves the surface, a removal) and completes the Phase A library sequence; D14's ordering note already shipped with step 2. Declared in `enhancement.yaml` as `[D1]`.
- **Complexity (Principle VII):** net removal: one file, one method, one parameter, one test.
