package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

#JobConfigTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/workload"
		version:     "v1"
		name:        "job-config"
		description: "A trait to configure Job-specific settings for task workloads"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: jobConfig: schemas.#JobConfigSchema
}

#JobConfig: c.#Component & {
	#traits: (#JobConfigTrait.metadata.fqn): #JobConfigTrait
}
