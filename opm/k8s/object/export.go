package object

import (
	"encoding/json"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Exported is one Resource after its single export: the JSON bytes CUE
// emitted, the object decoded from those same bytes, and the provenance. It
// holds no CUE value, so it does not pin the build.
type Exported struct {
	// JSON is the CUE export of the value, field order as CUE emits it.
	JSON []byte

	// Object is decoded from JSON, not exported again. Object.Object is
	// never nil.
	Object *unstructured.Unstructured

	// Instance, Component and Transformer are the Resource's provenance.
	Instance    string
	Component   string
	Transformer string
}

// ExportStep names the step of [Export] that failed.
type ExportStep int

const (
	// ExportMarshal is the CUE export: the value would not export to JSON.
	ExportMarshal ExportStep = iota
	// ExportDecode is the decode: the JSON would not decode to an object.
	ExportDecode
)

// String returns "cue export" or "json decode".
func (s ExportStep) String() string {
	switch s {
	case ExportMarshal:
		return "cue export"
	case ExportDecode:
		return "json decode"
	default:
		return fmt.Sprintf("ExportStep(%d)", int(s))
	}
}

// ExportError is the failure of [Export] for one Resource: its position in
// the input, its summary, the step that failed and the cause.
type ExportError struct {
	// Index is the failing Resource's position in the input.
	Index int
	// Resource is the failing Resource's String().
	Resource string
	// Step is the step that failed.
	Step ExportStep
	// Err is the cause.
	Err error
}

// Error names the resource, its position and the step.
func (e *ExportError) Error() string {
	return fmt.Sprintf("export resource %d (%s): %s: %v", e.Index, e.Resource, e.Step, e.Err)
}

// Unwrap returns the cause.
func (e *ExportError) Unwrap() error { return e.Err }

// errNotObject is the cause of a decode that yields no object map.
var errNotObject = errors.New("the exported JSON is not an object")

// Export exports each resource from CUE exactly once, in input order, and
// returns one [Exported] per input, index-aligned. Each object is decoded from
// the bytes of that one export. A value whose JSON is not an object (a list,
// a string or null) fails at [ExportDecode]. Export stops at the first
// failure and returns an *[ExportError].
//
// Export never changes or drops its input. The caller drops its Resources
// after the export to release the CUE build they pin. A nil Resource in the
// input fails at [ExportMarshal].
func Export(resources []*Resource) ([]Exported, error) {
	out := make([]Exported, len(resources))
	for i, r := range resources {
		if r == nil {
			return nil, &ExportError{Index: i, Resource: "<nil>", Step: ExportMarshal, Err: errors.New("nil resource")}
		}
		b, err := r.MarshalJSON()
		if err != nil {
			return nil, &ExportError{Index: i, Resource: r.String(), Step: ExportMarshal, Err: err}
		}
		var obj map[string]any
		if err := json.Unmarshal(b, &obj); err != nil {
			return nil, &ExportError{Index: i, Resource: r.String(), Step: ExportDecode, Err: err}
		}
		if obj == nil {
			return nil, &ExportError{Index: i, Resource: r.String(), Step: ExportDecode, Err: errNotObject}
		}
		out[i] = Exported{
			JSON:        b,
			Object:      &unstructured.Unstructured{Object: obj},
			Instance:    r.Instance,
			Component:   r.Component,
			Transformer: r.Transformer,
		}
	}
	return out, nil
}
