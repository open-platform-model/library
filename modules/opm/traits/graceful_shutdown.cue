package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#GracefulShutdownTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "graceful-shutdown"
		description: "Termination grace period and pre-stop lifecycle hooks"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: gracefulShutdown: #GracefulShutdownSchema
}

#GracefulShutdown: c.#Component & {
	#traits: (#GracefulShutdownTrait.metadata.fqn): #GracefulShutdownTrait
}

#GracefulShutdownSchema: {
	terminationGracePeriodSeconds: uint
}
