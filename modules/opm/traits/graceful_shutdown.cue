package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#GracefulShutdownTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
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
