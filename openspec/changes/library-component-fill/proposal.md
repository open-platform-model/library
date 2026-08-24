## Why

The kernel hands `#transform` a stripped copy of the component: `compile.FinalizeValue` exports through `cue.Final()`, which drops every definition field, so `#component.#names`, `#resources`, `#traits` and `#instance` are unreachable inside a transformer even though core computes them and pure CUE unification passes them through. The parity harness (`library-parity-harness`) records this as the `names-probe :: web` divergence. Enhancement 0019 D4 binds the fix to a fixture repair: `opm/kernel/flow_integration_test.go` builds its instance with `LookupPath`+`FillPath`, which severs `#instance`, so exposing definitions before repairing it would turn that fixture from shipping no value into shipping a broken one (`#names.dns.fqdn: required field missing: namespace`).

## What Changes

- `executePair` fills `#component` with the unstripped component value (the one `Match` already reads) instead of the finalized copy. The strip stops being on the render path; the value a transformer sees is the value unification would see.
- `Kernel.Compile` stops forking one components value into two for execution. `compile.FinalizeValue` and `Kernel.Finalize` remain callable and unchanged this slice; removing them from the public surface is 0019 step 4 (`library-finalize-removal`, MAJOR). `compile.Module.Execute` keeps its signature: the `dataComponents` parameter is documented as ignored and deprecated, and the kernel passes the same value for both, so no consumer re-pins here.
- The flow fixture's instance is authored by import (as `testdata/parity/instance` is) so `#instance` resolves, and its hard-coded `uuid` goes, letting core derive it from the fqn. A regression test asserts a transformer can read `#component.#names` through the kernel.
- The parity harness's `names-probe :: web` row loses its `ExpectedDivergence` entry (the harness fails until it does). `instance-probe :: web` and the four context-label-order rows are untouched; they belong to later slices.
- `composed_open_test.go` (closed-fill corruption guard) and the CUE canary pair keep passing; ADR-003's no-cross-build-fill-into-closed-values rule is not relaxed. The component is filled into the transformer, as today; what changes is only that the filled value keeps its definitions. Experiment 00 measured that a genuinely closed component renders this way.

## Capabilities

### New Capabilities
- `transform-input-fill`: what the runtime owes each declared `#transform` input when it fills it: `#component` is the instance's component value with every field class preserved (no strip in transit), so a transformer reads what unification would give it.

### Modified Capabilities
- `kernel-runtime`: the requirement "Compile sources its cue.Context from the caller Kernel" describes the pipeline as "Finalize → Match → Execute" building "the finalized data components"; the pipeline no longer finalizes components for execution, so that requirement's description and scenario wording change (the context-sourcing rule itself is unchanged).

## Impact

- **SemVer: MINOR.** No `opm/` signature changes. Transformers receive strictly more than before; for every shipped transformer the rendered output is byte-identical, which the parity harness proves rather than assumes (no shipped transformer reads a definition field; re-verified 2026-08-20 at 50 transformers). `cli` and `opm-operator` need no change.
- **Packages:** `opm/compile` (`execute.go`, `module.go` doc), `opm/kernel` (`compile.go`, `flow_integration_test.go`, `parity_harness_test.go` / `parity_probe_test.go` row update).
- **Rendered output ordering:** unchanged by this slice. The measured ordering divergence comes from the Go-built `#context`, not from the strip, so no `MIGRATIONS.md` note is owed here; 0019 D14's migration note ships with `library-finalize-removal`.
- **Enhancement 0019:** implements D2 (execution unit stays one component; recorded, not changed), D3's `#component` half and D4's fixture-repair ordering; D1's first parity fix, in the direction D1 requires (removal of a kernel step from the render path). Declared in `enhancement.yaml`.
- **Complexity (Principle VII):** net removal; one code path fewer in `executePair`, one fewer value in `Kernel.Compile`.
- **Risk:** if filling an unfinalized closed component into `#transform` trips the ADR-003 corruption for some catalog member, the harness's shipped rows fail on that pair rather than silently rendering wrong; the design records the fallback (fill the schema value's non-definition projection is *not* acceptable under D1; the answer would be a fixture to reproduce and an upstream report).
