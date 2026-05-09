// Package module defines the Module and ModuleMetadata types, mirroring the
// #Module definition in the CUE catalog. A Module represents the parsed module
// definition before it is built into a release.
//
// Debug overlays. The CUE schema includes a `debugValues` field on every
// `#Module` for author-supplied example values used by build/validation
// tooling. `debugValues` is a Module field — NOT a separate kernel artifact —
// and it is read off Module.Package via the binding path
// api.Paths().DebugValues. Whether a frontend layers debugValues into the
// values stack is a policy decision that lives in the helper layer; the
// kernel itself never observes the distinction. The previously contemplated
// top-level `#ModuleDebug` artifact is retired (see enhancement
// 001-kernel-redesign-around-platform D6 and the retire-module-debug change).
package module

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/apiversion"
)

// ConfigSchema returns the module's #config schema reachable via Paths().Config
// on m.Package.
//
// All failure modes return the zero cue.Value (not an error): a nil receiver,
// an unregistered binding for m.APIVersion, or a missing #config definition on
// the module package. Callers detect failure via the returned value's Exists()
// method, mirroring [Release.ConfigSchema] and [Release.MatchComponents].
//
//nolint:revive // method receiver name 'm' is consistent with package convention
func (m *Module) ConfigSchema() cue.Value {
	if m == nil {
		return cue.Value{}
	}
	b, err := api.Lookup(m.APIVersion)
	if err != nil {
		return cue.Value{}
	}
	return m.Package.LookupPath(b.Paths().Config)
}

// Module represents an OPM #Module artifact in the unified artifact shape.
//
// Package is the source of truth: it is the loaded CUE value for the module
// and every kernel-internal read (e.g. the #config schema, components subtree)
// goes through Package.LookupPath via the version Binding's Paths().
//
// Metadata is an ergonomic decoded projection of the module-level metadata
// stamped at construction. It is a cache, not a parallel source of truth —
// when Metadata and the corresponding subtree of Package disagree, Package
// wins. Callers MUST NOT mutate Package's apiVersion field after construction;
// the APIVersion field is stamped from Package and would otherwise drift.
type Module struct {
	// APIVersion is the OPM schema version detected on the module artifact.
	// Set by NewModuleFromValue from Package's apiVersion field. The zero
	// value signals an unloaded or hand-constructed module.
	APIVersion apiversion.Version `json:"apiVersion,omitempty"`

	// Metadata is the decoded module-level metadata cache. Authoritative data
	// lives in Package; Metadata exists for hot-path access (logging, name
	// lookups). May be nil when the metadata could not be decoded.
	Metadata *ModuleMetadata `json:"metadata"`

	// Package is the loaded CUE value for the module artifact. Source of
	// truth for every field reachable via the binding's Paths().
	Package cue.Value `json:"-"`
}

// ModuleMetadata contains module-level identity and version information.
// This is the module's canonical metadata, distinct from the release it is deployed as.
//
//nolint:revive // stutter intentional: module.ModuleMetadata reads clearly at call sites
type ModuleMetadata struct {
	// Name is the canonical module name from module.metadata.name (kebab-case).
	Name string `json:"name"`

	// Description is a brief description of the module.
	Description string `json:"description,omitempty"`

	// ModulePath is the CUE registry module path from metadata.modulePath.
	// This is the registry path (e.g., "opmodel.dev/modules"), NOT a filesystem path.
	ModulePath string `json:"modulePath"`

	// Version is the module version (semver).
	Version string `json:"version"`

	// FQN is the fully qualified module name (modulePath/name:version).
	// Example: "opmodel.dev/modules/my-app:1.0.0"
	FQN string `json:"fqn"`

	// UUID is the module identity UUID (from #Module.metadata.identity).
	UUID string `json:"uuid"`

	// Labels from the module definition (pre-build, author-declared).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the module definition.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// CueContextOwner is the minimal context-owner interface accepted by the
// constructor helpers. *kernel.Kernel satisfies it; tests may pass any value
// exposing a *cue.Context. The interface lives in pkg/module to keep the
// constructor's import surface free of pkg/kernel.
type CueContextOwner interface {
	CueContext() *cue.Context
}

// NewModuleFromValue builds a *Module from a raw CUE artifact value. The
// supplied k is currently unused but preserved in the signature so future
// kernel-scoped state (logger, tracer, clock) can be threaded without an
// API break.
//
// The function detects apiVersion via apiversion.Detect, looks up the
// matching Binding, decodes ModuleMetadata via the binding, stamps APIVersion
// on the returned struct, and stores the input cue.Value unmodified in
// Package.
//
// Errors from any step return a nil *Module — partial values are never
// returned. Detection failures wrap apiversion.ErrUnknownAPIVersion.
func NewModuleFromValue(_ CueContextOwner, v cue.Value) (*Module, error) {
	ver, err := apiversion.Detect(v)
	if err != nil {
		return nil, fmt.Errorf("detecting apiVersion: %w", err)
	}
	b, err := api.Lookup(ver)
	if err != nil {
		return nil, fmt.Errorf("resolving binding for %q: %w", ver, err)
	}
	apiMeta, err := b.DecodeModuleMetadata(v)
	if err != nil {
		return nil, fmt.Errorf("decoding module metadata: %w", err)
	}
	return &Module{
		APIVersion: ver,
		Metadata:   moduleMetadataFromAPI(apiMeta),
		Package:    v,
	}, nil
}

// moduleMetadataFromAPI converts the binding's canonical api.ModuleMetadata
// into the pkg/module ModuleMetadata projection. The two shapes are
// deliberately identical so this is a copy; the indirection keeps pkg/api
// from leaking into pkg/module's exported struct field type.
func moduleMetadataFromAPI(m *api.ModuleMetadata) *ModuleMetadata {
	if m == nil {
		return nil
	}
	return &ModuleMetadata{
		Name:        m.Name,
		Description: m.Description,
		ModulePath:  m.ModulePath,
		FQN:         m.FQN,
		Version:     m.Version,
		UUID:        m.UUID,
		Labels:      m.Labels,
		Annotations: m.Annotations,
	}
}
