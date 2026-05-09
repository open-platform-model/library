package transformers

import (
	c "opmodel.dev/core/v1alpha2@v1"
	workload_resources "opmodel.dev/modules/opm/resources/workload@v1"
	network_traits "opmodel.dev/modules/opm/traits/network@v1"
	k8scorev1 "opmodel.dev/modules/opm/schemas/kubernetes/core/v1@v1"
)

// ServiceTransformer creates Kubernetes Services from components with Expose trait
#ServiceTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "service-transformer"
		description: "Creates Kubernetes Services for components with Expose trait"

		labels: {
			"core.opmodel.dev/trait-type":    "network"
			"core.opmodel.dev/resource-type": "service"
		}
	}

	requiredLabels: {} // No specific labels required; matches any component with Expose trait

	// Required resources - Container MUST be present to know which ports to expose
	requiredResources: {
		"opmodel.dev/modules/opm/resources/workload/container@v1": workload_resources.#ContainerResource
	}

	// No optional resources
	optionalResources: {}

	// Required traits - Expose is mandatory for Service creation
	requiredTraits: {
		"opmodel.dev/modules/opm/traits/network/expose@v1": network_traits.#ExposeTrait
	}

	// No optional traits
	optionalTraits: {}

	#transform: {
		#component: _ // Unconstrained; validated by matching, not by transform signature
		#context:   c.#TransformerContext

		// Extract required Container resource (will be bottom if not present)
		_container: #component.spec.container

		// Extract required Expose trait (will be bottom if not present)
		_expose: #component.spec.expose

		// Build port list from expose trait ports
		// Schema: targetPort = container port, exposedPort = optional external port
		// K8s Service: port = service port (external), targetPort = pod port
		_ports: [
			for portName, portConfig in _expose.ports {
				{
					name: portName
					// Service port: use exposedPort if specified, else targetPort
					port:       portConfig.exposedPort | *portConfig.targetPort
					targetPort: portConfig.targetPort
					protocol:   portConfig.protocol | *"TCP"
					if _expose.type == "NodePort" && portConfig.exposedPort != _|_ {
						nodePort: portConfig.exposedPort
					}
				}
			},
		]

		// Build Service resource
		output: k8scorev1.#Service & {
			apiVersion: "v1"
			kind:       "Service"
			metadata: {
				name:      "\(#context.#moduleReleaseMetadata.name)-\(#component.metadata.name)"
				namespace: #context.#moduleReleaseMetadata.namespace
				labels:    #context.labels
				// Include component annotations if present
				if len(#context.componentAnnotations) > 0 {
					annotations: #context.componentAnnotations
				}
			}
			spec: {
				type: _expose.type

				selector: #context.componentLabels

				ports: _ports
			}
		}
	}
}
