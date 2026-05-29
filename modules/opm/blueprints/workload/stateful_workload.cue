package workload

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
	tr "opmodel.dev/catalogs/opm/traits"
)

#StatefulWorkloadSchema: {
	container: res.#ContainerSchema
	volumes?: [string]: res.#VolumeSchema
	scaling?:        tr.#ScalingSchema
	restartPolicy?:  tr.#RestartPolicySchema
	updateStrategy?: tr.#UpdateStrategySchema
	sidecarContainers?: [...tr.#SidecarContainersSchema]
	initContainers?: [...tr.#InitContainersSchema]
}

#StatefulWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		modulePath:  "\(id.ModulePath)/blueprints/workload"
		version:     id.Version
		name:        "stateful-workload"
		description: "A stateful workload with stable identity and persistent storage requirements"
	}

	composedResources: [
		res.#ContainerResource,
		res.#VolumesResource,
	]

	composedTraits: [
		tr.#ScalingTrait,
		tr.#RestartPolicyTrait,
		tr.#UpdateStrategyTrait,
		tr.#SidecarContainersTrait,
		tr.#InitContainersTrait,
	]

	spec: statefulWorkload: #StatefulWorkloadSchema
}

#StatefulWorkload: c.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "stateful"
	}

	#blueprints: (#StatefulWorkloadBlueprint.metadata.fqn): #StatefulWorkloadBlueprint

	res.#Container
	res.#Volumes
	tr.#Scaling
	tr.#RestartPolicy
	tr.#UpdateStrategy
	tr.#SidecarContainers
	tr.#InitContainers

	// Override spec to propagate values from statefulWorkload
	spec: {
		statefulWorkload: #StatefulWorkloadSchema
		container:        spec.statefulWorkload.container
		if spec.statefulWorkload.volumes != _|_ {
			volumes: spec.statefulWorkload.volumes
		}
		if spec.statefulWorkload.scaling != _|_ {
			scaling: spec.statefulWorkload.scaling
		}
		if spec.statefulWorkload.restartPolicy != _|_ {
			restartPolicy: spec.statefulWorkload.restartPolicy
		}
		if spec.statefulWorkload.updateStrategy != _|_ {
			updateStrategy: spec.statefulWorkload.updateStrategy
		}
		if spec.statefulWorkload.sidecarContainers != _|_ {
			sidecarContainers: spec.statefulWorkload.sidecarContainers
		}
		if spec.statefulWorkload.initContainers != _|_ {
			initContainers: spec.statefulWorkload.initContainers
		}
	}
}
