# values-validation Specification

## Purpose

**This capability has been retired.** The `pkg/helper/values/` package and its `Layer`/`Stack`/`MultiSourceError`/`KernelOwner` types have been removed. The replacement lives in the `config-validation` capability, which exposes `Kernel.ValidateConfig`, `Kernel.ValidateConfigPartial`, and `Kernel.ValidateConfigDetailed` directly on `*Kernel` in `pkg/kernel/`.

Migration summary:

- `helpervalues.Layer{Name, Source, Value}` → `kernel.Source{Value, Name, Origin}` (field rename `Source` → `Origin`).
- `helpervalues.Stack` → `[]kernel.Source`.
- `helpervalues.ValidateAndUnify(k, schema, layers)` → `k.ValidateConfigDetailed(schema, sources)`. Pass `kernel.Partial()` for partial-mode behavior.
- `*MultiSourceError` → walk `cuelang.org/go/cue/errors.Errors(err)` and bucket by `pos.Filename()` at the call site if needed.
- `KernelOwner` interface — removed; depend on `*kernel.Kernel` directly.
- Per-source attribution now flows through `cue.Filename(Origin)` set at compile time (use `Kernel.LoadSourceFromFile`/`Bytes`/`String`).
- Errors are CUE-native (`cuelang.org/go/cue/errors.Error`); use `cueerrors.Errors(err)` to walk them and `pos.Filename()` to recover the originating source.

See the `config-validation` capability for the active requirements.

## Requirements

(None — all requirements removed; see `config-validation` for the active surface.)
