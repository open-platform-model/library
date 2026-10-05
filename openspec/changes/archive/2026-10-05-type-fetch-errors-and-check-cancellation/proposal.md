## Why

A frontend cannot tell why a registry fetch or a dependency resolution failed without reading the
error's text. The library wraps every such failure in a plain `fmt.Errorf`
(`opm/internal/loader/registry.go:101-103`, `opm/internal/loader/load.go:108-109`,
`opm/internal/renderstage/stage.go:215-216`, `opm/schema/loader.go:162-163`,
`opm/helper/platformmodule/closure.go:101-103`), and `cue/load` flattens the cause of a failed
import into a string before the library sees it. So the cli carries three text probes
(`internal/cuemod/connectivity.go` "cannot do HTTP request", `internal/publish/check.go`
"not found", `internal/publish/compat.go` "cannot find module providing package"), and the operator
cannot separate a registry blip from an author defect. 0021:D8:R12 makes this a GA exit
criterion: a caller tells each fetch or resolution failure by type, transient or not, and which
kind of fetch failed, without matching on message text.

Five kernel verbs also discard their context (`AcquireModuleFromDir`, `AcquireCatalogFromDir`,
`AcquirePlatformFromDir`, `AcquireInstanceFromDir` and `SynthesizeInstance` take `_ context.Context`).
A caller that cancels still pays for every stage. 0009:D9, as amended on 2026-10-03, allows
context checks at a verb's entry and between its stages to land on their own. Cancellation inside a
stage stays with 0009.

The owner decided both (ADR-013, decisions d1 and d3), and bundled them into one library
change:

- d1: "Sentinel ErrTransient + typed *FetchError (errors.Is/As); export opmerrors.Classify(err) for
  raw cue/load, modconfig and cuemod errors so the cli drops its 3 text probes."
- d3: "add ctx.Err() checks to the 5 verbs with a cancelled-ctx test each; godoc: cancellation lands
  between stages."

## What Changes

- `opm/errors` gains the fetch classification:
  - `ErrTransient`, a sentinel that is network-level only: the registry was unreachable (no HTTP
    response), a deadline expired, or the registry answered with a 5xx status;
  - `FetchKind` with `FetchOther`, `FetchNotFound`, `FetchUnauthorized` and `FetchUnreachable`;
  - `*FetchError{Kind, Coordinate, Err}`. Its `Error()` is the cause's text unchanged, `Unwrap`
    returns the cause, and `errors.Is(err, ErrTransient)` holds exactly for the transient cases;
  - `Classify(err error) error`. It reads the typed chain first (`modregistry.ErrNotFound`,
    `ociregistry.HTTPError` status, the ociregistry error codes, `net.Error`,
    `context.DeadlineExceeded`). Its one documented text fallback covers the forms `cue/load`
    flattens, and a test pins those forms against the embedded CUE version. A context cancellation
    and any error it does not recognise come back unchanged and are never transient.
- The library applies `Classify` where a fetch or resolution failure leaves it. That is the
  registry fetch in `FetchArtifact` (with `Coordinate` set), the `cue/load` resolution error in
  `loader.LoadDir` (which covers every acquire verb and the synth build), the load of a file-backed
  values source (`opm/kernel/source_loader.go`), the render module load, the schema `OCILoader`
  load and the platform-module dependency closure. Every wrap keeps its message text, and every existing `errors.Is` sentinel still matches.
- Each of the five verbs checks `ctx.Err()` at entry and between its stages, and returns the bare
  context error. `FetchArtifact` checks after the registry fetch, and `Render` checks after the
  build. The `opm/kernel` package doc says, once, that cancellation lands between stages: a
  running `cue/load` or build is not interrupted.
- `go.mod`: `cuelabs.dev/go/oci/ociregistry` moves from indirect to direct (same version, pinned by
  `cuelang.org/go`).
- Specs: a new `fetch-error-classification` capability, and an ADDED cancellation requirement in
  `kernel-runtime`.

Not in this change:

- The cli moving its three probes onto `Classify`, and the operator retry policy that reads it.
  Those are the frontend changes on the first library release that carries this one. Each cli site
  keeps today's exit code: `check.go` and `compat.go` read `FetchNotFound`, and
  `IsConnectivityError` reads `FetchUnreachable` (not `ErrTransient`, which also holds for a 5xx
  answer that the probe treats as an answer today). design.md D2 gives the mapping.
- The cli's `internal/config/platform.go` hint probe ("module not found", "cannot find package",
  "cannot expand module graph"). The owner's decision counts three probes. `Classify` covers these
  forms, so the cli may move it in the same change if it wants to.
- Cancellation inside a stage (0009:D9). `cue/load` takes no context.
- A migration fragment: `migrations/` is dormant until GA, and nothing here breaks.

## Capabilities

### New Capabilities

- `fetch-error-classification`: the typed fetch errors, `ErrTransient`, `Classify`, and the library
  sites that apply it.

### Modified Capabilities

- `kernel-runtime`: ADDED "Kernel verbs check cancellation at entry and between stages".

## Impact

- Packages: `opm/errors` (new `fetch.go`, `classify.go`, package doc), `opm/internal/loader`
  (`registry.go`, `load.go`), `opm/internal/renderstage` (`stage.go`), `opm/kernel` (`acquire.go`,
  `synth.go`, `render.go`, `source_loader.go`, `doc.go`), `opm/schema` (`loader.go`, `cache.go`
  godoc), `opm/helper/platformmodule` (`closure.go`), `opm/internal/registrytest`
  (a registry that answers every request with one status). `opm/errors` gains its first imports
  outside the standard library: `cuelang.org/go/mod/modregistry` and
  `cuelabs.dev/go/oci/ociregistry`. Both are already in every consumer's module graph.
- Public surface: additive. There are new exported symbols in `opm/errors`, and no signature
  changes. The `_ context.Context` parameters are renamed to `ctx`, which is not an API change.
- Behaviour: an error that `Classify` recognises is now wrapped in `*FetchError`, so its chain has
  one more link. `Error()` text is identical, and `errors.Is`/`errors.As` on the cause (CUE error
  lists included) still match. A verb called with an already-cancelled or expired context now fails
  fast with the context error instead of completing. Both frontends pass live contexts.
- Downstream: neither frontend needs a code change to keep building.
- SemVer: MINOR (additive). Release class `feat`. Satisfies 0021:D8:R12 on the library side.
