package transformers

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
	k8scorev1 "opmodel.dev/modules/opm/schemas/kubernetes/core/v1@v1"
)

// PVCTransformer creates standalone PersistentVolumeClaims from Volume resources
#PVCTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
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
		"opmodel.dev/modules/opm/resources/volumes@v1": res.#VolumesResource
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

		// Generate PVC for each volume that has a persistentClaim defined
		output: {
			for volumeName, volume in _volumes if volume.persistentClaim != _|_ {
				"\(volumeName)": k8scorev1.#PersistentVolumeClaim & {
					apiVersion: "v1"
					kind:       "PersistentVolumeClaim"
					metadata: {
						name:      "\(#context.#moduleReleaseMetadata.name)-\(#context.#componentMetadata.name)-\(volumeName)"
						namespace: #context.#moduleReleaseMetadata.namespace
						labels:    #context.labels
						// Include component annotations if present
						if len(#context.componentAnnotations) > 0 {
							annotations: #context.componentAnnotations
						}
					}
					spec: {
						// accessMode is singular in schema, K8s expects accessModes array
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
			}
		}
	}
}
