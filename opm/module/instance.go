package module

import (
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
	// so a follow-on build can import the instance as a package. Instances
	// are constructed only by the kernel, which stamps it at exactly two
	// sites: Kernel.SynthesizeInstance (overlay mode, the synthesized package
	// inside the module's staged root) and Kernel.AcquireInstanceFromDir
	// (on-disk mode, the loaded directory; overlay mode with WithValues).
	Source *Source
}

// InstanceMetadata is a re-export of [schema.InstanceMetadata] so callers can
// keep working with `module.InstanceMetadata`.
//
// Was: ReleaseMetadata
type InstanceMetadata = schema.InstanceMetadata

// Components returns the instance's components value as evaluated,
// definition fields (#resources, #traits, #blueprints, #names) included. It
// is a read for frontends and tests: the render build reads the same field
// in CUE, inside the generated glue, and never through this accessor.
func (r *Instance) Components() cue.Value {
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
