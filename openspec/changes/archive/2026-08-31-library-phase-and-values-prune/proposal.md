## Why

The kernel's four phase verbs were designed for frontends that never materialised (`vet` → `Validate`, `plan` → `Plan`, an operator admission webhook), and two of them, `Plan` and `Validate`, have had no caller in `cli` or `opm-operator` since they landed. `Plan` is `Compile` with the rendered slice discarded, so a caller pays a full render to be told it would succeed; `Compile` validates `CompileInput.Values` and then never applies it, so the field reads like a knob and behaves like an assertion. Six typed validation wrappers spell `k.ValidateConfig(m.ConfigSchema(), v)` six ways for callers that do not exist. The library kernel review (2026-08-30) put all of it on the removal list; this change removes the phase and input surface, and keeps the validation primitives the CLI is about to adopt.

## What Changes

- **BREAKING** `opm/kernel`: remove `Kernel.Plan`, `PlanInput`, `PlanResult`. `Compile` remains the single terminal verb; `Match` remains the single phase-only diagnostic verb (its `MatchPlan` is what the CLI prints, and 0019 D10 retires the Go matcher on its own schedule).
- **BREAKING** `opm/kernel`: remove `Kernel.Validate` and `ValidateInput`. `ProcessModuleInstance` is where user values are validated and filled; `Compile` no longer re-validates a value it does not apply.
- **BREAKING** `opm/kernel`: remove `CompileInput.Values`. `CompileInput` keeps `ModuleInstance`, `Platform`, `RuntimeName` and stays a struct (0019 D7 adds a skew-policy field to it later).
- **BREAKING** `opm/kernel`: remove the six typed wrappers `ValidateModuleValues`, `ValidateModuleValuesPartial`, `ValidateModuleValuesDetailed`, `ValidateInstanceValues`, `ValidateInstanceValuesPartial`, `ValidateInstanceValuesDetailed`. `Module.ConfigSchema()` and `Instance.ConfigSchema()` stay; a caller composes them with the primitive it wants.
- Kept, deliberately: `ValidateConfig`, `ValidateConfigPartial`, `ValidateConfigDetailed`, `Source`, `ValidateOption`, `Partial()`, `LoadSourceFromFile` / `LoadSourceFromBytes` / `LoadSourceFromString`. The `cli` sibling change `cli-layered-values` replaces the CLI's own values-file unification with this surface so `-f a.cue -f b.cue` conflicts report both file positions.
- Docs: delete `docs/design/kernel-validate-flow.md` (it documents `Kernel.Validate` and a binding layer that no longer exists); rewrite the phase table in `CLAUDE.md`, `README.md`, `docs/getting-started.md` and `opm/kernel/doc.go` to the two verbs; `MIGRATIONS.md` entry under Unreleased, Breaking.

Out of scope: `context.Context` threading and the logger, tracer and clock slots (enhancement 0009 D9); the dead-symbol removals in `library-dead-symbol-sweep`; any change to matching semantics or to `Match`'s output.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `kernel-runtime`: the phase-method requirement names two verbs (`Match`, `Compile`); the phase input structs drop `Values`, `ValidateInput` and `PlanInput`; Tier-2 validation is stated as `ProcessModuleInstance`'s obligation, not `Compile`'s.
- `config-validation`: the phase-method wrapping requirement is replaced by one on `ProcessModuleInstance`; the typed convenience methods requirement keeps the `ConfigSchema()` accessors and drops the six wrappers.
- `platform-artifact`: the phase-inputs requirement names `MatchInput` and `CompileInput` only.

## Impact

- Packages: `opm/kernel` (`phases.go`, `inputs.go`, `results.go`, `validate_typed.go` deleted), `opm/compile` (unchanged; `CompileResult` still carries `MatchPlan`, `Components`, `Unmatched`, `Warnings`).
- Consumers: neither `cli` nor `opm-operator` calls `Plan`, `Validate`, the typed wrappers, or sets `CompileInput.Values` (verified 2026-08-30); both keep compiling. MAJOR under SemVer (Principle VI); recorded in `MIGRATIONS.md`.
- Sibling: `cli-layered-values` (in `cli/openspec/`) consumes the kept `Source` / `ValidateConfigDetailed` surface; it is written beside this change so nothing this change deletes is something that proposal names.
