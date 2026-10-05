## Context

Line numbers are at `origin/main` `ca7c56b`, after wave 2 rounds 1 and 2 of the beta.1 kernel plan.

**Where a fetch or resolution failure leaves the library today.** Every site wraps it with a
plain `fmt.Errorf`:

| Site | Call | Wrap |
| --- | --- | --- |
| `opm/internal/loader/registry.go:101-103` | `reg.Fetch(ctx, mv)` | `fetching %s %s: %w` |
| `opm/internal/loader/load.go:108-109` | `load.Instances` → `instances[0].Err` | `loading %s package from %s (%s): %w` |
| `opm/kernel/source_loader.go:110-111` | `load.Instances` → `instances[0].Err` for a file-backed values source | returned raw (no wrap) |
| `opm/internal/renderstage/stage.go:215-216` | the render module load | `loading the render module: %w` (then `building render module: %w`, `opm/kernel/render.go:411`) |
| `opm/schema/loader.go:162-163` | the schema `OCILoader` load | `schema OCILoader: loading %q: %w` |
| `opm/helper/platformmodule/closure.go:101-103` | `ModFileSource.ModFile` | `resolving dependency %s: %w` |

`loader.LoadDir` is the one build step behind every directory acquire verb, the registry verbs
(through `FetchArtifact`) and the synth build (`opm/internal/synth/instance.go:191`). Classifying
there covers all of them. `compileSource` (`opm/kernel/source_loader.go`) is the other `cue/load`
call a kernel verb makes: it loads every file-backed values source, and a values file may import a
registry module through `WithRegistry` (`opm/kernel/doc.go` Surface). `AcquireInstanceFromDir` and
`SynthesizeInstance` reach it through `mergeSources`, and `ValidateConfigDetailed` reaches it
directly. The two `modconfig.NewRegistry` calls (`registry.go:94-96`,
`platformmodule/closure.go:43-50`) only parse configuration and are not fetch sites (D4).

**What survives from CUE v0.17.1.** A direct `reg.Fetch` keeps a typed chain:
`mod/modregistry/client.go:48` exports `ErrNotFound` and wraps it with `%w` (`:243-250`). The
OCI client wraps a transport failure as `cannot do HTTP request: %w` around a `net.Error`
(`ociregistry/ociclient/client.go:308`). A registry answer becomes an `ociregistry.HTTPError`
carrying `StatusCode()` (`ociclient/error.go:50`), and `ociregistry.WireError.Is` matches the error
codes (`ErrNameUnknown`, `ErrManifestUnknown`, `ErrUnauthorized`, `ErrDenied`, ...). Through
`cue/load` the cause is mostly flattened: `internal/mod/modpkgload/import.go:164` formats
`cannot expand module graph: %v`, `:415` returns the text `cannot find module providing package P`,
and only `:212` (`cannot fetch %v: %w`) wraps. Whether that wrap survives into `instances[0].Err`,
which is a CUE error list, is not verified. Section 1 measures it.

**The consumers' probes** at cli `origin/main`:

- `internal/cuemod/connectivity.go:18-27`: `IsConnectivityError` is true for a `net.Error` in the
  chain or the text `cannot do HTTP request`. A registry that answered, even with 401 or 5xx, is
  not a connectivity failure there.
- `internal/publish/check.go:226-231`: after `reg.Fetch`, the text `not found` means
  `ErrNotPublished`, and anything else is a `*ConnectivityError`.
- `internal/publish/compat.go:229-236`: after `load.Instances`, the text
  `cannot find module providing package` means "absent", and anything else is a
  `*ConnectivityError`.

**Cancellation today.** `Render` checks `ctx.Err()` after input validation (`render.go:362`) and
after staging (`:404`), and returns the bare context error. The other five verbs discard the
context (`acquire.go:96, :228, :263, :320`, `synth.go:109`). The two registry verbs pass it to
`reg.Fetch` only. `cue/load` takes no context, so nothing can reach inside a load (0009:D9).

## Goals / Non-Goals

**Goals:**

- A caller tells a fetch or resolution failure by type: `errors.Is(err, oerrors.ErrTransient)`
  for "retrying may help", and `errors.As(err, &fe)` with `fe.Kind` for which kind of failure.
  No message text is needed (0021:D8:R12).
