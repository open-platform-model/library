package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

// Enables hostNetwork: true on the pod spec, sharing the node's network
// namespace. Required for workloads that must bind to host interfaces
// directly (e.g. MetalLB speaker for ARP/NDP).
#HostNetworkTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
		name:        "host-network"
		description: "Share the node's network namespace (hostNetwork: true)"
		labels: {
			"trait.opmodel.dev/category": "network"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: hostNetwork: bool
}

#HostNetwork: c.#Component & {
	#traits: (#HostNetworkTrait.metadata.fqn): #HostNetworkTrait
}
