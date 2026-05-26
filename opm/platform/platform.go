package platform

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/schema"
)

// Platform represents an OPM #Platform artifact in the unified artifact
// shape: { Metadata, Package }.
//
// Package is the source of truth: it is the loaded CUE value for the
// platform and every kernel-internal read (Registry, computed views) goes
// through Package.LookupPath with paths from opm/schema. The CUE-computed
// views (#composedTransformers, #matchers) are NOT decoded into Go fields;
// consumers iterate them lazily off Package.
//
// Metadata is an ergonomic decoded projection of the platform-level metadata
// stamped at construction. It is a cache, not a parallel source of truth —
// when Metadata and the corresponding subtree of Package disagree, Package
// wins.
type Platform struct {
	// Metadata is the decoded platform-level metadata cache. Authoritative
	// data lives in Package; Metadata exists for hot-path access (logging,
	// name lookups). May be nil when the metadata could not be decoded.
	Metadata *PlatformMetadata `json:"metadata"`

	// Package is the loaded CUE value for the platform artifact. Source of
	// truth for every field reachable via opm/schema, including #registry
	// and the computed views.
	Package cue.Value `json:"-"`
}

// PlatformMetadata is a re-export of [schema.PlatformMetadata] so callers
// can keep working with `platform.PlatformMetadata`.
//
//nolint:revive // stutter intentional: platform.PlatformMetadata reads clearly at call sites
type PlatformMetadata = schema.PlatformMetadata

// CueContextOwner is the minimal context-owner interface accepted by the
// constructor helpers. *kernel.Kernel satisfies it; tests may pass any
// value exposing a *cue.Context. The interface lives in opm/platform to
// keep the constructor's import surface free of opm/kernel.
type CueContextOwner interface {
	CueContext() *cue.Context
}

// NewPlatformFromValue builds a *Platform from a raw CUE artifact value.
// The supplied k is currently unused but preserved in the signature so
// future kernel-scoped state (logger, tracer, clock) can be threaded
// without an API break.
//
// The function decodes PlatformMetadata via schema.DecodePlatformMetadata
// and stores the input cue.Value unmodified in Package. Errors return a nil
// *Platform — partial values are never returned.
func NewPlatformFromValue(_ CueContextOwner, v cue.Value) (*Platform, error) {
	meta, err := schema.DecodePlatformMetadata(v)
	if err != nil {
		return nil, fmt.Errorf("decoding platform metadata: %w", err)
	}
	return &Platform{
		Metadata: meta,
		Package:  v,
	}, nil
}
