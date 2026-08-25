package compile

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/core"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// executeTransforms runs the CUE #transform for each matched (component, transformer)
// pair in the plan and returns the rendered values.
//
// schemaComponents is the instance's evaluated components value: the same value
// Match reads. Each pair's #component is filled from it with every field class
// intact (definitions such as #names, hidden fields, constraints), and #context
// metadata is read from it (0019 D1).
//
// Execution is sequential: *cue.Context is not goroutine-safe.
// Resources are returned in the deterministic order produced by MatchedPairs().
// Per-pair errors are collected and returned alongside any successful resources.
func executeTransforms(
	ctx context.Context,
	cueCtx *cue.Context,
	plan *MatchPlan,
	composedVal cue.Value,
	schemaComponents cue.Value,
	inst *module.Instance,
	runtimeName string,
) ([]*core.Compiled, []string, []error) {
	compiled := make([]*core.Compiled, 0)
	var warnings []string
	var errs []error

	for _, pair := range plan.MatchedPairs() {
		select {
		case <-ctx.Done():
			return compiled, warnings, append(errs, ctx.Err())
		default:
		}

		res, pairWarnings, err := executePair(cueCtx, composedVal, schemaComponents, inst, pair, runtimeName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		compiled = append(compiled, res...)
		warnings = append(warnings, pairWarnings...)
	}

	return compiled, warnings, errs
}

// executePair runs the CUE #transform for a single (component, transformer) matched pair.
//
// composedVal is the native #composedTransformers map (FQN → #ComponentTransformer)
// from materialize.MaterializedPlatform.Transformers, built in the owner context.
// Reading a #transform off it — including output-local hidden fields — renders
// concrete (the federation guarantee; see that field's docs and ADR-003).
//
// The flow:
//  1. Look up the transformer's #transform from the composed map (by FQN).
//  2. Look up the component in schemaComponents, the instance's evaluated
//     components value as Match reads it (no finalized copy).
//  3. FillPath #component with that value as-is: definition fields (#names,
//     #instance, ...), hidden fields and constraints all reach the transformer,
//     exactly as plain CUE unification would give them (0019 D1).
//  4. FillPath #context.* fields (#moduleInstanceMetadata, #componentMetadata, #runtimeName),
//     read from the same component value.
//  5. Look up and decode the output field.
func executePair(
	cueCtx *cue.Context,
	composedVal cue.Value,
	schemaComponents cue.Value,
	inst *module.Instance,
	pair MatchedPair,
	runtimeName string,
) ([]*core.Compiled, []string, error) {
	compName := pair.ComponentName
	tfFQN := pair.TransformerFQN

	// Retrieve the transformer's #transform from the open composed map.
	transformVal := composedVal.
		LookupPath(cue.MakePath(cue.Str(tfFQN))).
		LookupPath(schema.Transform)

	if !transformVal.Exists() {
		return nil, nil, fmt.Errorf("component %q / transformer %q: #transform not found in #composedTransformers", compName, tfFQN)
	}
	if err := transformVal.Err(); err != nil {
		return nil, nil, fmt.Errorf("component %q / transformer %q: #transform error: %w", compName, tfFQN, err)
	}

	// Retrieve the component from the evaluated components value: the one
	// value Match read, with every field class preserved.
	schemaComp := schemaComponents.LookupPath(cue.MakePath(cue.Str(compName)))
	if !schemaComp.Exists() {
		return nil, nil, fmt.Errorf("component %q not found in components value", compName)
	}

	// Inject #component with the evaluated value directly (0019 D1).
	unified := transformVal.FillPath(schema.Component, schemaComp)
	if err := unified.Err(); err != nil {
		return nil, nil, fmt.Errorf("component %q / transformer %q: filling #component: %w", compName, tfFQN, err)
	}

	// Build and inject #context. opm/schema owns the shape; the renderer
	// only fills the resulting value at schema.Context.
	ctxVal, warnings, err := schema.BuildTransformerContext(cueCtx, inst, compName, schemaComp, runtimeName)
	if err != nil {
		return nil, nil, fmt.Errorf("component %q / transformer %q: injecting #context: %w", compName, tfFQN, err)
	}
	unified = unified.FillPath(schema.Context, ctxVal)
	if err := unified.Err(); err != nil {
		return nil, nil, fmt.Errorf("component %q / transformer %q: filling #context: %w", compName, tfFQN, err)
	}

	// Extract the output field.
	outputVal := unified.LookupPath(schema.Output)
	if !outputVal.Exists() {
		return []*core.Compiled{}, warnings, nil
	}
	if err := outputVal.Err(); err != nil {
		return nil, nil, fmt.Errorf("component %q / transformer %q: evaluating output: %w", compName, tfFQN, err)
	}

	// #ComponentTransformer.#transform.output is either a single resource
	// (struct) or a list of resources (list). Dispatch on cue.Kind:
	//   StructKind → one Compiled, Value = the whole struct verbatim
	//   ListKind   → one Compiled per item, Value = the list element verbatim
	// The renderer never inspects fields inside Value — apply-layer code
	// (binding-specific) is responsible for interpreting the resource shape.
	instanceName := inst.Metadata.Name
	switch outputVal.Kind() {
	case cue.StructKind:
		return []*core.Compiled{{
			Value:       outputVal,
			Instance:    instanceName,
			Component:   compName,
			Transformer: tfFQN,
		}}, warnings, nil
	case cue.ListKind:
		iter, err := outputVal.List()
		if err != nil {
			return nil, nil, fmt.Errorf(
				"component %q / transformer %q: iterating output list: %w",
				compName, tfFQN, err,
			)
		}
		var compiled []*core.Compiled
		for iter.Next() {
			compiled = append(compiled, &core.Compiled{
				Value:       iter.Value(),
				Instance:    instanceName,
				Component:   compName,
				Transformer: tfFQN,
			})
		}
		return compiled, warnings, nil
	default:
		return nil, nil, fmt.Errorf(
			"component %q / transformer %q: unexpected output kind %s (must be struct for a single resource or list for multiple)",
			compName, tfFQN, outputVal.Kind(),
		)
	}
}
