package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

#GracefulShutdownTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/workload"
		version:     "v1"
		name:        "graceful-shutdown"
		description: "Termination grace period and pre-stop lifecycle hooks"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: gracefulShutdown: schemas.#GracefulShutdownSchema
}

#GracefulShutdown: c.#Component & {
	#traits: (#GracefulShutdownTrait.metadata.fqn): #GracefulShutdownTrait
}
