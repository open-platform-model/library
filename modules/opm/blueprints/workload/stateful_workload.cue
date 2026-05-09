package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
	res_storage "opmodel.dev/modules/opm/resources/storage"
	tr_workload "opmodel.dev/modules/opm/traits/workload"
)

#StatefulWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/blueprints/workload"
		version:     "v1"
		name:        "stateful-workload"
		description: "A stateful workload with stable identity and persistent storage requirements"
	}

	composedResources: [
		res_workload.#ContainerResource,
		res_storage.#VolumesResource,
	]

	composedTraits: [
		tr_workload.#ScalingTrait,
		tr_workload.#RestartPolicyTrait,
		tr_workload.#UpdateStrategyTrait,
		tr_workload.#SidecarContainersTrait,
		tr_workload.#InitContainersTrait,
	]

	spec: statefulWorkload: schemas.#StatefulWorkloadSchema
}
