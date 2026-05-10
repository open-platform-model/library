package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
	tr_workload "opmodel.dev/modules/opm/traits/workload"
)

#DaemonWorkloadSchema: {
	container:       schemas.#ContainerSchema
	restartPolicy?:  schemas.#RestartPolicySchema
	updateStrategy?: schemas.#UpdateStrategySchema
	sidecarContainers?: [...schemas.#SidecarContainersSchema]
	initContainers?: [...schemas.#InitContainersSchema]
}

#DaemonWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/blueprints/workload"
		version:     "v1"
		name:        "daemon-workload"
		description: "A daemon workload that runs on all (or selected) nodes in a cluster"
	}

	composedResources: [
		res_workload.#ContainerResource,
	]

	composedTraits: [
		tr_workload.#RestartPolicyTrait,
		tr_workload.#UpdateStrategyTrait,
		tr_workload.#SidecarContainersTrait,
		tr_workload.#InitContainersTrait,
	]

	spec: daemonWorkload: #DaemonWorkloadSchema
}

#DaemonWorkload: c.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "daemon"
	}

	#blueprints: (#DaemonWorkloadBlueprint.metadata.fqn): #DaemonWorkloadBlueprint

	res_workload.#Container
	tr_workload.#RestartPolicy
	tr_workload.#UpdateStrategy
	tr_workload.#SidecarContainers
	tr_workload.#InitContainers

	// Override spec to propagate values from daemonWorkload
	spec: {
		daemonWorkload: #DaemonWorkloadSchema
		container:      spec.daemonWorkload.container
		if spec.daemonWorkload.restartPolicy != _|_ {
			restartPolicy: spec.daemonWorkload.restartPolicy
		}
		if spec.daemonWorkload.updateStrategy != _|_ {
			updateStrategy: spec.daemonWorkload.updateStrategy
		}
		if spec.daemonWorkload.sidecarContainers != _|_ {
			sidecarContainers: spec.daemonWorkload.sidecarContainers
		}
		if spec.daemonWorkload.initContainers != _|_ {
			initContainers: spec.daemonWorkload.initContainers
		}
	}
}
