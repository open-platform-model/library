package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#JobConfigTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "job-config"
		description: "A trait to configure Job-specific settings for task workloads"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: jobConfig: #JobConfigSchema
}

#JobConfig: c.#Component & {
	#traits: (#JobConfigTrait.metadata.fqn): #JobConfigTrait
}

#JobConfigSchema: {
	completions?:             uint
	parallelism?:             uint
	backoffLimit?:            uint
	activeDeadlineSeconds?:   uint
	ttlSecondsAfterFinished?: uint
}
