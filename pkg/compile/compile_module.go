package compile

import (
	"context"
	"fmt"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/platform"
)

// CompileModuleRelease compiles a prepared release with the given platform.
// The release must already be fully prepared via module.ParseModuleRelease.
//
// runtimeName identifies the runtime executing this compile (e.g. "opm-cli",
// "opm-controller", "compose-runtime"). It MUST be non-empty — the catalog
// declares #context.#runtimeName as mandatory and CUE evaluation fails on
// empty values.
//
// The binding for the release's apiVersion is looked up internally via
// api.Lookup(rel.APIVersion); callers do not pass a binding. If the release
// and platform declare different apiVersions, CompileModuleRelease returns
// an error before invoking any transformer.
//
// Deprecated: use Kernel.Compile.
func CompileModuleRelease(
	ctx context.Context,
	rel *module.Release,
	plat *platform.Platform,
	runtimeName string,
) (*CompileResult, error) {
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

	dataComponents, err := FinalizeValue(plat.Package.Context(), schemaComponents)
	if err != nil {
		return nil, fmt.Errorf("finalizing components: %w", err)
	}

	plan, err := Match(schemaComponents, plat, binding)
	if err != nil {
		return nil, err
	}

	return NewModule(plat, runtimeName).Execute(ctx, rel, schemaComponents, dataComponents, plan)
}
