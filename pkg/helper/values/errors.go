package values

import (
	"fmt"
	"strings"

	oerrors "github.com/open-platform-model/library/pkg/errors"
)

// LayerError pairs a per-layer schema diagnostic with the layer it came
// from. Frontends consume the slice returned by [MultiSourceError.Errors]
// to format diagnostics in their preferred shape.
type LayerError struct {
	// LayerName mirrors [Layer.Name] (human-friendly).
	LayerName string

	// Source mirrors [Layer.Source] (stable machine identifier).
	Source string

	// Err is the layer's underlying [*oerrors.ConfigError]. Non-nil for
	// every entry returned from [MultiSourceError.Errors].
	Err *oerrors.ConfigError
}

// MultiSourceError aggregates per-layer Tier-1 validation failures from
// [ValidateAndUnify]. It carries one [LayerError] per failing layer in
// stack order so frontends can render every problem at once.
type MultiSourceError struct {
	errors []LayerError
}

// Error returns a human-readable summary listing every failing layer and
// the underlying [*oerrors.ConfigError] block for each.
func (e *MultiSourceError) Error() string {
	if e == nil {
		return "<nil>"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "values validation failed in %d layer(s):\n", len(e.errors))
	for _, le := range e.errors {
		if le.Source != "" {
			fmt.Fprintf(&sb, "layer %q (source %q):\n", le.LayerName, le.Source)
		} else {
			fmt.Fprintf(&sb, "layer %q:\n", le.LayerName)
		}
		fmt.Fprintf(&sb, "%s\n", le.Err.Error())
	}
	return strings.TrimRight(sb.String(), "\n")
}

// Errors returns the structured per-layer entries, in stack order. The
// returned slice MUST NOT be mutated.
func (e *MultiSourceError) Errors() []LayerError {
	if e == nil {
		return nil
	}
	return e.errors
}

// Unwrap exposes the underlying [*oerrors.ConfigError]s so stdlib
// `errors.Is` / `errors.As` walks reach them.
func (e *MultiSourceError) Unwrap() []error {
	if e == nil {
		return nil
	}
	out := make([]error, 0, len(e.errors))
	for _, le := range e.errors {
		out = append(out, le.Err)
	}
	return out
}
