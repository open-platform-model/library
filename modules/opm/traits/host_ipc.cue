package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

// Enables hostIPC: true on the pod spec, sharing the node's IPC namespace.
// Required for workloads that use shared memory or IPC mechanisms with host
// processes.
#HostIPCTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
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
