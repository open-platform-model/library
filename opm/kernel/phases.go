package kernel

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/compile"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// Validate performs Tier-2 schema validation of [ValidateInput.Values]
// against the module's `#config` schema. It does NOT perform matching,
// execution, or finalization.
//
// Returns nil on success. On failure the returned error wraps the raw CUE
// error tree with the module name as a `module %q:` prefix; callers can
// reach the underlying [cuelang.org/go/cue/errors.Error] tree via
// [errors.As] / [cuelang.org/go/cue/errors.Errors] for structured access.
func (k *Kernel) Validate(_ context.Context, in ValidateInput) error {
	if in.Module == nil {
		return fmt.Errorf("ValidateInput.Module is required")
	}
	if in.ModuleInstance == nil {
		return fmt.Errorf("ValidateInput.ModuleInstance is required")
	}
	if !in.Values.Exists() {
		return nil
	}
	configSchema := in.Module.Package.LookupPath(schema.Config)
	if !configSchema.Exists() {
		return nil
	}
	name := instanceDisplayName(in.ModuleInstance)
	if _, vErr := k.ValidateConfig(configSchema, in.Values); vErr != nil {
		return fmt.Errorf("module %q: %w", name, vErr)
	}
	return nil
}

// Match produces a [*MatchPlan] describing matched and non-matched
// component / transformer pairs. It does NOT execute any transformer.
func (k *Kernel) Match(_ context.Context, in MatchInput) (*MatchPlan, error) {
	if in.ModuleInstance == nil {
		return nil, fmt.Errorf("MatchInput.ModuleInstance is required")
	}
	if in.Platform == nil {
		return nil, fmt.Errorf("MatchInput.Platform is required")
	}
	components := in.ModuleInstance.MatchComponents()
	if !components.Exists() {
		return nil, fmt.Errorf("instance %q: no components field in instance spec", in.ModuleInstance.Metadata.Name)
	}
	return compile.Match(components, in.Platform, in.ModuleInstance.Metadata.Name)
}

// Plan runs Validate + Match + Execute (dry-run) and returns a
// [*PlanResult] containing component summaries, unmatched FQNs, ambiguous
// FQNs, and warnings. It does NOT return rendered values.
//
// Internally Plan reuses [Compile] and discards the rendered slice. This
// keeps Plan and Compile pinned to a single execution path so a Plan that
// succeeds gives the caller strong confidence that a subsequent Compile
// will also succeed.
func (k *Kernel) Plan(ctx context.Context, in PlanInput) (*PlanResult, error) {
	if in.ModuleInstance == nil {
		return nil, fmt.Errorf("PlanInput.ModuleInstance is required")
	}
	if in.Platform == nil {
		return nil, fmt.Errorf("PlanInput.Platform is required")
	}
	if in.RuntimeName == "" {
		return nil, fmt.Errorf("PlanInput.RuntimeName must be non-empty")
	}

	out, err := k.Compile(ctx, CompileInput(in))
	if err != nil {
		return nil, err
	}

	return &PlanResult{
		MatchPlan:  out.MatchPlan,
		Components: nonNilComponentSummaries(out.Components),
		Unmatched:  nonNilStrings(out.Unmatched),
		Warnings:   nonNilStrings(out.Warnings),
	}, nil
}

// Compile runs the full pipeline (Validate + Match + Execute + Finalize)
// and returns a [*CompileResult] containing rendered values, component
// summaries, unmatched FQNs, and warnings.
//
// The Tier-2 #config schema validation is sourced from the embedded #module
// reference on `in.ModuleInstance.Package` (see [module.Instance.ConfigSchema]).
// No standalone `*module.Module` is required.
func (k *Kernel) Compile(ctx context.Context, in CompileInput) (*CompileResult, error) {
	if in.ModuleInstance == nil {
		return nil, fmt.Errorf("CompileInput.ModuleInstance is required")
	}
	if in.Platform == nil {
		return nil, fmt.Errorf("CompileInput.Platform is required")
	}
	if in.RuntimeName == "" {
		return nil, fmt.Errorf("CompileInput.RuntimeName must be non-empty")
	}

	mod, err := moduleFromInstance(k, in.ModuleInstance)
	if err != nil {
		return nil, err
	}
	if err := k.Validate(ctx, ValidateInput{
		Module:         mod,
		ModuleInstance: in.ModuleInstance,
		Values:         in.Values,
	}); err != nil {
		return nil, err
	}

	return k.compileModuleInstance(ctx, in.ModuleInstance, in.Platform, in.RuntimeName)
}

// moduleFromInstance synthesizes a transient *module.Module from the embedded
// #module reference on inst.Package. The transient view is used by Compile to
// satisfy the [ValidateInput.Module] contract while the slim CompileInput
// surface drops the parallel field.
//
// No defensive fallback: production instances produced by
// [Kernel.ProcessModuleInstance] always embed #module at schema.Module, and
// hand-built test fixtures must do the same. A missing embedded module is a
// programming error and is surfaced loudly so it cannot mask a malformed
// fixture or a regressed parser.
func moduleFromInstance(k *Kernel, inst *module.Instance) (*module.Module, error) {
	embedded := inst.Package.LookupPath(schema.Module)
	if !embedded.Exists() {
		return nil, fmt.Errorf("instance %q: embedded #module reference not found at %q", instanceDisplayName(inst), schema.Module)
	}
	return module.NewModuleFromValue(k, embedded)
}

// Finalize converts v to its finalized, constraint-free form using the
// kernel's [*cue.Context]. See [compile.FinalizeValue].
func (k *Kernel) Finalize(v cue.Value) (cue.Value, error) {
	return compile.FinalizeValue(k.cueCtx, v)
}

func instanceDisplayName(inst *module.Instance) string {
	if inst == nil || inst.Metadata == nil || inst.Metadata.Name == "" {
		return "<unknown>"
	}
	return inst.Metadata.Name
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func nonNilComponentSummaries(s []compile.ComponentSummary) []compile.ComponentSummary {
	if s == nil {
		return []compile.ComponentSummary{}
	}
	return s
}
