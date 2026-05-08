package compile

import (
	"context"
	"fmt"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/provider"
)

// CompileModuleRelease compiles a prepared release with the given provider.
// The release must already be fully prepared via module.ParseModuleRelease.
//
// runtimeName identifies the runtime executing this compile (e.g. "opm-cli",
// "opm-controller", "compose-runtime"). It MUST be non-empty — the catalog
// declares #context.#runtimeName as mandatory and CUE evaluation fails on
// empty values.
//
// The binding for the release's apiVersion is looked up internally via
// api.Lookup(rel.APIVersion); callers do not pass a binding. If the release
// and provider declare different apiVersions, CompileModuleRelease returns
// an error before invoking any transformer.
//
// Deprecated: use Kernel.Compile.
func CompileModuleRelease(
	ctx context.Context,
	rel *module.Release,
	p *provider.Provider,
	runtimeName string,
) (*CompileResult, error) {
	if runtimeName == "" {
		return nil, fmt.Errorf("runtimeName must be non-empty")
	}
	if rel == nil {
		return nil, fmt.Errorf("release is required")
	}
	if p == nil {
		return nil, fmt.Errorf("provider is required")
	}
	if rel.APIVersion != p.APIVersion {
		return nil, fmt.Errorf(
			"apiVersion mismatch: release %q has %q but provider %q has %q",
			rel.Metadata.Name, rel.APIVersion, p.Metadata.Name, p.APIVersion,
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

	dataComponents, err := FinalizeValue(p.Data.Context(), schemaComponents)
	if err != nil {
		return nil, fmt.Errorf("finalizing components: %w", err)
	}

	plan, err := Match(schemaComponents, p, binding)
	if err != nil {
		return nil, err
	}

	return NewModule(p, runtimeName).Execute(ctx, rel, schemaComponents, dataComponents, plan)
}

// ProcessModuleRelease is an alias for [CompileModuleRelease].
//
// Deprecated: use [CompileModuleRelease] or [Kernel.Compile].
func ProcessModuleRelease(
	ctx context.Context,
	rel *module.Release,
	p *provider.Provider,
	runtimeName string,
) (*ModuleResult, error) {
	return CompileModuleRelease(ctx, rel, p, runtimeName)
}
