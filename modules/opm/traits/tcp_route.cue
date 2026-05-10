package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#TcpRouteTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
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
