package network

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

#GrpcRouteTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/network"
		version:     "v1"
		name:        "grpc-route"
		description: "gRPC routing rules for a workload"
		labels: {
			"trait.opmodel.dev/category": "network"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: grpcRoute: schemas.#GrpcRouteSchema
}

#GrpcRoute: c.#Component & {
	#traits: (#GrpcRouteTrait.metadata.fqn): #GrpcRouteTrait
}
