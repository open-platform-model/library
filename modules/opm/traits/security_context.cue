package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#SecurityContextTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
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
