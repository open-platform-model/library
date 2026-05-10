package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

#RestartPolicyTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
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
