package config

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
)

#ConfigMapsResource: c.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/resources/config"
		version:     "v1"
		name:        "config-maps"
		description: "A ConfigMap definition for external configuration"
		labels: {
			"resource.opmodel.dev/category": "config"
		}
	}

	spec: configMaps: [cmName=string]: schemas.#ConfigMapSchema & {name: string | *cmName}
}

#ConfigMaps: c.#Component & {
	#resources: (#ConfigMapsResource.metadata.fqn): #ConfigMapsResource
}
