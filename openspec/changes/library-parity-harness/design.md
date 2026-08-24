## Context

See `proposal.md` for motivation. The constraints that shape the approach:

- The kernel renders a pair in `opm/compile/execute.go` (`executePair`): it fills `#component` with the finalized component from `compile.FinalizeValue`, fills `#context` from `opm/schema/context.go`'s Go decoding, and never fills `#moduleInstance`. `opm/kernel/compile.go` forks one components value into `schemaComponents` (for `Match`) and `dataComponents` (for `Execute`).
- The oracle already exists as CUE: `enhancements/0019/experiments/01-purecue-render-flow/render.cue` expresses match (predicate rung), `#context` projection, and execute in about 80 lines against the real published catalog, with `cue vet -c` exit 0.
- The kernel accepts an import-authored `#ModuleInstance` package via `Kernel.LoadInstancePackage` followed by `ProcessModuleInstance` and `Compile`; `opm/kernel/flow_synth_imported_test.go` already exercises that path. The existing flow fixture instead assembles its instance with `LookupPath`+`FillPath`, which severs `#instance` and hardcodes a `uuid`; that fixture is repaired by the next 0019 slice, not this one.
- None of the 50 published transformers reads a definition field, so a shipped-catalog-only harness is green on landing and cannot show the strip. The D1 evidence needs transformers that read `#component.#names` and `#moduleInstance`.
- Hermetic catalogs are already served in tests through `opm/internal/registrytest` (`standardCatalog`, `UniquePath`), and both `cue/load` and the kernel loaders accept a registry mapping, so a probe catalog can be resolved identically by both renderers without publishing anything.
- Ordering is part of the contract (0019 D14). `Syntax(cue.Final())` is the very pass that reorders, so the comparison must not route either value through it.

## Goals / Non-Goals

**Goals:**

- One table-driven test whose rows are `#ParityCase` values from `enhancements/0019/contracts/contracts.cue` transcribed into Go: `name`, fixture names for the three inputs, `equality`, `expectedDivergence`.
- Both renderers consume the same fixture bytes and the same `cue.mod` pins; the only per-side difference is which evaluator runs.
- The suite is green on landing, with the strip and the unfilled `#moduleInstance` captured as expected divergences that later slices retire.

**Non-Goals:**

- Changing any kernel behaviour, including repairing the existing flow fixture.
- Specifying matching. The oracle's match is the predicate rung only; the pair-set check exists to keep the two universes visibly aligned on the fixtures, not to be the matcher's spec (0019 D10 owns that).
- Comparing `#context`. Until 0019 D12 lands the two sides derive it differently by construction; every case starts at `output-fields-only`.
- Performance. The harness runs the web_app fixture and a two-transformer probe catalog; that is seconds, not a benchmark.

## Decisions

### D1: The harness lives in `opm/kernel` as an integration-style test

**Choice:** `opm/kernel/parity_harness_test.go`, in package `kernel_test`, gated the same way as the flow tests (`testing.Short()` skip, `skipUnlessRegistry`, `OPM_FLOW_TEST_FORCE=1` to force).

**Why:** the kernel side must run the real public path (`LoadModulePackage`, `LoadPlatformPackage`, `Materialize`, `LoadInstancePackage`, `ProcessModuleInstance`, `Compile`) so that what is compared is what consumers get. `opm/compile` would let the test reach `executePair` directly but would compare a seam, not the product, and would need the platform materialized by hand anyway.

**Alternatives:** an `opm/internal/parity` package with its own helpers (rejected: nothing is reused yet, Principle VII); a Go-side re-implementation of the oracle (rejected outright: the oracle must be CUE or it is not the oracle).

### D2: Fixtures form one CUE module under `testdata/parity/`, with the oracle and the instance as sibling packages

**Layout:**

```
testdata/parity/
  cue.mod/module.cue      module: testing.opmodel.dev/library-parity@v0
                          deps: core v2.0.0-alpha.4, catalogs/opm v2.0.0-alpha.3
  web_app/                copied from testdata/modules/web_app (module.cue, components.cue)
  opm_platform/           copied from modules/opm_platform/platform.cue
  instance/instance.cue   #ModuleInstance; #module: web_app by import; name web-app-demo / default
  oracle/render.cue       experiment 01's glue: match, _contextFor, rendered, pairs
```

**Why:** intra-module imports need no registry, so the oracle can import `instance`, `web_app`, `opm_platform`, and the catalog with one `cue.mod`; the kernel loads `web_app/`, `opm_platform/`, and `instance/` as packages from the same tree. `web_app` and `opm_platform` are copied rather than imported from `testdata/modules` because that module is `opmodel.dev/library-testdata@v0`, which resolves only from a local publish. The copies carry a header naming their source; the existing `testdata/modules/*` drift check in `Taskfile.yml` is the model if drift ever bites.

**Module path:** `testing.opmodel.dev/...` per the workspace registry policy (fixtures never squat `opmodel.dev/*`). It is never published; the path only needs to be a valid module identity for intra-module imports.

