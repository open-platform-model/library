package errors

import (
	"fmt"
)

// MissingFQN is the hard structured diagnostic for a demanded primitive FQN
// that no transformer on the materialized platform requires (its #matchers
// bucket is empty). It is distinct from the soft non-match recorded when a
// transformer is found but its requiredLabels are not satisfied.
//
// One MissingFQN is recorded per (Instance, Component, FQN) triple. Alternatives
// lists every primitive FQN sharing the same modulePath/name at other SemVers
// that the platform did materialize, sorted by SemVer — a hint that a different
// version of the same primitive is available.
type MissingFQN struct {
	// Instance is the ModuleInstance name the demanding component belongs to.
	// Was: Release
	Instance string

	// Component is the component name that demanded the FQN.
	Component string

	// FQN is the demanded primitive FQN with no transformer requiring it.
	FQN string

	// Alternatives is the same-modulePath/name FQN set materialized on the
	// platform at other versions, sorted by SemVer. May be empty.
	Alternatives []string
}

func (e MissingFQN) Error() string {
	if len(e.Alternatives) > 0 {
		return fmt.Sprintf("instance %q, component %q: no transformer requires %q (alternatives: %v)",
			e.Instance, e.Component, e.FQN, e.Alternatives)
	}
	return fmt.Sprintf("instance %q, component %q: no transformer requires %q",
		e.Instance, e.Component, e.FQN)
}

// UnifyError is the structured diagnostic for a unification failure between a
// consumer component's primitive body and a candidate transformer's required
// primitive body at the same FQN (the always-unify rung).
//
// Cause carries the CUE error tree verbatim — no Go-side reformatting — so a
// frontend can walk it via [Unwrap] / [errors.As] for a
// cuelang.org/go/cue/errors.Error and render the native
// `conflicting values "X" and "Y": file:line file:line` message.
type UnifyError struct {
	// Component is the component whose body diverged.
	Component string

	// FQN is the primitive FQN at which the bodies conflicted.
	FQN string

	// Cause is the verbatim CUE error tree, reachable via errors.As for
	// cuelang.org/go/cue/errors.Error.
	Cause error
}

func (e UnifyError) Error() string {
	return fmt.Sprintf("component %q, fqn %q: %v", e.Component, e.FQN, e.Cause)
}

func (e UnifyError) Unwrap() error {
	return e.Cause
}
