package security

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

// Enables hostPID: true on the pod spec, sharing the node's PID namespace.
// Required for workloads that must observe or signal host processes.
#HostPIDTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/security"
		version:     "v1"
		name:        "host-pid"
		description: "Share the node's PID namespace (hostPID: true)"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: hostPid: bool
}
