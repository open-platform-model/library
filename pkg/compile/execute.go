package compile

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/core"
	"github.com/open-platform-model/library/pkg/module"
)

// executeTransforms runs the CUE #transform for each matched (component, transformer)
// pair in the plan and returns the rendered values.
//
// schemaComponents is the original (non-finalized) components value — used for
// reading definition fields (metadata.labels, metadata.annotations) for #context.
// dataComponents is the finalized, constraint-free components value — used for
// FillPath injection into transformer #transform without schema conflicts.
//
// Execution is sequential: *cue.Context is not goroutine-safe.
// Resources are returned in the deterministic order produced by MatchedPairs().
// Per-pair errors are collected and returned alongside any successful resources.
func executeTransforms(
	ctx context.Context,
	cueCtx *cue.Context,
	plan *MatchPlan,
	platformVal cue.Value,
	schemaComponents cue.Value,
	dataComponents cue.Value,
	rel *module.Release,
	runtimeName string,
	binding api.Binding,
) ([]*core.Rendered, []string, []error) {
	rendered := make([]*core.Rendered, 0)
	var warnings []string
	var errs []error

	for _, pair := range plan.MatchedPairs() {
		select {
		case <-ctx.Done():
			return rendered, warnings, append(errs, ctx.Err())
		default:
		}

		res, pairWarnings, err := executePair(cueCtx, platformVal, schemaComponents, dataComponents, rel, pair, runtimeName, binding)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		rendered = append(rendered, res...)
		warnings = append(warnings, pairWarnings...)
	}

	return rendered, warnings, errs
}

// executePair runs the CUE #transform for a single (component, transformer) matched pair.
//
// The flow:
//  1. Look up the transformer's #transform from Platform.#composedTransformers.
//  2. Look up the component from dataComponents (already finalized — no constraints).
//  3. FillPath #component with the data component value directly (no materialize needed).
//  4. FillPath #context.* fields (#moduleReleaseMetadata, #componentMetadata, #runtimeName).
//     Metadata is read from schemaComponents which preserves definition fields.
//  5. Look up and decode the output field.
func executePair(
	cueCtx *cue.Context,
	platformVal cue.Value,
	schemaComponents cue.Value,
	dataComponents cue.Value,
	rel *module.Release,
	pair MatchedPair,
	runtimeName string,
	binding api.Binding,
) ([]*core.Rendered, []string, error) {
	compName := pair.ComponentName
	tfFQN := pair.TransformerFQN
	paths := binding.Paths()

	// Retrieve the transformer's #transform definition from
	// Platform.#composedTransformers.
	transformVal := platformVal.
		LookupPath(paths.ComposedTransformers).
		LookupPath(cue.MakePath(cue.Str(tfFQN))).
		LookupPath(paths.Transform)

	if !transformVal.Exists() {
		return nil, nil, fmt.Errorf("component %q / transformer %q: #transform not found in platform.#composedTransformers", compName, tfFQN)
	}
	if err := transformVal.Err(); err != nil {
		return nil, nil, fmt.Errorf("component %q / transformer %q: #transform error: %w", compName, tfFQN, err)
	}

	// Retrieve the finalized (constraint-free) component value from dataComponents.
	// No materialize() round-trip needed — components were finalized at load time.
	dataComp := dataComponents.LookupPath(cue.MakePath(cue.Str(compName)))
	if !dataComp.Exists() {
		return nil, nil, fmt.Errorf("component %q not found in data components value", compName)
	}

	// Retrieve the schema component value for metadata extraction (#context injection).
	// schemaComponents preserves definition fields that are stripped by finalization.
	schemaComp := schemaComponents.LookupPath(cue.MakePath(cue.Str(compName)))

	// Inject #component using the finalized data value — safe for FillPath without
	// schema constraint conflicts.
	unified := transformVal.FillPath(paths.Component, dataComp)
	if err := unified.Err(); err != nil {
		return nil, nil, fmt.Errorf("component %q / transformer %q: filling #component: %w", compName, tfFQN, err)
	}

	// Build and inject #context. The binding owns the shape; the renderer
	// only fills the resulting value at Paths().Context.
	ctxVal, warnings, err := binding.BuildTransformerContext(cueCtx, rel, compName, schemaComp, runtimeName)
	if err != nil {
		return nil, nil, fmt.Errorf("component %q / transformer %q: injecting #context: %w", compName, tfFQN, err)
	}
	unified = unified.FillPath(paths.Context, ctxVal)
	if err := unified.Err(); err != nil {
		return nil, nil, fmt.Errorf("component %q / transformer %q: filling #context: %w", compName, tfFQN, err)
	}

	// Extract the output field.
	outputVal := unified.LookupPath(paths.Output)
	if !outputVal.Exists() {
		return []*core.Rendered{}, warnings, nil
	}
	if err := outputVal.Err(); err != nil {
		return nil, nil, fmt.Errorf("component %q / transformer %q: evaluating output: %w", compName, tfFQN, err)
	}

	releaseName := rel.Metadata.Name

	// Decode the output into rendered items. Two supported forms:
	//   1. List of items   — cue.ListKind
	//   2. Map of items    — cue.StructKind keyed by stable id
	//
	// Singleton outputs MUST wrap themselves in a one-element list. The
	// pipeline does not auto-detect singletons because the heuristic
	// ("apiVersion + kind at root") is k8s-shape-specific and would
	// misclassify outputs of other platforms (compose service, nomad job,
	// terraform resource, ...).
	switch outputVal.Kind() {
	case cue.ListKind:
		res, err := collectRenderedList(outputVal, releaseName, compName, tfFQN)
		return res, warnings, err
	case cue.StructKind:
		res, err := collectRenderedMap(outputVal, releaseName, compName, tfFQN)
		return res, warnings, err
	default:
		return nil, nil, fmt.Errorf(
			"component %q / transformer %q: unexpected output kind %s (must be list or struct)",
			compName, tfFQN, outputVal.Kind(),
		)
	}
}

// collectRenderedList wraps each item in a CUE list as a Rendered,
// keeping the CUE value intact without any intermediate decoding.
func collectRenderedList(v cue.Value, releaseName, compName, tfFQN string) ([]*core.Rendered, error) {
	var rendered []*core.Rendered
	iter, err := v.List()
	if err != nil {
		return nil, fmt.Errorf("component %q / transformer %q: iterating output list: %w", compName, tfFQN, err)
	}
	for iter.Next() {
		rendered = append(rendered, &core.Rendered{
			Value: iter.Value(), Release: releaseName,
			Component: compName, Transformer: tfFQN,
		})
	}
	return rendered, nil
}

// collectRenderedMap wraps each field value in a CUE struct as a Rendered,
// keeping the CUE value intact without any intermediate decoding.
func collectRenderedMap(v cue.Value, releaseName, compName, tfFQN string) ([]*core.Rendered, error) {
	var rendered []*core.Rendered
	iter, err := v.Fields()
	if err != nil {
		return nil, fmt.Errorf(
			"component %q / transformer %q: iterating output map: %w",
			compName, tfFQN, err,
		)
	}
	for iter.Next() {
		rendered = append(rendered, &core.Rendered{
			Value: iter.Value(), Release: releaseName,
			Component: compName, Transformer: tfFQN,
		})
	}
	return rendered, nil
}
