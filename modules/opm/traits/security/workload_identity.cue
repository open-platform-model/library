package security

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

#WorkloadIdentityTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/security"
		version:     "v1"
		name:        "workload-identity"
		description: "A workload identity definition for service identity"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: workloadIdentity: schemas.#WorkloadIdentitySchema
}

#WorkloadIdentity: c.#Component & {
	#traits: (#WorkloadIdentityTrait.metadata.fqn): #WorkloadIdentityTrait
}
