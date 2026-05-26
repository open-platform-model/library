// Package module defines the Module type, mirroring the #Module definition
// in the OPM core schema. A Module represents the parsed module definition
// before it is built into a release.
//
// Debug overlays. The CUE schema includes a `debugValues` field on every
// `#Module` for author-supplied example values used by build/validation
// tooling. `debugValues` is a Module field — NOT a separate kernel artifact —
// and it is read off Module.Package via schema.DebugValues. Whether a
// frontend layers debugValues into the values stack is a policy decision
// that lives in the helper layer; the kernel itself never observes the
// distinction.
package module

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/schema"
)

// ConfigSchema returns the module's #config schema reachable via
// schema.Config on m.Package.
//
// All failure modes return the zero cue.Value (not an error): a nil receiver
// or a missing #config definition on the module package. Callers detect
// failure via the returned value's Exists() method.
//
//nolint:revive // method receiver name 'm' is consistent with package convention
func (m *Module) ConfigSchema() cue.Value {
	if m == nil {
		return cue.Value{}
	}
	return m.Package.LookupPath(schema.Config)
}

// Module represents an OPM #Module artifact in the unified artifact shape.
//
// Package is the source of truth: it is the loaded CUE value for the module
// and every kernel-internal read (the #config schema, components subtree)
// goes through Package.LookupPath with paths from opm/schema.
//
// Metadata is an ergonomic decoded projection of the module-level metadata
// stamped at construction. It is a cache, not a parallel source of truth —
// when Metadata and the corresponding subtree of Package disagree, Package
// wins.
type Module struct {
	// Metadata is the decoded module-level metadata cache. Authoritative data
	// lives in Package; Metadata exists for hot-path access (logging, name
	// lookups). May be nil when the metadata could not be decoded.
	Metadata *ModuleMetadata `json:"metadata"`

	// Package is the loaded CUE value for the module artifact. Source of
	// truth for every field reachable via opm/schema's path vars.
	Package cue.Value `json:"-"`
}

// ModuleMetadata is the decoded module-level identity record. It is a
// re-export of [schema.ModuleMetadata] so callers can keep working with
// `module.ModuleMetadata` without taking a transitive dependency on opm/schema
// at every reference site.
//
//nolint:revive // stutter intentional: module.ModuleMetadata reads clearly at call sites
type ModuleMetadata = schema.ModuleMetadata

// CueContextOwner is the minimal context-owner interface accepted by the
// constructor helpers. *kernel.Kernel satisfies it; tests may pass any value
// exposing a *cue.Context. The interface lives in opm/module to keep the
// constructor's import surface free of opm/kernel.
type CueContextOwner interface {
	CueContext() *cue.Context
}

// NewModuleFromValue builds a *Module from a raw CUE artifact value. The
// supplied k is currently unused but preserved in the signature so future
// kernel-scoped state (logger, tracer, clock) can be threaded without an
// API break.
//
// The function decodes ModuleMetadata via schema.DecodeModuleMetadata and
// stores the input cue.Value unmodified in Package. Errors return a nil
// *Module — partial values are never returned.
func NewModuleFromValue(_ CueContextOwner, v cue.Value) (*Module, error) {
	meta, err := schema.DecodeModuleMetadata(v)
	if err != nil {
		return nil, fmt.Errorf("decoding module metadata: %w", err)
	}
	return &Module{
		Metadata: meta,
		Package:  v,
	}, nil
}
