package security

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
)

#RoleResource: c.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/resources/security"
		version:     "v1"
		name:        "role"
		description: "An RBAC Role definition with rules and CUE-referenced subjects"
		labels: {
			"resource.opmodel.dev/category": "security"
		}
	}

	spec: role: schemas.#RoleSchema
}
