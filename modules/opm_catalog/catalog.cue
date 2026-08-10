// Catalog manifest for the library's fixture catalog: a minimal, core-v2
// authored stand-in for opmodel.dev/catalogs/opm, published under the
// testing.opmodel.dev prefix (workspace Registry Policy — tests serve it from
// the in-process registry; local diagnostic flows publish it to
// localhost:5000). It exists because no v2-built real catalog can exist until
// 0010's catalogs-republish, which depends on library slices downstream of
// library-core-retarget.
//
// Embeds bare c.#Catalog (modules pattern), sources metadata from the sibling
// identity/ package, and enumerates every transformer keyed by its own
// authored fqn. The #Catalog pattern constraint stamps each entry's
// modulePath/catalogVersion in lockstep.
package opm

import (
	c "opmodel.dev/core@v2"
	id "testing.opmodel.dev/catalogs/opm/identity"
	t "testing.opmodel.dev/catalogs/opm/transformers"
)

c.#Catalog
metadata: {
	modulePath:  id.ModulePath
	version:     id.Version
	description: "Library fixture catalog — minimal opm-shaped members authored against core v2"
}

#transformers: {
	(t.#ConfigMapTransformer.metadata.fqn):  t.#ConfigMapTransformer
	(t.#DeploymentTransformer.metadata.fqn): t.#DeploymentTransformer
	(t.#HttpRouteTransformer.metadata.fqn):  t.#HttpRouteTransformer
	(t.#ServiceTransformer.metadata.fqn):    t.#ServiceTransformer
}
