package platform

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/apiversion"
)

// Platform represents an OPM #Platform artifact in the unified artifact
// shape: { APIVersion, Metadata, Package }.
//
// Package is the source of truth: it is the loaded CUE value for the
// platform and every kernel-internal read (Registry, computed views) goes
// through Package.LookupPath via the version Binding's Paths(). The
// CUE-computed views (#knownResources, #knownTraits, #composedTransformers,
// #matchers) are NOT decoded into Go fields; consumers iterate them lazily
// off Package.
//
// Metadata is an ergonomic decoded projection of the platform-level
// metadata stamped at construction. It is a cache, not a parallel source
// of truth — when Metadata and the corresponding subtree of Package
// disagree, Package wins. Callers MUST NOT mutate Package's apiVersion
// field after construction; APIVersion is stamped from Package and would
// otherwise drift.
type Platform struct {
	// APIVersion is the OPM schema version detected on the platform
	// artifact. Set by NewPlatformFromValue from Package's apiVersion
	// field. The zero value signals an unloaded or hand-constructed
	// platform.
	APIVersion apiversion.Version `json:"apiVersion,omitempty"`

	// Metadata is the decoded platform-level metadata cache. Authoritative
	// data lives in Package; Metadata exists for hot-path access (logging,
	// name lookups). May be nil when the metadata could not be decoded.
	Metadata *PlatformMetadata `json:"metadata"`

	// Package is the loaded CUE value for the platform artifact. Source of
	// truth for every field reachable via the binding's Paths(), including
	// #registry and the four computed views.
	Package cue.Value `json:"-"`
}

// PlatformMetadata contains platform-level identity information decoded from
// the #Platform artifact. Fields mirror catalog enhancement 014's metadata
// block plus the top-level #Platform.type field hoisted into the projection.
//
//nolint:revive // stutter intentional: platform.PlatformMetadata reads clearly at call sites
type PlatformMetadata struct {
	// Name is the platform identifier (kebab-case) from metadata.name.
	Name string `json:"name"`

	// Type is the platform target type (e.g. "kubernetes", "crossplane").
	// Hoisted from #Platform.type at construction. Per catalog 014 the
	// field is informational today; future enhancements may enforce
	// per-Module compatibility against this field.
	Type string `json:"type"`

	// Description is a brief human-readable summary.
	Description string `json:"description,omitempty"`

	// Labels from the platform definition.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the platform definition.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// CueContextOwner is the minimal context-owner interface accepted by the
// constructor helpers. *kernel.Kernel satisfies it; tests may pass any
// value exposing a *cue.Context. The interface lives in pkg/platform to
// keep the constructor's import surface free of pkg/kernel.
type CueContextOwner interface {
	CueContext() *cue.Context
}

// NewPlatformFromValue builds a *Platform from a raw CUE artifact value.
// The supplied k is currently unused but preserved in the signature so
// future kernel-scoped state (logger, tracer, clock) can be threaded
// without an API break.
//
// The function detects apiVersion via apiversion.Detect, looks up the
// matching Binding, decodes PlatformMetadata via the binding, stamps
// APIVersion on the returned struct, and stores the input cue.Value
// unmodified in Package.
//
// Errors from any step return a nil *Platform — partial values are never
// returned. Detection failures wrap apiversion.ErrUnknownAPIVersion.
func NewPlatformFromValue(_ CueContextOwner, v cue.Value) (*Platform, error) {
	ver, err := apiversion.Detect(v)
	if err != nil {
		return nil, fmt.Errorf("detecting apiVersion: %w", err)
	}
	b, err := api.Lookup(ver)
	if err != nil {
		return nil, fmt.Errorf("resolving binding for %q: %w", ver, err)
	}
	apiMeta, err := b.DecodePlatformMetadata(v)
	if err != nil {
		return nil, fmt.Errorf("decoding platform metadata: %w", err)
	}
	return &Platform{
		APIVersion: ver,
		Metadata:   platformMetadataFromAPI(apiMeta),
		Package:    v,
	}, nil
}

// platformMetadataFromAPI converts the binding's canonical
// api.PlatformMetadata into the pkg/platform PlatformMetadata projection.
// The two shapes are deliberately identical so this is a copy; the
// indirection keeps pkg/api from leaking into pkg/platform's exported
// struct field type.
func platformMetadataFromAPI(m *api.PlatformMetadata) *PlatformMetadata {
	if m == nil {
		return nil
	}
	return &PlatformMetadata{
		Name:        m.Name,
		Type:        m.Type,
		Description: m.Description,
		Labels:      m.Labels,
		Annotations: m.Annotations,
	}
}
