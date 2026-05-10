package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
	tr "opmodel.dev/modules/opm/traits"
)

#TaskWorkloadSchema: {
	container:      res.#ContainerSchema
	jobConfig?:     tr.#JobConfigSchema
	restartPolicy?: tr.#RestartPolicySchema
	sidecarContainers?: [...tr.#SidecarContainersSchema]
	initContainers?: [...tr.#InitContainersSchema]
}

#TaskWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/blueprints/workload"
		version:     "v1"
		name:        "task-workload"
		description: "A one-time task workload that runs to completion (Job)"
	}

	composedResources: [
		res.#ContainerResource,
	]

	composedTraits: [
		tr.#JobConfigTrait,
		tr.#RestartPolicyTrait,
		tr.#SidecarContainersTrait,
		tr.#InitContainersTrait,
	]

	spec: taskWorkload: #TaskWorkloadSchema
}

#TaskWorkload: c.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "task"
	}

	#blueprints: (#TaskWorkloadBlueprint.metadata.fqn): #TaskWorkloadBlueprint

	res.#Container
	tr.#JobConfig
	tr.#RestartPolicy
	tr.#SidecarContainers
	tr.#InitContainers

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
