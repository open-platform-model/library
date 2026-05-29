package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#UpdateStrategyTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
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
