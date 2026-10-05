## Context

Line numbers are at `origin/main` `88afdfb`.

**What #205 left.** `opm/errors/classify.go` `Classify` (`:34-52`) returns an error unchanged when
neither its typed branch nor its text fallback (`classifyText`, `:129-147`) recognises a failed
registry interaction. The fallback skips `cannot find module providing package P` when `P` carries
no exact version (`textVersionNotProvided`, `:122`), and it skips `cannot expand module graph` on
its own (`:124-128`). `opm/errors/classify_cue_test.go` pins these forms against CUE v0.17.1 as
returned unchanged:

| Form (`classify_cue_test.go` case) | Text |
| --- | --- |
| `load/undeclared-import` | `cannot find module providing package test.example/other@v0` |
| `load/own-path-missing-package` | `cannot find module providing package spike.example/main/missing` |
| `load/dependency-missing-package` | `cannot find module providing package test.example/dep/missing` |
| `load/malformed-dependency-module-file` | `cannot expand module graph: P@V: cannot parse module file from P@V: bogus: field not allowed` |

`opm/kernel/fetch_classify_test.go` `TestFetchClassify_UnresolvableImportStaysPlain` (`:118-139`)
pins the first two through `AcquireModuleFromDir`, and `opm/errors/classify_test.go`
`TestClassify_UnrecognisedIsUnchanged` (`:43-62`) pins the constructed forms.

