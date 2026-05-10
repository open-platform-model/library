package traits

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
)

// Enables hostIPC: true on the pod spec, sharing the node's IPC namespace.
// Required for workloads that use shared memory or IPC mechanisms with host
// processes.
#HostIPCTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits"
		version:     "v1"
		name:        "host-ipc"
		description: "Share the node's IPC namespace (hostIPC: true)"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: hostIpc: bool
}

#HostIPC: c.#Component & {
	#traits: (#HostIPCTrait.metadata.fqn): #HostIPCTrait
}
