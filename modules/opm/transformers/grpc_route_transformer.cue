package transformers

import (
	c "opmodel.dev/core/v1alpha2@v1"
	tr "opmodel.dev/modules/opm/traits"
)

// GrpcRouteTransformer creates Gateway API GRPCRoutes from components with GrpcRoute trait.
// Untyped struct output — see #HttpRouteTransformer for rationale.
#GrpcRouteTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "grpc-route-transformer"
		description: "Creates Gateway API GRPCRoutes for components with GrpcRoute trait"

		labels: {
			"core.opmodel.dev/trait-type":    "network"
			"core.opmodel.dev/resource-type": "grpc-route"
		}
	}

	requiredLabels: {}
	requiredResources: {}
	optionalResources: {}

	requiredTraits: {
		"opmodel.dev/modules/opm/traits/grpc-route@v1": tr.#GrpcRouteTrait
	}
	optionalTraits: {}

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_grpcRoute: #component.spec.grpcRoute
		_name:      "\(#context.#moduleReleaseMetadata.name)-\(#component.metadata.name)"

		_tlsAnnotations: {
			if _grpcRoute.tls != _|_ {
				if _grpcRoute.tls.mode != _|_ {
					"route.opmodel.dev/tls-mode": _grpcRoute.tls.mode
				}
				if _grpcRoute.tls.certificateRef != _|_ {
					if _grpcRoute.tls.certificateRef.namespace != _|_ {
						"route.opmodel.dev/tls-certificate-ref": "\(_grpcRoute.tls.certificateRef.namespace)/\(_grpcRoute.tls.certificateRef.name)"
					}
					if _grpcRoute.tls.certificateRef.namespace == _|_ {
						"route.opmodel.dev/tls-certificate-ref": _grpcRoute.tls.certificateRef.name
					}
				}
			}
		}

		_routeAnnotations: {
			if len(#context.componentAnnotations) > 0 {
				#context.componentAnnotations
			}
			_tlsAnnotations
		}

		output: {
			apiVersion: "gateway.networking.k8s.io/v1"
			kind:       "GRPCRoute"
			metadata: {
				name:      _name
				namespace: #context.#moduleReleaseMetadata.namespace
				labels:    #context.labels
				if len(_routeAnnotations) > 0 {
					annotations: _routeAnnotations
				}
			}
			spec: {
				if _grpcRoute.gatewayRef != _|_ {
					parentRefs: [{
						name: _grpcRoute.gatewayRef.name
						if _grpcRoute.gatewayRef.namespace != _|_ {
							namespace: _grpcRoute.gatewayRef.namespace
						}
					}]
				}

				if _grpcRoute.hostnames != _|_ {
					hostnames: _grpcRoute.hostnames
				}

				rules: [for rule in _grpcRoute.rules {
					backendRefs: [{
						name: _name
						port: rule.backendPort
					}]
					if rule.matches != _|_ {
						matches: [for m in rule.matches {
							if m.service != _|_ || m.method != _|_ {
								method: {
									if m.service != _|_ {
										service: m.service
									}
									if m.method != _|_ {
										method: m.method
									}
								}
							}
							if m.headers != _|_ {
								headers: [for h in m.headers {
									name:  h.name
									value: h.value
								}]
							}
						}]
					}
				}]
			}
		}
	}
}
