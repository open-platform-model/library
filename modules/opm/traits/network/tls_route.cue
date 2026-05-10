package network

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

#TlsRouteTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/network"
		version:     "v1"
		name:        "tls-route"
		description: "TLS routing rules (passthrough or terminate) for a workload"
		labels: {
			"trait.opmodel.dev/category": "network"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: tlsRoute: schemas.#TlsRouteSchema
}

#TlsRoute: c.#Component & {
	#traits: (#TlsRouteTrait.metadata.fqn): #TlsRouteTrait
}
