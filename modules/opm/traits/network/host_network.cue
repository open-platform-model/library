package network

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

// Enables hostNetwork: true on the pod spec, sharing the node's network
// namespace. Required for workloads that must bind to host interfaces
// directly (e.g. MetalLB speaker for ARP/NDP).
#HostNetworkTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/network"
		version:     "v1"
		name:        "host-network"
		description: "Share the node's network namespace (hostNetwork: true)"
		labels: {
			"trait.opmodel.dev/category": "network"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: hostNetwork: bool
}

#HostNetwork: c.#Component & {
	#traits: (#HostNetworkTrait.metadata.fqn): #HostNetworkTrait
}
