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
`TestClassify_UnrecognisedIsUnchanged` (`:41-59`) pins the constructed forms.

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
  `internal/publish/registryfailure_pin_test.go` `TestLoadPublishedPackage_Pinned` (`:114-153`;
  the rows "import its dependency does not provide" and "import of a missing own-path package"
  read `absent`) and `TestUnprovidedImport_RegistryFailureIsNotAbsent` (`:216-225`). The second
  test drives `unprovidedImport` with constructed text, and an unprovided-import text that also
  carries `cannot do HTTP request` or a 503 must not read as absent.
- cli `internal/config/platform.go:98-117` `platformBuildHint`: matches `cannot find package`
  (cue/load's prefix for every failed import) and `cannot expand module graph` to pick a hint,
  never an exit code. It also matches `#registry`, an evaluation error: no library work, but the
  cli follow-up drops it too by reading the CUE error path (see "Frontend follow-ups").
- opm-operator `internal/reconcile/resolution.go` `IsTransientFailure` (`:70-76`) retries any
  `*FetchError` and treats everything else, the unclassified author defects included, as
  terminal. It cannot say why: no type tells an import defect from a syntax error. Its
  `isTerminalCause` runs before the `*FetchError` check.

## Goals / Non-Goals

**Goals:**

- A caller tells an author-defect resolution failure by type (`errors.As(err, &re)`) and its
  kind by `re.Kind`, with no text matching (0021:D8:R12).
- Every `*FetchError` and `ErrTransient` answer stays as it is, and every message stays
  byte-identical.
- Both cli text matches the 0021 delivery log records can move onto the types with their pinned
  answers unchanged: `unprovidedImport` becomes `Kind == ResolutionImportUnprovided`, and the
  two import matches of `platformBuildHint` become "any `*FetchError` or `*ResolutionError`".
  Section 4 proves both against the cli's own pin tests.

**Non-Goals:**

- The frontend changes themselves (cli `compat.go` and `platform.go`, operator stall reason).
- Telling the three unprovided-import causes apart (D2).
- Classifying evaluation errors (a conflict, a missing field, the platform `#registry` key check).
  The cli reads the `#registry` check's CUE error path instead; that needs no library type.

## Decisions

### D1. The API

```go
// opm/errors/resolution.go
type ResolutionKind int

const (
    ResolutionOther             ResolutionKind = iota // zero value; Classify never builds it
    ResolutionImportUnprovided                        // no module of the build provides an imported package
    ResolutionImportAmbiguous                         // more than one module of the build provides it
    ResolutionModuleFileInvalid                       // a dependency's module file (published or local replacement) does not parse
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
adopts it, it uses `*ResolutionError` only for its stall reason and condition message. It must
not add it to `isTerminalCause`: that check runs before the `*FetchError` check, so a
kernel-joined error holding a `*ResolutionError` from one site and a `*FetchError` from another
would stall instead of retrying. These errors are already terminal there, because the operator
does not classify them as transient.

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
    if kind, status, ok := classifyText(msg); ok { return &FetchError{...} }   // today's fallback, unchanged
    if kind, ok := classifyResolutionText(msg); ok { return &ResolutionError{Kind: kind, Err: err} }
    return err
}
```

The fallback is today's, whole and first, and the author-defect forms come after it:

1. The fetch forms, unchanged and in today's order: `cannot do HTTP request`, an HTTP status,
   `module not found`, `cannot find module providing package P@vX.Y.Z`, and the generic
   `cannot fetch `. Any fetch form anywhere in the text wins. A CUE error list may carry one error
   per failed import, and a registry failure in it is the one a retry may cure. This is also
   exactly the guard the cli's `TestUnprovidedImport_RegistryFailureIsNotAbsent` pins. Because
   the whole fetch fallback runs first, every text that is a `*FetchError` today stays one: a
   directory load whose list holds `cannot fetch m@v0.0.1: ...: zip: not a valid zip file` beside
   `cannot find module providing package a.b/c` is `FetchOther`, as today, and the cli and the
   operator keep their answers for it (retry, exit 3).
2. The author-defect forms, in this order:
   - `cannot find module providing package ` → `ResolutionImportUnprovided`. Step 1 has already
     taken the exact-version form, so whatever reaches this step carries at most a major version;
   - `ambiguous import: ` → `ResolutionImportAmbiguous`;
   - `cannot parse module file`, or cue/build's `import failed: ` followed directly by a
     module coordinate at an exact version and `: ` (`import failed: P@vX.Y.Z: `) →
     `ResolutionModuleFileInvalid`. The first is graph expansion's form; the second is the direct
     import path's form, which section 1 found (see "Spike findings").

   When one text carries several author-defect forms, the first in this order decides. A test pins
   the order with one constructed text.

Section 1 measured the direct import path for a dependency whose module file does not parse:
CUE v0.17.1 does not say `cannot fetch P@V: cannot parse module file`. It says
`import failed: P@vX.Y.Z: <parse error>`, which `Classify` returns unchanged today. So typing it
moves no `*FetchError` answer, and the order question the plan review raised does not arise. A
`cannot fetch ` text stays `FetchOther` whatever it carries.

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

1. Clone cli `main` and opm-operator `main` into the scratch directory, and run
   `GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> <worktree> <workdir>` for each,
   unpatched. Both must build and vet.
2. In a second cli clone, replace the body of `unprovidedImport` with
   `errors.As(liberrors.Classify(err), &re) && re.Kind == liberrors.ResolutionImportUnprovided`,
   and point the comment's last paragraph at the type. Replace the `cannot find package` and
   `cannot expand module graph` matches of `platformBuildHint` with "the chain holds a
   `*FetchError` or a `*ResolutionError`" (`errors.As` on the error as it arrives, since the kernel
   verb already classified it).
