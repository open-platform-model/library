package errors

import (
	"fmt"
)

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

// UnresolvedDemand is the structured diagnostic for a demanded contract key
// the platform does not resolve (0010 D28): the matcher index holds no
// candidate for it, or every candidate was disqualified (by unification or by
// predicate). Every declared resource is a required demand; a trait demand is
// unresolved only when its effective `optional` posture is load-bearing —
// including the fail-closed case of a posture the catalog never stated.
//
// The D4 contract-key diagnostic is carried by Alternatives: empty means
// nothing on this platform implements the contract at any version; non-empty
// means the contract base is implemented at a different apiVersion only.
type UnresolvedDemand struct {
	// Component is the component whose demand went unresolved.
	Component string

	// FQN is the demanded contract key.
	FQN string

	// Kind is "resource" or "trait".
	Kind string

	// Alternatives is the same-base contract-key set the platform does
	// implement, in contract-key order (kube-aware apiVersion ladder,
	// D34/D4). Empty when nothing implements the contract.
	Alternatives []string

	// Disqualified carries, when candidates existed, the unify failures that
	// disqualified them. Predicate-disqualified candidates contribute no
	// entry (there is no unify cause to carry).
	Disqualified []UnifyError

	// UnstatedPosture marks a trait demand that failed closed because its
	// effective `optional` was not concrete — the declaring catalog stated
	// no posture, so the trait is treated as load-bearing (D28).
	UnstatedPosture bool
}

func (e UnresolvedDemand) Error() string {
	msg := fmt.Sprintf("component %q: unresolved %s demand %q", e.Component, e.Kind, e.FQN)
	if len(e.Alternatives) > 0 {
		msg += fmt.Sprintf(": implemented at a different apiVersion (alternatives: %v)", e.Alternatives)
	} else {
		msg += ": nothing on this platform implements this contract"
	}
	if len(e.Disqualified) > 0 {
		msg += fmt.Sprintf("; %d candidate(s) disqualified", len(e.Disqualified))
	}
	if e.UnstatedPosture {
		msg += "; trait states no optional posture (treated as load-bearing)"
	}
	return msg
}

// UnresolvedDemandsError aggregates every unresolved demand of a plan into
// the typed error Render fails with through the gate (D28). Each demand is reachable via
// Unwrap() []error for errors.As routing.
type UnresolvedDemandsError struct {
	// Demands is the full unresolved-demand set, in plan order.
	Demands []UnresolvedDemand
}

func (e *UnresolvedDemandsError) Error() string {
	msg := fmt.Sprintf("%d unresolved demand(s):", len(e.Demands))
	for _, d := range e.Demands {
		msg += "\n  " + d.Error()
	}
	return msg
}

// Unwrap exposes each UnresolvedDemand so callers can route on the value type
// via errors.As.
func (e *UnresolvedDemandsError) Unwrap() []error {
	errs := make([]error, 0, len(e.Demands))
	for _, d := range e.Demands {
		errs = append(errs, d)
	}
	return errs
}
