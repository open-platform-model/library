# Pure-CUE control for the transformer output-local hidden field bug

This is a **no-Go** harness proving the bug in
`../transformer-output-hidden-field-scope-bug.md` is **not** a CUE evaluator bug —
it is in the library's Go value plumbing (filling the composed map into a closed,
separately-built `c.#Platform`). See §11–§13 of that doc.

It expresses the kernel's "glue" (fill `#component` / `#context` into a
transformer's `#transform`) as plain CUE unification (`&`) and renders the
**buggy** `deployment-transformer@0.5.2` (whose `_convertedSidecars` hidden field
is declared *inside* `output`).

## Files

- `repro.cue` — minimal `#component`/`#context` applied to the real v0.5.2
  transformer; plus a `_viaMap` variant mimicking `#composedTransformers`.
- `realcomp.cue` — the **exact real finalized `web` component** dumped from the
  kernel (`OPM_DUMP_CUE`), to rule out component shape as the trigger.
- `closed.cue` — wraps the transformer in the **real closed `c.#TransformerMap`
  and full closed `c.#Platform`** (what Materialize fills into). Both still render
  concretely → the closed schema is fine *in CUE*; only the Go `FillPath`-into-a-
  closed-separately-built-value path corrupts it.

## Running

Requires a local OCI registry at `localhost:5000` holding
`opmodel.dev/catalogs/opm@v0.5.2` (the pre-fix catalog), `opmodel.dev/core@v0`,
and `cue.dev/x/k8s.io@v0`.

```bash
export CUE_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'

# Match the library's pinned CUE version exactly — install the alpha.1 CLI:
GOBIN=/tmp/cuebin go install cuelang.org/go/cmd/cue@v0.17.0-alpha.1

cue mod tidy                                  # populate transitive deps
/tmp/cuebin/cue export -e containers          ./...   # minimal component
/tmp/cuebin/cue export -e containersReal      ./...   # exact real component
/tmp/cuebin/cue export -e containersViaPlatform ./... # through closed c.#Platform
/tmp/cuebin/cue vet -c                         ./...  # whole thing is concrete
```

## Result

All render `containers` **concretely** and `cue vet -c` passes — on the *same*
CUE version (`v0.17.0-alpha.1`) the Go library uses. The Go kernel, fed the
identical transformer + component + context, produced
`list.Concat: non-concrete value _` until the fix. The fix (kernel) reads the
`#transform` from the open `MaterializedPlatform.Composed` map instead of out of
the closed `Package`.
