package workload

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
	tr "opmodel.dev/catalogs/opm/traits"
)

#ScheduledTaskWorkloadSchema: {
	container:      res.#ContainerSchema
	cronJobConfig:  tr.#CronJobConfigSchema
	restartPolicy?: tr.#RestartPolicySchema
	sidecarContainers?: [...tr.#SidecarContainersSchema]
	initContainers?: [...tr.#InitContainersSchema]
}

#ScheduledTaskWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		modulePath:  "\(id.ModulePath)/blueprints/workload"
		version:     id.Version
		name:        "scheduled-task-workload"
		description: "A scheduled task workload that runs on a cron schedule (CronJob)"
	}

	composedResources: [
		res.#ContainerResource,
	]

	composedTraits: [
		tr.#CronJobConfigTrait,
		tr.#RestartPolicyTrait,
		tr.#SidecarContainersTrait,
		tr.#InitContainersTrait,
	]

	spec: scheduledTaskWorkload: #ScheduledTaskWorkloadSchema
}

#ScheduledTaskWorkload: c.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "scheduled-task"
	}

	#blueprints: (#ScheduledTaskWorkloadBlueprint.metadata.fqn): #ScheduledTaskWorkloadBlueprint

	res.#Container
	tr.#CronJobConfig
	tr.#RestartPolicy
	tr.#SidecarContainers
	tr.#InitContainers

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
