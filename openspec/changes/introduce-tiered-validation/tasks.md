## 1. Package Skeleton

- [ ] 1.1 Create `pkg/helper/values/` directory with package doc comment explaining the Tier-1 / Tier-2 split and pointing at the umbrella enhancement
- [ ] 1.2 Define `Layer` struct: `Name string; Source string; Value cue.Value`
- [ ] 1.3 Define `Stack []Layer`

## 2. ValidateAndUnify Implementation

- [ ] 2.1 Implement `func ValidateAndUnify(k *kernel.Kernel, schema cue.Value, layers Stack) (cue.Value, *MultiSourceError)`
- [ ] 2.2 For each layer, call `validate.Config(schema, layer.Value, "values", layer.Name)`; collect any `*oerrors.ConfigError`
- [ ] 2.3 If any layer has errors, return `cue.Value{}` and a populated `*MultiSourceError`
- [ ] 2.4 If all layers pass, unify in order: `merged = layers[0].Value`; `for i := 1; i < len(layers); i++ { merged = merged.Unify(layers[i].Value) }`
- [ ] 2.5 Handle empty Stack: return zero-value `cue.Value{}` and nil error

## 3. MultiSourceError Type

- [ ] 3.1 Define `MultiSourceError` struct holding a slice of per-layer error records
- [ ] 3.2 Implement `Error() string` that summarizes per-layer issues
- [ ] 3.3 Implement `Errors() []LayerError` returning the structured per-layer entries; `LayerError` carries `LayerName`, `Source`, and `*oerrors.ConfigError`
- [ ] 3.4 Implement `Unwrap() []error` returning the underlying `*ConfigError`s for stdlib `errors.Is/As` walking

## 4. Kernel Convenience Method

- [ ] 4.1 Add `(k *Kernel) ValidateAndUnify(schema cue.Value, layers Stack) (cue.Value, *MultiSourceError)` delegating to the helper
- [ ] 4.2 Confirm godoc points to the helper package as the canonical implementation

## 5. Remove validate.UnifyAndValidate

- [ ] 5.1 Delete `validate.UnifyAndValidate` and its tests
- [ ] 5.2 Confirm no remaining references in kernel internals (`grep -rn "UnifyAndValidate" pkg/`)
- [ ] 5.3 Confirm no remaining references in fixtures or examples

## 6. Tests

- [ ] 6.1 Unit test: empty Stack → zero `cue.Value`, nil error
- [ ] 6.2 Unit test: single valid Layer → unified result, nil error
- [ ] 6.3 Unit test: multiple valid Layers in order → expected merge result
- [ ] 6.4 Unit test: one Layer with schema violation → zero value, `*MultiSourceError` containing one entry
- [ ] 6.5 Unit test: multiple Layers with violations → aggregated `*MultiSourceError`
- [ ] 6.6 Unit test: layer override semantics — later layer wins for conflicting field
- [ ] 6.7 Integration test: round-trip through Kernel.ValidateAndUnify → Kernel.ValidateConfig (Tier-2) → confirm Tier-2 passes when Tier-1 passed

## 7. Documentation

- [ ] 7.1 CHANGELOG entry: "Removed `validate.UnifyAndValidate`; use `pkg/helper/values.ValidateAndUnify`"; show before/after migration recipe
- [ ] 7.2 Update `library/README.md` to demonstrate the helper in the Quick Start
- [ ] 7.3 `pkg/helper/values/doc.go` package doc covering: tier split, layer ordering, when to use, example with three layers
- [ ] 7.4 Cross-reference umbrella D1 / D5 in package doc

## 8. Validation

- [ ] 8.1 Run `task fmt`
- [ ] 8.2 Run `task vet`
- [ ] 8.3 Run `task lint`
- [ ] 8.4 Run `task test`
- [ ] 8.5 Run `task check`
