## Why

Every render evaluates each matched pair's transform twice. The glue builds `rendered` from
`_transformers[p.transformer].#transform & {#moduleInstance, #component, #context}` for every pair
(`opm/internal/renderstage/render.cue.tmpl:513-525`), and `diagnostics.failedPairs` repeats the same
unification for every pair to test `applied == _|_` (`render.cue.tmpl:550-561`). The second copy is
not shared with the first, and it sits inside `diagnostics`, which
`decodeRenderDiagnostics` forces with `dv.Validate(cue.Concrete(true))`
(`opm/kernel/render_decode.go:65`). So every render pays it, refusals included.

The kernel does not need the glue's list. `decodeRendered` already reads each pair's output and
checks `out.Err()` (`render_decode.go:170`), which catches the same pairs; `failed[p]` is only a
fallback there (`:155-158`).

The cost is large. With the memprobe harness, a library-side simulation of the operator's render
path, at this change's base: one cert-manager render (5 components, 42 objects, CRD-heavy) peaks at
1839 MiB of heap and about 2.0 GiB RSS, and two at once at about 3.7 GiB RSS against the operator's
4 Gi pod limit. `modules/k8up` (20 objects) peaks at 1039 MiB. This change cuts the cert-manager
peak heap about 6x (1839 to 304 MiB) and its user CPU about 3.6x, two cert-manager renders at once
to about 560 MiB RSS, and k8up's peak heap about 3x. `modules/web_app` (2 objects) does not change.
The cost follows output size, not component count. design.md records the before and after numbers.

The same glue also evaluates the always-unify rung (`_unify`, `:176-192`) and the predicate rung
(`_pred`, `:197-220`) for every transformer on the platform, per component. Only candidates (the
transformers the demand walk reaches, `_candidates`, `:224-231`) are ever read. Keying the two rungs on
the candidates saves another 0.3-0.4 s of CPU per cert-manager render on a two-catalog platform.

The owner decided this in the kernel checklist walkthrough (task g4): measure first, on a
multi-component and a cert-manager-sized module, then drop `failedPairs` from the glue and fill
`FailedPairs` in Go, on the gate-refusal path too, and key the matching rungs on candidates. The
parity harness must keep the verdicts identical. It lands before any other change to
`decodeRendered` or `render.go`. The same walkthrough asked for an operator memory measurement before
and after this fix ("before and after j2, nil-out, shared limit and g4"). design.md records
memprobe's operator columns for the wave-1 baseline, this change's base and its head.

## What Changes

- `render.cue.tmpl`: delete `diagnostics.failedPairs` and its comment. Each matched pair is now
  unified once, in `rendered`.
- `render.cue.tmpl`: build `_unify` and `_pred` over `_candidates` instead of over all of
  `#transformers`. `unifyFailures` keeps today's row order (see design.md D3).
- `opm/kernel`: `RenderDiagnostics.FailedPairs` is filled in Go. A pair is failed when its
  `rendered` output is an error (`Value.Err()` non-nil). This is the same set the glue's
  `== _|_` guard reported. The kernel fills it on both `RenderError` paths after the build: the gate
  refusal (`render.go:398-399`, which today returns before `decodeRendered` runs) and the pair
  failure (`render.go:401-404`). An incomplete output stays out of the list, as today.
- `opm/kernel`: the `glueDiagnostics.FailedPairs` field, the `failed[p]` map in `decodeRendered` and
  its unreachable fallback cause `"transformer output is an error"` go away. The doc comment on
  `RenderDiagnostics.FailedPairs` says when it is filled.
- Tests: a gate refusal beside a failing pair reports the failing pair; the existing failed-pair
  test asserts that the cause is the pair's own CUE error, which pins that `Err()` sees a bottom
  nested inside the output. Every existing `FailedPairs` assertion stays as it is.
  A test on the built render value pins the two evaluation rules: `diagnostics` carries no
  `failedPairs` field, and a component's rung entries cover exactly its candidates.
- `docs/site/diagnostics/transform-failed.md`: its maintainer comment drops the fallback cause from
  the list of causes. No reader-visible text changes.

Out of scope:

- The operator memory package (one render semaphore shared by both controllers, `GOMEMLIMIT` in
  the manifest). Those are operator changes. They should be sized against the numbers this change
  records, not the older baseline.
- Any other change to `decodeRendered` or `render.go` (rendered bytes, values, in-memory staging).
  Those changes rebase onto this one.
- Committing memprobe or its edits. The harness stays outside every repo.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `single-build-render`: the requirement "Matching runs inside the build with verdicts as data" now
  says that the kernel names failed pairs from each pair's output, not from a glue verdict, on a
  gate refusal as well as on a pair failure. It also says the two candidate rungs are evaluated only
  for candidates. Every existing scenario stays, and one is added: "Failed pairs are reported on a
  gate refusal".

## Impact

- Packages: `opm/internal/renderstage` (the glue template), `opm/kernel` (`render.go`,
  `render_decode.go`, tests), `testdata/render/scenarios` (one new scenario package),
  `docs/site/diagnostics/transform-failed.md` (maintainer comment only).
- Public surface: none changes. `RenderDiagnostics.FailedPairs` keeps its type and meaning. It is
  still filled on every `RenderError` raised after the build, and is still empty on success.
  Only the glue's internal `diagnostics.failedPairs` field disappears, and nothing outside the
  kernel reads it (the render module is generated per render).
- Downstream: the cli prints `component %q: transformer %s failed` for each `FailedPairs` entry on
  any `RenderError` (`cli/internal/workflow/render/validation.go:186-188`). It sees the same entries,
  in the same pair order. The operator does not read `FailedPairs`. Neither needs a code change.
  Both see the lower render cost after they bump the library.
- SemVer: PATCH, a performance change with no behaviour change. Release class `perf`.
- Risk: a refused render still evaluates each pair's output once to name the failed pairs. Today it
  does so twice. The fill is not skipped, because the cli prints `FailedPairs` on refusals.
