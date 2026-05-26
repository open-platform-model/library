package core

// #CatalogFQNType: catalog-level FQN — modulePath@semver (no name segment).
// Distinct from #FQNType which is modulePath/name@semver for primitives.
// Both accept SemVer 2.0. Note: the two regexes are NOT structurally
// disjoint — a string like "opmodel.dev/catalogs/opm/transformer@1.0.0"
// matches both. They are semantically distinguished by usage, not regex.
//
// Introduced by enhancement 0001 (D19).
#CatalogFQNType: string & =~"^[a-z0-9.-]+(/[a-z0-9.-]+)*@\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

// #Catalog: top-level catalog definition. Authoring shape uses the modules
// pattern — bare `c.#Catalog` at file root, fields written at package level,
// no `Catalog:` wrapper:
//
//   // library/modules/opm/catalog.cue
//   package opm
//
//   import (
//       c  "opmodel.dev/core@v0"
//       id "opmodel.dev/catalogs/opm/identity"
//       t  "opmodel.dev/catalogs/opm/transformers"
//   )
//
//   c.#Catalog
//   metadata: {
//       modulePath:  id.ModulePath
//       version:     id.Version
//       description: "OPM core catalog"
//   }
//   #transformers: {
//       (t.#ConfigMapTransformer.metadata.fqn):  t.#ConfigMapTransformer
//       (t.#DeploymentTransformer.metadata.fqn): t.#DeploymentTransformer
//   }
//
// Catalog identity lives in a sibling `identity/` subpackage so transformer
// subpackages can source it without a circular import. Publish-time stamping
// targets `identity/version_override.cue`.
//
// The pattern constraint on `#transformers` stamps every entry's
// `metadata.modulePath` to "\(M.modulePath)/transformers" and
// `metadata.version` to the catalog's version. It does NOT stamp
// `metadata.fqn` — fqn derives from modulePath/name/version, and the map
// key already carries the transformer's own fqn. Author discipline replaced
// by schema enforcement: D18 lockstep is enforced structurally.
//
// `M=metadata` is a field-label alias (enhancement 0001 D25). It binds the
// label `M` to the metadata field path so the pattern constraint can reach
// `M.modulePath` and `M.version` across the nested struct boundary. The
// value-alias form `metadata: M={...}` does NOT carry across this boundary
// and fails cue vet with "reference M not found". Experiment 09 validated
// both sound forms (label alias + hidden mirror); the label alias is
// chosen for inline locality.
//
// Resources / Traits / Blueprints are surfaced transitively via each
// transformer's required/optional maps. Adding sibling maps (#resources,
// #traits, #blueprints) is an additive extension if introspection demand
// surfaces later.
#Catalog: {
	kind: "Catalog"
	M=metadata: {
		modulePath!:  #ModulePathType
		version!:     #VersionType | *"0.0.0-dev"
		fqn:          #CatalogFQNType & "\(modulePath)@\(version)"
		description?: string
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}

	#transformers: [#FQNType]: #ComponentTransformer & {
		metadata: {
			modulePath: "\(M.modulePath)/transformers"
			version:    M.version
		}
	}
}
