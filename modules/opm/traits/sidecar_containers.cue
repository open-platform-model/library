package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#SidecarContainersTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "sidecar-containers"
		description: "A trait to specify sidecar containers for a workload"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: sidecarContainers: [...#SidecarContainersSchema]
}

#SidecarContainers: c.#Component & {
	#traits: (#SidecarContainersTrait.metadata.fqn): #SidecarContainersTrait
}

// Sidecar container shape — alias of #ContainerSchema.
#SidecarContainersSchema: res.#ContainerSchema