3. Run `GOTOOLCHAIN=local bash .tasks/consumer-build.sh <patched clone> <worktree> <workdir>`, so
   the script builds and vets the patched cli, its `_test.go` files included.
4. Under `GOWORK=<workdir>/go.work` and an absolute `TMPDIR`, run
   `go test ./internal/publish/ -run 'TestLoadPublishedPackage_Pinned|TestUnprovidedImport_RegistryFailureIsNotAbsent|TestProbedPackageAbsent|TestCompatScan|TestGateCompat'`,
   `go test ./internal/config/ -run 'TestPlatformBuildHint|TestBuildPlatformModule'` and
   `go test ./internal/cuemod/ -run 'TestIsConnectivityError|TestIsVersionNotHeld'`.

These are unit tests with in-process registries, not the cluster suites. Every row must pass
unchanged, and both runs go into "Verification". The `platformBuildHint` replacement is required,
not tried: if a hint row fails, section 1 pins the form it meets untyped, and section 2 or 3 types
it in this change, so the cli can drop both matches on this library release.

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

### Where resolution ends

0021:D8:R12 asks for each fetch or resolution failure by type. This change counts as a resolution
failure a failure to find, choose or read the module that provides an imported package. These
`cue/load` failures are not resolution failures, and `Classify` keeps returning them unchanged:

- `no files in package directory` (the cli pins it for a directory whose only `.cue` file
  cue/load ignores): the package was found, and the provider resolved; its content is what the
  load cannot use.
- A package-name mismatch (`found packages a (a.cue) and b (b.cue)`): a defect in the package's
  own files.
- An import cycle (`import cycle not allowed`): the imports all resolved; their structure is the
  defect.
