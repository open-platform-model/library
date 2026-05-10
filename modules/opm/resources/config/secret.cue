package config

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
)

#SecretsResource: c.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/resources/config"
		version:     "v1"
		name:        "secrets"
		description: "A Secret definition for sensitive configuration"
		labels: {
			"resource.opmodel.dev/category": "config"
		}
	}

	spec: secrets: [secretName=string]: schemas.#SecretSchema & {name: string | *secretName}
}

#Secrets: c.#Component & {
	#resources: (#SecretsResource.metadata.fqn): #SecretsResource
}