- The text matching the flattened CUE forms need lives in one library function, pinned against the
  embedded CUE version, so the frontends drop their probes.
- An author defect is never marked transient, and an unrecognised error is never wrapped.
- Every existing message, and every existing `errors.Is` on a sentinel or a CUE error, stays as it
  is.
- The five verbs check the context at entry and between stages. `FetchArtifact` checks it after
  the fetch, and `Render` checks it after the build.

**Non-Goals:**

- Frontend migration (the cli probes, the operator retry policy).
- Cancellation inside a stage (0009:D9).
- Retrying inside the library. The caller owns retries.

## Decisions

### D1. The API

```go
// opm/errors/fetch.go
var ErrTransient = errors.New("transient registry failure")

type FetchKind int

const (
    FetchOther        FetchKind = iota // a fetch or resolution failure of no narrower kind
    FetchNotFound                      // the module, version or package is absent (a 403 on a tag lookup too, see below)
    FetchUnauthorized                  // the registry refused the credentials (401), or a 403 whose HTTPError survives
    FetchUnreachable                   // no HTTP response: refused, DNS, TLS, timeout, deadline
)

func (k FetchKind) String() string // "other", "not found", "unauthorized", "unreachable"

type FetchError struct {
    Kind       FetchKind
    Coordinate string // "path@vX.Y.Z" when the failing site knows it, else ""
    Status     int    // HTTP status the registry answered with, 0 when none or unknown
    Err        error  // the cause, unchanged
}

func (e *FetchError) Error() string        // e.Err.Error(), unchanged
func (e *FetchError) Unwrap() error        // e.Err
func (e *FetchError) Is(target error) bool // target == ErrTransient && e.Transient()
func (e *FetchError) Transient() bool      // Kind == FetchUnreachable || Status >= 500

// opm/errors/classify.go
func Classify(err error) error
```

`Error()` returns the cause's text, so a site that wraps `Classify(err)` prints exactly what it
printed before. `Status` is not in the owner's wording. It is added because the transient rule
needs the 5xx answer as data. A `FetchError` built by hand states its own status, and one that
`Classify` builds records the status it found, from the typed chain or from the text.

A `FetchError` with a nil `Err` is a caller bug. `Error()` then returns `registry fetch failed:`
and the kind's name, so it never panics.

CUE's registry client decides one case before `Classify` sees it. In CUE v0.17.1
`modregistry.Client.GetModule` (`mod/modregistry/client.go:248-250`) runs `isNotExist`
(`:566-583`), which treats any 403 answer, like any 404, as not-exist and returns
`module %v: %w` around `modregistry.ErrNotFound`. The `HTTPError` leaves the chain there. So a 403
on a tag lookup classifies as `FetchNotFound`, which is also what the cli's `check.go` probe does
today (its text `not found` gives `ErrNotPublished`). `FetchUnauthorized` covers a 401, and a 403
only where an `HTTPError` with that status survives in the chain.

### D2. Which kinds are transient, and how each cli site keeps its exit code

`ErrTransient` is network-level only. It holds when the registry could not be reached (no HTTP
response, `FetchUnreachable`, which includes `context.DeadlineExceeded`), or when the registry
answered with a 5xx status. `FetchNotFound` and `FetchUnauthorized` are never transient: retrying
an absent version or a refused credential is the stall the operator defect was about. A 429 answer
is `FetchOther` and not transient, because the rule names network-level failures only. The
frontend that wants to retry on 429 reads `Status`.

The cli moves onto the kinds, not onto `ErrTransient`, so each site's semantics stay as they are:

| cli site | Today | With this change |
| --- | --- | --- |
| `IsConnectivityError` | `net.Error` in chain, or `cannot do HTTP request` | `Kind == FetchUnreachable` |
| `check.go` after `reg.Fetch` | text `not found` → `ErrNotPublished`, else connectivity | `Kind == FetchNotFound` → `ErrNotPublished`, else connectivity |
| `compat.go` after `load.Instances` | text `cannot find module providing package` → absent, else connectivity | `Kind == FetchNotFound` → absent, else connectivity |

`compat.go` loads a standalone `importPath@version`, so its "absent" form names an exact version
and stays `FetchNotFound` (D3); the parity holds for that form. One edge moves, and it is the cli
change's to map, not this one's: a dependency of the probed module that the registry answers with
404 or 403 (`cannot fetch D@V: module D@V: module not found`) is a `*ConnectivityError` today
(exit 3), and under `Kind == FetchNotFound` it would read as "absent" (found=false). The cli change
either accepts that with a pinning test, or keeps connectivity when the not-found failure is not
the probed coordinate itself.

