package transformers

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
	k8scorev1 "opmodel.dev/catalogs/opm/schemas/kubernetes/core/v1"
)

// PVCTransformer creates standalone PersistentVolumeClaims from Volume resources
#PVCTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "\(id.ModulePath)/transformers"
		version:     id.Version
		name:        "pvc-transformer"
		description: "Creates standalone Kubernetes PersistentVolumeClaims from Volume resources"

		labels: {
			"core.opmodel.dev/resource-category": "storage"
			"core.opmodel.dev/resource-type":     "persistentvolumeclaim"
		}
	}

	requiredLabels: {} // No specific labels required; matches any component with Volumes resource

	// Required resources - Volumes MUST be present
	requiredResources: {
		(res.#VolumesResource.metadata.fqn): res.#VolumesResource
	}

	// No optional resources
	optionalResources: {}

	// No required traits
	requiredTraits: {}

	// No optional traits
	optionalTraits: {}

	#transform: {
		#component: _ // Unconstrained; validated by matching, not by transform signature
		#context:   c.#TransformerContext

		// Extract required Volumes resource (will be bottom if not present)
		_volumes: #component.spec.volumes

		// Emit one PVC per volume that declares a persistentClaim. Output is
		// a list of resources; the renderer dispatches on cue.Kind and
		// produces one Compiled per list element.
		output: [
			for volumeName, volume in _volumes if volume.persistentClaim != _|_ {
				k8scorev1.#PersistentVolumeClaim & {
					apiVersion: "v1"
					kind:       "PersistentVolumeClaim"
					metadata: {
						name:      "\(#context.#moduleReleaseMetadata.name)-\(#context.#componentMetadata.name)-\(volumeName)"
						namespace: #context.#moduleReleaseMetadata.namespace
						labels:    #context.labels
						if len(#context.componentAnnotations) > 0 {
							annotations: #context.componentAnnotations
						}
					}
					spec: {
						accessModes: [volume.persistentClaim.accessMode | *"ReadWriteOnce"]
						resources: {
							requests: {
								storage: volume.persistentClaim.size
							}
						}
						if volume.persistentClaim.storageClass != _|_ {
							storageClassName: volume.persistentClaim.storageClass
						}
					}
				}
			},
		]
	}
}
