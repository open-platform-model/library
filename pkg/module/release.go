package module

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/apiversion"
)

// Release is an OPM #ModuleRelease artifact in the unified artifact shape.
//
// Package is the source of truth: it is the concrete, values-filled CUE value
// for the release and every kernel-internal read (components subtree, source
// module, transformer match data) goes through Package.LookupPath via the
// version Binding's Paths().
//
// Metadata is an ergonomic decoded projection of the release-level metadata
// stamped at construction. It is a cache, not a parallel source of truth —
// when Metadata and the corresponding subtree of Package disagree, Package
// wins. Callers MUST NOT mutate Package's apiVersion field after construction.
type Release struct {
	// APIVersion is the OPM schema version detected on the release. Set by
	// NewReleaseFromValue from Package's apiVersion field. The zero value
	// signals an unloaded or hand-constructed release and is rejected by the
	// render entry point.
	APIVersion apiversion.Version

	// Metadata is the decoded release-level metadata cache. May be nil when
	// the metadata could not be decoded.
	Metadata *ReleaseMetadata

	// Package is the loaded, concrete CUE value for the release artifact.
	// Source of truth for every field reachable via the binding's Paths(),
	// including the embedded #module reference at Paths().Module.
	Package cue.Value
}

// MatchComponents returns the schema-preserving components value used for
// matching. The returned value keeps definition fields such as #resources,
// #traits, and #blueprints. Path lookup goes through the binding registered
// for r.APIVersion.
func (r *Release) MatchComponents() cue.Value {
	if r == nil {
		return cue.Value{}
	}
	b, err := api.Lookup(r.APIVersion)
	if err != nil {
		return cue.Value{}
	}
	return r.Package.LookupPath(b.Paths().Components)
}

// ConfigSchema returns the embedded source module's #config schema reachable
// via Paths().Module followed by Paths().Config on r.Package.
//
// All failure modes return the zero cue.Value (not an error): a nil receiver,
// an unregistered binding for r.APIVersion, a missing #module reference, or a
// missing #config definition on the embedded module. Callers detect failure
// via the returned value's Exists() method, mirroring MatchComponents.
func (r *Release) ConfigSchema() cue.Value {
	if r == nil {
		return cue.Value{}
	}
	b, err := api.Lookup(r.APIVersion)
	if err != nil {
		return cue.Value{}
	}
	mod := r.Package.LookupPath(b.Paths().Module)
	if !mod.Exists() {
		return cue.Value{}
	}
	return mod.LookupPath(b.Paths().Config)
}

// The methods below let *Release satisfy api.ReleaseView. They are the surface
// the per-version Binding uses to assemble a transformer context without
// dragging the whole module package's types behind it.

// ReleaseName returns the release's metadata.name.
func (r *Release) ReleaseName() string {
	if r == nil || r.Metadata == nil {
		return ""
	}
	return r.Metadata.Name
}

// Namespace returns the release's metadata.namespace.
func (r *Release) Namespace() string {
	if r == nil || r.Metadata == nil {
		return ""
	}
	return r.Metadata.Namespace
}

// ReleaseUUID returns the release's metadata.uuid.
func (r *Release) ReleaseUUID() string {
	if r == nil || r.Metadata == nil {
		return ""
	}
	return r.Metadata.UUID
}

// ModuleFQN returns the source module's fully qualified name. The value is
// read from Package.LookupPath(binding.Paths().ModuleMetadata).fqn so that
// Package remains the source of truth for module identity.
func (r *Release) ModuleFQN() string {
	return r.lookupModuleMetadataString("fqn")
}

// ModuleVersion returns the source module's version, read from Package via
// the binding's ModuleMetadata path.
func (r *Release) ModuleVersion() string {
	return r.lookupModuleMetadataString("version")
}

// Labels returns the release-level labels (already merged with module labels
// at CUE evaluation time).
func (r *Release) Labels() map[string]string {
	if r == nil || r.Metadata == nil {
		return nil
	}
	return r.Metadata.Labels
}

// Annotations returns the release-level annotations.
func (r *Release) Annotations() map[string]string {
	if r == nil || r.Metadata == nil {
		return nil
	}
	return r.Metadata.Annotations
}

// lookupModuleMetadataString reads a string field under the binding's
// ModuleMetadata path. Returns the empty string on any lookup or decode
// failure — callers treat this as "metadata not available".
func (r *Release) lookupModuleMetadataString(field string) string {
	if r == nil {
		return ""
	}
	b, err := api.Lookup(r.APIVersion)
	if err != nil {
		return ""
	}
	mm := r.Package.LookupPath(b.Paths().ModuleMetadata)
	if !mm.Exists() {
		return ""
	}
	v := mm.LookupPath(cue.ParsePath(field))
	if !v.Exists() {
		return ""
	}
	s, err := v.String()
	if err != nil {
		return ""
	}
	return s
}

// ReleaseMetadata contains release-level identity information.
// Used for inventory tracking, resource labeling, and CLI output.
type ReleaseMetadata struct {
	// Name is the release name (from --name or module.metadata.name).
	Name string `json:"name"`

	// Namespace is the target namespace.
	Namespace string `json:"namespace"`

	// UUID is the release identity UUID.
	// Computed by CUE as SHA1(OPMNamespace, moduleUUID:name:namespace).
	UUID string `json:"uuid"`

	// Labels are the merged release labels (module labels + standard opm labels).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the merged release annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NewReleaseFromValue builds a *Release from a raw CUE artifact value. The
// supplied k is currently unused but preserved in the signature so future
// kernel-scoped state can be threaded without an API break.
//
// The function detects apiVersion via apiversion.Detect, looks up the
// matching Binding, decodes ReleaseMetadata via the binding, stamps APIVersion
// on the returned struct, and stores the input cue.Value unmodified in
// Package.
//
// Errors from any step return a nil *Release. Detection failures wrap
// apiversion.ErrUnknownAPIVersion.
func NewReleaseFromValue(_ CueContextOwner, v cue.Value) (*Release, error) {
	ver, err := apiversion.Detect(v)
	if err != nil {
		return nil, fmt.Errorf("detecting apiVersion: %w", err)
	}
	b, err := api.Lookup(ver)
	if err != nil {
		return nil, fmt.Errorf("resolving binding for %q: %w", ver, err)
	}
	apiMeta, err := b.DecodeReleaseMetadata(v)
	if err != nil {
		return nil, fmt.Errorf("decoding release metadata: %w", err)
	}
	return &Release{
		APIVersion: ver,
		Metadata:   ReleaseMetadataFromAPI(apiMeta),
		Package:    v,
	}, nil
}

// ReleaseMetadataFromAPI converts the binding's canonical api.ReleaseMetadata
// into the pkg/module ReleaseMetadata projection.
func ReleaseMetadataFromAPI(m *api.ReleaseMetadata) *ReleaseMetadata {
	if m == nil {
		return nil
	}
	return &ReleaseMetadata{
		Name:        m.Name,
		Namespace:   m.Namespace,
		UUID:        m.UUID,
		Labels:      m.Labels,
		Annotations: m.Annotations,
	}
}
