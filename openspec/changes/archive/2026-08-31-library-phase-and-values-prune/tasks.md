## 1. Verify the premise (all packages)

- [x] 1.1 Re-run the reference sweep against current `main` of `cli` and `opm-operator`: no call to `Plan`, `Validate`, `ValidateModuleValues*`, `ValidateInstanceValues*`, and no `CompileInput{... Values: ...}` literal.

## 2. opm/kernel: phase surface

- [x] 2.1 `phases.go`: delete `Plan`, `Validate`, `moduleFromInstance`, `nonNilStrings`, `nonNilComponentSummaries`; `Compile` calls `compileModuleInstance` directly after its nil and `RuntimeName` guards.
- [x] 2.2 `inputs.go`: delete `ValidateInput` and `PlanInput`; remove the `Values` field from `CompileInput`; rewrite the `MatchInput` / `CompileInput` docs to say the instance is rendered as processed.
- [x] 2.3 `results.go`: delete `PlanResult`; keep the `MatchPlan` and `CompileResult` aliases.
- [x] 2.4 Delete `validate_typed.go`.
- [x] 2.5 `doc.go`: rewrite the "Phase methods" and "Configuration validation" sections to the surviving surface (two verbs; three primitives plus `Source`; `ProcessModuleInstance` as the validated entry).

## 3. Tests (opm/kernel)

- [x] 3.1 `phase_test.go`: delete `TestKernel_Validate_*`, `TestKernel_Plan_*`; keep `TestKernel_Match_OK`, `TestKernel_Compile_OK`, `TestKernel_Compile_FromInstanceOnly`; add a reflect-based negative test that `*Kernel` has no `Plan` or `Validate` method and `CompileInput` has no `Values` field.
- [x] 3.2 Delete `validate_typed_test.go`; fold any assertion not covered by `validate_test.go` into it against the primitives.
- [x] 3.3 `integration_validate_test.go` and `flow_integration_test.go`: replace `Plan` phases with `Match` where the assertion was about pairing, drop them where it duplicated `Compile`.

## 4. Docs

- [x] 4.1 Delete `docs/design/kernel-validate-flow.md`; remove its entries from `CLAUDE.md` Entrypoint and `README.md` Further reading.
- [x] 4.2 `CLAUDE.md` "Kernel API surface" and "Compile pipeline": two verbs, `ProcessModuleInstance` validates; drop the typed-shortcuts sentence. `README.md` and `docs/getting-started.md` phase tables likewise; getting-started's layered-values example calls `k.ValidateConfigDetailed(mod.ConfigSchema(), sources)`.
- [x] 4.3 `MIGRATIONS.md`: `### Removed — library-phase-and-values-prune` under `## Unreleased — Breaking` with one replacement per removed identifier.

## 5. Validation

- [x] 5.1 `task fmt`, `task vet`, `task lint`, `task test` (parity harness and the `-race` leg included; rendered output unchanged).
- [x] 5.2 `go doc -all ./opm/kernel` diff against `main`: only the listed removals.
- [x] 5.3 Build `cli` and `opm-operator` against this branch via a temporary `replace` (not committed): both compile unchanged.
