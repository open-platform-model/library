package workload

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
	tr "opmodel.dev/catalogs/opm/traits"
)

#StatelessWorkloadSchema: {
	container:       res.#ContainerSchema
	scaling?:        tr.#ScalingSchema
	restartPolicy?:  tr.#RestartPolicySchema
	updateStrategy?: tr.#UpdateStrategySchema
	sidecarContainers?: [...tr.#SidecarContainersSchema]
	initContainers?: [...tr.#InitContainersSchema]
	securityContext?: res.#SecurityContextSchema
	hostPid?:         bool
	hostIpc?:         bool
}

#StatelessWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		modulePath:  "\(id.ModulePath)/blueprints/workload"
		version:     id.Version
		name:        "stateless-workload"
		description: "A stateless workload with no requirement for stable identity or storage"
	}

	composedResources: [
		res.#ContainerResource,
	]

	composedTraits: [
		tr.#ScalingTrait,
		tr.#RestartPolicyTrait,
		tr.#UpdateStrategyTrait,
		tr.#SidecarContainersTrait,
		tr.#InitContainersTrait,
	]

	spec: statelessWorkload: #StatelessWorkloadSchema
}

#StatelessWorkload: c.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "stateless"
	}

	#blueprints: (#StatelessWorkloadBlueprint.metadata.fqn): #StatelessWorkloadBlueprint

	res.#Container
	tr.#Scaling
	tr.#RestartPolicy
	tr.#UpdateStrategy
	tr.#SidecarContainers
	tr.#InitContainers

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
