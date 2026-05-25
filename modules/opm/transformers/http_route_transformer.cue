package transformers

import (
	c "opmodel.dev/core/v1alpha2@v1"
	tr "opmodel.dev/modules/opm/traits"
)

// HttpRouteTransformer creates Gateway API HTTPRoutes from components with HttpRoute trait.
// Output is an untyped struct literal — no Gateway API schema lives in modules/opm/schemas/,
// and the renderer dispatches on cue.Kind.
#HttpRouteTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "http-route-transformer"
		description: "Creates Gateway API HTTPRoutes for components with HttpRoute trait"

		labels: {
			"core.opmodel.dev/trait-type":    "network"
			"core.opmodel.dev/resource-type": "http-route"
		}
	}

	requiredLabels: {}
	requiredResources: {}
	optionalResources: {}

	requiredTraits: {
		(tr.#HttpRouteTrait.metadata.fqn): tr.#HttpRouteTrait
	}
	optionalTraits: {}

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_httpRoute: #component.spec.httpRoute
		_name:      "\(#context.#moduleReleaseMetadata.name)-\(#component.metadata.name)"

		// TLS hints: Gateway API HTTPRoute has no tls field (TLS lives on the
		// Gateway listener), so surface the trait's tls attachment as
		// annotations downstream controllers can read.
		_tlsAnnotations: {
			if _httpRoute.tls != _|_ {
				if _httpRoute.tls.mode != _|_ {
					"route.opmodel.dev/tls-mode": _httpRoute.tls.mode
				}
				if _httpRoute.tls.certificateRef != _|_ {
					if _httpRoute.tls.certificateRef.namespace != _|_ {
						"route.opmodel.dev/tls-certificate-ref": "\(_httpRoute.tls.certificateRef.namespace)/\(_httpRoute.tls.certificateRef.name)"
					}
					if _httpRoute.tls.certificateRef.namespace == _|_ {
						"route.opmodel.dev/tls-certificate-ref": _httpRoute.tls.certificateRef.name
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
			kind:       "HTTPRoute"
			metadata: {
				name:      _name
				namespace: #context.#moduleReleaseMetadata.namespace
				labels:    #context.labels
				if len(_routeAnnotations) > 0 {
					annotations: _routeAnnotations
				}
			}
			spec: {
				if _httpRoute.gatewayRef != _|_ {
					parentRefs: [{
						name: _httpRoute.gatewayRef.name
						if _httpRoute.gatewayRef.namespace != _|_ {
							namespace: _httpRoute.gatewayRef.namespace
						}
					}]
				}

				if _httpRoute.hostnames != _|_ {
					hostnames: _httpRoute.hostnames
				}

				rules: [for rule in _httpRoute.rules {
					backendRefs: [{
						name: _name
						port: rule.backendPort
					}]
					if rule.matches != _|_ {
						matches: [for m in rule.matches {
							if m.path != _|_ {
								path: {
									type:  m.path.type
									value: m.path.value
								}
							}
							if m.method != _|_ {
								method: m.method
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
