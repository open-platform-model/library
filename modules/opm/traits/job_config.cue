package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#JobConfigTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
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
