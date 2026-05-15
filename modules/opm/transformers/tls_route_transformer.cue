package transformers

import (
	c "opmodel.dev/core/v1alpha2@v1"
	tr "opmodel.dev/modules/opm/traits"
)

// TlsRouteTransformer creates Gateway API TLSRoutes from components with TlsRoute trait.
// Untyped struct output — see #HttpRouteTransformer for rationale.
#TlsRouteTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "tls-route-transformer"
		description: "Creates Gateway API TLSRoutes for components with TlsRoute trait"

		labels: {
			"core.opmodel.dev/trait-type":    "network"
			"core.opmodel.dev/resource-type": "tls-route"
		}
	}

	requiredLabels: {}
	requiredResources: {}
	optionalResources: {}

	requiredTraits: {
		"opmodel.dev/modules/opm/traits/tls-route@v1": tr.#TlsRouteTrait
	}
	optionalTraits: {}

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_tlsRoute: #component.spec.tlsRoute
		_name:     "\(#context.#moduleReleaseMetadata.name)-\(#component.metadata.name)"

		_tlsAnnotations: {
			if _tlsRoute.tls != _|_ {
				if _tlsRoute.tls.mode != _|_ {
					"route.opmodel.dev/tls-mode": _tlsRoute.tls.mode
				}
				if _tlsRoute.tls.certificateRef != _|_ {
					if _tlsRoute.tls.certificateRef.namespace != _|_ {
						"route.opmodel.dev/tls-certificate-ref": "\(_tlsRoute.tls.certificateRef.namespace)/\(_tlsRoute.tls.certificateRef.name)"
					}
					if _tlsRoute.tls.certificateRef.namespace == _|_ {
						"route.opmodel.dev/tls-certificate-ref": _tlsRoute.tls.certificateRef.name
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
			apiVersion: "gateway.networking.k8s.io/v1alpha2"
			kind:       "TLSRoute"
			metadata: {
				name:      _name
				namespace: #context.#moduleReleaseMetadata.namespace
				labels:    #context.labels
				if len(_routeAnnotations) > 0 {
					annotations: _routeAnnotations
				}
			}
			spec: {
				if _tlsRoute.gatewayRef != _|_ {
					parentRefs: [{
						name: _tlsRoute.gatewayRef.name
						if _tlsRoute.gatewayRef.namespace != _|_ {
							namespace: _tlsRoute.gatewayRef.namespace
						}
					}]
				}

				if _tlsRoute.hostnames != _|_ {
					hostnames: _tlsRoute.hostnames
				}

				rules: [for rule in _tlsRoute.rules {
					backendRefs: [{
						name: _name
						port: rule.backendPort
					}]
				}]
			}
		}
	}
}
