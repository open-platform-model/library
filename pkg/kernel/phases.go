package kernel

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/apiversion"
	"github.com/open-platform-model/library/pkg/compile"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/validate"
)

// Validate performs Tier-2 schema validation of [ValidateInput.Values]
// against the module's `#config` schema. It does NOT perform matching,
// execution, or finalization.
//
// Returns nil on success. On failure the returned error is a
// [*oerrors.ConfigError] carrying grouped CUE diagnostics; callers can use
// [errors.As] to extract it.
func (k *Kernel) Validate(_ context.Context, in ValidateInput) error {
	if in.Module == nil {
		return fmt.Errorf("ValidateInput.Module is required")
	}
	if in.ModuleRelease == nil {
		return fmt.Errorf("ValidateInput.ModuleRelease is required")
	}
	if !in.Values.Exists() {
		return nil
	}
	b, err := api.Lookup(in.Module.APIVersion)
	if err != nil {
		return fmt.Errorf("resolving binding for %q: %w", in.Module.APIVersion, err)
	}
	schema := in.Module.Package.LookupPath(b.Paths().Config)
	if !schema.Exists() {
		return nil
	}
	name := releaseDisplayName(in.ModuleRelease)
	if _, cfgErr := validate.Config(schema, []cue.Value{in.Values}, "module", name); cfgErr != nil { //nolint:staticcheck // SA1019: kernel method wraps the deprecated free function
		return cfgErr
	}
	return nil
}

// Match produces a [*MatchPlan] describing matched and non-matched
// component / transformer pairs. It does NOT execute any transformer.
func (k *Kernel) Match(_ context.Context, in MatchInput) (*MatchPlan, error) {
	if in.ModuleRelease == nil {
		return nil, fmt.Errorf("MatchInput.ModuleRelease is required")
	}
	if in.Platform == nil {
		return nil, fmt.Errorf("MatchInput.Platform is required")
	}
	if in.ModuleRelease.APIVersion != in.Platform.APIVersion {
		return nil, fmt.Errorf(
			"apiVersion mismatch: release %q has %q but platform %q has %q",
			in.ModuleRelease.Metadata.Name, in.ModuleRelease.APIVersion,
			in.Platform.Metadata.Name, in.Platform.APIVersion,
		)
	}
	b, err := api.Lookup(in.ModuleRelease.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("resolving binding for %q: %w", in.ModuleRelease.APIVersion, err)
	}
	components := in.ModuleRelease.MatchComponents()
	if !components.Exists() {
		return nil, fmt.Errorf("release %q: no components field in release spec", in.ModuleRelease.Metadata.Name)
	}
	return compile.Match(components, in.Platform, b)
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
	if in.Module == nil {
		return nil, fmt.Errorf("PlanInput.Module is required")
	}
	if in.ModuleRelease == nil {
		return nil, fmt.Errorf("PlanInput.ModuleRelease is required")
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
		Ambiguous:  nonNilStrings(out.Ambiguous),
		Warnings:   nonNilStrings(out.Warnings),
	}, nil
}

// Compile runs the full pipeline (Validate + Match + Execute + Finalize)
// and returns a [*CompileResult] containing rendered values, component
// summaries, unmatched FQNs, ambiguous FQNs, and warnings.
func (k *Kernel) Compile(ctx context.Context, in CompileInput) (*CompileResult, error) {
	if in.Module == nil {
		return nil, fmt.Errorf("CompileInput.Module is required")
	}
	if in.ModuleRelease == nil {
		return nil, fmt.Errorf("CompileInput.ModuleRelease is required")
	}
	if in.Platform == nil {
		return nil, fmt.Errorf("CompileInput.Platform is required")
	}
	if in.RuntimeName == "" {
		return nil, fmt.Errorf("CompileInput.RuntimeName must be non-empty")
	}

	if err := k.Validate(ctx, ValidateInput{
		Module:        in.Module,
		ModuleRelease: in.ModuleRelease,
		Values:        in.Values,
	}); err != nil {
		return nil, err
	}

	return compile.CompileModuleRelease(ctx, in.ModuleRelease, in.Platform, in.RuntimeName) //nolint:staticcheck // SA1019: compile.CompileModuleRelease is the underlying implementation for this method
}

// DetectAPIVersion reads the apiVersion literal from the root of v and
// returns the matching [apiversion.Version]. Delegates to
// [apiversion.Detect]; exposed as a method so callers find the operation
// through the Kernel anchor.
func (k *Kernel) DetectAPIVersion(v cue.Value) (apiversion.Version, error) {
	return apiversion.Detect(v)
}

// Finalize converts v to its finalized, constraint-free form using the
// kernel's [*cue.Context]. See [compile.FinalizeValue].
func (k *Kernel) Finalize(v cue.Value) (cue.Value, error) {
	return compile.FinalizeValue(k.cueCtx, v)
}

func releaseDisplayName(rel *module.Release) string {
	if rel == nil || rel.Metadata == nil || rel.Metadata.Name == "" {
		return "<unknown>"
	}
	return rel.Metadata.Name
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
