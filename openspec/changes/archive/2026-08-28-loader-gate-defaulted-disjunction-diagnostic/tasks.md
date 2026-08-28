## 1. opm/helper/loader/internal/shape

- [x] 1.1 In `requireConcrete`, on the `!IsConcrete()` branch call `f.Default()`; when a default exists return `required field %q is a defaulted disjunction (default %v), not a concrete value: identity fields must be concrete literals: %w` wrapping `ErrMissingRequiredField`; otherwise keep the existing message.

## 2. opm/helper/loader/file tests

- [x] 2.1 Add a `TestShapeGate_RejectsMalformedPackages` case: module package with `metadata.version: #T | *"1.0.1"`; assert `errors.Is(err, file.ErrMissingRequiredField)` and that the message contains `metadata.version`, `"1.0.1"` and `concrete literals`.
- [x] 2.2 Add a passing case: identity-style package with `Version: "1.0.1"` referenced as `metadata.version: Version`; assert the gate passes.

## 3. Docs

- [x] 3.1 Update the `shape` package doc comment and `requireConcrete`'s comment to state the defaulted-disjunction rule (mirrors the spec clause).

## 4. Validation gates

- [x] 4.1 `task fmt`
- [x] 4.2 `task vet`
- [x] 4.3 `task lint`
- [x] 4.4 `task test`
