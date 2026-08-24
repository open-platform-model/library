## Context

See `proposal.md` for motivation. The mechanics that shape the approach:

- `opm/kernel/compile.go` forks `inst.MatchComponents()` into `schemaComponents` (read by `compile.Match`) and `dataComponents = compile.FinalizeValue(schemaComponents)` (filled into `#component` by `executePair`). `executePair` already reads `schemaComponents` for the `#context` metadata, so the unstripped value is in hand at the fill site.
- `compile.Module.Execute(ctx, inst, schemaComponents, dataComponents, plan)` is public in `opm/compile`; `cli` and `opm-operator` reach it only through `Kernel.Compile`, but the signature is SemVer surface.
- `Kernel.Finalize` / `compile.FinalizeValue` are public and have a `kernel-runtime` requirement of their own ("Utility Methods on Kernel"). Their removal is 0019 step 4, a MAJOR.
- `opm/kernel/flow_integration_test.go` builds its instance from a `CompileString` skeleton plus three `FillPath`s and a hard-coded `uuid`; `testdata/parity/instance/instance.cue` is the import-authored shape that resolves `#instance`, and `Kernel.LoadInstancePackage` already accepts it.
- The parity harness records `names-probe :: web` as expected to diverge because of the strip, and fails when a recorded divergence stops reproducing.
- ADR-003 forbids `FillPath` into a *closed, independently built* value. The transformer `#transform` is that value today already; what changes is only the filled argument. Experiment 00 (`enhancements/0019/experiments/00-purecue-definitions/`) measured a closed component rendering through unification with its definitions intact.

## Goals / Non-Goals

**Goals:**

- The value `#component` is filled with is the value `Match` reads: one components value through the pipeline.
- The flow fixture becomes an import-authored instance package, with `#instance` resolving and `uuid` derived.
- The harness's `names-probe` row loses its expected divergence and the suite is green.

**Non-Goals:**

- Removing `FinalizeValue`, `Kernel.Finalize`, or the `dataComponents` parameter (step 4, `library-finalize-removal`, MAJOR).
- Filling `#moduleInstance` (step 3).
- Any change to `#context` construction, so the label-order rows stay recorded (D12).
- Touching the env hoisting: it comes from the finalized value, which after this change no transformer receives, so `worker env order is hoisted by finalization` may turn green here. If it does, that subtest and `divergenceFinalizeHoisting` are retired in this slice (the harness forces it), and the D14 migration note moves here from step 4. Measured on first run, not assumed.

## Decisions

### D1: Fill `#component` from the unstripped value; keep `Execute`'s signature and ignore `dataComponents`

`executePair` looks the component up in `schemaComponents` and fills it. `Execute` keeps `(ctx, inst, schemaComponents, dataComponents, plan)`; the doc comment marks `dataComponents` "ignored; deprecated, removed with FinalizeValue". `Kernel.compileModuleInstance` stops calling `FinalizeValue` and passes `schemaComponents` for both arguments.

**Why:** the fix is a removal on the render path (0019 D1's direction) without a public break. Dropping the parameter now would force this slice to MAJOR and couple it to the `FinalizeValue` removal that D4 deliberately sequences later.

**Alternative:** rename `schemaComponents`/`dataComponents` to `components` throughout now. Rejected: it is the same MAJOR in disguise for `Execute`, and pure churn inside the package; done once, in step 4.

### D2: The flow fixture's instance becomes an on-disk package inside the `web_app` fixture module

`testdata/modules/web_app/instance/instance.cue`: `package instance`, imports `opmodel.dev/core@v2` and the `web_app` root package by its module path, `core.#ModuleInstance`, `metadata: {name: "web-app-demo", namespace: "default"}`, `#module: webapp`, `values:` equal to today's `debugValues`. The test loads it with `Kernel.LoadInstancePackage` and processes it with `ProcessModuleInstance(ctx, instVal, *mod, cue.Value{})`, exactly the authored path `flow_synth_imported_test.go` already exercises. `uuid` is no longer authored; core derives it from the fqn.

**Why inside the module:** an intra-module import needs no registry and no second `cue.mod`; `LoadModulePackage(web_app)` loads only the root package, so the subdirectory does not change what the module fixture is. The parity module's instance is not reused because its `web_app` copy carries the parity-only `worker` component and the flow test asserts two components.

**Consequence:** the fixture's `.cue` checksum in `cue-versions.yml` changes, so `cue:publish:smart` will see it as modified; that is the tooling working as designed, not drift.

**Alternative:** keep the skeleton and add a fourth `FillPath` for `#instance`. Rejected: it patches the symptom the wrong way round (a Go-side fill emulating what import gives for free), the exact pattern D1 forbids, and it leaves the hard-coded `uuid`.

### D3: The regression test is a transformer reading `#names` through the kernel, hermetic

A `registrytest`-served catalog with one transformer whose `#transform` re-declares `#component: _` and emits `#component.#names.dns.fqdn` and `#names.resourceName`, rendered through `Kernel.Compile` against an import-authored instance, asserting the concrete values. This is the probe the parity harness already builds; the regression test is the kernel-only half of it, so it runs without the oracle and without GHCR, and it lives next to the other hermetic kernel integration tests.

### D4: Closedness is guarded, not assumed

`opm/materialize/composed_open_test.go` (closed-fill corruption guard) and `opm/internal/cueregression` (canary pair) are required green. The shipped parity rows are the broad guard: every published transformer renders through the new fill, and any output-local hidden-field corruption shows as a value divergence on that pair, classified "beyond ordering" by the comparator. If one appears, this change stops and reports; there is no workaround that is not kernel behaviour added.

### D5: Harness bookkeeping is part of the change

Delete `ExpectedDivergence` on `names-probe :: web`; the harness's "no longer reproduces" failure is the trigger. Re-run `worker env order is hoisted by finalization`: if it passes (the hoisting was in the finalized value), delete that subtest, remove `divergenceFinalizeHoisting` from the worker row, and add the D14 migration note to `MIGRATIONS.md` under `## Unreleased — Additive` (ordering change, no consumer action). If it still fails, leave both and record why in this design.

## Risks / Trade-offs

- [A published transformer trips the ADR-003 corruption once it receives a closed, unfinalized component] → shipped parity rows catch it as a genuine value divergence; the change halts and the pair is reproduced as a fixture and reported upstream (0019 05-risks names this hazard as probed-and-not-reproduced).
- [`cue.Final()`-derived defaults: a transformer relying on defaults having been resolved in the finalized value] → unification resolves defaults at read time the same way; the parity rows are byte-identical for every shipped transformer or the harness says which one is not.
- [Flow fixture assertions depend on the hand-built instance's shape (component count, labels)] → the import-authored instance carries the same two components and labels; the test's assertions are unchanged, only the construction is.
- [`compile_test.go` helpers still call `FinalizeValue` and pass the result to `Execute`] → harmless while the parameter is ignored; they are cleaned in step 4 with the parameter. Not touched here to keep the slice small.

## Migration Plan

Code change plus fixture; no artifact publishes. Rollback is a revert. If D5's env-order retirement happens here, `MIGRATIONS.md` gains the D14 note in this slice (one server-side-apply diff on the first reconcile for modules whose env is assembled from guarded sources; nothing afterwards).

## Open Questions

- Whether the `worker` env hoisting is retired by this slice or by `library-finalize-removal`. Answered by running the harness (D5); it changes which slice carries the migration note, not what is built.
