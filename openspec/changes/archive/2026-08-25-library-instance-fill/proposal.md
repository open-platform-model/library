## Why

`core/src/transformer.cue` declares three inputs on `#transform` and states that the runtime supplies all three concretely, but the kernel fills only `#component` and `#context`: `#moduleInstance` is never filled, so a transformer that reads the declared slot vets clean, publishes, and fails at render with an error naming a disjunction rather than the slot (open-platform-model/library#65). The render-parity harness records this as the `instance-probe :: web` divergence; with `library-component-fill` landed it is the only Phase A divergence left, and it is the gate on `library-finalize-removal` (0019 step 4).

## What Changes

- `executePair` fills `#transform.#moduleInstance` with the instance's evaluated `#ModuleInstance` value, whole: `components` (siblings included), `#module`, `metadata`, `values`. No masking, no metadata-only wrapper (0019 D3 and D11). `opm/schema` gains the path constant for the slot.
- The self-referential read is covered: the instance filled into `#moduleInstance` contains the component filled into `#component`, and a transformer reading its own component through the instance renders concretely with no cycle. A hermetic regression test asserts both the plain read (`#moduleInstance.metadata.*`) and the self-referential one.
- The parity harness's `instance-probe :: web` row loses its `ExpectedDivergence` entry (the harness fails until it does). The four `divergenceContextLabelOrder` rows are untouched; they are D12's.
- `#context` construction is unchanged. It remains a Go projection of the same values until D12's `core` slice replaces it.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `transform-input-fill`: gains the requirement that `#moduleInstance` is filled with the whole evaluated instance (the third input the capability's purpose already covers). Existing requirements are unchanged.
- `render-parity`: the requirement "Probe transformers expose definition and instance inputs" currently states that the probe cases record the kernel's failure to supply the inputs as expected divergences, with a scenario "Instance probe under the kernel today"; after this change neither probe diverges, so the requirement's landing clause and that scenario change.

## Impact

- **SemVer: MINOR.** One new exported path constant in `opm/schema`; no signature changes. Transformers receive strictly more than before; every shipped transformer's rendered output is unchanged because none reads `#moduleInstance` (re-verified 2026-08-20 at 50 transformers), and the parity harness's shipped rows prove it rather than assume it. `cli` and `opm-operator` need no change.
- **Packages:** `opm/schema` (`paths.go`), `opm/compile` (`execute.go`), `opm/kernel` (regression test, `parity_probe_test.go` row update).
- **Enhancement 0019:** implements D3's `#moduleInstance` half, in D1's direction (the kernel supplies what unification supplies). D11 is the authoring contract this fill makes reachable; it is excused in the entry's `no_work` and not declared here. Closes library#65. Declared in `enhancement.yaml`.
- **Complexity (Principle VII):** one `FillPath` and one path constant; no new type, no new phase.
- **Risk:** filling a closed, independently built instance value into the transformer is the same shape as the `#component` fill this repo already carries under ADR-003's guard; the shipped parity rows and `composed_open_test.go` catch corruption as a value divergence rather than a silent wrong render.
