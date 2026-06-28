// Package compile executes matched transforms and emits compiled values.
//
// The package is platform-neutral: its public output type is *core.Compiled,
// a CUE value plus provenance. Adapters wrap each Compiled with their
// platform-specific Resource implementation (see opm/core).
package compile

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/core"
	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// ComponentSummary contains display-oriented summary data extracted from a component
// after the compile pipeline. It captures the key properties useful for verbose output
// without exposing cue.Value fields.
type ComponentSummary struct {
	// Name is the component name.
	Name string

	// Labels are the component-level labels from metadata.labels.
	// Example: {"core.opmodel.dev/workload-type": "stateless"}
	Labels map[string]string

	// ResourceFQNs are the FQNs of resource types declared by the component.
	// Sorted for deterministic output.
	// Example: ["opmodel.dev/catalogs/opm/resources/container@v1"]
	ResourceFQNs []string

	// TraitFQNs are the FQNs of traits declared by the component.
	// Sorted for deterministic output.
	// Example: ["opmodel.dev/catalogs/opm/traits/expose@v1"]
	TraitFQNs []string
}

// Module drives the OPM compile pipeline for a single ModuleInstance.
//
// A Module is constructed once per platform and reused across multiple
// Execute calls. It is not safe for concurrent use (CUE context is single-threaded).
type Module struct {
	cueCtx      *cue.Context // build context owned by the caller Kernel
	platform    *materialize.MaterializedPlatform
	runtimeName string // identity of the runtime executing this compile
}

// CompileResult holds the output of a successful Execute call.
type CompileResult struct {
	// Compiled is the ordered list of compiled values from the pipeline.
	// Each entry carries Component and Transformer provenance for inventory
	// tracking. Adapters wrap each Compiled with a platform-specific
	// core.Resource implementation.
	Compiled []*core.Compiled

	// MatchPlan is the decoded result of matching components against transformers.
	// Nil if matching was not performed (e.g. no components).
	MatchPlan *MatchPlan

	// Components is a per-component summary for verbose output, sorted by name.
	Components []ComponentSummary

	// Unmatched is the list of component FQNs that found no matching
	// transformer.
	Unmatched []string

	// Warnings is a list of human-readable advisory messages (e.g. unhandled traits).
	// A non-empty Warnings slice does NOT indicate failure.
	Warnings []string
}

// ModuleResult is an alias for [CompileResult].
//
// Deprecated: use [CompileResult].
type ModuleResult = CompileResult

// NewModule creates a Module for the given materialized platform and runtime
// identity. cueCtx is the caller Kernel's owned build context — Execute builds
// the finalized data, the transformer #context.* view, and the rendered output
// in it, consuming mp.Transformers / mp.Matchers as read-only input (the
// cross-read source) rather than borrowing their context. The Execute path
// reads #transforms off mp.Transformers — the native composed map — so callers
// must Materialize before compiling. runtimeName must be non-empty — the
// catalog requires #context.#runtimeName to be populated and CUE evaluation
// fails on empty values.
func NewModule(cueCtx *cue.Context, mp *materialize.MaterializedPlatform, runtimeName string) *Module {
	return &Module{cueCtx: cueCtx, platform: mp, runtimeName: runtimeName}
}

// Execute runs matched transformers against the provided component views and
// returns rendered values, component summaries, and warnings.
//
// schemaComponents is the non-finalized components value (from inst.MatchComponents())
// preserving CUE definition fields needed for metadata extraction.
// dataComponents is the finalized, constraint-free components value for FillPath injection.
func (r *Module) Execute(
	ctx context.Context,
	inst *module.Instance,
	schemaComponents cue.Value,
	dataComponents cue.Value,
	plan *MatchPlan,
) (*CompileResult, error) {
	if inst == nil {
		return nil, fmt.Errorf("instance is required")
	}
	if r.platform == nil {
		return nil, fmt.Errorf("platform is required")
	}

	// Build in the caller Kernel's context; the platform is read-only input.
	cueCtx := r.cueCtx

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
	// Passes the native composed transformer map (mp.Transformers), built in the
	// owner context so #transforms — including output-local hidden fields —
	// render concrete. Also passes both schemaComponents (for metadata
	// extraction) and dataComponents (already finalized, no materialize() needed).
	compiled, warnings, errs := executeTransforms(
		ctx, cueCtx, plan, r.platform.Transformers,
		schemaComponents, dataComponents, inst,
		r.runtimeName,
	)
	if len(errs) > 0 {
		return nil, fmt.Errorf("executing transforms: %w", errors.Join(errs...))
	}

	allWarnings := nonNilWarnings(plan.Warnings())
	allWarnings = append(allWarnings, warnings...)

	return &CompileResult{
		Compiled:   nonNilCompiled(compiled),
		MatchPlan:  plan,
		Components: nonNilComponentSummaries(extractComponentSummaries(schemaComponents)),
		Unmatched:  []string{},
		Warnings:   allWarnings,
	}, nil
}

func nonNilCompiled(compiled []*core.Compiled) []*core.Compiled {
	if compiled == nil {
		return []*core.Compiled{}
	}
	return compiled
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
// schemaComponents is the value from inst.MatchComponents() which preserves the
// definition fields (#resources, #traits) that carry FQN keys.
func extractComponentSummaries(schemaComponents cue.Value) []ComponentSummary {
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
		if labelsVal := compVal.LookupPath(schema.MetadataLabels); labelsVal.Exists() {
			var labels map[string]string
			if err := labelsVal.Decode(&labels); err == nil {
				summary.Labels = labels
			}
		}

		// Extract #resources keys (definition field — FQN keys).
		if resourcesVal := compVal.LookupPath(schema.ComponentResources); resourcesVal.Exists() {
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
		if traitsVal := compVal.LookupPath(schema.ComponentTraits); traitsVal.Exists() {
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
