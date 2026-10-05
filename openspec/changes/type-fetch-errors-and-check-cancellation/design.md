## Context

Line numbers are at `origin/main` `ca7c56b`, after wave 2 rounds 1 and 2 of the beta.1 kernel plan.

**Where a fetch or resolution failure leaves the library today.** Every site wraps it with a
plain `fmt.Errorf`:

| Site | Call | Wrap |
| --- | --- | --- |
| `opm/internal/loader/registry.go:94-96` | `modconfig.NewRegistry` | `building module registry resolver: %w` |
| `opm/internal/loader/registry.go:101-103` | `reg.Fetch(ctx, mv)` | `fetching %s %s: %w` |
| `opm/internal/loader/load.go:108-109` | `load.Instances` → `instances[0].Err` | `loading %s package from %s (%s): %w` |
| `opm/internal/renderstage/stage.go:215-216` | the render module load | `loading the render module: %w` (then `building render module: %w`, `opm/kernel/render.go:411`) |
| `opm/schema/loader.go:162-163` | the schema `OCILoader` load | `schema OCILoader: loading %q: %w` |
| `opm/helper/platformmodule/closure.go:101-103` | `ModFileSource.ModFile` | `resolving dependency %s: %w` |

`loader.LoadDir` is the one build step behind every directory acquire verb, the registry verbs
(through `FetchArtifact`) and the synth build (`opm/internal/synth/instance.go:191`). Classifying
there covers all of them.

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
    FetchNotFound                      // the module, version or package is absent
    FetchUnauthorized                  // the registry refused the credentials (401) or the access (403)
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

A `FetchError` with a nil `Err` is a caller bug. `Error()` then returns the kind's name, so it
never panics.

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

`context.DeadlineExceeded` satisfies `net.Error`, so `IsConnectivityError` already treats it as
connectivity, and `FetchUnreachable` keeps that. A 5xx answer is `FetchOther`. That is what the
probe says today (an answer, not connectivity), even though it is transient. The cli change pins
each row with a test. This library change pins the kind for each failure form (D3, section 3).

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
   anything else → `FetchOther`. `Status` is the code;
3. `modregistry.ErrNotFound`, `ociregistry.ErrNameUnknown`, `ErrManifestUnknown` or
   `ErrBlobUnknown` → `FetchNotFound`;
4. `ociregistry.ErrUnauthorized` or `ErrDenied` → `FetchUnauthorized`;
5. `net.Error` (which covers `*url.Error`, `*net.OpError` and `*net.DNSError`) → `FetchUnreachable`.

`classifyText` is the one text fallback in the library. It matches only the forms that section 1
pins against CUE v0.17.1, in this order, so the most specific form wins:

1. `cannot do HTTP request` → `FetchUnreachable` (the OCI client's transport prefix);
2. the 401 and 403 forms the spike records → `FetchUnauthorized`;
3. `cannot find module providing package` or `module not found`, and the forms the spike records
   for a 404 → `FetchNotFound`;
4. the 5xx status prefix, if the spike shows that a flattened 5xx keeps it → `FetchOther` with that
   `Status`;
5. `cannot fetch ` or `cannot expand module graph` → `FetchOther`.

A test in `opm/errors` (`classify_cue_test.go`) produces each form through the embedded CUE and
asserts the kind. It fails when a CUE bump changes a form, which is the signal to update the
fallback. A comment at the fallback names the test.

**Never transient by accident.** An error that matches neither branch is returned as it is, so a
syntax error, a conflict or a missing field from `cue/load` stays a plain error. `FetchNotFound`
includes "cannot find module providing package", which is also what a missing dependency in an
author's `cue.mod/module.cue` reports. That error is a fetch failure of kind not-found, and it is
never transient. Whether the operator retries it is the operator's policy.

### D4. Where the library applies it, and where it does not

`Classify` wraps the cause at the sites in the Context table, inside the existing `%w`:

- `FetchArtifact`: the `reg.Fetch` error, with `Coordinate` set to the canonical `mv.String()`
  (D5), and the `modconfig.NewRegistry` error (a bad `CUE_REGISTRY` stays unrecognised and
  unchanged; an auth-config failure that is a `net.Error` does not occur there);
- `loader.LoadDir`: `instances[0].Err` only;
- `renderstage.Build`: `instances[0].Err` only;
- `schema.OCILoader.Load`: `instances[0].Err` only;
- `platformmodule.Closure`: the `ModFile` error.

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
| `AcquireModuleFromDir`, `AcquireCatalogFromDir`, `AcquirePlatformFromDir` | entry; inside `acquireDir`, after `dirSource` reads the tree and before `LoadDir` |
| `AcquireInstanceFromDir` | entry; after the tree read (both paths), and in `loadInstanceWithValues` after `mergeSources` and before `LoadDir`; after the build, before `checkInstanceValues`; before `processInstance` |
| `SynthesizeInstance` | entry (after the argument checks); after `resolveCoreVersion`; after `mergeSources`; after `synth.Instance` (before the attribution rebuild); before `processInstance` |
| `FetchArtifact` (both registry verbs) | after `reg.Fetch`, before staging and `LoadDir` |
| `Render` | after `renderstage.Build`, before decoding (new; the two existing checks stay) |

`acquireDir` gains a `ctx` parameter. `ValidateConfigDetailed` is not one of the five verbs and is
unchanged.

The godoc gets one paragraph in `opm/kernel/doc.go` (a "# Cancellation" section), and one
sentence on each of the five verbs, `AcquireModuleFromRegistry`, `AcquireCatalogFromRegistry` and
`Render`: "ctx is checked at entry and between stages; a running load or build is not interrupted,
so cancellation lands at the next stage boundary." The `Surface` list and the three code
examples stay as they are (`kernel-runtime` "The kernel package doc renders its verb list and
examples").

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
by a person. The operator's retry policy still sees every `*FetchError` and can retry more widely.

### Spike before code

**Context**: Three forms are unverified: whether the `%w` at `modpkgload/import.go:212` survives
into `instances[0].Err`, what a flattened 401/403 and 5xx look like, and whether a load against an
unreachable registry reports `cannot do HTTP request`.
**Decision**: Section 1 drives each failure through the real loaders against a local status
registry and pins what it sees. The findings are written under "Spike findings" below before
section 2 starts.

## Spike findings

To be filled in by section 1.

## Risks / Trade-offs

- [A CUE bump changes a flattened form] → `classify_cue_test.go` fails on the bump, and the workspace
  `RELEASING.md` rule moves `cuelang.org/go` only through a library release, so the test runs
  before any frontend sees the new CUE.
- [An extra link in the chain breaks a caller that type-switches on the direct cause] → `errors.As`
  and `cueerrors.Errors` (which uses `As`) still find it. A test asserts `cueerrors.Errors` on a
  classified load error returns the same positions.
- [The operator retries an author's missing dependency (`FetchNotFound`) forever] → It is not
  transient. Whether the operator retries any `*FetchError` with backoff is its own policy, and the
  operator change decides it.
- [A pre-cancelled context now fails a call that used to complete] → Both frontends pass live
  contexts. The behaviour is the documented contract of a context parameter.
- [`opm/errors` grows a CUE dependency] → Both modules are already required. D7 records why.

## Verification

To be filled in by section 5.
