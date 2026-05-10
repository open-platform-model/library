package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#WorkloadIdentityTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "workload-identity"
		description: "A workload identity definition for service identity"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [res.#ContainerResource]

	// Schema shape lives in resources/service_account.cue alongside #ServiceAccountSchema
	// because #RoleSubjectSchema (resources package) also references it.
	spec: workloadIdentity: res.#WorkloadIdentitySchema
}

#WorkloadIdentity: c.#Component & {
	#traits: (#WorkloadIdentityTrait.metadata.fqn): #WorkloadIdentityTrait
}
