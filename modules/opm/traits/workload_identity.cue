package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#WorkloadIdentityTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
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
