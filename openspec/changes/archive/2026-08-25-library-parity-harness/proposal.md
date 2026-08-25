## Why

Enhancement 0019 (D1) makes plain CUE unification of `#transform` with its three inputs the reference semantics of the render path, and nothing in `library` measures that today: the definition strip in `compile.FinalizeValue` diverged from CUE for three months and two refactors because no test compared the kernel's rendered value against unification of the same inputs. D4 orders the harness first, before any fill or removal lands, so that every later Phase A slice is checked against the oracle rather than the existing suite, and so the harness's first red run is preserved as the evidence for D1 instead of being discarded by fixing first.

## What Changes

- Add a differential parity harness to `library`: a Go test that renders a fixed set of (instance, component, transformer) cases through the kernel's compile path and through a single pure-CUE build of the same inputs, and asserts the two rendered values agree.
- Vendor the pure-CUE oracle as test fixtures: the render glue from `enhancements/0019/experiments/01-purecue-render-flow/` (match, context projection, execute, expressed as CUE) plus an import-authored `#ModuleInstance` fixture. The instance is authored by import rather than assembled with `LookupPath`+`FillPath`, so both renderers resolve identical inputs; the existing flow fixture's severed `#instance` wiring is left for the next slice (D4's fixture-repair-plus-fill slice), not fixed here.
- Enumerate cases per the entry's `#ParityCase` contract: each names its inputs by fixture, states `equality` (`output-fields-only` for the interim, because `#context` is built in Go on the kernel side and projected in CUE on the oracle side until D12 lands), compares order-sensitively (D14), and carries `expectedDivergence` where the kernel is known to differ today (the definition strip, the unfilled `#moduleInstance`, the finalization reordering of map-derived lists).
- Cases with `expectedDivergence` set are asserted to diverge with the named cause; the suite is green on landing and the divergences are recorded, not hidden. Each later Phase A slice empties the entries it fixes, and the harness fails when an expected divergence stops reproducing, which is how a slice proves it closed one.
- No kernel behaviour change. `FinalizeValue`, `executePair`, and `Match` are untouched; the harness only reads what they produce.

## Capabilities

### New Capabilities
- `render-parity`: the differential parity contract between the kernel's render path and pure-CUE unification: what a case names, how inputs are resolved identically for both renderers, what "equal" means (structural versus output-fields-only, order-sensitive), how a known divergence is recorded and retired, and the rule that a divergence is closed by removing kernel behaviour rather than by loosening the comparison.

### Modified Capabilities
<!-- none: no existing requirement changes; the render path's behaviour is unchanged by this slice -->

## Impact

- **SemVer: PATCH.** Test code and testdata only; no `opm/` type, signature, or behaviour changes. No downstream migration for `cli` or `opm-operator`.
- **Packages:** a new test file (and any helper) alongside `opm/compile` or `opm/kernel`, whichever the design chooses as the seam that exposes the per-pair rendered value; new fixtures under `testdata/` (oracle `render.cue`, import-authored instance, a `cue.mod` pinning exact published `opmodel.dev/core` and `opmodel.dev/catalogs/opm` versions, resolved from GHCR per the workspace registry policy).
- **Dependencies:** none new. The oracle runs through the module's existing `cuelang.org/go` via `cue/load`, the same way `schema-testing` and `cue-regression-canary` already do.
- **Enhancement 0019:** implements D1 (the oracle and the direction of fixes), D4 (harness-first ordering), and D14 (order-sensitive comparison). Declared in `enhancement.yaml` so archive-time logging to `enhancements/0019/delivery.yaml` is mechanical.
- **Complexity justification (Principle VII):** one table-driven test and a fixture directory. The only structure added is the case table, which the entry's contract already fixes the shape of; no new public symbols.
- **Risk:** the harness's value depends on both sides resolving identical inputs. If the kernel side can only be reached through the kernel's own instance construction, the design must show the two inputs are the same value and not merely equivalent-looking fixtures.
