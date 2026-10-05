## Why

0021:D8:R12 is a GA exit criterion: a caller tells each fetch or resolution failure the library
returns by type, without matching on message text, including which kind of resolution failed.
Library PR #205 typed every failed registry interaction as a `*FetchError`, and left the
author-defect resolution failures unclassified on purpose: `opm/errors.Classify` returns them
unchanged. Those are an import of a missing package under the module's own path, an import that no
declared module provides, and a package missing from a fetched declared dependency. CUE v0.17.1
reports all three as `cannot find module providing package P`, through an internal type that it
flattens into the instance error, so a caller that wants to tell one apart has only the text.

So the cli still matches text: `internal/publish/compat.go` `unprovidedImport` reads
`cannot find module providing package` to treat a predecessor whose import no module provides as
absent. Its `internal/config/platform.go` hint also reads `cannot find package` and
`cannot expand module graph`. The operator cannot tell such a defect from any other unclassified
error, so it stalls on it without saying why.

The 0021 delivery log records that R12 waited on an owner ruling about these failures. The ruling
is to read R12 strictly: the library types the author-defect resolution failures too, and the cli
drops its last text match.

## What Changes

- `opm/errors` gains `*ResolutionError{Kind, Err}` and `ResolutionKind`, distinct from
  `*FetchError`. `Error()` is the cause's text unchanged, and `Unwrap` returns the cause. A
  `*ResolutionError` is never transient, and `errors.Is(err, ErrTransient)` is false for it. The
  kinds are:
  - `ResolutionImportUnprovided`: no module of the build provides an imported package. This
    covers the three cases from #205, which share one CUE form;
  - `ResolutionImportAmbiguous`: more than one module of the build provides an imported package;
  - `ResolutionModuleFileInvalid`: a dependency's published module file does not parse;
  - `ResolutionOther`, the zero value. `Classify` never builds it.
- `Classify` also recognises these forms and returns them wrapped in a `*ResolutionError`. A fetch
  form anywhere in the same text still wins, so a registry failure never reads as an author defect.
  It stays idempotent for a chain that already holds either type. A syntax error, a conflict and
  every other error it does not recognise still come back unchanged.
- No library call site changes: every site that returns a `cue/load` resolution failure already
  passes it through `Classify` (#205), so each now returns the new type.
- Specs: `fetch-error-classification` REMOVES the `Classify` requirement that left author
  defects unchanged, and ADDS it back under a new name with the author-defect scenario rewritten.
  It ADDS the `*ResolutionError` type requirement and a site-level requirement.

Not in this change:

- The cli and operator moves onto the new type. Each is a frontend change on the first library
  release that carries this one. Section 4 proves that the cli's `unprovidedImport` can be replaced
  by `Kind == ResolutionImportUnprovided` with every pinned exit code unchanged. It runs the cli's
  pinning tests against this branch through a temporary Go workspace, and nothing is committed to
  the cli. The operator adds `*ResolutionError` to its terminal causes.
- The cli's `#registry` hint match in `internal/config/platform.go`. That matches an evaluation
  error (the platform author's own field), not a fetch or resolution failure, so 0021:D8:R12 does
  not cover it.
- Telling the three unprovided-import causes apart. See design.md D2.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `fetch-error-classification`: author-defect resolution failures become a typed
  `*ResolutionError`. The `Classify` requirement is replaced (REMOVED + ADDED), and two
  requirements are ADDED.

## Impact

- Packages: `opm/errors` (new `resolution.go`, `classify.go`, the package doc) and the tests in
  `opm/errors` and `opm/kernel` (`fetch_classify_test.go`). No other package changes.
- Public surface: additive. New exported symbols in `opm/errors`, and no signature changes.
- Behaviour: an error `Classify` used to return unchanged as an author defect now comes back with
  one `*ResolutionError` link around it. `Error()` text is identical, `errors.As` on the cause
  (CUE error lists included) still matches, and no `*FetchError` or `ErrTransient` answer
  changes. A caller that compared `Classify(err) == err` for these forms sees a different value.
  No frontend does: the cli's `unprovidedImport` checks only for a `*FetchError`.
- Downstream: neither frontend needs a code change to keep building or to keep its exit codes.
- SemVer: MINOR (additive). Release class `feat`. Completes 0021:D8:R12 on the library side.
