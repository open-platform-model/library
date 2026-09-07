package errors

import (
	"fmt"
)

// TransformError indicates transformer execution failed for one matched
// (component, transformer) pair. Unlike the verdict rows, it wraps a real
// cause: the CUE error the pair's output carried, or the kernel's own
// concreteness refusal.
type TransformError struct {
	Component   string
	Transformer string
	Cause       error
}

func (e *TransformError) Error() string {
	return fmt.Sprintf("component %q, transformer %q: %v",
		e.Component, e.Transformer, e.Cause)
}

func (e *TransformError) Unwrap() error {
	return e.Cause
}
