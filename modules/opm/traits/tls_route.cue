package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#TlsRouteTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "tls-route"
		description: "TLS routing rules (passthrough or terminate) for a workload"
		labels: {
			"trait.opmodel.dev/category": "network"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: tlsRoute: #TlsRouteSchema
}

#TlsRoute: c.#Component & {
	#traits: (#TlsRouteTrait.metadata.fqn): #TlsRouteTrait
}

// No L7 match fields for TLS.
#TlsRouteRuleSchema: #RouteRuleBase

#TlsRouteSchema: #RouteAttachmentSchema & {
	hostnames?: [...string]
	rules: [#TlsRouteRuleSchema, ...#TlsRouteRuleSchema]
}
