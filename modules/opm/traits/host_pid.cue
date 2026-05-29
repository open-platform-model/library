package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

// Enables hostPID: true on the pod spec, sharing the node's PID namespace.
// Required for workloads that must observe or signal host processes.
#HostPIDTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
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
