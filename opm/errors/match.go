package errors

import (
	"fmt"
)

// UnifyRefusal is one row of the always-unify rung: a candidate transformer
// whose required primitive bodies conflict with the component's own bodies.
// It is data, not an error — the rung's verdicts are carried on the render
// diagnostics and aggregated into a gate cause, never raised on their own.
//
// The conflict is reported as the FQNs it occurred at, not as a CUE error
// tree: the glue decides the verdict inside the build and a CUE error is not
// exportable from there (0019 D10).
type UnifyRefusal struct {
	// Component is the component whose bodies diverged.
	Component string

	// Transformer is the FQN of the candidate that was disqualified.
	Transformer string

	// Conflicts are the primitive FQNs at which the bodies conflicted, in
	// the build's order.
	Conflicts []string
}

// UnresolvedDemand is the structured diagnostic for a demanded contract key
// the platform does not resolve (0010 D28): the matcher index holds no
// candidate for it, or every candidate was disqualified (by unification or by
// predicate). Every declared resource is a required demand; a trait demand is
// unresolved only when its effective `optional` posture is load-bearing (a
// trait whose posture the catalog never stated is a build error, not a
// diagnostics row).
//
// The D4 contract-key diagnostic is carried by Alternatives: empty means
// nothing on this platform implements the contract at any version; non-empty
// means the contract base is implemented at a different apiVersion only.
// DefinedBy carries the 0015 D18 arm beside it: which enabled catalog lists
// the demanded key in its contract maps, so the refusal can say "defined by
// this catalog and implemented by nothing" rather than "unknown".
//
// It is data, not an error; [UnresolvedDemandsError] is the gate cause.
type UnresolvedDemand struct {
	// Component is the component whose demand went unresolved.
	Component string

	// FQN is the demanded contract key.
	FQN string

	// Kind is "resource" or "trait".
	Kind string

	// Alternatives is the same-base contract-key set the platform does
	// implement, in contract-key order (kube-aware apiVersion ladder,
	// D34/D4, applied inside the build). Empty when nothing implements the
	// contract.
	Alternatives []string

	// Disqualified carries, when candidates existed, the unify refusals that
	// disqualified them. Predicate-disqualified candidates contribute no
	// entry (there is no unify conflict to carry).
	Disqualified []UnifyRefusal

	// DefinedBy is the registry key (module path) of the enabled catalog
	// whose contract maps list the demanded key, read inside the build from
	// the platform's derived contract inventory (#contracts.definedBy,
	// enhancement 0015 D18) and never parsed off the FQN. Empty when no
	// enabled catalog lists it. Diagnostic only: its presence or absence
	// never changes whether the demand refuses.
	DefinedBy string
}

// describe renders one unresolved demand as a line of an aggregate's
// message, wording one of three cases: implemented at a different apiVersion
// (alternatives listed, the defining catalog named when one lists the key);
// defined by a named catalog and implemented by nothing; or no enabled
// catalog defines the contract at all.
func (d UnresolvedDemand) describe() string {
	msg := fmt.Sprintf("component %q: unresolved %s demand %q", d.Component, d.Kind, d.FQN)
	switch {
	case len(d.Alternatives) > 0 && d.DefinedBy != "":
		msg += fmt.Sprintf(": defined by %q, implemented at a different apiVersion (alternatives: %v)", d.DefinedBy, d.Alternatives)
	case len(d.Alternatives) > 0:
		msg += fmt.Sprintf(": implemented at a different apiVersion (alternatives: %v)", d.Alternatives)
	case d.DefinedBy != "":
		msg += fmt.Sprintf(": defined by %q and nothing on this platform implements it", d.DefinedBy)
	default:
		msg += ": no enabled catalog defines this contract"
	}
	if len(d.Disqualified) > 0 {
		msg += fmt.Sprintf("; %d candidate(s) disqualified", len(d.Disqualified))
	}
	return msg
}

// UnresolvedDemandsError aggregates every unresolved demand of a render into
// the typed cause Render fails with through the fail-closed gate (D28). It
// carries the diagnostics' rows unchanged and wraps nothing: a row is data,
// so there is no cause of another kind underneath.
type UnresolvedDemandsError struct {
	// Demands is the full unresolved-demand set, in build order.
	Demands []UnresolvedDemand
}

func (e *UnresolvedDemandsError) Error() string {
	msg := fmt.Sprintf("%d unresolved demand(s):", len(e.Demands))
	for _, d := range e.Demands {
		msg += "\n  " + d.describe()
	}
	return msg
}
