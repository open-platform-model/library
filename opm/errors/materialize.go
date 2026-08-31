package errors

import (
	"fmt"
)

// MaterializeError kinds. Kind discriminates where in the Materialize flow a
// failure originated so frontends can route diagnostics without string
// matching (D24).
const (
	// MaterializeKindCatalog marks a subscription-resolution failure: a
	// catalog path could not be enumerated, pulled, decoded, or its
	// transformers conflicted. Subscription and Version are populated.
	// It is the only kind Materialize emits.
	MaterializeKindCatalog = "catalog"
)

// MaterializeError is a structured failure from the platform Materialize
// flow. It carries a Kind discriminator plus the subscription path and the
// attempted/resolved version, and wraps the underlying cause so callers can
// reach it via [errors.Unwrap] / [errors.As].
type MaterializeError struct {
	// Kind is MaterializeKindCatalog.
	Kind string

	// Subscription is the catalog subscription path that failed.
	Subscription string

	// Version is the resolved or attempted version, when known.
	Version string

	// Cause is the wrapped underlying error.
	Cause error
}

func (e *MaterializeError) Error() string {
	switch {
	case e.Subscription != "" && e.Version != "":
		return fmt.Sprintf("materialize %s: subscription %q at version %q: %v",
			e.Kind, e.Subscription, e.Version, e.Cause)
	case e.Subscription != "":
		return fmt.Sprintf("materialize %s: subscription %q: %v",
			e.Kind, e.Subscription, e.Cause)
	default:
		return fmt.Sprintf("materialize %s: %v", e.Kind, e.Cause)
	}
}

func (e *MaterializeError) Unwrap() error {
	return e.Cause
}
