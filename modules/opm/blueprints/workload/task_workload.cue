package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
	tr_workload "opmodel.dev/modules/opm/traits/workload"
)

#TaskWorkloadSchema: {
	container:      schemas.#ContainerSchema
	jobConfig?:     schemas.#JobConfigSchema
	restartPolicy?: schemas.#RestartPolicySchema
	sidecarContainers?: [...schemas.#SidecarContainersSchema]
	initContainers?: [...schemas.#InitContainersSchema]
}

#TaskWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/blueprints/workload"
		version:     "v1"
		name:        "task-workload"
		description: "A one-time task workload that runs to completion (Job)"
	}

	composedResources: [
		res_workload.#ContainerResource,
	]

	composedTraits: [
		tr_workload.#JobConfigTrait,
		tr_workload.#RestartPolicyTrait,
		tr_workload.#SidecarContainersTrait,
		tr_workload.#InitContainersTrait,
	]

	spec: taskWorkload: #TaskWorkloadSchema
}

#TaskWorkload: c.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "task"
	}

	#blueprints: (#TaskWorkloadBlueprint.metadata.fqn): #TaskWorkloadBlueprint

	res_workload.#Container
	tr_workload.#JobConfig
	tr_workload.#RestartPolicy
	tr_workload.#SidecarContainers
	tr_workload.#InitContainers

	// Override spec to propagate values from taskWorkload
	spec: {
		taskWorkload: #TaskWorkloadSchema
		container:    spec.taskWorkload.container
		if spec.taskWorkload.jobConfig != _|_ {
			jobConfig: spec.taskWorkload.jobConfig
		}
		if spec.taskWorkload.restartPolicy != _|_ {
			restartPolicy: spec.taskWorkload.restartPolicy
		}
		if spec.taskWorkload.sidecarContainers != _|_ {
			sidecarContainers: spec.taskWorkload.sidecarContainers
		}
		if spec.taskWorkload.initContainers != _|_ {
			initContainers: spec.taskWorkload.initContainers
		}
	}
}
