package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

#ScalingTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/workload"
		version:     "v1"
		name:        "scaling"
		description: "A trait to specify scaling behavior for a workload"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: scaling: schemas.#ScalingSchema
}

#Scaling: c.#Component & {
	#traits: (#ScalingTrait.metadata.fqn): #ScalingTrait
}
