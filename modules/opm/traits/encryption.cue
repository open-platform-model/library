package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#EncryptionConfigTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
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
