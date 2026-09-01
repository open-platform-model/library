package module

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/schema"
)

// Instance is an OPM #ModuleInstance artifact in the unified artifact shape.
//
// Package is the source of truth: it is the concrete, values-filled CUE
// value for the instance and every kernel-internal read (components subtree,
// source module, transformer match data) goes through Package.LookupPath
// with paths from opm/schema.
//
// Metadata is an ergonomic decoded projection of the instance-level metadata
// stamped at construction. It is a cache, not a parallel source of truth —
// when Metadata and the corresponding subtree of Package disagree, Package
// wins.
//
// Was: Release
type Instance struct {
	// Metadata is the decoded instance-level metadata cache. May be nil when
	// the metadata could not be decoded.
	Metadata *InstanceMetadata

	// Package is the loaded, concrete CUE value for the instance artifact.
	// Source of truth for every field reachable via opm/schema, including
	// the embedded #module reference at schema.Module.
	Package cue.Value

	// Source is the staged source tree the instance package was built from,
	// so a follow-on build can import the instance as a package. It is
	// stamped at exactly two sites: Kernel.SynthesizeInstance (overlay mode,
	// the synthesized package inside the module's staged root) and
	// Kernel.AcquireInstanceFromDir (on-disk mode, the loaded directory). It
	// is nil for instances constructed from a bare value
	// (NewInstanceFromValue, Kernel.ProcessModuleInstance called directly).
	Source *Source
}

// InstanceMetadata is a re-export of [schema.InstanceMetadata] so callers can
// keep working with `module.InstanceMetadata`.
//
// Was: ReleaseMetadata
type InstanceMetadata = schema.InstanceMetadata

// MatchComponents returns the schema-preserving components value used for
// matching. The returned value keeps definition fields such as #resources,
// #traits, and #blueprints.
func (r *Instance) MatchComponents() cue.Value {
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
func (r *Instance) ConfigSchema() cue.Value {
	if r == nil {
		return cue.Value{}
	}
	mod := r.Package.LookupPath(schema.Module)
	if !mod.Exists() {
		return cue.Value{}
	}
	return mod.LookupPath(schema.Config)
}

// The methods below let *Instance satisfy schema.InstanceView — the surface
// that BuildTransformerContext uses to assemble a transformer context
// without dragging opm/module's types behind it.

// InstanceName returns the instance's metadata.name.
//
// Was: ReleaseName
func (r *Instance) InstanceName() string {
	if r == nil || r.Metadata == nil {
		return ""
	}
	return r.Metadata.Name
}

// Namespace returns the instance's metadata.namespace.
func (r *Instance) Namespace() string {
	if r == nil || r.Metadata == nil {
		return ""
	}
	return r.Metadata.Namespace
}

// InstanceUUID returns the instance's metadata.uuid.
//
// Was: ReleaseUUID
func (r *Instance) InstanceUUID() string {
	if r == nil || r.Metadata == nil {
		return ""
	}
	return r.Metadata.UUID
}

// InstanceFQN returns the instance's OWN fully qualified name — metadata.fqn
// (registryPath:name:namespace, core v2), decoded into the metadata cache.
// Distinct from the source module's identity, which remains reachable through
// Package at the schema.ModuleMetadataPath path.
func (r *Instance) InstanceFQN() string {
	if r == nil || r.Metadata == nil {
		return ""
	}
	return r.Metadata.FQN
}

// ModuleVersion returns the source module's version, read from Package via
// the ModuleMetadataPath path.
func (r *Instance) ModuleVersion() string {
	return r.lookupModuleMetadataString("version")
}

// Labels returns the instance-level labels (already merged with module labels
// at CUE evaluation time).
func (r *Instance) Labels() map[string]string {
	if r == nil || r.Metadata == nil {
		return nil
	}
	return r.Metadata.Labels
}

// Annotations returns the instance-level annotations.
func (r *Instance) Annotations() map[string]string {
	if r == nil || r.Metadata == nil {
		return nil
	}
	return r.Metadata.Annotations
}

// lookupModuleMetadataString reads a string field under the schema's
// ModuleMetadataPath. Returns the empty string on any lookup or decode
// failure — callers treat this as "metadata not available".
func (r *Instance) lookupModuleMetadataString(field string) string {
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

// NewInstanceFromValue builds a *Instance from a raw CUE artifact value. The
// supplied k is currently unused but preserved in the signature so future
// kernel-scoped state can be threaded without an API break.
//
// The function decodes InstanceMetadata via schema.DecodeInstanceMetadata and
// stores the input cue.Value unmodified in Package. Errors return a nil
// *Instance.
//
// Was: NewReleaseFromValue
func NewInstanceFromValue(_ CueContextOwner, v cue.Value) (*Instance, error) {
	meta, err := schema.DecodeInstanceMetadata(v)
	if err != nil {
		return nil, fmt.Errorf("decoding instance metadata: %w", err)
	}
	return &Instance{
		Metadata: meta,
		Package:  v,
	}, nil
}
