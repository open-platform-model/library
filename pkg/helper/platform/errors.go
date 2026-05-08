package platform

import (
	"fmt"
	"strings"
)

// MultiFulfillerError is returned by [Compose] when two or more registered
// Modules' transformers claim the same primitive FQN, violating catalog
// enhancement 014 D13.
//
// FQN, ConflictingModules, and ConflictingTransformers carry the parsed
// attribution when the underlying CUE diagnostic could be classified. They
// MAY be empty when classification fell back to wrapping the raw error;
// in that case [Unwrap] returns the original CUE diagnostic so callers can
// still surface it. Frontends should prefer the structured fields when
// non-empty and fall through to err.Error() / Unwrap() otherwise.
type MultiFulfillerError struct {
	// FQN is the offending primitive FQN (e.g. "example.com/r/echo@v0").
	// Empty when the underlying CUE error could not be parsed.
	FQN string

	// ConflictingModules names the Modules whose transformers contributed
	// to the conflict, in registry order. Empty when not extractable.
	ConflictingModules []string

	// ConflictingTransformers names the transformer FQNs that claim the
	// offending primitive. Empty when not extractable.
	ConflictingTransformers []string

	// rawErr is the original CUE evaluation error. Surfaced via Unwrap so
	// errors.Is/As walks reach it; included in Error() when the structured
	// fields are empty.
	rawErr error
}

// Error renders a human-readable diagnostic. When FQN is set, the message
// names the offending primitive and the contributing Modules / transformers
// when available. Otherwise it falls back to the raw CUE diagnostic.
func (e *MultiFulfillerError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.FQN == "" {
		if e.rawErr != nil {
			return fmt.Sprintf("multi-fulfiller violation: %s", e.rawErr.Error())
		}
		return "multi-fulfiller violation"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "multi-fulfiller violation for %q", e.FQN)
	if len(e.ConflictingModules) > 0 {
		fmt.Fprintf(&b, " (modules: %s)", strings.Join(e.ConflictingModules, ", "))
	}
	if len(e.ConflictingTransformers) > 0 {
		fmt.Fprintf(&b, " (transformers: %s)", strings.Join(e.ConflictingTransformers, ", "))
	}
	return b.String()
}

// Unwrap exposes the wrapped CUE diagnostic for errors.Is / errors.As walks.
func (e *MultiFulfillerError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.rawErr
}
