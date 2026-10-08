package errors

// ConfigValidationError marks values that do not satisfy the configuration
// schema they were validated against: a disallowed field, a type or
// constraint violation, or a required field left non-concrete. The kernel's
// ValidateConfigDetailed returns it, so a frontend tests for the failure
// class with [errors.As] and a *ConfigValidationError target, through any
// number of %w wraps.
//
// It is a marker and nothing more. Err is the CUE error tree exactly as CUE
// and the kernel's disallowed-field walk produced it; Error returns that
// tree's own text and Unwrap returns the tree, so the helpers of
// [cuelang.org/go/cue/errors] (Errors, Positions, Details, Print) give the
// same result on a ConfigValidationError, or on an error that wraps one, as
// on Err itself. The type does not group, re-order or reword the CUE errors,
// and the library ships no walker or formatter beside those helpers.
//
// A source that fails to compile or to load is not marked: that failure is
// not a statement about the values against the schema, and for a file-backed
// source it can be a registry failure, which [Classify] names.
type ConfigValidationError struct {
	// Err is the CUE error tree. It is never nil on an error the library
	// returns.
	Err error
}

// Error returns the text of the wrapped CUE error tree, unchanged.
func (e *ConfigValidationError) Error() string { return e.Err.Error() }

// Unwrap returns the CUE error tree.
func (e *ConfigValidationError) Unwrap() error { return e.Err }
