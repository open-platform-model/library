package errors

import (
	"fmt"
)

// TransformError indicates transformer execution failed.
type TransformError struct {
	ComponentName  string
	TransformerFQN string
	Cause          error
}

func (e *TransformError) Error() string {
	return fmt.Sprintf("component %q, transformer %q: %v",
		e.ComponentName, e.TransformerFQN, e.Cause)
}

func (e *TransformError) Unwrap() error {
	return e.Cause
}
