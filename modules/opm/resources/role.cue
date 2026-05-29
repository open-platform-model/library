package resources

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
)

/////////////////////////////////////////////////////////////////
//// Role Resource
/////////////////////////////////////////////////////////////////

#RoleResource: c.#Resource & {
	metadata: {
		modulePath:  "\(id.ModulePath)/resources"
		version:     id.Version
		name:        "role"
		description: "An RBAC Role definition with rules and CUE-referenced subjects"
		labels: {
			"resource.opmodel.dev/category": "security"
		}
	}

	spec: role: #RoleSchema
}

#Role: c.#Component & {
	#resources: (#RoleResource.metadata.fqn): #RoleResource
}

/////////////////////////////////////////////////////////////////
//// Role Schemas
/////////////////////////////////////////////////////////////////

// Single RBAC permission rule.
#PolicyRuleSchema: {
	apiGroups!: [...string]
	resources!: [...string]
	verbs!: [...string]
}

// Role subject — embeds an identity directly via CUE reference.
// References sibling primitives (#WorkloadIdentitySchema, #ServiceAccountSchema)
// in the same package.
#RoleSubjectSchema: {#WorkloadIdentitySchema | #ServiceAccountSchema}

#RoleSchema: {
	name!: string
	scope: "namespace" | "cluster"
	rules!: [...#PolicyRuleSchema] & [_, ...]
	subjects!: [...#RoleSubjectSchema] & [_, ...]
}

#RoleDefaults: #RoleSchema & {
	scope: "namespace"
}
