## REMOVED Requirements

### Requirement: One validation marker and no projection of CUE errors

**Reason**: The requirement held two rules, and one of them was a transition: `IdentityError` kept a value receiver, an `As` method and a deprecated value target while opm-operator at `main` used the value forms. opm-operator left them in opm-operator#273, so the transition ends, and its scenario "The deprecated value target still matches" no longer holds. The two rules return below as two requirements, unchanged except for the end of the transition.

**Migration**: Build the error as `&errors.IdentityError{...}` and match it with a `*errors.IdentityError` target (`errors.As(err, &ptr)` or `errors.AsType[*errors.IdentityError](err)`). A value-typed target no longer matches: `errors.AsType[errors.IdentityError]` does not compile, and `errors.As` with a value-typed target is reported by `go vet` and panics at run time.

## ADDED Requirements

### Requirement: A validation failure has one marker type and no projection

The library SHALL define exactly one Go type for a configuration validation failure: `ConfigValidationError` in `opm/errors`, a pointer-receiver marker that wraps the CUE error tree unchanged. It SHALL carry the tree in its `Err` field, return it from `Unwrap`, and return the tree's own text from `Error`. It SHALL NOT project, group, re-order or reword the CUE errors, and the library SHALL NOT provide a walking or formatting API beside `cuelang.org/go/cue/errors`. The names `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, `MultiSourceError`, `LayerError`, and `DetailedError` SHALL NOT exist as exported symbols anywhere in the library.

#### Scenario: opm/errors carries no validation projections

- **WHEN** a developer reads `opm/errors/`
- **THEN** its exported identifiers are the acquisition and synthesis sentinels, the acquisition identity error (`IdentityError`), `TransformError`, the fetch and resolution classification, the render verdict rows and refusal causes (skew, routing, contract, demand, unmatched-component and core-floor), and the one validation marker `ConfigValidationError`, which wraps a CUE validation error and projects nothing
- **AND** no `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, or `GroupedError` types are present

#### Scenario: Frontends rely on cuelang.org/go/cue/errors

- **WHEN** a frontend wants per-position iteration over validation errors
- **THEN** it imports `cuelang.org/go/cue/errors` and uses `errors.Errors(err)` plus `errors.Positions(ce)` to walk the tree
- **AND** the library does not provide a parallel walking API

#### Scenario: A validation failure is matched by type

- **WHEN** `ValidateConfigDetailed` refuses values that do not satisfy the schema, and the caller wraps the error with `%w` any number of times
- **THEN** `errors.As` with a `*ConfigValidationError` target succeeds
- **AND** `cueerrors.Errors` on the same error returns the same CUE errors, in the same order, as on the `Err` field

### Requirement: Every typed error has a pointer receiver

Every error type in `opm/errors` that has an `Error` method SHALL declare it on the pointer receiver, with no exception, so a caller matches each of them with `errors.As` and a pointer target and the value type is not an error. `IdentityError` SHALL follow the rule: the library returns `*IdentityError`, the type declares `Error` on its pointer receiver, and it SHALL NOT declare an `As` method. A value-typed `IdentityError` target SHALL NOT match: `errors.AsType[IdentityError]` does not compile, because the value type does not implement `error`.

#### Scenario: Every typed error matches through a pointer target

- **WHEN** a module acquired from a registry declares another path or version than the coordinate it was fetched by
- **THEN** `errors.As` with a `*IdentityError` target succeeds on the returned error
- **AND** every other type in `opm/errors` that implements `error` does so on its pointer receiver only

#### Scenario: No typed error keeps a value receiver

- **WHEN** a test parses every non-test file of `opm/errors`
- **THEN** every `Error` method has a pointer receiver, and the test lists no exception
- **AND** no type of the package declares an `As` method

#### Scenario: The value type is not an error

- **WHEN** a test asks whether the type `IdentityError` implements `error`
- **THEN** only `*IdentityError` does