An import no registry interaction failed is not classified at all (D3): a package missing from the
main module's own path, an import the module never declared, or a package missing from a declared
dependency that was fetched. These are author defects. No frontend retries them, because no
`*FetchError` reaches it.

`context.DeadlineExceeded` satisfies `net.Error`, so `IsConnectivityError` already treats it as
connectivity, and `FetchUnreachable` keeps that. A 5xx answer is `FetchOther`. That is what the
probe says today (an answer, not connectivity), even though it is transient. The cli change pins
each row with a test. This library change pins the kind for each failure form (D3, section 3).

One case is not exact. Today a cancelled request counts as connectivity: `*url.Error{Err:
context.Canceled}` satisfies `net.Error` (`connectivity.go:22-25`). `Classify` passes
`context.Canceled` through unchanged (D3), so `Kind == FetchUnreachable` is false for a Ctrl-C'd
tidy or fetch. The cli row that adopts `IsConnectivityError` keeps parity with
`Kind == FetchUnreachable || errors.Is(err, context.Canceled)`, or records the change for
cancellation and accepts it.

### D3. How `Classify` decides

```go
func Classify(err error) error {
    if err == nil { return nil }
    var fe *FetchError
    if errors.As(err, &fe) { return err }                  // already classified: idempotent
    if errors.Is(err, context.Canceled) { return err }     // the caller's cancellation, not a fetch failure
    if kind, status, ok := classifyTyped(err); ok { return &FetchError{Kind: kind, Status: status, Err: err} }
    if kind, status, ok := classifyText(err.Error()); ok { return &FetchError{Kind: kind, Status: status, Err: err} }
    return err                                              // unrecognised: unchanged, never transient
}
```

`classifyTyped` reads, in this order:

1. `context.DeadlineExceeded` → `FetchUnreachable`;
2. `ociregistry.HTTPError` in the chain: 404 → `FetchNotFound`, 401 or 403 → `FetchUnauthorized`,
   anything else → `FetchOther`. `Status` is the code. A 403 on a tag lookup never reaches this
   step with its `HTTPError`: CUE's registry client has already turned it into
   `modregistry.ErrNotFound` (D1), so step 3 makes it `FetchNotFound`;
3. `modregistry.ErrNotFound`, `ociregistry.ErrNameUnknown`, `ErrManifestUnknown` or
   `ErrBlobUnknown` → `FetchNotFound`;
4. `ociregistry.ErrUnauthorized` or `ErrDenied` → `FetchUnauthorized`;
5. `net.Error` (which covers `*url.Error`, `*net.OpError` and `*net.DNSError`) → `FetchUnreachable`.

`classifyText` is the one text fallback in the library. It matches only the forms that section 1
pins against CUE v0.17.1 (Spike findings), in this order, so the most specific form wins:

