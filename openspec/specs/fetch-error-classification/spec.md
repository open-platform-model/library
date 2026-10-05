# fetch-error-classification Specification

## Purpose
A registry fetch failure that the library returns is a typed `*FetchError` with a kind, and `ErrTransient` marks the network-level ones. An author-defect resolution failure (an import no module provides, an ambiguous import, a dependency module file that does not parse) is a typed `*ResolutionError` with a kind, and is never transient. A frontend decides retry, exit code or stall by type rather than by matching message text. `opm/errors.Classify` is the one place that reads CUE's registry and `cue/load` failure forms; a fetch form wins over an author-defect form in the same text, and every other author defect comes back unchanged (0021:D8:R12).

## Requirements

### Requirement: A fetch or resolution failure is a typed error

`opm/errors` SHALL export `FetchKind` with the values `FetchOther`, `FetchNotFound`, `FetchUnauthorized` and `FetchUnreachable`, and the type `*FetchError` with the fields `Kind`, `Coordinate`, `Status` and `Err`. `FetchNotFound` means the module, version or package is absent. `FetchUnauthorized` means the registry refused the credentials (401), or refused the access (403) where the 403 answer survives in the chain. CUE's registry client reports a 403 on a module tag lookup as not-found, so that case SHALL classify as `FetchNotFound`. `FetchUnreachable` means no HTTP response was received: a refused connection, a DNS or TLS failure, a timeout or an expired deadline. `FetchOther` is any other failed registry interaction, such as a 429 answer or a fetched archive that does not unzip. An author-defect resolution failure, which no failed registry interaction caused, is a `*ResolutionError` and never a `*FetchError`. `Status` is the HTTP status the registry answered with, or 0 when there was none or it is unknown. Source: 0021:D8:R12.

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

### Requirement: An author-defect resolution failure is a typed error

`opm/errors` SHALL export `ResolutionKind` with the values `ResolutionOther`, `ResolutionImportUnprovided`, `ResolutionImportAmbiguous` and `ResolutionModuleFileInvalid`, and the type `*ResolutionError` with the fields `Kind` and `Err`. `ResolutionImportUnprovided` means no module of the build provides an imported package. It covers an import of a package missing from the main module's own path, an import of a module the main module does not declare, and an import of a package missing from a declared dependency that was fetched. `ResolutionImportAmbiguous` means more than one module of the build provides an imported package. `ResolutionModuleFileInvalid` means a dependency's module file does not parse, whether the dependency is published or served from a local replacement directory. `ResolutionOther` is the zero value, and `Classify` SHALL NOT build it. Source: 0021:D8:R12.

`(*ResolutionError).Error()` SHALL return the text of `Err` unchanged, and `Unwrap` SHALL return `Err`. A `*ResolutionError` whose `Err` is nil SHALL NOT panic in `Error()`. A `*ResolutionError` SHALL NOT satisfy `errors.Is(err, ErrTransient)`, and `errors.As` SHALL NOT find a `*FetchError` through it unless its cause holds one.

#### Scenario: The message is the cause's message

- **WHEN** a `*ResolutionError` wraps a cause whose text is `T`
- **THEN** its `Error()` returns `T`, and `errors.As` finds a CUE error list in the cause through it

#### Scenario: An author defect is never transient

- **WHEN** an error chain holds a `*ResolutionError` of any kind and no `*FetchError`
- **THEN** `errors.Is(err, ErrTransient)` is false, and `errors.As(err, &fe)` for a `*FetchError` is false

### Requirement: Classify recognises fetch failures and author-defect resolution failures, and leaves every other error unchanged

`opm/errors` SHALL export `Classify(err error) error`. It SHALL return nil for nil. It SHALL return `err` unchanged when the chain already holds a `*FetchError` or a `*ResolutionError`, when the chain holds `context.Canceled`, and when it recognises nothing. It SHALL return a `*FetchError` that wraps `err` for a failed registry interaction: a fetch that got no response, a registry answer that refused or did not hold what was asked for, or a fetched archive that could not be used. It SHALL return a `*ResolutionError` that wraps `err` for an author-defect resolution failure that no failed registry interaction caused. When the text carries both a fetch form and an author-defect form, the fetch form SHALL win, the generic `cannot fetch` form included, so a registry failure never reads as an author defect and no text that classified as a `*FetchError` before stops being one. When the text carries several author-defect forms, the first in the order of step 2 below SHALL decide. Source: 0021:D8:R12.

`Classify` SHALL read the typed chain first: `context.DeadlineExceeded`, an `ociregistry.HTTPError` status, `modregistry.ErrNotFound`, the ociregistry not-found, unauthorized and denied codes, and `net.Error`. Only when no typed cause is found SHALL it match text. The text fallback SHALL be the only place in the library that matches the text of a registry or `cue/load` error. It SHALL cover the forms the embedded CUE version produces when `cue/load` flattens a failure, in this order:

