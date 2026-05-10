package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#EncryptionConfigTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "encryption"
		description: "Enforces encryption requirements"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: encryption: #EncryptionConfigSchema
}

#EncryptionConfig: c.#Component & {
	#traits: (#EncryptionConfigTrait.metadata.fqn): #EncryptionConfigTrait
}

#EncryptionConfigSchema: {
	atRest:    bool
	inTransit: bool
}
