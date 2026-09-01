# Tasks: library-plan-diagnostics-prune

## 1. opm/compile + opm/errors (miss diagnostics)

- [x] 1.1 `opm/compile/match.go`: remove the `Missing` field from `MatchPlan` and the `plan.Missing = append(...)` recording in `walk`; keep `alternativesFor` (the empty-bucket `UnresolvedDemand` keeps carrying the alternatives). Update the package doc's algorithm prose.
- [x] 1.2 `opm/errors/match.go`: remove the `MissingFQN` type; adjust `opm/errors/match_test.go` and any `plan.Missing` assertions in `opm/compile` and `opm/kernel` tests.
- [x] 1.3 `opm/compile/module.go`: remove the `Unmatched` field from `CompileResult` and its `[]string{}` initialization; drop the two `assert.Empty(t, out.Unmatched)` lines in `opm/kernel` integration/flow tests.

## 2. opm/errors (TransformError)

- [x] 2.1 `opm/errors/domain.go`: remove the `Component()` method (keep the `ComponentName` field and `Error`/`Unwrap`).

## 3. opm/module + opm/schema (ModuleFQN)

- [x] 3.1 `opm/module/instance.go`: remove `ModuleFQN()`; keep `ModuleVersion()` and `lookupModuleMetadataString`. Drop the accessor assertion in `opm/module/module_test.go`.
- [x] 3.2 `opm/schema/metadata.go`: remove the `ModuleFQN` member from `InstanceView`; update any test fakes implementing the interface (an extra method on a fake is harmless, but remove it where present).

## 4. opm/core (neutral adapter contract)

- [x] 4.1 Delete `opm/core/resource.go` (the `Resource` interface, `Identity`, `Identity.String()`) and `opm/core/identity_test.go`; move a trimmed package doc onto `compiled.go` stating `Compiled` is the terminal output and platform identity is the frontend's concern.
- [x] 4.2 `opm/compile/module.go`: reword the `core.Resource` reference in the `CompileResult.Compiled` comment.

## 5. Docs and cross-references

- [x] 5.1 `README.md` + `CLAUDE.md`: remove the "adapters wrap each Compiled with a platform-specific core.Resource filling core.Identity" story; state that frontends own platform identity for compiled output.
- [x] 5.2 `enhancements/0012/07-questions.md`: record OQ3's in-place resolution (neutral contract deleted by this change; delete-vs-retain no longer blocks promotion) with a matching `config.yaml` history event. 0012 is draft; revise in place per the enhancements rules.

## 6. Validation

- [x] 6.1 Verify zero references to removed identifiers remain in `cli` and `opm-operator` (grep `MissingFQN|\.Missing\b|CompileResult.Unmatched|ModuleFQN|core.Identity|core.Resource\b|\.Component()`), and that both consumers build.
- [x] 6.2 `task check` in `library` (fmt, vet, lint, test).
