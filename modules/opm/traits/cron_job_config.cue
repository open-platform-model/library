package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#CronJobConfigTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
		name:        "cron-job-config"
		description: "A trait to configure CronJob-specific settings for scheduled task workloads"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: cronJobConfig: #CronJobConfigSchema
}

#CronJobConfig: c.#Component & {
	#traits: (#CronJobConfigTrait.metadata.fqn): #CronJobConfigTrait
}

#CronJobConfigSchema: {
	scheduleCron!:               string
	concurrencyPolicy?:          "Allow" | "Forbid" | "Replace"
	startingDeadlineSeconds?:    uint
	successfulJobsHistoryLimit?: uint
	failedJobsHistoryLimit?:     uint
}
