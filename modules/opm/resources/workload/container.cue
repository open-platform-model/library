package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
)

#ContainerResource: c.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/resources/workload"
		version:     "v1"
		name:        "container"
		description: "A container definition for workloads"
		labels: {
			"resource.opmodel.dev/category": "workload"
		}
	}

	spec: container: schemas.#ContainerSchema
}

#Container: c.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type"!: "stateless" | "stateful" | "daemon" | "task" | "scheduled-task"
	}

	#resources: (#ContainerResource.metadata.fqn): #ContainerResource
}
