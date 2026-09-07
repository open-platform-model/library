package errors

import (
	"fmt"
	"strings"
)

// CandidateVerdict is one candidate the render build's demand walk reached
// for a component: whether it matched and, when the label predicate refused
// it, which required labels the component's matchLabels lacked or carried
// with a different value. A candidate the always-unify rung refused appears
// unmatched with no missing labels; its conflict is a [UnifyRefusal] on the
// render diagnostics. It is data, not an error.
type CandidateVerdict struct {
	// Transformer is the candidate's FQN.
	Transformer string

	// Matched is the combined verdict of the unify and predicate rungs.
	Matched bool

	// MissingLabels are the required labels the predicate rung found
	// missing or divergent, in the build's order.
	MissingLabels []string
}

// UnmatchedComponent is one component no transformer matched, with every
// candidate the demand walk reached for it. It is data, not an error;
// [UnmatchedComponentsError] is the gate cause.
type UnmatchedComponent struct {
	// Component is the component name.
	Component string

	// Candidates are the verdicts on every transformer the build evaluated
	// for the component, in transformer order. Empty when no transformer
	// was a candidate at all (the component's demands are then on the
	// unresolved rows).
	Candidates []CandidateVerdict
}

// UnmatchedComponentsError is the render gate's refusal for components no
// transformer matched. It carries the diagnostics' rows unchanged, so a
// frontend can list which candidates were evaluated and why each was
// refused, and wraps nothing: a row is data, so there is no cause of another
// kind underneath. Reachable via errors.As from *kernel.RenderError.
type UnmatchedComponentsError struct {
	// Components is the unmatched set, in build order.
	Components []UnmatchedComponent
}

func (e *UnmatchedComponentsError) Error() string {
	names := make([]string, 0, len(e.Components))
	for _, c := range e.Components {
		names = append(names, c.Component)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%d component(s) have no matching transformer: %v\n", len(e.Components), names)
	for _, c := range e.Components {
		fmt.Fprintf(&sb, "  component %q:\n", c.Component)
		for _, v := range c.Candidates {
			if v.Matched {
				continue
			}
			fmt.Fprintf(&sb, "    transformer %q did not match:\n", v.Transformer)
			if len(v.MissingLabels) > 0 {
				fmt.Fprintf(&sb, "      missing labels:    %v\n", v.MissingLabels)
			}
		}
	}
	return sb.String()
}