1. `cannot do HTTP request` → `FetchUnreachable` (the OCI client's transport prefix);
2. an HTTP status as the OCI client writes it, `: <code> <status text>: ` with the status text
   `net/http` gives that code (`: 401 Unauthorized: `, `: 503 Service Unavailable: `). 401 or 403
   → `FetchUnauthorized`, 404 → `FetchNotFound`, any other code from 400 up → `FetchOther`.
   `Status` is the code, so a flattened 5xx stays transient;
3. `module not found` → `FetchNotFound` (a flattened 403 or 404 tag lookup reads `module not
   found`), and `cannot find module providing package P@vX.Y.Z` → `FetchNotFound`. That second form
   names an exact version, which only a standalone `path@version` load produces (the schema
   loader's, and the cli's `compat.go` probe): the registry was asked for that version and does
   not provide the package. In a directory load the same words carry an import path, which holds
   at most a major version, and report an author defect no registry interaction failed (an own-path
   package that does not exist, an undeclared import, a package missing from a fetched dependency).
   That form is never matched;
4. `cannot fetch ` → `FetchOther` (what is left is a fetch with an unrecognised cause, such as a
   published archive that does not unzip).

`cannot expand module graph` is not matched on its own. `modpkgload/import.go:164` formats it with
`%v` around any requirements-graph error, a malformed `module.cue` in a published dependency
included, which is a defect rather than a fetch. It classifies only through the fetch form it
carries (steps 1-4 match anywhere in the text), and the spike records a malformed-dependency case
that stays unclassified.

The tidy forms the cli's `IsConnectivityError` reads come from `cmd/cue`'s own printer, not from
`cue/load`. Driving `cmd/cue/cmd` in a library test would add its command-line dependencies to
`go.mod`, so the library pins only the `cue/load` and `reg.Fetch` forms. The spike records the
tidy forms once (by running the same-version `cue mod tidy`), and the cli's behavioural tests
`TestIsConnectivityError_UnreachableRegistry` and `TestIsConnectivityError_RegistryAnswered` stay
as the pin of the tidy forms when the cli moves onto `Classify`.

A test in `opm/errors` (`classify_cue_test.go`) produces each form through the embedded CUE and
asserts the kind. It fails when a CUE bump changes a form, which is the signal to update the
fallback. A comment at the fallback names the test.

**Never transient by accident.** An error that matches neither branch is returned as it is, so a
syntax error, a conflict or a missing field from `cue/load` stays a plain error. `Classify`
recognises only a failed registry interaction, so an import that no module provides in a directory
load (an own-path package that does not exist, or an undeclared dependency) stays a plain error too;
`classify_cue_test.go` and `TestFetchClassify_UnresolvableImportStaysPlain` pin it. A frontend that
retries every `*FetchError` therefore never retries an author's import typo.

### D4. Where the library applies it, and where it does not

`Classify` wraps the cause at the sites in the Context table, inside the existing `%w`:

- `FetchArtifact`: the `reg.Fetch` error, with `Coordinate` set to the canonical `mv.String()`
  (D5);
- `loader.LoadDir`: `instances[0].Err` only;
- `compileSource` (`opm/kernel/source_loader.go`): `instances[0].Err` of a file-backed values
  source only, never the `v.Err()` after its build;
- `renderstage.Build`: `instances[0].Err` only;
- `schema.OCILoader.Load`: `instances[0].Err` only;
- `platformmodule.Closure`: the `ModFile` error.

Neither `modconfig.NewRegistry` call (`FetchArtifact`, `platformmodule.NewRegistry`) is
classified. It parses `CUE_REGISTRY` and the auth configuration and dials nothing, so its error is
a configuration error `Classify` would leave unchanged anyway.

It is never applied to an evaluation error (`val.Err()` after `BuildInstance`, the shape gate,
`processInstance`). An evaluation error can carry an author's own strings in its message (a
conflict quotes both values), so the text fallback could misfire on it, and no evaluation error
is a fetch failure.

`Classify` returns the same `error` when it does not recognise one, so each site changes by one
call: `fmt.Errorf("fetching %s %s: %w", spec.Label, mv, withCoordinate(oerrors.Classify(err), mv))`.

### D5. `Coordinate` is set where the site knows it

Only `FetchArtifact` knows the exact coordinate. It sets `Coordinate` on the `*FetchError` that
`Classify` returned. The load sites leave it empty: a `cue/load` error may concern any module in
the dependency graph, and the CUE text already names it.

### D6. Cancellation points

Each verb returns the bare `ctx.Err()`, as `Render` already does, so
`errors.Is(err, context.Canceled)` and `context.DeadlineExceeded` work without unwrapping a frame.
The entry check runs after the verb's argument checks, mirroring `Render`: an argument error is a
caller bug that does not depend on time, and both checks do no work.

| Verb | Checks |
| --- | --- |
| `AcquireModuleFromDir`, `AcquireCatalogFromDir`, `AcquirePlatformFromDir` | entry; inside `acquireDir`, after `dirSource` reads the tree and before `LoadDir`, and after `LoadDir` builds the package |
| `AcquireInstanceFromDir` | entry; after the tree read (both paths), and in `loadInstanceWithValues` after `mergeSources` and before `LoadDir`; after the build, before `checkInstanceValues`; before `processInstance` |
| `SynthesizeInstance` | entry (after the argument checks); after `resolveCoreVersion`; after `mergeSources`; after `synth.Instance` (before the attribution rebuild); before `processInstance` |
| `FetchArtifact` (both registry verbs) | after `reg.Fetch`, before staging and `LoadDir`; after `LoadDir` builds the package |
| `Render` | after `renderstage.Build`, before decoding (new; the two existing checks stay) |

