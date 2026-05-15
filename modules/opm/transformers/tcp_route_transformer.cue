package transformers

import (
	c "opmodel.dev/core/v1alpha2@v1"
	tr "opmodel.dev/modules/opm/traits"
)

// TcpRouteTransformer creates Gateway API TCPRoutes from components with TcpRoute trait.
// Untyped struct output — see #HttpRouteTransformer for rationale.
#TcpRouteTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "tcp-route-transformer"
		description: "Creates Gateway API TCPRoutes for components with TcpRoute trait"

		labels: {
			"core.opmodel.dev/trait-type":    "network"
			"core.opmodel.dev/resource-type": "tcp-route"
		}
	}

	requiredLabels: {}
	requiredResources: {}
	optionalResources: {}

	requiredTraits: {
		"opmodel.dev/modules/opm/traits/tcp-route@v1": tr.#TcpRouteTrait
	}
	optionalTraits: {}

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_tcpRoute: #component.spec.tcpRoute
		_name:     "\(#context.#moduleReleaseMetadata.name)-\(#component.metadata.name)"

		_tlsAnnotations: {
			if _tcpRoute.tls != _|_ {
				if _tcpRoute.tls.mode != _|_ {
					"route.opmodel.dev/tls-mode": _tcpRoute.tls.mode
				}
				if _tcpRoute.tls.certificateRef != _|_ {
					if _tcpRoute.tls.certificateRef.namespace != _|_ {
						"route.opmodel.dev/tls-certificate-ref": "\(_tcpRoute.tls.certificateRef.namespace)/\(_tcpRoute.tls.certificateRef.name)"
					}
					if _tcpRoute.tls.certificateRef.namespace == _|_ {
						"route.opmodel.dev/tls-certificate-ref": _tcpRoute.tls.certificateRef.name
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
			kind:       "TCPRoute"
			metadata: {
				name:      _name
				namespace: #context.#moduleReleaseMetadata.namespace
				labels:    #context.labels
				if len(_routeAnnotations) > 0 {
					annotations: _routeAnnotations
				}
			}
			spec: {
				if _tcpRoute.gatewayRef != _|_ {
					parentRefs: [{
						name: _tcpRoute.gatewayRef.name
						if _tcpRoute.gatewayRef.namespace != _|_ {
							namespace: _tcpRoute.gatewayRef.namespace
						}
					}]
				}

				rules: [for rule in _tcpRoute.rules {
					backendRefs: [{
						name: _name
						port: rule.backendPort
					}]
				}]
			}
		}
	}
}
