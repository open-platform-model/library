package network

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

#ExposeTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/network"
		version:     "v1"
		name:        "expose"
		description: "A trait to expose a workload via a service"
		labels: {
			"trait.opmodel.dev/category": "network"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: expose: schemas.#ExposeSchema
}

#Expose: c.#Component & {
	#traits: (#ExposeTrait.metadata.fqn): #ExposeTrait
}
