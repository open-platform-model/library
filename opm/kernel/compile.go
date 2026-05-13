package kernel

import (
	"context"
	"fmt"

	"github.com/open-platform-model/library/opm/api"
	"github.com/open-platform-model/library/opm/compile"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
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
	plat *platform.Platform,
	runtimeName string,
) (*compile.CompileResult, error) {
	if runtimeName == "" {
		return nil, fmt.Errorf("runtimeName must be non-empty")
	}
	if rel == nil {
		return nil, fmt.Errorf("release is required")
	}
	if plat == nil {
		return nil, fmt.Errorf("platform is required")
	}
	if rel.APIVersion != plat.APIVersion {
		return nil, fmt.Errorf(
			"apiVersion mismatch: release %q has %q but platform %q has %q",
			rel.Metadata.Name, rel.APIVersion, plat.Metadata.Name, plat.APIVersion,
		)
	}

	binding, err := api.Lookup(rel.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("resolving binding for release %q: %w", rel.Metadata.Name, err)
	}

	schemaComponents := rel.MatchComponents()
	if !schemaComponents.Exists() {
		return nil, fmt.Errorf("release %q: no components field in release spec", rel.Metadata.Name)
	}

	dataComponents, err := compile.FinalizeValue(plat.Package.Context(), schemaComponents)
	if err != nil {
		return nil, fmt.Errorf("finalizing components: %w", err)
	}

	plan, err := compile.Match(schemaComponents, plat, binding)
	if err != nil {
		return nil, err
	}

	return compile.NewModule(plat, runtimeName).Execute(ctx, rel, schemaComponents, dataComponents, plan) //nolint:staticcheck // SA1019: compile.NewModule constructor is on its own deprecation arc; replacing it is out of scope for this change.
}
