package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#ExposeTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "expose"
		description: "A trait to expose a workload via a service"
		labels: {
			"trait.opmodel.dev/category": "network"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: expose: #ExposeSchema
}

#Expose: c.#Component & {
	#traits: (#ExposeTrait.metadata.fqn): #ExposeTrait
}

// Service expose specification.
#ExposeSchema: {
	ports: [portName=string]: res.#PortSchema & {name: portName}
	type: "ClusterIP" | "NodePort" | "LoadBalancer"
}
