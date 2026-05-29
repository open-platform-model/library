package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#SizingTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
		name:        "sizing"
		description: "A trait to specify vertical sizing behavior for a workload"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: sizing: #SizingSchema
}

#Sizing: c.#Component & {
	#traits: (#SizingTrait.metadata.fqn): #SizingTrait
}

#SizingSchema: {
	resources?:   res.#ResourceRequirementsSchema
	autoScaling?: #VerticalScalingSchema
}

// Placeholder for future VPA support.
#VerticalScalingSchema: {}
