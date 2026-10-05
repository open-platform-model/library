package platform

import (
	"fmt"
	"sync"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// Platform represents an OPM #Platform artifact in the unified artifact
// shape: { Metadata, Package }.
//
// Package is the source of truth: it is the loaded CUE value for the
// platform, and metadata decoding reads it. No exported field holds a
// derived CUE view: the render build imports the platform package and the
// glue reads #composedTransformers in CUE.
//
// Metadata is an ergonomic decoded projection of the platform-level metadata
// stamped at construction. It is a cache, not a parallel source of truth —
// when Metadata and the corresponding subtree of Package disagree, Package
// wins.
//
// The core floor (whether #contracts carries providedBy) and the contract
// inventory, or its refusal, are a second cache stamped at construction:
// [NewPlatformFromValue] decodes them from Package once, and
// [Platform.Contracts] and [Platform.CoreFloor] read the recorded result
// without reading Package again. A Platform the constructor did not build
// (a struct literal, or the zero value) decodes them from its Package on the
// first Contracts or CoreFloor call, once, and every later call reads that
// result. Like Metadata, the cache does not follow a later change to
// Package: a caller that changes Package re-runs the constructor.
//
// A Platform is used through a pointer and never copied: it carries a
// [sync.Once], so go vet's copylocks check reports a copied value.
type Platform struct {
	// Metadata is the decoded platform-level metadata cache. Authoritative
	// data lives in Package; Metadata exists for hot-path access (logging,
	// name lookups). May be nil when the metadata could not be decoded.
	Metadata *PlatformMetadata `json:"metadata"`

	// Package is the loaded CUE value for the platform artifact, the
	// evaluated form of the package Source points at.
	Package cue.Value `json:"-"`

	// Source is the staged source tree the platform package was loaded from,
	// so a follow-on build can import the platform as a package. It is
	// stamped by Kernel.AcquirePlatformFromDir (on-disk mode, the loaded
	// directory) and is nil for platforms constructed from a bare value
	// (NewPlatformFromValue). Source is the render input: Kernel.Render
	// imports the platform package from it, so a platform without Source
	// cannot be rendered against. No other kernel operation reads it.
	Source *Source `json:"-"`

	// once guards recorded: the constructor runs it, and on a Platform the
	// constructor did not build the first Contracts or CoreFloor call does.
	once sync.Once
	// recorded is the core floor and the contract inventory (or its
	// refusal), decoded from Package once.
	recorded facts
}

// Source is a re-export of [module.Source] so callers can keep working with
// `platform.Source`, mirroring the [PlatformMetadata] re-export. One type
// describes the staged source tree of every artifact; see [module.Source]
// for the overlay and on-disk modes.
type Source = module.Source

// PlatformMetadata is a re-export of [schema.PlatformMetadata] so callers
// can keep working with `platform.PlatformMetadata`.
//
//nolint:revive // stutter intentional: platform.PlatformMetadata reads clearly at call sites
type PlatformMetadata = schema.PlatformMetadata

// NewPlatformFromValue builds a *Platform from a raw CUE artifact value: it
// decodes PlatformMetadata from the value's metadata field (with the
// top-level type hoisted in), stores the input cue.Value unmodified in
// Package, and records the core floor and the contract inventory decoded
// from it (see [Platform]). Errors return a nil *Platform — partial values
// are never returned. Only the metadata can fail construction: a missing,
// too-old or non-evaluating #contracts is recorded and returned later by
// [Platform.Contracts] and [Platform.CoreFloor], so an older-core platform
// still constructs. The returned Platform carries no Source.
func NewPlatformFromValue(v cue.Value) (*Platform, error) {
	meta, err := decodePlatformMetadata(v)
	if err != nil {
		// The decoder's error already names the artifact; a second prefix
		// would read "decoding platform metadata: decoding platform metadata: ...".
		return nil, err
	}
	p := &Platform{
		Metadata: meta,
		Package:  v,
	}
	p.facts()
	return p, nil
}

// decodePlatformMetadata extracts PlatformMetadata from a #Platform value.
// metadata.{name,description,labels,annotations} is decoded directly into the
// struct; the top-level #Platform.type field is read separately and merged
// into Metadata.Type so callers see one identity record per Platform.
// A missing metadata field is fatal.
func decodePlatformMetadata(v cue.Value) (*PlatformMetadata, error) {
	metaVal := v.LookupPath(schema.Metadata)
	if !metaVal.Exists() {
		return nil, fmt.Errorf("platform metadata field is required")
	}
	meta := &PlatformMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding platform metadata: %w", err)
	}
	if typeVal := v.LookupPath(cue.ParsePath("type")); typeVal.Exists() {
		s, err := typeVal.String()
		if err != nil {
			return nil, fmt.Errorf("decoding platform type: %w", err)
		}
		meta.Type = s
	}
	return meta, nil
}
