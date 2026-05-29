package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#DisruptionBudgetTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
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
