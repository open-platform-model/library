package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
	tr_workload "opmodel.dev/modules/opm/traits/workload"
)

#ScheduledTaskWorkloadSchema: {
	container:      schemas.#ContainerSchema
	cronJobConfig:  schemas.#CronJobConfigSchema
	restartPolicy?: schemas.#RestartPolicySchema
	sidecarContainers?: [...schemas.#SidecarContainersSchema]
	initContainers?: [...schemas.#InitContainersSchema]
}

#ScheduledTaskWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/blueprints/workload"
		version:     "v1"
		name:        "scheduled-task-workload"
		description: "A scheduled task workload that runs on a cron schedule (CronJob)"
	}

	composedResources: [
		res_workload.#ContainerResource,
	]

	composedTraits: [
		tr_workload.#CronJobConfigTrait,
		tr_workload.#RestartPolicyTrait,
		tr_workload.#SidecarContainersTrait,
		tr_workload.#InitContainersTrait,
	]

	spec: scheduledTaskWorkload: #ScheduledTaskWorkloadSchema
}

#ScheduledTaskWorkload: c.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "scheduled-task"
	}

	#blueprints: (#ScheduledTaskWorkloadBlueprint.metadata.fqn): #ScheduledTaskWorkloadBlueprint

	res_workload.#Container
	tr_workload.#CronJobConfig
	tr_workload.#RestartPolicy
	tr_workload.#SidecarContainers
	tr_workload.#InitContainers

	// Override spec to propagate values from scheduledTaskWorkload
	spec: {
		scheduledTaskWorkload: #ScheduledTaskWorkloadSchema
		container:             spec.scheduledTaskWorkload.container
		cronJobConfig:         spec.scheduledTaskWorkload.cronJobConfig
		if spec.scheduledTaskWorkload.restartPolicy != _|_ {
			restartPolicy: spec.scheduledTaskWorkload.restartPolicy
		}
		if spec.scheduledTaskWorkload.sidecarContainers != _|_ {
			sidecarContainers: spec.scheduledTaskWorkload.sidecarContainers
		}
		if spec.scheduledTaskWorkload.initContainers != _|_ {
			initContainers: spec.scheduledTaskWorkload.initContainers
		}
	}
}
