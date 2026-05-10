package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#CronJobConfigTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
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