1. The fetch forms. These are `cannot do HTTP request`, and an HTTP status in the form `<code> <status text>: `, which the embedded CUE keeps for 401, 429 and 5xx answers, so a flattened 5xx stays transient. They also include `module not found`, and `cannot find module providing package` followed by a package path at an exact version (`P@vX.Y.Z`). Only a standalone `path@version` load produces that form, after it asks the registry for that version. Last among them is `cannot fetch`, which is `FetchOther`.
2. The author-defect forms, in this order:
   - `cannot find module providing package` followed by a path without an exact version, which is `ResolutionImportUnprovided`;
   - `ambiguous import`, which is `ResolutionImportAmbiguous`;
   - `cannot parse module file`, or `import failed: ` followed directly by a module coordinate at an exact version and `: `, which is `ResolutionModuleFileInvalid`.

`cannot expand module graph` SHALL NOT be matched on its own, because it wraps both fetch failures and a malformed dependency's module file. It classifies through the form it carries. A test SHALL produce each covered form through the embedded CUE and SHALL fail when a CUE version changes one.

#### Scenario: An author defect stays a plain error

- **WHEN** `Classify` receives a `cue/load` error for a syntax error or a conflicting value
- **THEN** it returns the error unchanged, with neither a `*FetchError` nor a `*ResolutionError` in the chain, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A flattened not-found form is classified

- **WHEN** `Classify` receives a `cue/load` error whose text says it cannot fetch a dependency because the module was not found
- **THEN** it returns a `*FetchError` of kind `FetchNotFound`

#### Scenario: An exact version the registry does not provide is not found

- **WHEN** `Classify` receives the error of a standalone `path@version` load whose text says no module provides the package at that exact version
- **THEN** it returns a `*FetchError` of kind `FetchNotFound`

#### Scenario: An unresolvable import is a typed author defect

- **WHEN** `Classify` receives a `cue/load` error for an import of a package missing from the main module's own path, of a module the main module does not declare, or of a package missing from a declared dependency that was fetched
- **THEN** it returns a `*ResolutionError` of kind `ResolutionImportUnprovided` with the same message, with no `*FetchError` in the chain, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A dependency whose module file does not parse is a typed author defect

- **WHEN** `Classify` receives a `cue/load` error saying the module graph cannot be expanded because a dependency's module file does not parse, published or in a local replacement directory
- **THEN** it returns a `*ResolutionError` of kind `ResolutionModuleFileInvalid`, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A dependency module file that does not parse on the direct import path is a typed author defect

- **WHEN** `Classify` receives a `cue/load` error for a directly imported, declared dependency whose module file does not parse, published or in a local replacement directory
- **THEN** it returns a `*ResolutionError` of kind `ResolutionModuleFileInvalid`, and no `*FetchError` is in the chain

#### Scenario: An ambiguous import is a typed author defect

- **WHEN** `Classify` receives a `cue/load` error saying an imported package is found in more than one module of the build
- **THEN** it returns a `*ResolutionError` of kind `ResolutionImportAmbiguous`, and `errors.Is(err, ErrTransient)` is false

#### Scenario: The first author-defect form decides

- **WHEN** `Classify` receives a text that carries an unprovided import, an ambiguous import and an unparsable module file, and no fetch form
- **THEN** it returns a `*ResolutionError` of kind `ResolutionImportUnprovided`

#### Scenario: A registry failure beside an unprovided import is a fetch failure

- **WHEN** `Classify` receives an error whose text says no module provides a package and also carries `cannot do HTTP request` or a 503 status
- **THEN** it returns a `*FetchError`, and no `*ResolutionError`

#### Scenario: An archive that does not unzip beside an unprovided import stays a fetch failure

- **WHEN** `Classify` receives a CUE error list holding `cannot fetch m@v0.0.1: unzip ...: zip: not a valid zip file` and `cannot find module providing package a.b/c`
- **THEN** it returns a `*FetchError` of kind `FetchOther`, and no `*ResolutionError`

#### Scenario: Cancellation is not a fetch failure

- **WHEN** `Classify` receives an error that wraps `context.Canceled`
- **THEN** it returns the error unchanged, and `errors.Is(err, context.Canceled)` still holds

#### Scenario: Classifying twice changes nothing

- **WHEN** `Classify` receives an error whose chain already holds a `*FetchError` or a `*ResolutionError`
- **THEN** it returns the same error

### Requirement: The library returns author-defect resolution failures typed

Every library path that the requirement "The library classifies every fetch and resolution failure it returns" names SHALL return an author-defect resolution failure as a `*ResolutionError` inside its existing wrap. The wrap text SHALL be unchanged. Source: 0021:D8:R12.

#### Scenario: A directory module importing an undeclared module

- **WHEN** `Kernel.AcquireModuleFromDir` loads a module whose package imports a module that its `cue.mod/module.cue` does not declare
- **THEN** `errors.As` yields a `*ResolutionError` of kind `ResolutionImportUnprovided`, no `*FetchError` is in the chain, the error is not transient, and the message text is the text the verb returned before this change

#### Scenario: A directory module importing a missing own-path package

- **WHEN** `Kernel.AcquireModuleFromDir` loads a module whose package imports a package under the module's own path that does not exist
- **THEN** `errors.As` yields a `*ResolutionError` of kind `ResolutionImportUnprovided`, and the error is not transient
