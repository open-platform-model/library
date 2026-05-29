package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#TlsRouteTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
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
