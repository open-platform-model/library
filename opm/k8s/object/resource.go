package object

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/library/opm/kernel"
)

// Resource is a single rendered object with its provenance. It is the
// Kubernetes wrapper of a [kernel.Compiled], field for field.
//
// Value holds the raw CUE output from a transformer, avoiding premature
// conversion to Go-native formats. Callers convert to the format they need
// (JSON, *unstructured.Unstructured) only when needed, through [Export] or
// the conversion methods.
//
// A Resource keeps its whole CUE build alive for as long as it is held. A
// long-lived caller exports and then drops its Resources.
type Resource struct {
	// Value is the CUE value of the rendered object (e.g. a Kubernetes
	// manifest). Concrete and fully evaluated: safe to encode to JSON.
	Value cue.Value

	// Instance is the name of the ModuleInstance that produced this object.
	Instance string

	// Component is the source component name within the instance.
	Component string

	// Transformer is the FQN of the transformer that produced this object.
	Transformer string
}

// NewResource wraps one compiled object, copying its value and provenance.
// A nil input yields a nil result.
func NewResource(c *kernel.Compiled) *Resource {
	if c == nil {
		return nil
	}
	return &Resource{
		Value:       c.Value,
		Instance:    c.Instance,
		Component:   c.Component,
		Transformer: c.Transformer,
	}
}

// Resources wraps a render's compiled objects in order, skipping nil entries.
func Resources(compiled []*kernel.Compiled) []*Resource {
	out := make([]*Resource, 0, len(compiled))
	for _, c := range compiled {
		if r := NewResource(c); r != nil {
			out = append(out, r)
		}
	}
	return out
}

// Kind returns the object's kind (e.g. "Deployment"), or "" when absent.
func (r *Resource) Kind() string {
	s, _ := r.Value.LookupPath(cue.ParsePath("kind")).String() //nolint:errcheck // best-effort; empty on non-concrete
	return s
}

// Name returns the object's metadata.name, or "" when absent.
func (r *Resource) Name() string {
	s, _ := r.Value.LookupPath(cue.ParsePath("metadata.name")).String() //nolint:errcheck // best-effort
	return s
}

// Namespace returns the object's metadata.namespace. Empty for a
// cluster-scoped object or one that names no namespace.
func (r *Resource) Namespace() string {
	s, _ := r.Value.LookupPath(cue.ParsePath("metadata.namespace")).String() //nolint:errcheck // best-effort
	return s
}

// APIVersion returns the object's apiVersion (e.g. "apps/v1"), or "" when
// absent.
func (r *Resource) APIVersion() string {
	s, _ := r.Value.LookupPath(cue.ParsePath("apiVersion")).String() //nolint:errcheck // best-effort
	return s
}

// GVK returns the GroupVersionKind parsed from apiVersion and kind.
func (r *Resource) GVK() schema.GroupVersionKind {
	group, version := parseAPIVersion(r.APIVersion())
	return schema.GroupVersionKind{Group: group, Version: version, Kind: r.Kind()}
}

// Labels returns the object's metadata.labels, or nil when absent or not a
// map of strings.
func (r *Resource) Labels() map[string]string {
	return r.decodeStringMap("metadata.labels")
}

// Annotations returns the object's metadata.annotations, or nil when absent
// or not a map of strings.
func (r *Resource) Annotations() map[string]string {
	return r.decodeStringMap("metadata.annotations")
}

// decodeStringMap decodes a CUE path into a map[string]string.
func (r *Resource) decodeStringMap(path string) map[string]string {
	v := r.Value.LookupPath(cue.ParsePath(path))
	if !v.Exists() {
		return nil
	}
	var m map[string]string
	if err := v.Decode(&m); err != nil {
		return nil
	}
	return m
}

// parseAPIVersion splits "group/version" or "version" into group and version.
func parseAPIVersion(apiVersion string) (group, version string) {
	if idx := strings.LastIndex(apiVersion, "/"); idx >= 0 {
		return apiVersion[:idx], apiVersion[idx+1:]
	}
	return "", apiVersion
}

// String returns a human-readable summary: "Kind/namespace/name", or
// "Kind/name" when there is no namespace.
func (r *Resource) String() string {
	ns := r.Namespace()
	if ns != "" {
		return fmt.Sprintf("%s/%s/%s", r.Kind(), ns, r.Name())
	}
	return fmt.Sprintf("%s/%s", r.Kind(), r.Name())
}
