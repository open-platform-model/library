package transformers

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
	k8sapiextv1 "opmodel.dev/modules/opm/schemas/kubernetes/apiextensions/v1@v1"
)

// CRDTransformer converts CRDs resources to Kubernetes CustomResourceDefinitions
#CRDTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "crd-transformer"
		description: "Converts CRDs resources to Kubernetes CustomResourceDefinitions"

		labels: {
			"core.opmodel.dev/resource-category": "extension"
			"core.opmodel.dev/resource-type":     "crd"
		}
	}

	requiredLabels: {}

	// Required resources - CRDs MUST be present
	requiredResources: {
		(res.#CRDsResource.metadata.fqn): res.#CRDsResource
	}

	optionalResources: {}
	requiredTraits: {}
	optionalTraits: {}

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_crds: #component.spec.crds

		// Emit one K8s CustomResourceDefinition per entry in the component's
		// crds map. Output is a list of resources; the renderer dispatches
		// on cue.Kind and produces one Compiled per list element.
		output: [
			for crdName, crd in _crds {
				k8sapiextv1.#CustomResourceDefinition & {
					apiVersion: "apiextensions.k8s.io/v1"
					kind:       "CustomResourceDefinition"
					metadata: {
						name:   crdName
						labels: #context.labels
					}
					spec: {
						group: crd.group
						names: {
							kind:   crd.names.kind
							plural: crd.names.plural
							if crd.names.singular != _|_ {
								singular: crd.names.singular
							}
							if crd.names.shortNames != _|_ {
								shortNames: crd.names.shortNames
							}
							if crd.names.categories != _|_ {
								categories: crd.names.categories
							}
						}
						scope:    crd.scope
						versions: crd.versions
					}
				}
			},
		]
	}
}
