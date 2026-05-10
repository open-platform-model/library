package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#DisruptionBudgetTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "disruption-budget"
		description: "Availability constraints during voluntary disruptions"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: disruptionBudget: #DisruptionBudgetSchema
}

#DisruptionBudget: c.#Component & {
	#traits: (#DisruptionBudgetTrait.metadata.fqn): #DisruptionBudgetTrait
}

// Exactly one of minAvailable or maxUnavailable must be set.
#DisruptionBudgetSchema: {
	minAvailable!: int | string & =~"^[0-9]+%$"
} | {maxUnavailable!: int | string & =~"^[0-9]+%$"
}
