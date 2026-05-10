package resources

import (
	c "opmodel.dev/core/v1alpha2@v1"
)

/////////////////////////////////////////////////////////////////
//// ConfigMaps Resource
/////////////////////////////////////////////////////////////////

#ConfigMapsResource: c.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/resources"
		version:     "v1"
		name:        "config-maps"
		description: "A ConfigMap definition for external configuration"
		labels: {
			"resource.opmodel.dev/category": "config"
		}
	}

	spec: configMaps: [cmName=string]: #ConfigMapSchema & {name: string | *cmName}
}

#ConfigMaps: c.#Component & {
	#resources: (#ConfigMapsResource.metadata.fqn): #ConfigMapsResource
}

/////////////////////////////////////////////////////////////////
//// ConfigMap Schema
/////////////////////////////////////////////////////////////////

// ConfigMap specification.
// `name` is auto-populated from the map key in the resource spec.
#ConfigMapSchema: {
	name!:     string
	immutable: bool
	data: [string]: string
}

#ConfigMapDefaults: #ConfigMapSchema & {
	immutable: false
}
