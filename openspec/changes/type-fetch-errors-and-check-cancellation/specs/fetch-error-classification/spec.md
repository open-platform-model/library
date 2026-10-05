## ADDED Requirements

### Requirement: A fetch or resolution failure is a typed error

`opm/errors` SHALL export `FetchKind` with the values `FetchOther`, `FetchNotFound`, `FetchUnauthorized` and `FetchUnreachable`, and the type `*FetchError` with the fields `Kind`, `Coordinate`, `Status` and `Err`. `FetchNotFound` means the module, version or package is absent. `FetchUnauthorized` means the registry refused the credentials (401), or refused the access (403) where the 403 answer survives in the chain. CUE's registry client reports a 403 on a module tag lookup as not-found, so that case SHALL classify as `FetchNotFound`. `FetchUnreachable` means no HTTP response was received: a refused connection, a DNS or TLS failure, a timeout or an expired deadline. `FetchOther` is any other fetch or resolution failure. `Status` is the HTTP status the registry answered with, or 0 when there was none or it is unknown. Source: 0021:D8:R12.

`(*FetchError).Error()` SHALL return the text of `Err` unchanged, and `Unwrap` SHALL return `Err`, so `errors.Is` and `errors.As` on the cause keep matching through it. A `*FetchError` whose `Err` is nil SHALL NOT panic in `Error()`.

#### Scenario: The message is the cause's message

- **WHEN** a `*FetchError` wraps a cause whose text is `T`
- **THEN** its `Error()` returns `T`

#### Scenario: The cause stays reachable

- **WHEN** a `*FetchError` wraps an error chain that holds a CUE error list
- **THEN** `errors.As` finds the list through it, and `cueerrors.Errors` returns the same errors and positions it returns for the unwrapped cause

### Requirement: ErrTransient marks network-level failures only

`opm/errors` SHALL export the sentinel `ErrTransient`. `errors.Is(err, ErrTransient)` SHALL hold for a `*FetchError` in the chain exactly when its `Kind` is `FetchUnreachable` or its `Status` is 500 or higher. It SHALL NOT hold for `FetchNotFound`, for `FetchUnauthorized`, for a 429 answer, for an error with no `*FetchError` in its chain, or for a context cancellation. `(*FetchError).Transient()` SHALL report the same answer. Source: 0021:D8:R12.

#### Scenario: An unreachable registry is transient

- **WHEN** a fetch fails because the registry's port refuses the connection
- **THEN** the error is a `*FetchError` of kind `FetchUnreachable`, and `errors.Is(err, ErrTransient)` is true

#### Scenario: A server error is transient

- **WHEN** the registry answers a fetch with status 503
- **THEN** the error is a `*FetchError` with `Status` 503, and `errors.Is(err, ErrTransient)` is true

#### Scenario: An absent version is not transient

- **WHEN** a fetch asks for a version the registry does not hold
- **THEN** the error is a `*FetchError` of kind `FetchNotFound`, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A refused credential is not transient

- **WHEN** the registry answers a fetch with status 401
- **THEN** the error is a `*FetchError` of kind `FetchUnauthorized`, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A forbidden tag lookup is not found

- **WHEN** the registry answers a module fetch with status 403
- **THEN** the error is a `*FetchError` of kind `FetchNotFound`, and `errors.Is(err, ErrTransient)` is false

### Requirement: Classify recognises fetch and resolution failures and leaves every other error unchanged

`opm/errors` SHALL export `Classify(err error) error`. It SHALL return nil for nil. It SHALL return `err` unchanged when the chain already holds a `*FetchError`, when the chain holds `context.Canceled`, and when it recognises nothing. Otherwise it SHALL return a `*FetchError` that wraps `err`.

`Classify` SHALL read the typed chain first: `context.DeadlineExceeded`, an `ociregistry.HTTPError` status, `modregistry.ErrNotFound`, the ociregistry not-found, unauthorized and denied codes, and `net.Error`. Only when no typed cause is found SHALL it match text. The text fallback SHALL be the only place in the library that matches the text of a registry or `cue/load` error, and it SHALL cover the forms that the embedded CUE version produces when `cue/load` flattens a fetch failure (`cannot do HTTP request`, `cannot find module providing package`, `module not found`, `cannot fetch`, and the 401 and 5xx forms where the embedded CUE preserves them; a form it flattens beyond recognition is `FetchOther` or stays unclassified). `cannot expand module graph` SHALL NOT be matched on its own, because it also wraps a malformed dependency's module file. A test SHALL produce each covered form through the embedded CUE and SHALL fail when a CUE version changes one. Source: 0021:D8:R12.

#### Scenario: An author defect stays a plain error

- **WHEN** `Classify` receives a `cue/load` error for a syntax error or a conflicting value
- **THEN** it returns the error unchanged, with no `*FetchError` in the chain, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A flattened not-found form is classified

- **WHEN** `Classify` receives a `cue/load` error whose text says it cannot find the module providing an imported package
- **THEN** it returns a `*FetchError` of kind `FetchNotFound`

#### Scenario: Cancellation is not a fetch failure

- **WHEN** `Classify` receives an error that wraps `context.Canceled`
- **THEN** it returns the error unchanged, and `errors.Is(err, context.Canceled)` still holds

#### Scenario: Classifying twice changes nothing

- **WHEN** `Classify` receives an error whose chain already holds a `*FetchError`
- **THEN** it returns the same error

### Requirement: The library classifies every fetch and resolution failure it returns

Every library path that returns a registry fetch failure or a `cue/load` dependency resolution failure SHALL pass the cause through `Classify` inside its existing wrap. These paths are the registry acquire verbs (`FetchArtifact`, with `Coordinate` set to the fetched `path@version`), every directory acquire verb and the synth build (through `loader.LoadDir`), the load of a file-backed values source, the render module load, the schema `OCILoader` load, and the platform-module dependency closure. The wrap text SHALL be unchanged, and every existing sentinel (`ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField` and the synthesis sentinels) SHALL still match with `errors.Is`. An evaluation error (a built value's error, a shape-gate or concreteness failure) SHALL NOT be classified. Source: 0021:D8:R12.

#### Scenario: Acquiring an unpublished version

- **WHEN** `Kernel.AcquireModuleFromRegistry` asks for a version the registry does not hold
- **THEN** `errors.As` yields a `*FetchError` of kind `FetchNotFound` whose `Coordinate` names the module path and the canonical version, and the message text is the text the verb returned before this change

#### Scenario: A directory module whose dependency registry is down

- **WHEN** `Kernel.AcquireModuleFromDir` loads a module whose dependency must be fetched from a registry that refuses connections
- **THEN** `errors.Is(err, oerrors.ErrTransient)` is true

#### Scenario: A values file whose import registry is down

- **WHEN** `Kernel.AcquireInstanceFromDir` receives a file-backed values source that imports a module from a registry that refuses connections
- **THEN** `errors.Is(err, oerrors.ErrTransient)` is true

#### Scenario: A malformed package is not a fetch failure

- **WHEN** an acquire verb loads a package that does not evaluate (a conflict between two concrete values)
- **THEN** the error holds no `*FetchError` and is not transient
