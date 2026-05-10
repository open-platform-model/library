package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
	res_storage "opmodel.dev/modules/opm/resources/storage"
	tr_workload "opmodel.dev/modules/opm/traits/workload"
)

#StatefulWorkloadSchema: {
	container: schemas.#ContainerSchema
	volumes?: [string]: schemas.#VolumeSchema
	scaling?:        schemas.#ScalingSchema
	restartPolicy?:  schemas.#RestartPolicySchema
	updateStrategy?: schemas.#UpdateStrategySchema
	sidecarContainers?: [...schemas.#SidecarContainersSchema]
	initContainers?: [...schemas.#InitContainersSchema]
}

#StatefulWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/blueprints/workload"
		version:     "v1"
		name:        "stateful-workload"
		description: "A stateful workload with stable identity and persistent storage requirements"
	}

	composedResources: [
		res_workload.#ContainerResource,
		res_storage.#VolumesResource,
	]

	composedTraits: [
		tr_workload.#ScalingTrait,
		tr_workload.#RestartPolicyTrait,
		tr_workload.#UpdateStrategyTrait,
		tr_workload.#SidecarContainersTrait,
		tr_workload.#InitContainersTrait,
	]

	spec: statefulWorkload: #StatefulWorkloadSchema
}

#StatefulWorkload: c.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "stateful"
	}

	#blueprints: (#StatefulWorkloadBlueprint.metadata.fqn): #StatefulWorkloadBlueprint

	res_workload.#Container
	res_storage.#Volumes
	tr_workload.#Scaling
	tr_workload.#RestartPolicy
	tr_workload.#UpdateStrategy
	tr_workload.#SidecarContainers
	tr_workload.#InitContainers

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
