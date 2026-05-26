package module

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/schema"
)

// Release is an OPM #ModuleRelease artifact in the unified artifact shape.
//
// Package is the source of truth: it is the concrete, values-filled CUE
// value for the release and every kernel-internal read (components subtree,
// source module, transformer match data) goes through Package.LookupPath
// with paths from opm/schema.
//
// Metadata is an ergonomic decoded projection of the release-level metadata
// stamped at construction. It is a cache, not a parallel source of truth —
// when Metadata and the corresponding subtree of Package disagree, Package
// wins.
type Release struct {
	// Metadata is the decoded release-level metadata cache. May be nil when
	// the metadata could not be decoded.
	Metadata *ReleaseMetadata

	// Package is the loaded, concrete CUE value for the release artifact.
	// Source of truth for every field reachable via opm/schema, including
	// the embedded #module reference at schema.Module.
	Package cue.Value
}

// ReleaseMetadata is a re-export of [schema.ReleaseMetadata] so callers can
// keep working with `module.ReleaseMetadata`.
type ReleaseMetadata = schema.ReleaseMetadata

// MatchComponents returns the schema-preserving components value used for
// matching. The returned value keeps definition fields such as #resources,
// #traits, and #blueprints.
func (r *Release) MatchComponents() cue.Value {
	if r == nil {
		return cue.Value{}
	}
	return r.Package.LookupPath(schema.Components)
}

// ConfigSchema returns the embedded source module's #config schema reachable
// via schema.Module followed by schema.Config on r.Package.
//
// All failure modes return the zero cue.Value (not an error): a nil
// receiver, a missing #module reference, or a missing #config definition on
// the embedded module.
func (r *Release) ConfigSchema() cue.Value {
	if r == nil {
		return cue.Value{}
	}
	mod := r.Package.LookupPath(schema.Module)
	if !mod.Exists() {
		return cue.Value{}
	}
	return mod.LookupPath(schema.Config)
}

// The methods below let *Release satisfy schema.ReleaseView — the surface
// that BuildTransformerContext uses to assemble a transformer context
// without dragging opm/module's types behind it.

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
// read from Package.LookupPath(schema.ModuleMetadataPath).fqn so that
// Package remains the source of truth for module identity.
func (r *Release) ModuleFQN() string {
	return r.lookupModuleMetadataString("fqn")
}

// ModuleVersion returns the source module's version, read from Package via
// the ModuleMetadataPath path.
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

// lookupModuleMetadataString reads a string field under the schema's
// ModuleMetadataPath. Returns the empty string on any lookup or decode
// failure — callers treat this as "metadata not available".
func (r *Release) lookupModuleMetadataString(field string) string {
	if r == nil {
		return ""
	}
	mm := r.Package.LookupPath(schema.ModuleMetadataPath)
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

// NewReleaseFromValue builds a *Release from a raw CUE artifact value. The
// supplied k is currently unused but preserved in the signature so future
// kernel-scoped state can be threaded without an API break.
//
// The function decodes ReleaseMetadata via schema.DecodeReleaseMetadata and
// stores the input cue.Value unmodified in Package. Errors return a nil
// *Release.
func NewReleaseFromValue(_ CueContextOwner, v cue.Value) (*Release, error) {
	meta, err := schema.DecodeReleaseMetadata(v)
	if err != nil {
		return nil, fmt.Errorf("decoding release metadata: %w", err)
	}
	return &Release{
		Metadata: meta,
		Package:  v,
	}, nil
}