- The main module's own `cue.mod/module.cue` failing its schema: the library checks it before
  any import resolves, and the text is the module file's evaluation error alone (`bogus: field
  not allowed`, section 1), with no structural form to match without reading author content.
- A syntax error, and every evaluation error (#205 D4).

Each is an author defect in package content, no retry cures it, and none of them sits in a cli or
operator text match. A later change can type them additively if a consumer needs it.

## Spike findings

Measured against the embedded CUE v0.17.1 in `classify_cue_test.go` (section 1). "Today" is the
answer at `88afdfb`.

| Case | Text (the part that decides) | Today | After this change |
| --- | --- | --- | --- |
| `standalone/unprovided-import` | `cannot find module providing package test.example/other@v0` (a major version only) | unchanged | `ResolutionImportUnprovided` |
| `load/ambiguous-import` | `ambiguous import: found package spike.example/main/dep in multiple locations:` | unchanged | `ResolutionImportAmbiguous` |
| `load/malformed-dependency-module-file-direct` | `import failed: test.example/dep@v0.0.2: bogus: field not allowed`; no `cannot fetch `, no `cannot parse module file` | unchanged | `ResolutionModuleFileInvalid` |
| `load/malformed-replacement-module-file-direct` | `import failed: test.example/dep@v0.0.2: bogus: field not allowed` (a local replacement directory); no `cannot fetch `, no `cannot parse module file` | unchanged | `ResolutionModuleFileInvalid` |
| `load/malformed-replacement-module-file` | `cannot parse module file in replacement directory: ...: bogus: field not allowed` | unchanged | `ResolutionModuleFileInvalid` |
| `load/malformed-main-module-file` | `loading module package from <dir> (.): bogus: field not allowed`; no `cannot parse module file` | unchanged | unchanged ("Where resolution ends") |
| `load/corrupt-archive-beside-unprovided-import` | `cannot fetch test.example/dep@v0.0.2: unzip ...: zip: not a valid zip file`; the undeclared import's error is not in the text | `FetchOther` | `FetchOther` |

- The ambiguous form is real, so `ResolutionImportAmbiguous` stays.
- The direct-path module-file form comes from `cue/load/modfilecache.go:53`
  (`fmt.Errorf("%v: %v", mv, err)`), reached from `newInstance` (`cue/load/import.go:426`), and
  cue/build wraps it as `import failed` (`cue/build/import.go:103`). At that site the only other
  failures are `no location for P@V` and a file read error, neither of which starts with a
  coordinate and `: `. A local replacement directory (`cue.mod/local-module.cue` `replaceWith`)
  whose module file does not parse reaches the same site and takes the same form, naming the
  replaced coordinate (`import failed: test.example/dep@v0.0.2: bogus: field not allowed`); graph
  expansion reports it as `cannot parse module file in replacement directory: ...`. Both are a
  dependency module file that does not parse, so both are `ResolutionModuleFileInvalid`, pinned
  by `load/malformed-replacement-module-file-direct` and `load/malformed-replacement-module-file`. The import-failed form with a file position
  (`import failed: <file>:3:8: cannot find package`) does not match it, because the position
  follows the version.
- The main module's malformed `module.cue` does not carry `cannot parse module file`, so the
  matched text comes only from dependency module files (fetched or local replacement), and the
  risk that the kind also covers the main module's file is dropped. The spec says "a dependency's
  module file, published or in a local replacement directory".
- A directory load with a corrupt dependency archive and an undeclared import in another file
  reports only the fetch failure: cue/load stops there. The constructed mixed text (both forms in
  one string) is pinned in `TestClassify_Resolution`, and stays `FetchOther`.
- `CUE errors.Error()` of a list is the first error's message only, so the text fallback reads the
  first error of a list. That is today's behaviour and this change keeps it.

## Risks / Trade-offs

- [A CUE bump rewords a form] → `classify_cue_test.go` fails on the bump. A CUE move reaches
  frontends only through a library release (workspace `RELEASING.md`), so the test runs first.
- [The direct-path module-file form is matched by its prefix] → `import failed: P@vX.Y.Z: ` is a
  shape cue/load and cue/build compose, not a message. A CUE bump that changes either fails
  `load/malformed-dependency-module-file-direct` first, and the text after the prefix (the
  author's module file content) is never read.
- [A caller compared `Classify(err)` with `err` to detect "unrecognised"] → that caller now sees
  a new value for these forms. Neither frontend does this; the cli checks for a `*FetchError`.
- [An error list with an unprovided import and a fetch failure reads as a fetch failure] → this is
  intended (D3). The fetch failure may be why the import is unprovided, and a retry may cure it.

## Verification

Run on 2026-10-05 against branch head `b08cf1c` (sections 1-3), base `origin/main` `88afdfb`, with
Go 1.26.5, an absolute private `TMPDIR` and the worktree's own copy of the main checkout's
`.cue-cache`.

- **Full suite (4.1).** `OPM_FLOW_TEST_FORCE=1 go test -race -count=1 -v ./opm/...`: exit 0, 24 packages ok, no data race and no skipped test. `TestParity_ShippedCatalog`, `TestParity_ShippedCatalogDiscriminated`, `TestParity_Probes`, `TestRender_InventoryParity` and the `TestFlow_*` tests ran and passed.
- **Consumer builds (4.2).** `GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> <worktree> <workdir>`,
  unpatched: cli `cd10f2d2` builds and vets; opm-operator `c303a460` builds and vets.
- **API diff (4.2).** `task api:diff` against `v1.0.0-beta.6`: "This change adds no incompatible
  change" (the 13 listed entries are inherited `opm/k8s/object` weights, not charged). `apidiff`
  run without `-incompatible` between `origin/main` and the work tree lists only compatible
  additions: `ResolutionError`, `ResolutionKind`, `ResolutionOther`, `ResolutionImportUnprovided`,
  `ResolutionImportAmbiguous` and `ResolutionModuleFileInvalid` in `opm/errors`.
- **cli replacement proof (4.3).** In a scratch clone of cli `cd10f2d2`:
  - `internal/publish/compat.go` `unprovidedImport` became
    `errors.As(liberrors.Classify(err), &re) && re.Kind == liberrors.ResolutionImportUnprovided`,
    with no text match;
  - `internal/config/platform.go` `platformBuildHint` replaced
    `strings.Contains(msg, "cannot find package")` and
    `strings.Contains(msg, "cannot expand module graph")` with "the chain holds a `*FetchError` or
    a `*ResolutionError`".

  `consumer-build.sh` built and vetted that patched clone (its `_test.go` files included). Under
  `GOWORK=<workdir>/go.work`:
  - `go test ./internal/publish/ -run 'TestLoadPublishedPackage_Pinned|TestUnprovidedImport_RegistryFailureIsNotAbsent|TestProbedPackageAbsent|TestCompatScan|TestGateCompat'`:
    pass, all 18 rows of `TestLoadPublishedPackage_Pinned` unchanged (both unprovided-import rows
    still read `absent`, every registry-failure row still `connectivity`);
  - `go test ./internal/config/ -run 'TestPlatformBuildHint|TestBuildPlatformModule'`: pass,
    all five `TestPlatformBuildHint_Pinned` rows unchanged;
  - `go test ./internal/cuemod/ -run 'TestIsConnectivityError|TestIsVersionNotHeld'`: pass;
  - the three packages whole (`./internal/publish/ ./internal/config/ ./internal/cuemod/`): pass.

  Nothing was committed to the cli, and the clones were removed. Two hint answers change with no
  pin row, and the cli follow-up pins both or decides each hint on purpose:
  - Direct-path module file: a directly imported dependency whose module file does not parse
    (`import failed: P@vX.Y.Z: ...`, published or local replacement) carried neither matched
    text, so it got the default hint; with the types it gets the "Pin a published build" hint.
  - Package qualifier: an import whose package name does not match
    (`import failed: <file>:3:8: cannot find package "test.example/dep": no files in package
    directory with package name "dep"`) matched `cannot find package` and got the "Pin a
    published build" hint; it is neither type ("Where resolution ends"), so it now gets the
    default hint. The other `cannot find package` forms listed there move the same way.

### Frontend follow-ups (4.4)

- cli: `compat.go` `unprovidedImport` moves onto `Kind == ResolutionImportUnprovided`, and the two
  import matches of `platform.go` `platformBuildHint` (`cannot find package`,
  `cannot expand module graph`) move onto "a `*FetchError` or a `*ResolutionError` in the chain",
  both proved above. This drops both text matches 0021's delivery log records. Required in the
  same change: the `#registry` match of `platform.go` moves off the message text onto the CUE
  error path (the strict reading leaves no text match in the cli). It is an evaluation error, so
  it needs no library work. The change pins the two hint rows above (direct-path module file,
  package qualifier) or decides each hint on purpose.
- opm-operator: `*ResolutionError` names the stall reason and condition message only. It is not
  added to `isTerminalCause`, so any `*FetchError` in a joined chain still retries.
