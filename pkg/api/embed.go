package api

import (
	"fmt"
	"io/fs"

	"github.com/open-platform-model/library/pkg/apiversion"
)

// EmbeddedSchema returns the embedded CUE schema filesystem for v, looked up
// through the registered binding. Returns a non-nil error if no binding is
// registered for v, or if the binding does not ship an embed (returns nil from
// Binding.EmbeddedSchema).
//
// Callers can use the returned fs.FS with cuelang.org/go/cue/load to drive
// schema validation without depending on CUE_REGISTRY.
func EmbeddedSchema(v apiversion.Version) (fs.FS, error) {
	b, err := Lookup(v)
	if err != nil {
		return nil, err
	}
	embedded := b.EmbeddedSchema()
	if embedded == nil {
		return nil, fmt.Errorf("no embedded schema available for %q", v)
	}
	return embedded, nil
}
