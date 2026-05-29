package resources

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
)

/////////////////////////////////////////////////////////////////
//// ServiceAccount Resource
/////////////////////////////////////////////////////////////////

#ServiceAccountResource: c.#Resource & {
	metadata: {
		modulePath:  "\(id.ModulePath)/resources"
		version:     id.Version
		name:        "service-account"
		description: "A standalone ServiceAccount definition for identity"
		labels: {
			"resource.opmodel.dev/category": "security"
		}
	}

	spec: serviceAccount: #ServiceAccountSchema
}

#ServiceAccount: c.#Component & {
	#resources: (#ServiceAccountResource.metadata.fqn): #ServiceAccountResource
}

/////////////////////////////////////////////////////////////////
//// Identity Schemas
//// Both #ServiceAccountSchema and #WorkloadIdentitySchema live here.
//// #WorkloadIdentitySchema is read by traits/workload_identity.cue
//// (the trait spec) and by #RoleSubjectSchema in role.cue.
/////////////////////////////////////////////////////////////////

#ServiceAccountSchema: {
	name!:           string
	automountToken?: bool
}

// Workload identity — used by #WorkloadIdentityTrait and as a #RoleSubjectSchema variant.
#WorkloadIdentitySchema: {
	name!:           string
	automountToken?: bool
}
