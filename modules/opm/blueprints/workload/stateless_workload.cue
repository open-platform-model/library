package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
	tr_workload "opmodel.dev/modules/opm/traits/workload"
)

#StatelessWorkloadSchema: {
	container:       schemas.#ContainerSchema
	scaling?:        schemas.#ScalingSchema
	restartPolicy?:  schemas.#RestartPolicySchema
	updateStrategy?: schemas.#UpdateStrategySchema
	sidecarContainers?: [...schemas.#SidecarContainersSchema]
	initContainers?: [...schemas.#InitContainersSchema]
	securityContext?: schemas.#SecurityContextSchema
	hostPid?:         bool
	hostIpc?:         bool
}

#StatelessWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/blueprints/workload"
		version:     "v1"
		name:        "stateless-workload"
		description: "A stateless workload with no requirement for stable identity or storage"
	}

	composedResources: [
		res_workload.#ContainerResource,
	]

	composedTraits: [
		tr_workload.#ScalingTrait,
		tr_workload.#RestartPolicyTrait,
		tr_workload.#UpdateStrategyTrait,
		tr_workload.#SidecarContainersTrait,
		tr_workload.#InitContainersTrait,
	]

	spec: statelessWorkload: #StatelessWorkloadSchema
}

#StatelessWorkload: c.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "stateless"
	}

	#blueprints: (#StatelessWorkloadBlueprint.metadata.fqn): #StatelessWorkloadBlueprint

	res_workload.#Container
	tr_workload.#Scaling
	tr_workload.#RestartPolicy
	tr_workload.#UpdateStrategy
	tr_workload.#SidecarContainers
	tr_workload.#InitContainers

	// Override spec to propagate values from statelessWorkload
	spec: {
		statelessWorkload: #StatelessWorkloadSchema
		container:         spec.statelessWorkload.container
		if spec.statelessWorkload.scaling != _|_ {
			scaling: spec.statelessWorkload.scaling
		}
		if spec.statelessWorkload.restartPolicy != _|_ {
			restartPolicy: spec.statelessWorkload.restartPolicy
		}
		if spec.statelessWorkload.updateStrategy != _|_ {
			updateStrategy: spec.statelessWorkload.updateStrategy
		}
		if spec.statelessWorkload.sidecarContainers != _|_ {
			sidecarContainers: spec.statelessWorkload.sidecarContainers
		}
		if spec.statelessWorkload.initContainers != _|_ {
			initContainers: spec.statelessWorkload.initContainers
		}
	}
}
