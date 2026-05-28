package kernel

import (
	"context"
	"fmt"

	"github.com/open-platform-model/library/opm/compile"
	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/module"
)

// compileModuleRelease is the canonical compile pipeline implementation. It
// runs the full Match → Finalize → Execute sequence against the supplied
// release and platform and returns the resulting [*compile.CompileResult].
//
// Callers go through [Kernel.Compile] (which adds Tier-2 validation in front
// of this helper). This function lives on the kernel package so the kernel
// owns the canonical impl rather than wrapping a deprecated free function.
func compileModuleRelease(
	ctx context.Context,
	rel *module.Release,
	mp *materialize.MaterializedPlatform,
	runtimeName string,
) (*compile.CompileResult, error) {
	if runtimeName == "" {
		return nil, fmt.Errorf("runtimeName must be non-empty")
	}
	if rel == nil {
		return nil, fmt.Errorf("release is required")
	}
	if mp == nil {
		return nil, fmt.Errorf("platform is required")
	}

	schemaComponents := rel.MatchComponents()
	if !schemaComponents.Exists() {
		return nil, fmt.Errorf("release %q: no components field in release spec", rel.Metadata.Name)
	}

	dataComponents, err := compile.FinalizeValue(mp.Package.Context(), schemaComponents)
	if err != nil {
		return nil, fmt.Errorf("finalizing components: %w", err)
	}

	plan, err := compile.Match(schemaComponents, mp, rel.Metadata.Name)
	if err != nil {
		return nil, err
	}

	return compile.NewModule(mp, runtimeName).Execute(ctx, rel, schemaComponents, dataComponents, plan) //nolint:staticcheck // SA1019: compile.NewModule constructor is on its own deprecation arc; replacing it is out of scope for this change.
}
