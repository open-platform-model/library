package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#SecurityContextTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
		name:        "security-context"
		description: "Container and pod-level security constraints"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [res.#ContainerResource]

	// Pod-level securityContext. The schema shape lives in resources/container.cue
	// because #ContainerSchema also embeds it (per-container scope).
	spec: securityContext: res.#SecurityContextSchema
}

#SecurityContext: c.#Component & {
	#traits: (#SecurityContextTrait.metadata.fqn): #SecurityContextTrait
}
