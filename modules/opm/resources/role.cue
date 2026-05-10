package resources

import (
	c "opmodel.dev/core/v1alpha2@v1"
)

/////////////////////////////////////////////////////////////////
//// Role Resource
/////////////////////////////////////////////////////////////////

#RoleResource: c.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/resources"
		version:     "v1"
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
