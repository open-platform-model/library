## Why

Configuration validation today is split across three packages (`opm/kernel/validate.go`, `opm/helper/values/`, `opm/errors/`) and exposes seven custom error types (`ConfigError`, `MultiSourceError`, `LayerError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`) that re-implement information CUE's `cuelang.org/go/cue/errors` already carries. The current shape is the accreted result of a Tier-1/Tier-2 split where each tier produces its own error language, leaving frontends (CLI, opm-operator, future XR composition fns) to translate between them. Worse, `runValidate`'s `(schema, values, ctxLabel, name)` signature mixes the validation primitive with display framing, and the library invents grouping logic (`groupCUEErrors`) that duplicates `cueerrors.Print`'s behavior. The consequence is a large surface that drifts from CUE's semantics and gives downstream consumers a Go-typed dialect of the same diagnostics CUE produces natively.

This redesign collapses validation onto a single CUE-native surface: three primitive functions on `*Kernel` plus typed convenience wrappers on `*Module`/`*Release`, returning `cuelang.org/go/cue/errors.Error` directly so frontends use `cueerrors.Print`/`Errors`/`Positions` instead of bespoke types. Per-source attribution is preserved by setting `cue.Filename(Origin)` at load time (not by inventing a per-layer error wrapper). The change is foundational: every other validation-adjacent feature (linting, IDE feedback, admission webhook plumbing) builds on it, and shipping the redesign first prevents new code from accreting onto the doomed surface.

## What Changes

- **BREAKING** Replace `Kernel.ValidateConfig(schema, values, ctxLabel, name)` and `Kernel.ValidateConfigPartial(schema, values, ctxLabel, name)` with primitives `Kernel.ValidateConfig(schema, values) (cue.Value, error)` and `Kernel.ValidateConfigPartial(schema, values) (cue.Value, error)`. `name`/`ctxLabel` move to caller-side `fmt.Errorf` wraps.
- **BREAKING** Add `Kernel.ValidateConfigDetailed(schema, sources, opts...) (cue.Value, error)` — unifies an ordered `[]Source`, then runs `ValidateConfig` (or `ValidateConfigPartial` under `Partial()` option). Returns CUE-native errors; per-source attribution flows through `cue.Filename(Source.Origin)` set at load.
- **BREAKING** Add `Source struct { Value cue.Value; Name, Origin string }` and option `Partial() Option` in `opm/kernel/`. Replaces the deleted `opm/helper/values/` `Layer` / `Stack` types.
- **BREAKING** Add typed convenience methods: `Module.ConfigSchema()`, `Module.ValidateValues`, `Module.ValidateValuesPartial`, `Module.ValidateValuesDetailed`, plus the four Release equivalents (eight new methods total). Each is a 2-line schema-lookup wrapper around the kernel primitives — no duplicated logic.
- **BREAKING** Add Source loader helpers `Kernel.LoadSourceFromFile`, `Kernel.LoadSourceFromBytes`, `Kernel.LoadSourceFromString` that bake `cue.Filename(Origin)` at compile time so per-source positions survive into errors.
- **BREAKING** Delete `opm/helper/values/` package entirely (`values.go`, `errors.go`, `doc.go`, `KernelOwner` interface, `Layer`, `Stack`, `MultiSourceError`, `LayerError`, `ValidateAndUnify`).
- **BREAKING** Delete custom validation error types from `opm/errors/`: `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, plus `groupCUEErrors`, `GroupedErrorsFromError`, `normalizeCUEPath`, `NewValidationError`. `TransformError` stays (execute-phase, unrelated).
- **BREAKING** Remove `Kernel.ValidateAndUnify` wrapper in `opm/kernel/wrappers.go` (replaced by `Kernel.ValidateConfigDetailed`).
- The library ships **no display helper**. Presentation belongs to the frontend per Constitution principles I (Kernel Neutrality) and IV (Composability via Stable Contracts: "Output formatting and presentation MUST stay outside the library."). Frontends use `cueerrors.Print` for raw CLI-style output or walk `cueerrors.Errors`/`Positions` themselves to render however they need (CLI prose, K8s status conditions, XR composition status, IDE squiggles).
- Internal `walkDisallowed` + `fieldNotAllowedError` stay private inside `opm/kernel/validate.go`; they implement `cueerrors.Error` and exist only to work around CUE's loss of source positions for closed-schema field rejections.
- `Kernel.Validate(ctx, ValidateInput)` (phase method) keeps its public signature; internally calls `ValidateConfig` and wraps with `fmt.Errorf("module %q: %w", name, err)`.
- `Kernel.ProcessModuleRelease` migrates step 1 to `ValidateConfig`; step 3's `spec.Validate(cue.Concrete(true))` and `Compose`'s `composed.Validate(cue.Concrete(false))` stay (both are CUE stdlib, not custom validators).
- `cuelang.org/go/cue/errors` becomes the canonical error format across the library's validation surface. Frontends use `cueerrors.Errors`/`Positions`/`Print` for traversal and rendering.

## Capabilities

### New Capabilities

- `config-validation`: The new CUE-native validation surface — three primitives on `*Kernel`, six typed shortcuts on `*Kernel` covering Module/Release, three Source loaders, the `Source`/`ValidateOption`/`Partial()` types. The library ships no display helper; presentation belongs to the frontend per Constitution principles I and IV. Owns every requirement that previously lived under `values-validation`.

### Modified Capabilities

- `values-validation`: All requirements removed. The capability is superseded by `config-validation`. The spec file remains as a tombstone pointing readers at the new capability.
- `kernel-runtime`: Update "Tier-2 Validation Always Runs", "Canonical Implementations Live on Kernel", and "Utility Methods on Kernel" requirements to reflect new signatures; add validation phase delegation through `ValidateConfig`.
- `helper-packages`: Remove any requirements describing `opm/helper/values/` (the package is deleted).

## Impact

**Affected library packages**

- `opm/kernel/`: `validate.go` rewritten; new `source.go`, `source_loader.go`, `print.go`; `wrappers.go` loses `ValidateAndUnify`; `phases.go` and `process.go` updated to call new primitives.
- `opm/module/`: `module.go` and `release.go` gain `ConfigSchema` accessors and four validation methods each.
- `opm/helper/values/`: deleted entirely.
- `opm/errors/`: shrinks to roughly `TransformError` + sentinels. `domain.go` and `config_error.go` lose validation-related types.

**Affected downstream consumers** (separate repos; migration recipes captured in design.md, code changes out of scope for this proposal)

- `cli/`: every callsite that imports `opm/helper/values` or constructs `*ConfigError`/`*MultiSourceError`. Switches to `[]kernel.Source` and `cueerrors.Errors`/`Print`.
- `opm-operator/`: reconcile loop and admission paths that surface validation diagnostics to status conditions. Updates to read CUE-native errors.

**SemVer**

MAJOR. Public types and function signatures in `opm/kernel/`, `opm/helper/values/` (full deletion), and `opm/errors/` change.

**Dependencies**

No new external dependencies. Leans more heavily on `cuelang.org/go/cue/errors` and `cuelang.org/go/cue/token`, both already imported transitively.

**Risk**

The change is large per Principle VIII (Small Batch Sizes) but cannot be split without leaving the public surface in a half-redesigned state across releases. Mitigation: the `tasks.md` execution sequence keeps the codebase compiling at every commit (additive new surface first, callsite migration second, demolition last), and tests are rewritten alongside the surface they cover so each commit lands with green CI.
