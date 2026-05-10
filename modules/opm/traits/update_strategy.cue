package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#UpdateStrategyTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "update-strategy"
		description: "A trait to specify the update strategy for a workload"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: updateStrategy: #UpdateStrategySchema
}

#UpdateStrategy: c.#Component & {
	#traits: (#UpdateStrategyTrait.metadata.fqn): #UpdateStrategyTrait
}

#UpdateStrategySchema: {
	type: "RollingUpdate" | "Recreate" | "OnDelete"
	if type == "RollingUpdate" {
		rollingUpdate?: {
			maxUnavailable?: uint | string
			maxSurge?:       uint | string
			partition?:      uint
		}
	}
}

#UpdateStrategyDefaults: #UpdateStrategySchema & {
	type: "RollingUpdate"
	rollingUpdate: {
		maxUnavailable: 1
		maxSurge:       1
	}
}
