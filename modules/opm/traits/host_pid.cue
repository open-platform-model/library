package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

// Enables hostPID: true on the pod spec, sharing the node's PID namespace.
// Required for workloads that must observe or signal host processes.
#HostPIDTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "host-pid"
		description: "Share the node's PID namespace (hostPID: true)"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: hostPid: bool
}

#HostPID: c.#Component & {
	#traits: (#HostPIDTrait.metadata.fqn): #HostPIDTrait
}