`TestCancel_EveryStageCheck` cancels each verb at its first, second, ... check in turn, through a
context whose `Err` turns non-nil after n calls, and pins the number of checks each verb makes, so
removing any one check fails the test.

`acquireDir` gains a `ctx` parameter. `ValidateConfigDetailed` is not one of the five verbs and is
unchanged.

The contract lives once, in a "# Cancellation" section of `opm/kernel/doc.go`: ctx is checked at
entry and between stages; a running load or build is not interrupted, so cancellation lands at the
next stage boundary. The verbs carry no copy of it; a runtime contract has one home, the owning
package's godoc. The `Surface` list and the three code examples stay as they are
(`kernel-runtime` "The kernel package doc renders its verb list and examples").

A registry verb checks after `reg.Fetch` returns, but a cancellation that `reg.Fetch` itself
observes on a cold cache comes back wrapped (`fetching %s %s: ... context canceled`). Only the
verb's own checks return the bare context error; a cancellation seen inside the fetch still
satisfies `errors.Is(err, context.Canceled)`.

### D7. `opm/errors` imports CUE's registry packages

The owner named `opmerrors.Classify`, so `Classify` lives in `opm/errors`. Its typed branch needs
`cuelang.org/go/mod/modregistry` and `cuelabs.dev/go/oci/ociregistry`, which are the package's
first imports outside the standard library. Both modules are already in the library's and both
frontends' module graphs, at the version `cuelang.org/go` selects. `ociregistry` moves from
`// indirect` to a direct require at the same version. A separate package was considered and
rejected: it would give the sentinel and the classifier two homes, and the sentinels already live
in `opm/errors` for the same reason (`sentinels.go:5-13`). Neither import reaches a Kubernetes
package or the helper tier, so no depguard rule changes.

## Research & Decisions

### Typed chain versus text

**Context**: 0021:D8:R12 forbids message matching in the caller. CUE flattens most `cue/load`
causes into strings.
**Explored**: CUE v0.17.1 `mod/modregistry/client.go`, `internal/mod/modpkgload/import.go`,
ociregistry `error.go` and `ociclient/client.go`. The cli probes and their comments, which record
what was measured against CUE v0.17.
**Decision**: Read the typed chain first and fall back to text in one function, pinned by a test
that drives the embedded CUE.
**Rationale**: The typed chain is exact where it exists (the direct fetch). The text is the only
signal after `cue/load`, and keeping it in the library is what lets the frontends stop matching
text. The pin turns a silent drift on a CUE bump into a failing test.

### Transient set

**Context**: Over-classifying hides an author defect behind endless retries. Under-classifying
stalls on a blip.
**Explored**: The cli's `IsConnectivityError`, which does not treat a 401 as connectivity, and the
two publish probes, which treat every answer other than not-found as connectivity (D2).
**Decision**: `ErrTransient` holds for no HTTP response, an expired deadline and a 5xx answer.
Not for 404, 401/403, 429 or any unrecognised error.
**Rationale**: These are the cases where the same request can succeed later with nothing changed
by a person (on the same Kernel only when no cache memoized the failure; see Risks). The operator's retry policy still sees every `*FetchError` and can retry more widely.

### Spike before code

**Context**: Three forms are unverified: whether the `%w` at `modpkgload/import.go:212` survives
into `instances[0].Err`, what a flattened 401/403 and 5xx look like, and whether a load against an
unreachable registry reports `cannot do HTTP request`. A deadline that expires during a fetch
cannot be driven through `cue/load` (it takes no context), so the deadline case is an
already-expired deadline on `FetchArtifact` only.
**Decision**: Section 1 drives each failure through the real loaders against a local status
registry and pins what it sees. The findings are written under "Spike findings" below before
section 2 starts.

## Spike findings

Measured against CUE v0.17.1 and pinned by `opm/errors/classify_cue_test.go`
(`TestCUEFailureForms`). Each failure was driven through `loader.FetchArtifact` (a direct
`reg.Fetch`) and through `loader.LoadDir` on a directory module requiring the failing dependency,
against a local registry only and a fresh module cache.

