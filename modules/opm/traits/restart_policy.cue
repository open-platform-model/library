package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

#RestartPolicyTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
		name:        "restart-policy"
		description: "A trait to specify the restart policy for a workload"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: restartPolicy: #RestartPolicySchema
}

#RestartPolicy: c.#Component & {
	#traits: (#RestartPolicyTrait.metadata.fqn): #RestartPolicyTrait
}

#RestartPolicySchema: "Always" | "OnFailure" | "Never"
