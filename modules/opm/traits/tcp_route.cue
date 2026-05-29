package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#TcpRouteTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
		name:        "tcp-route"
		description: "TCP port-forwarding rules for a workload"
		labels: {
			"trait.opmodel.dev/category": "network"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: tcpRoute: #TcpRouteSchema
}

#TcpRoute: c.#Component & {
	#traits: (#TcpRouteTrait.metadata.fqn): #TcpRouteTrait
}

// No L7 match fields for TCP.
#TcpRouteRuleSchema: #RouteRuleBase

#TcpRouteSchema: #RouteAttachmentSchema & {
	rules: [#TcpRouteRuleSchema, ...#TcpRouteRuleSchema]
}
