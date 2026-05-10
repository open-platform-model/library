package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

#PlacementTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/workload"
		version:     "v1"
		name:        "placement"
		description: "Workload placement intent across failure domains"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: placement: schemas.#PlacementSchema
}

#Placement: c.#Component & {
	#traits: (#PlacementTrait.metadata.fqn): #PlacementTrait
}
