## Context

See proposal.md for motivation. Facts the approach rests on, checked on 2026-09-13:

- `*kernel.Kernel` exports ten methods in production code (`AcquireInstanceFromDir`, `AcquireModuleFromDir`, `AcquireModuleFromRegistry`, `AcquirePlatformFromDir`, `LoadSourceFromBytes`, `LoadSourceFromFile`, `Render`, `SchemaCache`, `SynthesizeInstance`, `ValidateConfigDetailed`) plus `RenderForTest` from `export_test.go`, which the test binary compiles in.
- The absence pins live in `opm/kernel/kernel_test.go:298-345` (`NoFinalizeMethod`, `PrunedSurface`), `opm/kernel/registry_loader_test.go:64-71` and `opm/module/module_test.go:73`. `TestKernel_NoContextAccessor` (`kernel_test.go:34`) walks every method's parameter and result types.
- `parity_probe_test.go` serves catalog, module and oracle glue from the in-process registry and resolves core from the warm workspace cache; its `-short` skip message (line 87) still says it resolves core "from the workspace cache seeded from GHCR". `component_fill_test.go` and `instance_fill_test.go` use the same `probe_app` module, the same `names-regression` transformer text and assert the same two values, with no skip.
- `integration_validate_test.go` synthesizes one instance and asserts name, namespace and a non-empty UUID; `synth_schema_test.go` (`DerivedFieldsFromSchema`, `ParityWithAuthoredPackage`) asserts the UUID's derivation.
- `integration_live_test.go` and `flow_integration_test.go` both acquire `testdata/modules/web_app` behind `skipUnlessRegistry` and `-short`.
- `writeCatalogPlatform` (`integration_fixtures_test.go:79`) writes a `cue.mod/module.cue` with a core pin and a catalog pin and a `platform.cue` with one `#registry` entry; `platformmodule.Generate(Input{Name, Type, ModulePath, Entries, Deps})` produces the same two files and `Files.WriteTo` places them.
- `writeTempModuleDir`, `writeTempInstanceDir`, `writeTempPlatformDir` are defined in `opm/kernel/kernel_test.go` / `acquire_test.go` and again in `opm/internal/loader/load_test.go`; `opm/internal/schematest` is a leaf test-helper package both already import.
- `majorOf` (`integration_fixtures_test.go:62`), `registrytest.coreMajor` and `synth.major` are the same four lines.
- `schema/loader_test.go:97` loads `opmodel.dev/core@v1` from GHCR to prove that a major-only module string resolves.

## Goals / Non-Goals

**Goals:**
- One place states which methods the kernel exports; nothing else has to list what it does not.
- No test repeats a stronger sibling; no fixture imitates a generator by hand.
- Package docs describe the tree as it is.

**Non-Goals:**
- Touching production code paths (only `registrytest`, a test-only package, gains an export).
- Rewriting `opm/kernel/doc.go`.
- Merging the parity harness, the render fixtures and the flow test; each covers a different input set.

## Research & Decisions

### Exact-set surface test instead of deleting the absence pins

**Context**: the pins are cited by three spec scenarios; deleting them outright would need spec deltas for a test-only change.
**Explored**: the exported method set (`grep '^func (k \*Kernel) [A-Z]'`), the `export_test.go` addition, and the three scenarios' wording.
**Options considered**:
1. Delete the pins and the spec scenarios - a spec delta for no behaviour change, and the next prune brings a new pin back.
2. Keep the pins as they are - a 23-name list that grows with every removal and says nothing about what does exist.
3. One test asserting `reflect.TypeOf(&kernel.Kernel{})`'s exported method names equal a sorted literal list, with `*ForTest` names filtered out.
**Decision**: option 3, `TestKernel_ExportedSurface` in `kernel_test.go`, replacing `NoFinalizeMethod`, `PrunedSurface` and `NoLoadModuleFromRegistryMethod`; the `NewModuleFromValue` check in `module_test.go` is dropped because the exact set covers it. `NoContextAccessor` stays.
**Rationale**: an exact set is the strongest form of every absence pin at once, and a future removal edits one literal instead of adding a name.

### Retire the fill tests; spike the probe's skip

**Context**: the fills exist as the "kernel-only half" of a probe that no longer needs GHCR.
**Explored**: both files' fixtures and assertions against `parity_probe_test.go`; the skip's message.
**Decision**: delete both fill tests. Section 1 removes the probe's `-short` skip and times `go test -run TestParity_Probes -count=1`; the skip stays out if the probe takes under about five seconds on the warm cache, otherwise it stays in and the finding is recorded here. Either way the fills go: `-short` is not a CI mode.
**Rationale**: the probe pins the same values against the oracle, which is strictly more than the fills assert.
**Spike result (2026-09-13)**: without the skip, `go test ./opm/kernel/ -run TestParity_Probes -count=1` takes 0.12 s on the warm workspace cache (0.06 s per probe; about 1.3 s including package compilation on the first run). The skip stays out.

### Fold the live validate test into the flow test

**Decision**: `TestFlow_WebApp_OnOpmPlatform` gains, after acquiring `web_app`, the `debugValues`-satisfy-`#config` assertion from `integration_live_test.go` (render the field back to CUE with `format.Node`, wrap it with `LoadSourceFromBytes`, call `ValidateConfigDetailed`); the file is deleted.
**Rationale**: same module, same gate, one fewer GHCR round trip per run.

### Fixtures from the generator and shared helpers

**Decision**: `writeCatalogPlatform` builds `platformmodule.Input{Name: "hermetic", Type: "kubernetes", ModulePath: "testing.opmodel.dev/library-kernel-test/platform@v0", Entries: [{Path: dep, Version: version, Enable: true}], Deps: [{core pin}, {dep, "v"+version}]}`, calls `Generate` and `Files.WriteTo(platDir)`. `schematest` gains `WriteModuleDir`, `WriteInstanceDir`, `WritePlatformDir` (and the `WritePkgDir` they share), replacing both copies. `registrytest` exports `Major(version string) string`, used by `coreMajor`'s callers and by `integration_fixtures_test.go`; `synth.major` is production code in an internal package and stays.
**Rationale**: a hand-written copy of the generator's output cannot catch generator drift; the shared helpers already live in the package both test trees import.

### Doc trim

**Decision**: `opm/helper/doc.go` keeps the boundary statement, the `platformmodule` description and one line per folded subpackage saying where it went; it drops the "planned subpackages" list, the "added by their owning slices" process text and replaces the `enhancements/001-kernel-redesign-around-platform/` path with `legacy:001`. The six `// Was: Release…` lines go. CLAUDE.md's "Render contract" section is cut to a pointer at the `opm/kernel` package doc plus the two agent-only rules it carries (tests use the in-process registry; the parity harness is the oracle).
**Rationale**: the godoc is the published contract; two copies drift.

## Risks / Trade-offs

- [The exact-set test breaks on every deliberate surface change] → that is its job; the edit is one literal, and the failure message names the extra or missing method.
- [The probe is slow without the skip and CI time grows] → the spike measures it first; the skip is kept if it costs more than about five seconds.
- [Generating the test platform hides a bug in `Generate` that the hand-written file would have exposed] → `platformmodule/generate_test.go` covers the generator's text directly; the render tests now cover its output end to end, which the hand-written file never did.
