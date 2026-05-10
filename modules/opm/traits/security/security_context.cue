package security

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

#SecurityContextTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/security"
		version:     "v1"
		name:        "security-context"
		description: "Container and pod-level security constraints"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: securityContext: schemas.#SecurityContextSchema
}

#SecurityContext: c.#Component & {
	#traits: (#SecurityContextTrait.metadata.fqn): #SecurityContextTrait
}
