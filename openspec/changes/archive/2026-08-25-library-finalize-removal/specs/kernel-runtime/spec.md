## REMOVED Requirements

### Requirement: Utility Methods on Kernel
**Reason**: `Finalize` performed schema-constraint stripping for the render path; the render path no longer strips (`library-component-fill`), and the kernel exposes only the pipeline it runs (0019 D1). The requirement's other method, `DetectAPIVersion`, was already removed with the `opm/apiversion` package (commit 4276ec4); the spec text was stale on it. No utility method remains, pinned by the requirement below.
**Migration**: Drop calls to `Kernel.Finalize` / `compile.FinalizeValue`. A consumer that needs a constraint-free export of a value evaluates `v.Syntax(cue.Final())` and rebuilds it in its own `*cue.Context`; the kernel offers no wrapper.

## ADDED Requirements

### Requirement: No Utility Methods on Kernel

The Kernel SHALL expose only the pipeline it runs (acquire, load, process, validate, match, plan, compile) and SHALL NOT expose a finalization, constraint-stripping or other value-utility method.

#### Scenario: No finalization method on the Kernel

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/compile`
- **THEN** neither `Finalize` nor `FinalizeValue` exists, and `compile.Module.Execute` accepts one components value

## MODIFIED Requirements

### Requirement: Phase-Explicit Methods on Kernel

The `Kernel` SHALL expose four phase-explicit methods, each accepting a phase-specific input struct and returning a phase-appropriate result.

#### Scenario: Validate phase method

- **WHEN** a caller invokes `k.Validate(ctx, ValidateInput{Module, ModuleInstance, Values})`
- **THEN** the kernel performs Tier-2 schema validation of `Values` against `Module.Package`'s `#config` by calling `k.ValidateConfig` internally
- **AND** returns nil on success or a CUE-native error wrapped with `fmt.Errorf("module %q: %w", name, err)` on failure
- **AND** does not perform matching or execution

#### Scenario: Match phase method

- **WHEN** a caller invokes `k.Match(ctx, MatchInput{Module, ModuleInstance, Platform})`
- **THEN** the kernel produces a `*MatchPlan` describing matched and non-matched component/transformer pairs
- **AND** does not execute any transformer

#### Scenario: Plan phase method

- **WHEN** a caller invokes `k.Plan(ctx, PlanInput{Module, ModuleInstance, Values, Platform, RuntimeName})`
- **THEN** the kernel runs the full Compile pipeline (Validate + Match + Execute) and returns a `*PlanResult` containing component summaries, unmatched FQNs, ambiguous FQNs, and warnings
- **AND** does not return rendered values

#### Scenario: Compile phase method

- **WHEN** a caller invokes `k.Compile(ctx, CompileInput{Module, ModuleInstance, Values, Platform, RuntimeName})`
- **THEN** the kernel runs the full pipeline (Validate + Match + Execute) and returns a `*CompileResult` containing `Compiled []*core.Compiled`, component summaries, unmatched FQNs, ambiguous FQNs, and warnings
