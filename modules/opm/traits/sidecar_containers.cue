package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#SidecarContainersTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
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