| Failure | `FetchArtifact` (typed chain) | `LoadDir` (text only) |
| --- | --- | --- |
| version absent | `modregistry.ErrNotFound`; `module P@V: module not found` | `cannot fetch P@V: module P@V: module not found` |
| import of an undeclared module, of an own-path package that does not exist, or of a package missing from a fetched dependency | | `cannot find module providing package P` (P carries at most a major version; unclassified) |
| standalone `P@vX.Y.Z` load: version absent, package absent in it, or a 404 registry | | `cannot find module providing package P@vX.Y.Z` (`FetchNotFound`) |
| unreachable (refused) | `net.Error`; `cannot do HTTP request: ... connection refused` | `cannot fetch P@V: module P@V: cannot do HTTP request: ...` |
| 401 | `HTTPError` 401, `ErrUnauthorized`; `401 Unauthorized: unauthorized: ...` | `...: 401 Unauthorized: ...` |
| 403 | `modregistry.ErrNotFound`, no `HTTPError`; `module not found` | `...: module not found` |
| 404 | `modregistry.ErrNotFound`; `module not found` | `...: module not found` |
| 429 | `HTTPError` 429; `429 Too Many Requests: ...` | `...: 429 Too Many Requests: ...` |
| 500, 503 | `HTTPError` 500/503; `503 Service Unavailable: non-JSON error response ...` | `...: 503 Service Unavailable: ...` |
| expired deadline | `net.Error` and `context.DeadlineExceeded`; `cannot do HTTP request: ... context deadline exceeded` | not drivable (no context) |
| cancelled context | `net.Error` and `context.Canceled`; `cannot do HTTP request: ... context canceled` | not drivable |
| dependency module file does not parse | | `cannot expand module graph: P@V: cannot parse module file from P@V: bogus: field not allowed` |
| published archive does not unzip | | `cannot fetch P@V: unzip ...: zip: not a valid zip file` |

The schema `OCILoader` loads a standalone package by `path@version`, and that path keeps the typed
chain: an unreachable registry gives a `net.Error` behind `cannot fetch ...: cannot do HTTP
request`, and a 503 gives an `HTTPError` with status 503.

The answers to the three open questions:

- The `%w` at `modpkgload/import.go:212` does not survive into `instances[0].Err` of a directory
  load: the instance error carries the text only. It does survive the standalone-package load the
  schema loader makes.
- A flattened 401, 429 and 5xx keep the OCI client's `<code> <status text>: ` form, so the status
  can be read from the text. A 403 never reaches the text as a 403: CUE's registry client turns it
  into `module not found` before anything is flattened. A 401 is therefore always told apart from a
  404.
- A load against an unreachable registry says `cannot do HTTP request`, through both paths.

The `cue mod tidy` forms of the same CUE version (run once with `cue` v0.17.1, not in the suite):
an unreachable registry gives `failed to resolve "P": module M: cannot do HTTP request: ...`, a
registry answering 404 to everything gives `cannot find module providing package P`, and 401 and
503 give `module M: 401 Unauthorized: ...` and `module M: 503 Service Unavailable: ...`. The text
fallback matches the unreachable, 401 and 503 forms. The 404 form names an import path with no
exact version, so it stays unclassified; `IsConnectivityError` reads only the unreachable form, so
nothing in the cli depends on it. The cli's own tidy tests stay the pin of these forms (D3).

## Risks / Trade-offs

- [A CUE bump changes a flattened form] → `classify_cue_test.go` fails on the bump, and the workspace
  `RELEASING.md` rule moves `cuelang.org/go` only through a library release, so the test runs
  before any frontend sees the new CUE.
- [An extra link in the chain breaks a caller that type-switches on the direct cause] → `errors.As`
  and `cueerrors.Errors` (which uses `As`) still find it. A test asserts `cueerrors.Errors` on a
  classified load error returns the same positions.
