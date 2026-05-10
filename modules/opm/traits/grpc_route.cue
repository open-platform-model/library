package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#GrpcRouteTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "grpc-route"
		description: "gRPC routing rules for a workload"
		labels: {
			"trait.opmodel.dev/category": "network"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: grpcRoute: #GrpcRouteSchema
}

#GrpcRoute: c.#Component & {
	#traits: (#GrpcRouteTrait.metadata.fqn): #GrpcRouteTrait
}

#GrpcRouteMatchSchema: {
	service?: string
	method?:  string
	headers?: [...#RouteHeaderMatch]
}

#GrpcRouteRuleSchema: #RouteRuleBase & {
	matches?: [...#GrpcRouteMatchSchema]
}

#GrpcRouteSchema: #RouteAttachmentSchema & {
	hostnames?: [...string]
	rules: [#GrpcRouteRuleSchema, ...#GrpcRouteRuleSchema]
}
