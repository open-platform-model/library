package resources

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
)

/////////////////////////////////////////////////////////////////
//// ConfigMaps Resource
/////////////////////////////////////////////////////////////////

#ConfigMapsResource: c.#Resource & {
	metadata: {
		modulePath:  "\(id.ModulePath)/resources"
		version:     id.Version
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
