package transformers

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
	k8scorev1 "opmodel.dev/modules/opm/schemas/kubernetes/core/v1@v1"
)

// ConfigMapTransformer converts ConfigMaps resources to Kubernetes ConfigMaps.
// Supports immutable ConfigMaps with content-hash naming.
#ConfigMapTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "configmap-transformer"
		description: "Converts ConfigMaps resources to Kubernetes ConfigMaps"

		labels: {
			"core.opmodel.dev/resource-category": "config"
			"core.opmodel.dev/resource-type":     "configmap"
		}
	}

	requiredLabels: {}

	// Required resources - ConfigMaps MUST be present
	requiredResources: {
		"opmodel.dev/modules/opm/resources/config-maps@v1": res.#ConfigMapsResource
	}

	optionalResources: {}
	requiredTraits: {}
	optionalTraits: {}

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_configMaps: #component.spec.configMaps

		// Build the release-scoped prefix: {releaseName}-{componentName}
		// Mirrors the secret-transformer convention so all config resources
		// share the same namespace-isolation guarantee across releases.
		let _relName = #context.#moduleReleaseMetadata.name
		let _compName = #context.#componentMetadata.name

		// Emit one K8s ConfigMap per entry in the component's configMaps map.
		// Output is a list of resources; the renderer dispatches on cue.Kind
		// (see apis/core/v1alpha2/transformer.cue) and produces one Compiled
		// per list element.
		output: [
			for _, cm in _configMaps
			let _baseName = "\(_relName)-\(_compName)-\(cm.name)"
			let _k8sName = (res.#ImmutableName & {
				baseName:  _baseName
				data:      cm.data
				immutable: cm.immutable
			}).out {
				k8scorev1.#ConfigMap & {
					apiVersion: "v1"
					kind:       "ConfigMap"
					metadata: {
						name:      _k8sName
						namespace: #context.#moduleReleaseMetadata.namespace
						labels:    #context.labels
						if len(#context.componentAnnotations) > 0 {
							annotations: #context.componentAnnotations
						}
					}
					if cm.immutable == true {
						immutable: true
					}
					data: cm.data
				}
			},
		]
	}
}
