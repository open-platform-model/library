## REMOVED Requirements

### Requirement: Layer and Stack Types

**Reason**: Replaced by the `Source` struct and `[]Source` ordering convention defined in the new `config-validation` capability. `Layer{Name, Source, Value}` is renamed and relocated to `kernel.Source{Value, Name, Origin}`; `Stack` is replaced by `[]Source`.

**Migration**: Replace `helpervalues.Layer` with `kernel.Source` and `helpervalues.Stack` with `[]kernel.Source`. Field rename: `Source` → `Origin`. Per-source attribution now flows through `cue.Filename(Origin)` set at load time (use `Kernel.LoadSourceFromFile`/`Bytes`/`String`) rather than through a custom wrapper struct.

### Requirement: ValidateAndUnify Tier-1 Validation

**Reason**: The function is replaced by `Kernel.ValidateConfigDetailed(schema, sources, opts...)` defined in the new `config-validation` capability. The redesign collapses the per-layer-partial-then-unify flow into a unify-then-validate flow that produces equivalent diagnostics with simpler internals (CUE preserves source positions through `Unify`, so per-layer pre-validation is unnecessary for attribution purposes).

**Migration**: Replace `helpervalues.ValidateAndUnify(k, schema, layers)` with `k.ValidateConfigDetailed(schema, sources)`. For partial-mode behavior, pass the `kernel.Partial()` option. Errors are now CUE-native (`cuelang.org/go/cue/errors.Error`); use `cueerrors.Errors(err)` to walk them and `pos.Filename()` to recover the originating Source's `Origin`.

### Requirement: MultiSourceError Aggregation

**Reason**: Replaced by direct use of `cuelang.org/go/cue/errors`. CUE's error tree already carries per-position source attribution through `token.Pos.Filename()`; bucketing by source is a frontend concern and not imposed by the library.

**Migration**: Replace `*MultiSourceError.Errors()` with iteration via `cueerrors.Errors(err)` plus per-error position walking via `cueerrors.Positions(ce)`. To bucket errors by source, build a `map[string][]cueerrors.Error` keyed on `pos.Filename()` at the call site.

### Requirement: Kernel Convenience Method

**Reason**: `Kernel.ValidateAndUnify` is replaced by `Kernel.ValidateConfigDetailed` defined in the new `config-validation` capability.

**Migration**: Rename callsites from `k.ValidateAndUnify(schema, layers)` to `k.ValidateConfigDetailed(schema, sources)`. Construct `sources` as `[]kernel.Source` instead of `helpervalues.Stack`.

### Requirement: Removal of validate.UnifyAndValidate

**Reason**: Already removed in slice 05; no further action. The current change does not reintroduce the package.

**Migration**: None — this requirement documented work already completed.

### Requirement: KernelOwner Interface Surfaces Tier-1 Validation Method

**Reason**: With `ValidateConfigDetailed` living on `*Kernel` directly, no helper package needs to call back into the kernel through an interface. The `KernelOwner` interface and the `opm/helper/values/` package are deleted entirely.

**Migration**: Remove all imports of `opm/helper/values`. References to `values.KernelOwner` are unsatisfiable; replace any code that depended on the indirection with a direct dependency on `*kernel.Kernel`.
