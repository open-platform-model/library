// Package catalog defines the Catalog type, mirroring the #Catalog definition
// in the OPM core schema: the contracts a catalog defines (#resources,
// #traits, #blueprints) beside the transformers that implement them
// (#transformers).
//
// A catalog is the kernel's fourth acquired kind, admitted by ADR-009 on the
// terms recorded there: the kernel reads and derives, and every verdict about
// what it reads stays with the caller. Nothing here refuses a value for being
// unwelcome — [Catalog.Provides] reports the empty set for a catalog that
// implements no provider-fulfilled contract, because implementing none is a
// fact about the catalog and not a malformed value.
//
// The package mirrors opm/module's dependency direction: it imports opm/module
// for the shared Source type and opm/schema for paths and metadata, and it
// imports opm/kernel not at all. That direction is structural rather than a
// convention — opm/kernel imports this package to return what its catalog
// verbs acquire, so the reverse edge is an import cycle the compiler refuses.
package catalog

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// Catalog represents an OPM #Catalog artifact in the unified artifact shape.
//
// Package is the source of truth: it is the loaded CUE value for the catalog,
// and every derived view reads it by path on demand. Nothing but metadata is
// decoded at construction — a caller that never asks what a catalog provides
// pays nothing for the fold, exactly as platform.Platform.Contracts is not
// paid for by a render.
//
// Metadata is an ergonomic decoded projection of the catalog-level metadata
// stamped at construction. It is a cache, not a parallel source of truth —
// when Metadata and the corresponding subtree of Package disagree, Package
// wins.
type Catalog struct {
	// Metadata is the decoded catalog-level identity cache. Authoritative
	// data lives in Package; Metadata exists for hot-path access (logging,
	// provenance in diagnostics). May be nil when the metadata could not be
	// decoded.
	Metadata *CatalogMetadata `json:"metadata"`

	// Package is the loaded CUE value for the catalog artifact. Source of
	// truth for every field reachable via opm/schema's path vars.
	Package cue.Value `json:"-"`

	// Source is the catalog's staged source tree, populated only when the
	// catalog was acquired through a source-carrying path
	// (Kernel.AcquireCatalogFromRegistry, Kernel.AcquireCatalogFromDir):
	// always overlay mode, never on-disk. It is nil otherwise, and
	// [Catalog.Requires] is the one reader that needs it. See
	// [module.Source] for the full two-mode contract shared with every
	// artifact.
	Source *Source `json:"-"`
}

// Source is a re-export of [module.Source] so callers can keep working with
// `catalog.Source`, mirroring platform.Source. One type describes the staged
// source tree of every artifact.
type Source = module.Source

// CatalogMetadata is a re-export of [schema.CatalogMetadata] so callers can
// keep working with `catalog.CatalogMetadata` without taking a transitive
// dependency on opm/schema at every reference site.
//
//nolint:revive // stutter intentional: catalog.CatalogMetadata reads clearly at call sites
type CatalogMetadata = schema.CatalogMetadata

// NewCatalogFromValue builds a *Catalog from a raw CUE artifact value: it
// decodes CatalogMetadata from the value's metadata field and stores the input
// cue.Value unmodified in Package. Errors return a nil *Catalog — partial
// values are never returned. The returned Catalog carries no Source.
func NewCatalogFromValue(v cue.Value) (*Catalog, error) {
	meta, err := decodeCatalogMetadata(v)
	if err != nil {
		return nil, fmt.Errorf("decoding catalog metadata: %w", err)
	}
	return &Catalog{
		Metadata: meta,
		Package:  v,
	}, nil
}

// decodeCatalogMetadata extracts CatalogMetadata from a #Catalog artifact
// root. A missing metadata field is fatal.
func decodeCatalogMetadata(v cue.Value) (*CatalogMetadata, error) {
	metaVal := v.LookupPath(schema.Metadata)
	if !metaVal.Exists() {
		return nil, fmt.Errorf("catalog metadata field is required")
	}
	meta := &CatalogMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding catalog metadata: %w", err)
	}
	return meta, nil
}
