// Package render executes matched transforms and emits rendered values.
//
// The package is platform-neutral: its public output type is *core.Rendered,
// a CUE value plus provenance. Adapters wrap each Rendered with their
// platform-specific Resource implementation (see pkg/core).
package render

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/core"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/provider"
)

// ComponentSummary contains display-oriented summary data extracted from a component
// after the render pipeline. It captures the key properties useful for verbose output
// without exposing cue.Value fields.
type ComponentSummary struct {
	// Name is the component name.
	Name string

	// Labels are the component-level labels from metadata.labels.
	// Example: {"core.opmodel.dev/workload-type": "stateless"}
	Labels map[string]string

	// ResourceFQNs are the FQNs of resource types declared by the component.
	// Sorted for deterministic output.
	// Example: ["opmodel.dev/opm/v1alpha1/resources/workload/container@v1"]
	ResourceFQNs []string

	// TraitFQNs are the FQNs of traits declared by the component.
	// Sorted for deterministic output.
	// Example: ["opmodel.dev/opm/v1alpha1/traits/network/expose@v1"]
	TraitFQNs []string
}

// Module drives the OPM render pipeline for a single ModuleRelease.
//
// A Module is constructed once per provider and reused across multiple
// Execute calls. It is not safe for concurrent use (CUE context is single-threaded).
type Module struct {
	provider    *provider.Provider
	runtimeName string // identity of the runtime executing this render
}

// ModuleResult holds the output of a successful Execute call.
type ModuleResult struct {
	// Rendered is the ordered list of rendered values from the pipeline.
	// Each entry carries Component and Transformer provenance for inventory
	// tracking. Adapters wrap each Rendered with a platform-specific
	// core.Resource implementation.
	Rendered []*core.Rendered

	// MatchPlan is the decoded result of matching components against transformers.
	// Nil if matching was not performed (e.g. no components).
	MatchPlan *MatchPlan

	// Components is a per-component summary for verbose output, sorted by name.
	Components []ComponentSummary

	// Warnings is a list of human-readable advisory messages (e.g. unhandled traits).
	// A non-empty Warnings slice does NOT indicate failure.
	Warnings []string
}

// NewModule creates a Module for the given provider and runtime identity.
// runtimeName must be non-empty — the catalog requires #context.#runtimeName
// to be populated and CUE evaluation fails on empty values.
//
// Deprecated: use Kernel.NewRenderModule. The Kernel is the public anchor
// type for all OPM runtime operations.
func NewModule(p *provider.Provider, runtimeName string) *Module {
	return &Module{provider: p, runtimeName: runtimeName}
}

// Execute runs matched transformers against the provided component views and
// returns rendered values, component summaries, and warnings.
//
// schemaComponents is the non-finalized components value (from rel.MatchComponents())
// preserving CUE definition fields needed for metadata extraction.
// dataComponents is the finalized, constraint-free components value for FillPath injection.
//
// The api.Binding for rel.APIVersion is looked up internally so callers do
// not have to thread one through. ProcessModuleRelease has typically already
// performed the same lookup; the small duplicate cost (a map read) keeps the
// public Execute signature stable.
func (r *Module) Execute(
	ctx context.Context,
	rel *module.Release,
	schemaComponents cue.Value,
	dataComponents cue.Value,
	plan *MatchPlan,
) (*ModuleResult, error) {
	if rel == nil {
		return nil, fmt.Errorf("release is required")
	}
	binding, err := api.Lookup(rel.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("resolving binding for release %q: %w", rel.Metadata.Name, err)
	}

	// The CUE context lives on each cue.Value — extract it from the provider.
	cueCtx := r.provider.Data.Context()

	if plan == nil {
		return nil, fmt.Errorf("match plan is required")
	}

	// Error on unmatched components — these cannot be rendered.
	if len(plan.Unmatched) > 0 {
		return nil, &UnmatchedComponentsError{
			Components: plan.Unmatched,
			Matches:    plan.Matches,
		}
	}

	// Phase 2 — execution (CUE #transform per pair).
	// Passes both schemaComponents (for metadata extraction) and dataComponents
	// (already finalized, no materialize() needed).
	rendered, warnings, errs := executeTransforms(
		ctx, cueCtx, plan, r.provider.Data,
		schemaComponents, dataComponents, rel,
		r.runtimeName, binding,
	)
	if len(errs) > 0 {
		return nil, fmt.Errorf("executing transforms: %w", errors.Join(errs...))
	}

	allWarnings := nonNilWarnings(plan.Warnings())
	allWarnings = append(allWarnings, warnings...)

	return &ModuleResult{
		Rendered:   nonNilRendered(rendered),
		MatchPlan:  plan,
		Components: nonNilComponentSummaries(extractComponentSummaries(schemaComponents, binding)),
		Warnings:   allWarnings,
	}, nil
}

func nonNilRendered(rendered []*core.Rendered) []*core.Rendered {
	if rendered == nil {
		return []*core.Rendered{}
	}
	return rendered
}

func nonNilComponentSummaries(components []ComponentSummary) []ComponentSummary {
	if components == nil {
		return []ComponentSummary{}
	}
	return components
}

func nonNilWarnings(warnings []string) []string {
	if warnings == nil {
		return []string{}
	}
	return warnings
}

// extractComponentSummaries iterates the schemaComponents CUE value and builds
// a sorted []ComponentSummary for verbose output.
//
// schemaComponents is the value from rel.MatchComponents() which preserves the
// definition fields (#resources, #traits) that carry FQN keys.
func extractComponentSummaries(schemaComponents cue.Value, b api.Binding) []ComponentSummary {
	paths := b.Paths()
	iter, err := schemaComponents.Fields()
	if err != nil {
		return nil
	}

	var summaries []ComponentSummary
	for iter.Next() {
		compName := iter.Selector().Unquoted()
		compVal := iter.Value()

		summary := ComponentSummary{Name: compName}

		// Extract metadata.labels (optional field).
		if labelsVal := compVal.LookupPath(paths.MetadataLabels); labelsVal.Exists() {
			var labels map[string]string
			if err := labelsVal.Decode(&labels); err == nil {
				summary.Labels = labels
			}
		}

		// Extract #resources keys (definition field — FQN keys).
		if resourcesVal := compVal.LookupPath(paths.ComponentResources); resourcesVal.Exists() {
			var fqns []string
			ri, err := resourcesVal.Fields()
			if err == nil {
				for ri.Next() {
					fqns = append(fqns, ri.Selector().Unquoted())
				}
			}
			sort.Strings(fqns)
			summary.ResourceFQNs = fqns
		}

		// Extract #traits keys (definition field — FQN keys). Optional.
		if traitsVal := compVal.LookupPath(paths.ComponentTraits); traitsVal.Exists() {
			var fqns []string
			ti, err := traitsVal.Fields()
			if err == nil {
				for ti.Next() {
					fqns = append(fqns, ti.Selector().Unquoted())
				}
			}
			sort.Strings(fqns)
			summary.TraitFQNs = fqns
		}

		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})
	return summaries
}
