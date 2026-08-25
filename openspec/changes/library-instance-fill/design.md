## Context

See `proposal.md` for motivation. The mechanics that shape the approach:

- `executePair` (`opm/compile/execute.go`) already holds the whole instance: it receives `*module.Instance`, whose `Package` field is the loaded, evaluated `#ModuleInstance` value and the source of truth for every schema path (`MatchComponents()` is `Package.LookupPath(schema.Components)`). The value filled into `#component` is a sub-value of it.
- `opm/schema/paths.go` names the inputs the kernel fills (`Component`, `Context`, the three `Context*` sub-paths) but has no constant for `#moduleInstance`, because nothing ever filled it.
- The oracle (`testdata/parity/oracle/render.cue`) fills `#moduleInstance: #instance` with the import-authored instance, whole, and its comment records that this is the one input the kernel has never supplied. The `instance-probe :: web` row in `parity_probe_test.go` records that as its expected divergence and pins `probe-demo` as the oracle's value.
- `Instance.Package` is produced by either path the kernel offers: `ProcessModuleInstance` over a loaded package, or `SynthesizeInstance` (a `FillPath`-built instance). Both yield one evaluated value; the fill does not care which.
- ADR-003 forbids filling into a closed, independently built value. `#transform` is that value, as it is for the `#component` fill; the argument being filled is `_`-typed in core, so closedness of the argument is irrelevant to the fill itself.

## Goals / Non-Goals

**Goals:**

- `#moduleInstance` is filled with `inst.Package`, so all three declared inputs reach a transformer.
- The self-referential read (instance contains the very component in `#component`) is proven on the kernel path, not only on the oracle.
- `instance-probe :: web` loses its expected divergence and the harness is green.

**Non-Goals:**

- Changing `#context` construction (D12, a `core` slice in Phase B).
- Removing `FinalizeValue`, `Kernel.Finalize` or `Execute`'s `dataComponents` parameter (step 4, MAJOR).
- Any structural limit on sibling access (D11: discouraged by contract, never narrowed).
- Fixing the flow fixture or any test that never reads `#moduleInstance`.

## Decisions

### D1: Fill `inst.Package` at a new `schema.ModuleInstance` path, before `#component`

`opm/schema/paths.go` gains `ModuleInstance = cue.ParsePath("#moduleInstance")` under the "Inside #transform" group. `executePair` fills it with `inst.Package` as the first fill, then `#component`, then `#context`, and wraps the error as `filling #moduleInstance`. The function comment's numbered flow gains the step.

**Why `Package` and not a projection:** D3 rejected the metadata-only wrapper and D11 the sibling mask. The oracle fills the whole instance; anything less is a divergence D1 says to close by removal, not to create.

**Why before `#component`:** order does not change the result (unification is commutative), but filling the larger value first means a `#moduleInstance` error is attributed to that fill rather than surfacing later through the `#component` unification.

**Alternative:** fill `#moduleInstance` with `inst.Package` minus `components`, then rely on `#component` for the component. Rejected: it is a new strip, exactly the shape D11's `prevented` option rejects.

### D2: A `nil` or non-existent instance value is an error at the fill site

`inst.Package` is a zero `cue.Value` only when the caller bypassed every kernel entry point. `executePair` checks `Exists()` before filling and returns `component %q / transformer %q: instance value missing` rather than letting `FillPath` produce a bottom that surfaces as a transformer error. The check mirrors the existing `#transform not found` guard.

### D3: The regression test is the instance probe plus the self-reference, hermetic

`opm/kernel/instance_fill_test.go`, next to `component_fill_test.go` and built from the same `registrytest` helpers (`probeModuleFile`, `probeCatalogBody`, import-authored instance in a `t.TempDir()` module). One transformer emits `#moduleInstance.metadata.name`, `.namespace`, and `#moduleInstance.components[#component.metadata.name].metadata.name`; the test asserts the instance name and namespace and that the self-referential read equals `#component.metadata.name`. Runs without the oracle and without GHCR. Confirmed red on the current tree before the fill lands.

**Why the self-reference is a kernel test and not only a harness row:** the harness's probe pins only the plain read; D3 in 0019 names the self-referential case as the one that proves the slot is sound, and that proof should not depend on the GHCR-backed suite.

### D4: Harness bookkeeping is part of the change

Delete `ExpectedDivergence` on `instance-probe :: web`; the harness's "no longer reproduces" failure is the trigger. Update the row comment the way the `names-probe` row was updated. The oracle's comment ("which the kernel has never filled") is corrected in the same slice so the two renderers' documentation agrees.

## Risks / Trade-offs

- [A shipped transformer re-declares `#moduleInstance` with a constraint the filled instance violates] → measured absent (no shipped transformer reads the slot, 2026-08-20); the shipped parity rows would show it as a render failure on that pair, not a silent wrong render.
- [The whole instance value is large; filling it per pair costs evaluation time] → `FillPath` unifies lazily and the instance is already evaluated once per render; the transformer forces only what it reads. Experiment 08 measured the transformer step at 1.3 to 2.4 ms per object with all three inputs filled on the pure-CUE path. Not re-measured here; the harness and `task test` are the gate, and a regression would show as a timing change in the shipped rows.
- [`SynthesizeInstance`-built instances have a `Package` whose `#instance` may not resolve for every component] → unrelated to this fill: the value is handed over as is, and a transformer reading a field that is incomplete gets the same incomplete error unification gives. The flow fixture repair (step 2) already moved the kernel's own fixtures to import authoring.

## Migration Plan

Code change only; no artifact publishes. Rollback is a revert. No `MIGRATIONS.md` entry beyond a `### Changed` line under `## Unreleased — Additive` noting the third input is now filled (additive; a transformer that reads it starts rendering instead of failing).

## Open Questions

None. Whether a catalog transformer may read siblings through the filled instance is settled by 0019 D11 (reachable, discouraged by contract).
