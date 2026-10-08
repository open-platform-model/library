package catalog

// CatalogMetadata is the canonical decoded catalog-level metadata. A catalog
// carries no name: its identity is the module path it is published under and
// the version stamped on every member it ships, so ModulePath and Version are
// the two fields core declares required with no default. FQN is core's
// derivation and equals ModulePath (0010:D1); it is decoded so a
// caller reading provenance off the artifact does not have to know that.
//
//nolint:revive // stutter intentional: catalog.CatalogMetadata reads clearly at call sites
type CatalogMetadata struct {
	// ModulePath is the CUE registry module path the catalog is published
	// under, major suffix included. Example:
	// "opmodel.dev/catalogs/opm@v4".
	ModulePath string `json:"modulePath"`

	// Version is the catalog build version (semver), the value stamped onto
	// every member's metadata.catalogVersion.
	Version string `json:"version"`

	// FQN is the catalog's fully qualified name. Core derives it as the
	// module path itself; the version does not join it.
	FQN string `json:"fqn,omitempty"`

	// Description is a brief description of the catalog.
	Description string `json:"description,omitempty"`

	// Labels from the catalog definition.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the catalog definition.
	Annotations map[string]string `json:"annotations,omitempty"`
}
