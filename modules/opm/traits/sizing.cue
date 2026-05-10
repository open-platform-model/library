package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#SizingTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
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
