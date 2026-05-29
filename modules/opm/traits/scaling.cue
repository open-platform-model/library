package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#ScalingTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
		name:        "scaling"
		description: "A trait to specify scaling behavior for a workload"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: scaling: #ScalingSchema
}

#Scaling: c.#Component & {
	#traits: (#ScalingTrait.metadata.fqn): #ScalingTrait
}

/////////////////////////////////////////////////////////////////
//// Scaling Schemas
/////////////////////////////////////////////////////////////////

#ScalingSchema: {
	count: int & >=0 & <=1000
	auto?: #AutoscalingSpec
}

#AutoscalingSpec: {
	min!: int & >=1
	max!: int & >=1
	metrics!: [_, ...#MetricSpec]
	behavior?: {
		scaleUp?: {stabilizationWindowSeconds?: int}
		scaleDown?: {stabilizationWindowSeconds?: int}
	}
}

#MetricSpec: {
	type!:   "cpu" | "memory" | "custom"
	target!: #MetricTargetSpec
	if type == "custom" {
		metricName!: string
	}
}

#MetricTargetSpec: {
	averageUtilization?: int & >=1 & <=100
	averageValue?:       string
}
