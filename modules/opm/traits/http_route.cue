package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#HttpRouteTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
		name:        "http-route"
		description: "HTTP routing rules for a workload"
		labels: {
			"trait.opmodel.dev/category": "network"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: httpRoute: #HttpRouteSchema
}

#HttpRoute: c.#Component & {
	#traits: (#HttpRouteTrait.metadata.fqn): #HttpRouteTrait
}

/////////////////////////////////////////////////////////////////
//// HTTP Route Schemas
//// Shared route mixins (#RouteRuleBase, #RouteAttachmentSchema,
//// #RouteHeaderMatch) live in _route_common.cue.
/////////////////////////////////////////////////////////////////

#HttpRouteMatchSchema: {
	path?: {
		type:   "PathPrefix" | "Exact" | "RegularExpression"
		value!: string
	}
	headers?: [...#RouteHeaderMatch]
	method?: "GET" | "POST" | "PUT" | "DELETE" | "PATCH" | "HEAD" | "OPTIONS"
}

#HttpRouteRuleSchema: #RouteRuleBase & {
	matches?: [...#HttpRouteMatchSchema]
}

#HttpRouteSchema: #RouteAttachmentSchema & {
	hostnames?: [...string]
	rules: [#HttpRouteRuleSchema, ...#HttpRouteRuleSchema]
}
