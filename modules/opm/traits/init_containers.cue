package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#InitContainersTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
		name:        "init-containers"
		description: "A trait to specify init containers for a workload"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: initContainers: [...#InitContainersSchema]
}

#InitContainers: c.#Component & {
	#traits: (#InitContainersTrait.metadata.fqn): #InitContainersTrait
}

// Init container shape — alias of #ContainerSchema. Note: K8s only honours
// startupProbe on traditional init containers; native sidecar init containers
// (restartPolicy: Always, K8s >= 1.28) support all three probe types.
#InitContainersSchema: res.#ContainerSchema
