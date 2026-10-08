// Package module defines the Module type, mirroring the #Module definition
// in the OPM core schema. A Module represents the parsed module definition
// before it is built into an instance.
//
// Debug overlays. The CUE schema includes a `debugValues` field on every
// `#Module` for author-supplied example values used by build/validation
// tooling. `debugValues` is a Module field — NOT a separate kernel artifact —
// and it is read with Module.DebugValues. Whether a
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

// DebugValues returns the module's author-supplied debugValues at
// schema.DebugValues on m.Package.
//
// It returns the zero cue.Value (not an error) for a nil receiver or a
// module that declares none; callers test Exists(). The kernel never reads
// it: whether a frontend layers it into its values stack is the frontend's
// own policy.
//
//nolint:revive // method receiver name 'm' is consistent with package convention
func (m *Module) DebugValues() cue.Value {
	if m == nil {
		return cue.Value{}
	}
	return m.Package.LookupPath(schema.DebugValues)
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

	// Source is the module's staged source tree, populated only when the
	// module was acquired through a source-carrying path
	// (Kernel.AcquireModuleFromRegistry, Kernel.AcquireModuleFromDir):
	// always overlay mode, never on-disk. It is nil otherwise. Consumers
	// that must build inside the module's own root — Kernel.SynthesizeInstance
	// — gate on HasSource(). See [Source] for the
	// full two-mode contract shared with Instance and Platform.
	Source *Source `json:"-"`
}

// NewModuleFromValue builds a *Module from a raw CUE artifact value: it
// decodes ModuleMetadata from the value's metadata field and stores the input
// cue.Value unmodified in Package. Errors return a nil *Module — partial
// values are never returned. The returned Module carries no Source.
func NewModuleFromValue(v cue.Value) (*Module, error) {
	meta, err := decodeModuleMetadata(v)
	if err != nil {
		// The decoder's error already names the artifact; a second prefix
		// would read "decoding module metadata: decoding module metadata: ...".
		return nil, err
	}
	return &Module{
		Metadata: meta,
		Package:  v,
	}, nil
}

// decodeModuleMetadata extracts ModuleMetadata from a #Module artifact root.
// A missing metadata field is fatal.
func decodeModuleMetadata(v cue.Value) (*ModuleMetadata, error) {
	metaVal := v.LookupPath(schema.Metadata)
	if !metaVal.Exists() {
		return nil, fmt.Errorf("module metadata field is required")
	}
	meta := &ModuleMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding module metadata: %w", err)
	}
	return meta, nil
}
