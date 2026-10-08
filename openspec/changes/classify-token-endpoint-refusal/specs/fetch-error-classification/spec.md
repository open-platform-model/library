## ADDED Requirements

### Requirement: A token endpoint's answer is a registry answer

When a registry uses token authentication and its token endpoint answers the token request with an HTTP status, the failure SHALL classify by that status, the same as an answer from the registry itself: 401 and 403 are `FetchUnauthorized`, 404 is `FetchNotFound`, and every other status is `FetchOther`, each with `Status` set to the answered code. It SHALL NOT classify as `FetchUnreachable`, and it SHALL satisfy `errors.Is(err, ErrTransient)` only when the status is 500 or higher. This SHALL hold on a chain that keeps the typed cause and on a chain that the embedded CUE flattened into text, which includes the error of a module push (`modregistry.Client.PutModule`) that a frontend passes to `Classify`. A caller tells a refused credential from a refused permission by `Status` (401 or 403).

One case is outside this requirement. The embedded CUE's registry client reports a 403 or a 404 on a module version lookup as `modregistry.ErrNotFound` and drops the status, and it does so for the token endpoint's answer too. That case SHALL classify as `FetchNotFound`, as the requirement "A fetch or resolution failure is a typed error" says for a forbidden tag lookup.

#### Scenario: A push whose token endpoint refuses the credentials

- **WHEN** a module push goes to a registry that answers with a Bearer challenge, the token endpoint answers 401, and the push error is passed to `Classify`
- **THEN** the result is a `*FetchError` of kind `FetchUnauthorized` with `Status` 401, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A push whose token endpoint refuses the permission

- **WHEN** a module push goes to a registry that answers with a Bearer challenge, the token endpoint answers 403, and the push error is passed to `Classify`
- **THEN** the result is a `*FetchError` of kind `FetchUnauthorized` with `Status` 403, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A directory module whose dependency registry refuses the token

- **WHEN** `Kernel.AcquireModuleFromDir` loads a module whose dependency must be fetched from a registry that answers with a Bearer challenge and whose token endpoint answers 401
- **THEN** `errors.As` yields a `*FetchError` of kind `FetchUnauthorized` with `Status` 401, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A registry fetch whose token endpoint refuses the credentials

- **WHEN** `Kernel.AcquireModuleFromRegistry` asks a registry that answers with a Bearer challenge and whose token endpoint answers 401
- **THEN** `errors.As` yields a `*FetchError` of kind `FetchUnauthorized` with `Status` 401, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A token endpoint that fails stays transient

- **WHEN** a module push goes to a registry that answers with a Bearer challenge, the token endpoint answers 503, and the push error is passed to `Classify`
- **THEN** the result is a `*FetchError` of kind `FetchOther` with `Status` 503, and `errors.Is(err, ErrTransient)` is true

#### Scenario: A token endpoint that throttles is not transient

- **WHEN** a module push goes to a registry that answers with a Bearer challenge, the token endpoint answers 429, and the push error is passed to `Classify`
- **THEN** the result is a `*FetchError` of kind `FetchOther` with `Status` 429, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A forbidden token on a version lookup is not found

- **WHEN** `Kernel.AcquireModuleFromRegistry` asks a registry that answers with a Bearer challenge and whose token endpoint answers 403
- **THEN** the error is a `*FetchError` of kind `FetchNotFound` with `Status` 0

## MODIFIED Requirements

### Requirement: Classify recognises fetch failures and author-defect resolution failures, and leaves every other error unchanged

`opm/errors` SHALL export `Classify(err error) error`. It SHALL return nil for nil. It SHALL return `err` unchanged when the chain already holds a `*FetchError` or a `*ResolutionError`, when the chain holds `context.Canceled`, and when it recognises nothing. It SHALL return a `*FetchError` that wraps `err` for a failed registry interaction: a fetch that got no response, a registry answer that refused or did not hold what was asked for, or a fetched archive that could not be used. It SHALL return a `*ResolutionError` that wraps `err` for an author-defect resolution failure that no failed registry interaction caused. When the text carries both a fetch form and an author-defect form, the fetch form SHALL win, the generic `cannot fetch` form included, so a registry failure never reads as an author defect and no text that classified as a `*FetchError` before stops being one. When the text carries several author-defect forms, the first in the order of step 2 below SHALL decide. Source: 0021:D8:R12.

`Classify` SHALL read the typed chain first: `context.DeadlineExceeded`, an `ociregistry.HTTPError` status, `modregistry.ErrNotFound`, the ociregistry not-found, unauthorized and denied codes, and `net.Error`. Only when no typed cause is found SHALL it match text. The text fallback SHALL be the only place in the library that matches the text of a registry or `cue/load` error. It SHALL cover the forms the embedded CUE version produces when `cue/load` flattens a failure, in this order:

1. The fetch forms. First is the answered token request: `cannot do HTTP request: ` followed on the same line by a request and `: <code> <status text>` at the end of the line, where the status text is the one HTTP gives the code and the code is 400 or higher. The embedded registry client writes that form only when the token endpoint of a token-authenticated registry answered the token request with that status, so it classifies by the status, as a registry answer does, and never as `FetchUnreachable`. Then come `cannot do HTTP request` in any other form, which is `FetchUnreachable`, and an HTTP status in the form `<code> <status text>: `, which the embedded CUE keeps for 401, 429 and 5xx answers, so a flattened 5xx stays transient. They also include `module not found`, and `cannot find module providing package` followed by a package path at an exact version (`P@vX.Y.Z`). Only a standalone `path@version` load produces that form, after it asks the registry for that version. Last among them is `cannot fetch`, which is `FetchOther`.
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

#### Scenario: A flattened token refusal is not unreachable

- **WHEN** `Classify` receives an error with no typed cause whose text ends in `cannot do HTTP request: Post "<url>": 403 Forbidden`
- **THEN** it returns a `*FetchError` of kind `FetchUnauthorized` with `Status` 403, and `errors.Is(err, ErrTransient)` is false

#### Scenario: A request that got no response stays unreachable

- **WHEN** `Classify` receives an error with no typed cause whose text carries `cannot do HTTP request` and ends in a dial failure, and no line of it ends in an HTTP status
- **THEN** it returns a `*FetchError` of kind `FetchUnreachable`, and `errors.Is(err, ErrTransient)` is true
