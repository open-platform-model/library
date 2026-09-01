## Why

The 2026-09-01 kernel review swept every exported symbol under `opm/` against library production code, `cli` and `opm-operator` a second time and found a residue the two 2026-08-31 sweeps kept: diagnostics that are produced and never consumed anywhere (`MatchPlan.Missing`/`MissingFQN`, `CompileResult.Unmatched`), accessors with no caller (`TransformError.Component()`, `Instance.ModuleFQN()`), and an adapter contract with zero implementations (`core.Resource`/`core.Identity`; both frontends define their own Resource types and import `opm/core` only for `Compiled`). Each is SemVer surface maintained and spec-pinned for nobody.

## What Changes

All removals are **BREAKING** by SemVer rule; none has a caller in `cli` or `opm-operator` (verified 2026-09-01), so no consumer migration is needed.

- **BREAKING** `opm/compile` + `opm/errors`: remove `MatchPlan.Missing` and the `MissingFQN` type. An empty demand bucket is already carried by `UnresolvedDemand` (which records the same contract-key-ordered alternatives plus disqualification causes); the parallel `Missing` record was retained "for compatibility" and nothing reads it. The `alternativesFor` computation stays (it feeds `UnresolvedDemand.Alternatives`).
- **BREAKING** `opm/compile`: remove `CompileResult.Unmatched`. It has been hardcoded to an empty slice since the D28 gate made unmatched components fail `Compile`; `MatchPlan.Unmatched` (what the CLI prints) stays.
- **BREAKING** `opm/errors`: remove the `TransformError.Component()` method (the `ComponentName` field stays).
- **BREAKING** `opm/module` + `opm/schema`: remove `Instance.ModuleFQN()` and the `ModuleFQN` member of `schema.InstanceView`. `ModuleVersion()` stays (the transformer context reads it); `BuildTransformerContext` never read `ModuleFQN`.
- **BREAKING** `opm/core`: remove the `Resource` interface and the `Identity` struct (with `Identity.String()`). `Compiled` remains the kernel's terminal output; the package doc moves onto `compiled.go` and drops the "adapters implement Resource filling Identity" story. This answers enhancement 0012's OQ3 (delete or retain the neutral contract) in the delete direction, the direction 0012's own alternatives analysis leans; 0012 is draft, so its OQ3 is updated in place when this lands.
- Docs: `README.md` and `CLAUDE.md` lines describing `core.Resource`/`core.Identity` adapters; the `core.Resource` reference in `opm/compile/module.go`'s comment; recorded per the migrations policy (pre-GA: changelog and archive, no fragment).

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `platform-matching`: the empty-bucket lookup outcome is stated solely in unresolved-demand terms; the "Structured Missing-FQN Diagnostic" requirement is removed.
- `kernel-runtime`: the Compile phase scenario no longer lists unmatched components as a `CompileResult` field (the plan inside it still carries them).
- `artifact-types`: the instance module-identity accessor set drops `ModuleFQN()`; `ModuleVersion()` remains.
- `schema-dispatch`: `schema.InstanceView` drops the `ModuleFQN` member.

## Impact

- Packages: `opm/compile` (`match.go`, `module.go`), `opm/errors` (`match.go`, `domain.go`), `opm/module` (`instance.go`), `opm/schema` (`metadata.go`), `opm/core` (`resource.go` deleted, package doc to `compiled.go`).
- Consumers: zero references to any removed identifier in `cli` or `opm-operator`; both keep compiling unchanged. MAJOR under SemVer (Principle VI).
- Tests: tests exercising the removed symbols go with them; two `assert.Empty(out.Unmatched)` lines and the `ModuleFQN` accessor assertion are dropped; no behavior covered by a remaining test changes.
- Enhancements: 0012 OQ3 gets its in-place resolution note when this lands (tracked as a task; 0012 is draft, decisions and questions revise in place).
