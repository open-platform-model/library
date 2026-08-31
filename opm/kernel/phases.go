package kernel

import (
	"context"
	"fmt"

	"github.com/open-platform-model/library/opm/compile"
)

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

// Compile runs the full pipeline (Match + Execute) and returns a
// [*CompileResult] containing rendered values, component summaries,
// unmatched FQNs, and warnings.
//
// The instance is rendered as processed: values validation and filling
// happen in [Kernel.ProcessModuleInstance], and Compile performs no
// validation pass of its own.
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

	return k.compileModuleInstance(ctx, in.ModuleInstance, in.Platform, in.RuntimeName)
}
