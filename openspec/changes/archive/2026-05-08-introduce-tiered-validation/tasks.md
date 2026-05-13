## 1. Package Skeleton

- [x] 1.1 Create `opm/helper/values/` directory with package doc comment explaining the Tier-1 / Tier-2 split and pointing at the umbrella enhancement
- [x] 1.2 Define `Layer` struct: `Name string; Source string; Value cue.Value`
- [x] 1.3 Define `Stack []Layer`

## 2. ValidateAndUnify Implementation

- [x] 2.1 Implement `func ValidateAndUnify(k *kernel.Kernel, schema cue.Value, layers Stack) (cue.Value, *MultiSourceError)` — implemented as `ValidateAndUnify(KernelOwner, schema, layers)` per the helper/platform precedent (small interface avoids the kernel ↔ helper import cycle; `*kernel.Kernel` satisfies the interface so call sites are unchanged)
- [x] 2.2 For each layer, call `validate.ConfigPartial(schema, layer.Value, "values", layer.Name)`; collect any `*oerrors.ConfigError` — added `validate.ConfigPartial` (Tier-1 partial mode without `cue.Concrete(true)`); the original task said `validate.Config` but that hardcodes concreteness and would reject every partial overlay layer
- [x] 2.3 If any layer has errors, return `cue.Value{}` and a populated `*MultiSourceError`
- [x] 2.4 If all layers pass, unify in order: `merged = layers[0].Value`; `for i := 1; i < len(layers); i++ { merged = merged.Unify(layers[i].Value) }`
- [x] 2.5 Handle empty Stack: return zero-value `cue.Value{}` and nil error

## 3. MultiSourceError Type

- [x] 3.1 Define `MultiSourceError` struct holding a slice of per-layer error records
- [x] 3.2 Implement `Error() string` that summarizes per-layer issues
- [x] 3.3 Implement `Errors() []LayerError` returning the structured per-layer entries; `LayerError` carries `LayerName`, `Source`, and `*oerrors.ConfigError`
- [x] 3.4 Implement `Unwrap() []error` returning the underlying `*ConfigError`s for stdlib `errors.Is/As` walking

## 4. Kernel Convenience Method

- [x] 4.1 Add `(k *Kernel) ValidateAndUnify(schema cue.Value, layers Stack) (cue.Value, *MultiSourceError)` delegating to the helper
- [x] 4.2 Confirm godoc points to the helper package as the canonical implementation

## 5. Remove validate.UnifyAndValidate

- [x] 5.1 Delete `validate.UnifyAndValidate` and its tests
- [x] 5.2 Confirm no remaining references in kernel internals (`grep -rn "UnifyAndValidate" opm/`)
- [x] 5.3 Confirm no remaining references in fixtures or examples

## 6. Tests

- [x] 6.1 Unit test: empty Stack → zero `cue.Value`, nil error
- [x] 6.2 Unit test: single valid Layer → unified result, nil error
- [x] 6.3 Unit test: multiple valid Layers in order → expected merge result
- [x] 6.4 Unit test: one Layer with schema violation → zero value, `*MultiSourceError` containing one entry
- [x] 6.5 Unit test: multiple Layers with violations → aggregated `*MultiSourceError`
- [x] 6.6 Unit test: layer override semantics — later layer wins for conflicting field (via CUE `*default` disjunctions, the canonical OPM overlay pattern)
- [x] 6.7 Integration test: round-trip through Kernel.ValidateAndUnify → Kernel.ValidateConfig (Tier-2) → confirm Tier-2 passes when Tier-1 passed

## 7. Documentation

- [x] 7.1 CHANGELOG entry: "Removed `validate.UnifyAndValidate`; use `opm/helper/values.ValidateAndUnify`"; show before/after migration recipe
- [x] 7.2 Update `library/README.md` to demonstrate the helper in the Quick Start
- [x] 7.3 `opm/helper/values/doc.go` package doc covering: tier split, layer ordering, when to use, example with three layers
- [x] 7.4 Cross-reference umbrella D1 / D5 in package doc

## 8. Validation

- [x] 8.1 Run `task fmt`
- [x] 8.2 Run `task vet`
- [x] 8.3 Run `task lint`
- [x] 8.4 Run `task test`
- [x] 8.5 Run `task check`
