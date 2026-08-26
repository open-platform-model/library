## 1. Remove the surface (opm/compile, opm/kernel)

- [x] 1.1 Delete `opm/compile/finalize.go`.
- [x] 1.2 `opm/compile/module.go`: `Execute(ctx, inst, components, plan)`; rename the parameter, rewrite its doc, drop the `_ = dataComponents` line and the deprecation comment.
- [x] 1.3 `opm/kernel/phases.go`: delete `Kernel.Finalize`. `opm/kernel/compile.go`: pass `schemaComponents` once, drop the "deprecated, ignored" comment.

## 2. Tests (opm/compile, opm/kernel)

- [x] 2.1 `opm/compile/compile_test.go`: drop the `FinalizeValue` calls in `runExecute`, the inline site near line 916 and `TestExecute_NilGuards`; delete `TestExecute_DataComponentsIgnored`.
- [x] 2.2 `opm/kernel/phase_test.go`: delete `TestKernel_Finalize`; add a reflect-based test asserting `*Kernel` has no `Finalize` method (design D5).

## 3. Security-audit skill (.claude/skills/security-audit)

- [x] 3.1 Rewrite every finalization reference in `SKILL.md` per design D4 (lines 10, 30, 49, 53, 103, 107, 115, 122, 137, 179, 256, 259 at time of writing); keep line 142's statement as the single source and remove the contradiction.

## 4. Docs

- [x] 4.1 `MIGRATIONS.md`: `### Removed — library-finalize-removal` under `## Unreleased — Breaking` with a recipe per identifier (`FinalizeValue`, `Kernel.Finalize`, `Execute`'s fourth argument).
- [x] 4.2 `CLAUDE.md`, `README.md`, `CONSTITUTION.md`: pipeline wording "finalize → match → execute → emit" becomes "match → execute → emit".

## 5. Validation

- [x] 5.1 `task fmt`, `task vet`, `task lint`, `task test` (full parity harness included; rendered output must be unchanged).
- [x] 5.2 `go doc -all` diff against `main` for `opm/compile` and `opm/kernel`: exactly the three removals, nothing else. `grep -rn Finaliz opm/` hits only the D5 negative test (`TestKernel_NoFinalizeMethod`).
- [x] 5.3 Confirm `cli` and `opm-operator` still contain no reference to the removed identifiers (`grep` both repos).
