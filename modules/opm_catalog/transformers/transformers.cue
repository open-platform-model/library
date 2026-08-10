// Package transformers holds the fixture catalog's four transformers — the
// set the web_app flow tests pair against (deployment, service, http-route,
// configmap). Outputs are deliberately minimal: the kernel's renderer
// dispatches on cue.Kind and never inspects fields inside the value, and the
// flow tests assert pairing and Compiled counts, not full k8s shapes.
//
// A transformer's fqn is #ImplFQNType — keyed by the BUILD it shipped in —
// and is authored here from the identity package (enhancement 0010 D4/D21).
// It carries no apiVersion, structurally (D44).
package transformers

import (
	c "opmodel.dev/core@v2"
	id "testing.opmodel.dev/catalogs/opm/identity"
	res "testing.opmodel.dev/catalogs/opm/resources"
	tr "testing.opmodel.dev/catalogs/opm/traits"
)

/////////////////////////////////////////////////////////////////
//// Deployment
/////////////////////////////////////////////////////////////////

#DeploymentTransformer: c.#ComponentTransformer & {
	metadata: {
		name:           "deployment-transformer"
		modulePath:     id.kindPrefix.transformers
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.transformers)/deployment-transformer@\(id.Version)"
		description:    "Converts stateless workload components with a Container resource to Deployments"
		labels: {
			"core.opmodel.dev/workload-type": "stateless"
			"core.opmodel.dev/resource-type": "deployment"
		}
	}

	requiredLabels: "core.opmodel.dev/workload-type": "stateless"

	requiredResources: (res.#ContainerResource.metadata.fqn): res.#ContainerResource

	optionalTraits: {
		(tr.#ScalingTrait.metadata.fqn):        tr.#ScalingTrait
		(tr.#RestartPolicyTrait.metadata.fqn):  tr.#RestartPolicyTrait
		(tr.#UpdateStrategyTrait.metadata.fqn): tr.#UpdateStrategyTrait
	}

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_container: #component.spec.container

		_replicas: int | *1
		if #component.spec.scaling != _|_ {
			_replicas: #component.spec.scaling.count
		}

		output: {
			apiVersion: "apps/v1"
			kind:       "Deployment"
			metadata: {
				name:      "\(#context.#moduleInstanceMetadata.name)-\(#context.#componentMetadata.name)"
				namespace: #context.#moduleInstanceMetadata.namespace
				labels:    #context.labels
			}
			spec: {
				replicas: _replicas
				template: spec: containers: [{
					name:  _container.name
					image: _container.image.reference
					if _container.ports != _|_ {
						ports: [for _, p in _container.ports {containerPort: p.targetPort}]
					}
				}]
			}
		}
	}
}

/////////////////////////////////////////////////////////////////
//// Service
/////////////////////////////////////////////////////////////////

#ServiceTransformer: c.#ComponentTransformer & {
	metadata: {
		name:           "service-transformer"
		modulePath:     id.kindPrefix.transformers
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.transformers)/service-transformer@\(id.Version)"
		description:    "Creates Services for components with the Expose trait"
		labels: {
			"core.opmodel.dev/trait-type":    "network"
			"core.opmodel.dev/resource-type": "service"
		}
	}

	requiredResources: (res.#ContainerResource.metadata.fqn): res.#ContainerResource

	requiredTraits: (tr.#ExposeTrait.metadata.fqn): tr.#ExposeTrait

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_expose: #component.spec.expose

		output: {
			apiVersion: "v1"
			kind:       "Service"
			metadata: {
				name:      "\(#context.#moduleInstanceMetadata.name)-\(#context.#componentMetadata.name)"
				namespace: #context.#moduleInstanceMetadata.namespace
				labels:    #context.labels
			}
			spec: {
				type: _expose.type
				selector: "app.kubernetes.io/name": #context.#componentMetadata.name
				ports: [
					for portName, p in _expose.ports {
						{
							name: portName
							// Service port: exposedPort when the author set one,
							// else targetPort. The list-index form is load-bearing —
							// a disjunction default would silently win over the
							// concrete exposedPort arm.
							port: [
								if p.exposedPort != _|_ {p.exposedPort},
								p.targetPort,
							][0]
							targetPort: p.targetPort
							protocol:   p.protocol
						}
					},
				]
			}
		}
	}
}

/////////////////////////////////////////////////////////////////
//// HTTP Route
/////////////////////////////////////////////////////////////////

#HttpRouteTransformer: c.#ComponentTransformer & {
	metadata: {
		name:           "http-route-transformer"
		modulePath:     id.kindPrefix.transformers
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.transformers)/http-route-transformer@\(id.Version)"
		description:    "Creates Gateway API HTTPRoutes for components with the HttpRoute trait"
		labels: {
			"core.opmodel.dev/trait-type":    "network"
			"core.opmodel.dev/resource-type": "http-route"
		}
	}

	requiredTraits: (tr.#HttpRouteTrait.metadata.fqn): tr.#HttpRouteTrait

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_httpRoute: #component.spec.httpRoute
		_name:      "\(#context.#moduleInstanceMetadata.name)-\(#context.#componentMetadata.name)"

		output: {
			apiVersion: "gateway.networking.k8s.io/v1"
			kind:       "HTTPRoute"
			metadata: {
				name:      _name
				namespace: #context.#moduleInstanceMetadata.namespace
				labels:    #context.labels
			}
			spec: {
				if _httpRoute.hostnames != _|_ {
					hostnames: _httpRoute.hostnames
				}
				rules: [
					for r in _httpRoute.rules {
						{
							if r.matches != _|_ {
								matches: r.matches
							}
							backendRefs: [{
								name: _name
								port: r.backendPort
							}]
						}
					},
				]
			}
		}
	}
}

/////////////////////////////////////////////////////////////////
//// ConfigMap
/////////////////////////////////////////////////////////////////

#ConfigMapTransformer: c.#ComponentTransformer & {
	metadata: {
		name:           "configmap-transformer"
		modulePath:     id.kindPrefix.transformers
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.transformers)/configmap-transformer@\(id.Version)"
		description:    "Converts ConfigMaps resources to ConfigMaps"
		labels: {
			"core.opmodel.dev/resource-category": "config"
			"core.opmodel.dev/resource-type":     "configmap"
		}
	}

	requiredResources: (res.#ConfigMapsResource.metadata.fqn): res.#ConfigMapsResource

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_configMaps: #component.spec.configMaps

		let _relName = #context.#moduleInstanceMetadata.name
		let _compName = #context.#componentMetadata.name

		// One ConfigMap per entry — list output, so the renderer emits one
		// Compiled per element (the flow test's 2-entries → 2-Compiled pin).
		output: [
			for _, cm in _configMaps {
				{
					apiVersion: "v1"
					kind:       "ConfigMap"
					metadata: {
						name:      "\(_relName)-\(_compName)-\(cm.name)"
						namespace: #context.#moduleInstanceMetadata.namespace
						labels:    #context.labels
					}
					if cm.immutable {
						immutable: true
					}
					data: cm.data
				}
			},
		]
	}
}
