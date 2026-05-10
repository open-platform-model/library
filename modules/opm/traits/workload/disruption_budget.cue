package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

#DisruptionBudgetTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/workload"
		version:     "v1"
		name:        "disruption-budget"
		description: "Availability constraints during voluntary disruptions"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: disruptionBudget: schemas.#DisruptionBudgetSchema
}

#DisruptionBudget: c.#Component & {
	#traits: (#DisruptionBudgetTrait.metadata.fqn): #DisruptionBudgetTrait
}