- [The operator retries an author's missing dependency forever] → An import no registry
  interaction failed (an own-path package that does not exist, an undeclared dependency) is not
  classified (D3), so an operator that retries any `*FetchError` with backoff never retries it.
- [A pre-cancelled context now fails a call that used to complete] → Both frontends pass live
  contexts. The behaviour is the documented contract of a context parameter.
- [`opm/errors` grows a CUE dependency] → Both modules are already required. D7 records why.
- [`schema.Cache` memoizes a transient schema-load error] → `Cache` caches errors too and never
  retries a load (`opm/schema/cache.go:17-23`). An `OCILoader` failure that is now marked
  `ErrTransient` stays cached in the Kernel's cache, so retrying on the same Kernel cannot succeed
  (a bare-major `resolveCoreVersion`, the cli's `SchemaCache().Get()`, the operator's startup
  check). This change keeps `Cache`'s behaviour and says it in the `ErrTransient` and `Cache`
  godoc: a memoized error keeps its classification, and a retry needs a fresh `Cache` (a fresh
  Kernel). Whether `Cache` should stop memoizing transient errors is an owner question, left open.

## Verification

Recorded on 2026-10-05 against `origin/main` `ca7c56b`.

- Full suite (task 5.1): `OPM_FLOW_TEST_FORCE=1 go test -race -count=1 ./opm/...` passes in every
  package. `TestParity_ShippedCatalog`, `TestParity_ShippedCatalogDiscriminated`,
  `TestParity_Probes`, `TestRender_InventoryParity`, `TestFlow_WebApp_OnOpmPlatform` and both
  `TestFlow_ImportedModule_*` ran and passed; no test skipped.
- Consumer builds (task 5.2): `GOTOOLCHAIN=local bash .tasks/consumer-build.sh` builds and vets
  fresh clones of cli `main` at `bd4d1a7` and opm-operator `main` at `53ccaab` against this tree.
- `task api:diff` against `v1.0.0-beta.4`: no incompatible change; the only listed line is the
  allowed core-pin value change the base already carries. The new `opm/errors` symbols are
  additions, which the check does not list.
- `task check` is green after every section.
- After the code review fixes (post-build checks, unresolvable imports left unclassified), the
  full suite, `task check`, `task api:diff` and both consumer builds (cli `bd4d1a7`, opm-operator
  `53ccaab`) ran green again on 2026-10-05.

**The cli sites and the kind each moves onto** (D2), for the cli change that adopts `Classify` on
the first library release carrying this one:

| cli site | Moves onto |
| --- | --- |
| `internal/cuemod/connectivity.go` `IsConnectivityError` (after `Tidy`) | `Classify(err)`, then `Kind == FetchUnreachable` (or that, or `errors.Is(err, context.Canceled)`, to keep cancellation counting as connectivity) |
| `internal/publish/check.go` after `reg.Fetch` | `Kind == FetchNotFound` gives `ErrNotPublished`, anything else connectivity (a 403 is `FetchNotFound`, as the text probe reads it today) |
| `internal/publish/compat.go` after `load.Instances` | `Kind == FetchNotFound` means absent, anything else connectivity; a dependency answered 404 or 403 moves from connectivity to absent, and the cli change maps that edge (D2) |
| `internal/config/platform.go` hint probe (optional, outside the owner's three) | `Kind == FetchNotFound`; `cannot expand module graph` around a malformed dependency stays unclassified |

The tidy forms are pinned only by the cli's own `TestIsConnectivityError_UnreachableRegistry` and
`TestIsConnectivityError_RegistryAnswered`, which stay (D3).

**Callers of the five verbs** (task 5.3). Every call passes the caller's own context parameter:

- cli: `internal/cmd/module/eval.go:74` (`runModuleEval`; a nil ctx is replaced by
  `context.Background()`), `internal/cmdutil/instance_arg.go:105`, `internal/cmdutil/path_guard.go:55`,
  `internal/config/platform.go:75`, `internal/publish/kernel_gate.go:31`,
  `internal/scaffold/repair.go:249`, `internal/scaffold/scaffold.go:283`,
  `internal/workflow/render/module.go:69` and `:182`, `internal/workflow/render/render.go:73`,
  `internal/workflow/render/env.go:61`, and the integration programs
  `tests/integration/platform-build/main.go:124` and `tests/integration/render-parity/main.go:135`
  and `:151`.
- opm-operator: `internal/controller/platform_controller.go:259` (`Reconcile`),
  `internal/render/kernel_module_renderer.go:176`, `internal/render/kernel_package_renderer.go:106`.

One behaviour to note for the cli: `cmdutil.ModulePackageError` (`path_guard.go:55`) acquires the
directory as a module after an instance acquire failed with `ErrWrongKind`, to word the refusal.
If the command's context is already cancelled, that acquire now returns `context.Canceled` at once,
so the function returns nil and the caller reports the instance acquire's own error. The command
was being cancelled anyway.