**Alternatives:** `cue.mod/local-module.cue` directory replacements to avoid copying (rejected for this slice: the mechanism belongs to 0019 D9 and is not yet in the kernel's own loaders, so it would make the harness the first consumer of an unlanded mechanism); a generated fixture written into `t.TempDir()` (rejected for the shipped-catalog cases: reviewable on-disk CUE is the point of an oracle).

### D3: Probe transformers ride a hermetic catalog served by `registrytest`, with the oracle's `cue.mod` written per run

**Choice:** a second case group. The test builds a probe catalog (two transformers: `names-probe` emits an object carrying `#component.#names.dns.fqdn` and `#names.resourceName`; `instance-probe` emits an object carrying `#moduleInstance.metadata.name`) from Go strings with the `registrytest` helpers, at a `UniquePath`, and a probe platform subscribing to it. For this group both renderers receive the same registry mapping; the oracle side is a small CUE module written into `t.TempDir()` whose `cue.mod` pins the probe catalog at the served version and whose `render.cue` is the same glue with the catalog import swapped.

**Why:** the strip is not observable through shipped transformers, and a probe cannot be published to GHCR (catalog publishes are CI-only) nor placed under a checked-in `cue.mod` pin, because `UniquePath` is what keeps the CUE module cache from serving a stale build. This is the existing pattern in `flow_synth_imported_test.go`, applied to the oracle as well as the kernel.

**Expected divergences on landing, recorded in the table:**

| case | expectedDivergence |
| --- | --- |
| `names-probe :: web` | `FinalizeValue strips #names from #component` (kernel: incomplete `output`, oracle: concrete fqdn) |
| `instance-probe :: web` | `#moduleInstance is never filled` (kernel: incomplete `output`, oracle: concrete name) |

Ordering (0019 D14 / OQ14), measured on the first run (2026-08-24): four of the five shipped pairs (every object carrying the context-derived label set) diverge by field order only, same labels and values. The cause is not the strip but `opm/schema/context.go`, which decodes `metadata.labels` into a Go map and re-encodes it sorted, so the kernel hands `#transform` a sorted label map where unification hands it CUE's evaluation order. Recorded as `divergenceContextLabelOrder` on those four rows; the D12 projection slice retires it. The HPA pair carries no such labels and agrees. Experiment 07's guarded-env reordering did not reproduce on `web_app` and has no row.

**Alternatives:** a checked-in probe catalog under `testdata/parity/` imported intra-module by the oracle (rejected: the kernel side reaches transformers only through a platform's registry subscription, which needs a resolvable catalog module, so the kernel could not see it); adding the probes to `catalog_opm` (rejected: a published catalog is not a place for test probes, and the sweep that would read `#names` is gated on this harness existing first).

### D4: Comparison is per pair, over `output`, through an order-preserving encoding

**Kernel side:** `CompileResult.Compiled` filtered by `(Component, Transformer)`, in the order returned; each `Compiled.Value` is one object. **Oracle side:** `rendered["<component> :: <transformer>"]`, normalised to a list when it is a single object (mirroring how `executePair` flattens a list `output` into one `Compiled` per element).

**Encoding:** both values are serialised with the same order-preserving encoder before comparison, and the encoder must be verified not to sort struct fields. JSON marshalling of a `cue.Value` emits fields in evaluation order; that is the candidate, and the first task confirms it on a deliberately reordered value before anything else is built on it. The comparison is a byte diff of the two encodings, with the first differing path reported by walking the two decoded values in parallel.

**Why not `Syntax(cue.Final())` + `format.Node`:** that is the finalization pass D14 names as the source of reordering; routing both sides through it would hide the very difference the harness must be able to see.

**Pair sets:** the kernel's `MatchPlan.MatchedPairs()` and the oracle's `pairs` are compared as sets first. A kernel-side refusal caused by the always-unify rung (0019 D10, the D30 carve-out) is the single exemption and is reported, not silently accepted; on the shipped fixtures it is empty.

### D5: Case rows mirror `#ParityCase` field for field

A Go struct with `Name`, `Instance`, `Component`, `Transformer`, `Equality` (`structural` | `output-fields-only`), `ExpectedDivergence` (string, empty means "must agree"). `OrderSensitive` is not a field: it is fixed true by the contract, so it is a property of the comparator. A row with `ExpectedDivergence` set passes only when the kernel side fails or differs; when it agrees, the test fails with a message telling the author to delete the entry (0019 D4: every entry is emptied by the time the entry is `implemented`).

## Risks / Trade-offs

- [The oracle and the kernel could agree because both are wrong the same way] → the oracle is plain unification with no Go in the loop; its correctness rests on experiment 01's `cue vet -c` result, which the fixture's `cue vet` task re-runs.
- [`web_app` exercises D14 only through the context-label ordering, not the finalization hoisting experiment 07 measured] → a guarded-env fixture is the obvious addition when the `library-finalize-removal` slice needs it; adding a row is a fixture change, not a harness change.
- [Two fixture copies (`web_app`, `opm_platform`) can drift from their sources] → header comments name the source; the parity fixtures are pinned to the same published versions as the sources, so drift only matters when someone edits one, which the review of that edit sees.
- [The probe group needs `registrytest`, so it is heavier than the shipped group] → the two groups are independent subtests; the shipped group stays a fast reviewable on-disk oracle.
- [GHCR dependency, like the existing flow tests] → same gating (`-short` skip, reachability probe, force flag); nothing new for CI.
- [The encoder silently sorts and order regressions go unseen] → the first task is a negative test that proves the chosen encoder reports a reordering, before any case is written.

## Migration Plan

Test-only; no deployment, no consumer change. Rollback is reverting the commit. The `enhancement.yaml` declaration means archiving logs D1, D4, D14 to `enhancements/0019/delivery.yaml`; if the harness is reverted after archive, that log entry needs a counter-entry.

## Open Questions

- Whether the probe group's `t.TempDir()` oracle module should also be exercised by a `cue vet` task the way the on-disk fixtures are. Deferrable: it does not change the specs or the tasks, only whether one more command runs in CI.
