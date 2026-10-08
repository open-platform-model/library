package module

import (
	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/internal/corepath"
	"github.com/open-platform-model/library/opm/schema"
)

// Instance is an OPM #ModuleInstance artifact in the unified artifact shape.
//
// Package is the source of truth: it is the concrete, values-filled CUE
// value for the instance. Every kernel-internal read goes through
// Package.LookupPath with paths from opm/schema, and the accessors
// (Components, ConfigSchema, Values, ModuleMetadata) are those same reads,
// offered to frontends so they need not repeat the paths.
//
// Metadata is an ergonomic decoded projection of the instance-level metadata
// stamped at construction. It is a cache, not a parallel source of truth —
// when Metadata and the corresponding subtree of Package disagree, Package
// wins.
type Instance struct {
	// Metadata is the decoded instance-level metadata cache. May be nil when
	// the metadata could not be decoded.
	Metadata *InstanceMetadata

	// Package is the loaded, concrete CUE value for the instance artifact.
	// Source of truth for every field reachable via opm/schema, including
	// the embedded #module reference at schema.Module.
	Package cue.Value

	// Source is the staged source tree the instance package was built from,
	// so a follow-on build can import the instance as a package. Instances
	// are constructed only by the kernel, which stamps it at exactly two
	// sites: Kernel.SynthesizeInstance (overlay mode, the synthesized package
	// inside the module's staged root) and Kernel.AcquireInstanceFromDir
	// (on-disk mode, the loaded directory; overlay mode when values sources are layered on).
	Source *Source
}

// Components returns the instance's components value as evaluated,
// definition fields (#resources, #traits, #blueprints, #names) included. It
// is a read for frontends and tests: the render build reads the same field
// in CUE, inside the generated glue, and never through this accessor.
func (r *Instance) Components() cue.Value {
	if r == nil {
		return cue.Value{}
	}
	return r.Package.LookupPath(corepath.Components)
}

// ConfigSchema returns the embedded source module's #config schema reachable
// via schema.Module followed by corepath.Config on r.Package.
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
	return mod.LookupPath(corepath.Config)
}

// Values returns the instance's merged values at corepath.Values on
// r.Package, as evaluated.
//
// It returns the zero cue.Value (not an error) for a nil receiver or an
// instance with no values field; callers test Exists().
func (r *Instance) Values() cue.Value {
	if r == nil {
		return cue.Value{}
	}
	return r.Package.LookupPath(corepath.Values)
}

// ModuleMetadata returns the metadata of the module this instance was built
// from, decoded from the embedded #module (schema.Module) on r.Package.
//
// It returns nil for a nil receiver, an instance with no #module, or
// metadata that does not decode. The decode is all or nothing: there is no
// partial result. Each call decodes afresh from Package; the returned struct
// is a copy, and mutating it does not change Package.
func (r *Instance) ModuleMetadata() *ModuleMetadata {
	if r == nil {
		return nil
	}
	mod := r.Package.LookupPath(schema.Module)
	if !mod.Exists() {
		return nil
	}
	meta, err := decodeModuleMetadata(mod)
	if err != nil {
		return nil
	}
	return meta
}
