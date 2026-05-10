package traits

/////////////////////////////////////////////////////////////////
//// Route Shared Mixins
//// Used by #HttpRouteSchema, #GrpcRouteSchema, #TcpRouteSchema,
//// and #TlsRouteSchema.
/////////////////////////////////////////////////////////////////

// Header match for route rules.
#RouteHeaderMatch: {
	name!:  string
	value!: string
}

// Base fields shared by all route rules.
#RouteRuleBase: {
	backendPort!: uint & >=1 & <=65535
	...
}

// Shared attachment fields for route schemas (gateway, TLS, className).
#RouteAttachmentSchema: {
	gatewayRef?: {
		name!:      string
		namespace?: string
	}
	tls?: {
		mode?: "Terminate" | "Passthrough"
		certificateRef?: {
			name!:      string
			namespace?: string
		}
	}
	...
}