**Where CUE produces them.** `internal/mod/modpkgload/import.go` (CUE v0.17.1):
`ImportMissingError.Error()` (`:414-416`) is `cannot find module providing package ` + path,
and `importFromModules` returns it at `:153` once the full module graph holds no module that
provides the path. `AmbiguousImportError.Error()` (`:393-406`) starts
`ambiguous import: found package P in multiple locations:`. `mod/modcache/fetch.go:77` formats
`cannot parse module file from %v: %v`, and `import.go:164` wraps a graph error with
`cannot expand module graph: %v`. Both types sit in an internal package, and `cue/load` flattens
the instance error into a CUE error list (#205 spike), so `errors.As` cannot reach them from
outside CUE. The text is the only signal. The library already keeps that text matching in one
place, `classifyText`.

**What the frontends do today** (cli `origin/main` `cd10f2d2`, opm-operator `origin/main`):

- cli `internal/publish/compat.go:313-317` `unprovidedImport`: true when `Classify` finds no
  `*FetchError` and the text holds `cannot find module providing package`. Its caller
  `loadPublishedPackage` (`:242-254`) reads that as "absent" (found=false). Every other
  unrecognised failure is a `*ConnectivityError` (exit 3). The pins are
  `internal/publish/registryfailure_pin_test.go` `TestLoadPublishedPackage_Pinned` (`:114-163`;
  the rows "import its dependency does not provide" and "import of a missing own-path package"
  read `absent`) and `TestUnprovidedImport_RegistryFailureIsNotAbsent` (`:216-225`). The second
  test drives `unprovidedImport` with constructed text, and an unprovided-import text that also
  carries `cannot do HTTP request` or a 503 must not read as absent.
- cli `internal/config/platform.go:98-117` `platformBuildHint`: matches `cannot find package`
  (cue/load's prefix for every failed import) and `cannot expand module graph` to pick a hint,
  never an exit code. It also matches `#registry`, which is an evaluation error and outside this
  change.
- opm-operator `internal/reconcile/resolution.go` `IsTransientFailure` (`:70-76`) retries any
  `*FetchError` and treats everything else, the unclassified author defects included, as
  terminal. It cannot say why: no type tells an import defect from a syntax error.

## Goals / Non-Goals

**Goals:**

- A caller tells an author-defect resolution failure by type (`errors.As(err, &re)`) and its
  kind by `re.Kind`, with no text matching (0021:D8:R12).
- Every `*FetchError` and `ErrTransient` answer stays as it is, and every message stays
  byte-identical.
- The cli's `unprovidedImport` can become `Kind == ResolutionImportUnprovided` with its pinned
  exit codes unchanged. Section 4 proves this against the cli's own pin tests.

**Non-Goals:**

- The frontend changes themselves (cli `compat.go` and `platform.go`, operator terminal causes).
- Telling the three unprovided-import causes apart (D2).
- Classifying evaluation errors (a conflict, a missing field, the platform `#registry` key check).

## Decisions

### D1. The API

```go
// opm/errors/resolution.go
type ResolutionKind int

const (
    ResolutionOther             ResolutionKind = iota // zero value; Classify never builds it
    ResolutionImportUnprovided                        // no module of the build provides an imported package
    ResolutionImportAmbiguous                         // more than one module of the build provides it
    ResolutionModuleFileInvalid                       // a module file the resolution reads does not parse
)

func (k ResolutionKind) String() string // "other", "import unprovided", "import ambiguous", "module file invalid"

type ResolutionError struct {
    Kind ResolutionKind
    Err  error // the cause, unchanged
}

func (e *ResolutionError) Error() string // e.Err.Error(); "dependency resolution failed: <kind>" when Err is nil
func (e *ResolutionError) Unwrap() error // e.Err
```

`*ResolutionError` is a separate type from `*FetchError`. It is not a `FetchKind`, because a
frontend that retries every `*FetchError` (the operator does, `IsTransientFailure`) must never
retry an author defect. It has no `Is` method, so `errors.Is(err, ErrTransient)` is false through
it. The name follows the 0021:D8:R12 wording "fetch or resolution failure". The operator already
uses "typed resolution error" for its own identity and render-gate causes
(`isTypedResolutionError`). The library type does not change that function. When the operator
adopts it, it adds `*ResolutionError` to `isTerminalCause`.

`ResolutionOther` exists so that a zero `ResolutionError{}` built by hand does not claim a kind
it was never given, which mirrors `FetchOther`.

There is no import-path field. The cause's text names the path, CUE's list error carries its
position, and no consumer asked for the path as data (Principle VII). Adding the field later is
additive.

### D2. One kind for the three unprovided-import causes

#205 left three failures unclassified: a missing own-path package, an import that no declared
module provides, and a package missing from a fetched declared dependency. CUE reports all three
through the same `ImportMissingError` and the same text, and `Classify` receives only the error.
To tell them apart it would need the main module's path and its declared dependencies. The cli's
call site is a standalone `path@version` load with no main module of its own, so it does not have
them. A library site could infer the cause from its `cue.mod/module.cue`, but it would have to
handle a transitive provider and an import that fails inside a dependency's own package. That
inference would be a second, larger source of guesses. No consumer branches on the cause either:
the cli reads all three as "absent", and the operator stalls on all three. So all three are
`ResolutionImportUnprovided`, and the `ResolutionKind` doc names them. A later change can split the
kind additively if a consumer needs that.

One more producer has the same text. Run against a registry that lists no version of an imported
module, `cue mod tidy` says `cannot find module providing package P` (the #205 spike). That is an
unpublished or mistyped import, and it is not transient either. The kind doc says so. The cli's
tidy path reads only `FetchUnreachable` (`IsConnectivityError`), so nothing there changes.

### D3. How `Classify` decides

```go
func Classify(err error) error {
    if err == nil { return nil }
    if errors.As(err, new(*FetchError)) || errors.As(err, new(*ResolutionError)) { return err }
    if errors.Is(err, context.Canceled) { return err }
    if kind, status, ok := classifyTyped(err); ok { return &FetchError{...} }
    if kind, status, ok := classifyFetchText(msg); ok { return &FetchError{...} }   // steps 1-3 of today's fallback
    if kind, ok := classifyResolutionText(msg); ok { return &ResolutionError{Kind: kind, Err: err} }
    if strings.Contains(msg, textCannotFetch) { return &FetchError{Kind: FetchOther, Err: err} }
    return err
}
```

The fallback is today's order, with the author-defect forms placed between the specific fetch
forms and the generic `cannot fetch `:

1. The fetch forms, unchanged: `cannot do HTTP request`, an HTTP status, `module not found`, and
   `cannot find module providing package P@vX.Y.Z`. Any fetch form anywhere in the text wins. A
   CUE error list may carry one error per failed import, and a registry failure in it is the one a
   retry may cure. This is also exactly the guard the cli's
   `TestUnprovidedImport_RegistryFailureIsNotAbsent` pins.
2. The author-defect forms:
   - `cannot find module providing package ` → `ResolutionImportUnprovided`. Step 1 has already
     taken the exact-version form, so whatever reaches this step carries at most a major version;
   - `ambiguous import: ` → `ResolutionImportAmbiguous`;
   - `cannot parse module file` → `ResolutionModuleFileInvalid`.
3. `cannot fetch ` → `FetchOther`, unchanged.

The author-defect forms come before `cannot fetch ` on purpose. If a dependency's module file
fails to parse on the direct import path rather than in graph expansion, the text could read
`cannot fetch P@V: cannot parse module file from P@V: ...`. That is the same defect, and it must
read the same way. Section 1 measures whether CUE v0.17.1 produces that form. Either way the order
keeps the two consistent.

`Classify` is still applied only to load errors, never to an evaluation error (#205 D4). A
conflict message can quote an author's own strings, and the text fallback must never see one.
That rule is what keeps `cannot parse module file` and `ambiguous import: ` safe to match.

### D4. No call site changes

Every library site that returns a `cue/load` resolution failure already wraps
`oerrors.Classify(cause)`: `loader.LoadDir` (`opm/internal/loader/load.go:124`),
`compileSource` (`opm/kernel/source_loader.go:119`), `renderstage.Build`
(`opm/internal/renderstage/stage.go:226`), `schema.OCILoader.Load` (`opm/schema/loader.go:164`)
and `platformmodule.Closure` (`opm/helper/platformmodule/closure.go:105`). `classifyFetch`
(`opm/internal/loader/registry.go:222-231`) sets `Coordinate` only on a `*FetchError`, and a
`*ResolutionError` passes through it untouched. So the new type reaches every caller with no edit
outside `opm/errors`, and the kernel tests pin it through the public verbs.

### D5. Proving the cli replacement

The cli's text match must be replaceable with its exit codes unchanged, and the cli's own pin
tests are the proof. Section 4 does this without committing to
the cli:

1. Clone cli `main` into the scratch directory.
2. Replace the body of `unprovidedImport` with
   `errors.As(liberrors.Classify(err), &re) && re.Kind == liberrors.ResolutionImportUnprovided`.
   Point the comment's last paragraph at the type.
3. Run `go work init <clone> <library worktree>` in a scratch directory, the same setup
   `.tasks/consumer-build.sh` uses.
4. Under `GOWORK` and an absolute `TMPDIR`, run
   `go test ./internal/publish/ -run 'TestLoadPublishedPackage_Pinned|TestUnprovidedImport_RegistryFailureIsNotAbsent|TestProbedPackageAbsent|TestCompatScan|TestGateCompat'`
   and `go test ./internal/cuemod/ -run 'TestIsConnectivityError|TestIsVersionNotHeld'`.

These are unit tests with in-process registries, not the cluster suites. Every row must pass
unchanged, and the result goes into "Verification". `consumer-build.sh` itself then builds and vets
both consumers at `main`, unpatched, against the branch. The same scratch clone also tries the
`platformBuildHint` replacement (any `*FetchError` or `*ResolutionError` in place of the
`cannot find package` and `cannot expand module graph` matches) against the cli's hint pins
(`internal/config/platform_hint_pin_test.go`), and records the result. That replacement is the cli change's to decide.

## Research & Decisions

### A new type, not a new FetchKind

**Context**: A `FetchKind` value would give one type for every resolution failure.
**Explored**: The operator's `IsTransientFailure` retries any `*FetchError`. The cli's
`IsFetchNotFound`, `IsConnectivityError` and `check.go` branch on `*FetchError` kinds.
**Decision**: A separate `*ResolutionError`.
**Rationale**: Any new `FetchKind` would change the operator's retry answer for these failures
from stall to retry. The operator would then retry an author's import typo forever, which is the
failure 0021:D8:R12 exists to prevent. A separate type changes no existing answer.

### Three kinds, not one

**Context**: #205 names the three unprovided-import cases. The owner's ruling reads
R12 strictly, and R12 asks "which kind of fetch or resolution failed".
**Explored**: The other resolution failures `Classify` leaves unchanged today: a dependency module
file that does not parse (pinned in `classify_cue_test.go`), and an ambiguous import (CUE's
`AmbiguousImportError`). The cli's `platformBuildHint` matches the first through
`cannot expand module graph`.
**Decision**: Type those two as well, as their own kinds.
**Rationale**: Each one is a resolution failure no retry cures, and leaving either unclassified
would keep a cli text match alive (`cannot expand module graph`) or leave R12 open. Section 3 adds
them, separately from the required kind, so they can be dropped before implementation
without touching sections 1, 2 and 4. If section 1 cannot produce the ambiguous form through CUE
v0.17.1, that kind is dropped and the finding is written here.

### Spike before code

**Context**: Three forms are not yet pinned. The first is a standalone `path@version` load of a
published package whose import no module of its build provides, which is the cli's form. The
second is an ambiguous import. The third is a malformed module file reached through
`cannot fetch` instead of graph expansion, and the main module's own malformed `module.cue`.
**Decision**: Section 1 drives each through the embedded CUE and pins what it sees, before any
code changes. Its findings go under "Spike findings" below.

## Spike findings

(Section 1 writes this.)

## Risks / Trade-offs

- [A CUE bump rewords a form] → `classify_cue_test.go` fails on the bump. A CUE move reaches
  frontends only through a library release (workspace `RELEASING.md`), so the test runs first.
- [`cannot parse module file` also covers the main module's own `module.cue`] → that is an author
  defect as well, and the kind doc says "a module file the resolution reads". Section 1 records
  the form either way.
- [A caller compared `Classify(err)` with `err` to detect "unrecognised"] → that caller now sees
  a new value for these forms. Neither frontend does this; the cli checks for a `*FetchError`.
- [An error list with an unprovided import and a fetch failure reads as a fetch failure] → this is
  intended (D3). The fetch failure may be why the import is unprovided, and a retry may cure it.

## Verification

(Section 4 writes this.)
