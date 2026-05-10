package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#InitContainersTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
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
