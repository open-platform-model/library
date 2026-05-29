package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#GrpcRouteTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
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
