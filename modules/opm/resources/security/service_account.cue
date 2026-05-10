package security

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
)

#ServiceAccountResource: c.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/resources/security"
		version:     "v1"
		name:        "service-account"
		description: "A standalone ServiceAccount definition for identity"
		labels: {
			"resource.opmodel.dev/category": "security"
		}
	}

	spec: serviceAccount: schemas.#ServiceAccountSchema
}

#ServiceAccount: c.#Component & {
	#resources: (#ServiceAccountResource.metadata.fqn): #ServiceAccountResource
}
