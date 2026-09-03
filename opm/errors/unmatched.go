package errors

import (
	"fmt"
	"strings"
)

// MatchResult is the per-(component, transformer) verdict carried on
// [UnmatchedComponentsError.Matches]: whether the candidate matched and, when
// the label predicate refused it, which required labels were missing.
type MatchResult struct {
	Matched       bool     `json:"matched"`
	MissingLabels []string `json:"missingLabels"`
}

// UnmatchedComponentsError is the render gate's refusal for components no
// transformer matched. It carries the per-component match matrix so a
// frontend can list which candidates were evaluated and why each was
// refused; the render build reports the unmatched set as data
// (kernel.RenderDiagnostics.Unmatched) and the kernel raises this error
// through the fail-closed gate, reachable via errors.As from
// *kernel.RenderError.
//
// Each unmatched component is surfaced as a *TransformError via Unwrap(),
// enabling callers to use errors.As for typed handling of individual failures.
type UnmatchedComponentsError struct {
	// Components is the list of component names with no matching transformer.
	Components []string

	// Matches is the full match result matrix, used to build per-component diagnostics.
	Matches map[string]map[string]MatchResult
}

func (e *UnmatchedComponentsError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d component(s) have no matching transformer: %v\n",
		len(e.Components), e.Components)

	for _, compName := range e.Components {
		tfResults, ok := e.Matches[compName]
		if !ok {
			continue
		}
		fmt.Fprintf(&sb, "  component %q:\n", compName)
		for tfFQN, result := range tfResults {
			if result.Matched {
				continue
			}
			fmt.Fprintf(&sb, "    transformer %q did not match:\n", tfFQN)
			if len(result.MissingLabels) > 0 {
				fmt.Fprintf(&sb, "      missing labels:    %v\n", result.MissingLabels)
			}
		}
	}

	return sb.String()
}

// Unwrap returns a slice of *TransformError — one per unmatched component —
// so that callers can use errors.As to extract per-component failure details.
//
// Each TransformError carries the component name and the first non-matching
// transformer FQN. Its Cause is a plain terminal error describing the failure,
// not a nested UnmatchedComponentsError, to prevent infinite recursion when
// errors.As traverses the chain.
func (e *UnmatchedComponentsError) Unwrap() []error {
	errs := make([]error, 0, len(e.Components))
	for _, compName := range e.Components {
		compMatches := e.Matches[compName]
		// Find the first non-matching transformer FQN for context.
		// If the component had no transformers evaluated, leave TransformerFQN empty.
		tfFQN := ""
		for fqn, result := range compMatches {
			if !result.Matched {
				tfFQN = fqn
				break
			}
		}
		errs = append(errs, &TransformError{
			ComponentName:  compName,
			TransformerFQN: tfFQN,
			Cause:          fmt.Errorf("component %q has no matching transformer", compName),
		})
	}
	return errs
}
