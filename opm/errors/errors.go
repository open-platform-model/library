// Package errors provides sentinel errors and structured error types for OPM.
//
// Configuration validation errors are CUE-native — see
// [cuelang.org/go/cue/errors] for the canonical interface and helpers
// (Errors, Positions, Print). The library does not wrap CUE diagnostics
// in custom Go-typed projections, nor does it ship a presentation-layer
// formatter; frontends walk the CUE error tree and render however their
// consumer requires.
package errors

import (
	"fmt"
)

// Wrap wraps an error with a sentinel error type.
func Wrap(sentinel error, message string) error {
	return fmt.Errorf("%s: %w", message, sentinel)
}
