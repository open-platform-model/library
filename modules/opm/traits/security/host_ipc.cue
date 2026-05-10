package security

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

// Enables hostIPC: true on the pod spec, sharing the node's IPC namespace.
// Required for workloads that use shared memory or IPC mechanisms with host
// processes.
#HostIPCTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/security"
		version:     "v1"
		name:        "host-ipc"
		description: "Share the node's IPC namespace (hostIPC: true)"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: hostIpc: bool
}

#HostIPC: c.#Component & {
	#traits: (#HostIPCTrait.metadata.fqn): #HostIPCTrait
}
