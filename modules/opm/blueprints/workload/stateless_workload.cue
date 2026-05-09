package workload

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
	tr_workload "opmodel.dev/modules/opm/traits/workload"
)

#StatelessWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/blueprints/workload"
		version:     "v1"
		name:        "stateless-workload"
		description: "A stateless workload with no requirement for stable identity or storage"
	}

	composedResources: [
		res_workload.#ContainerResource,
	]

	composedTraits: [
		tr_workload.#ScalingTrait,
		tr_workload.#RestartPolicyTrait,
		tr_workload.#UpdateStrategyTrait,
		tr_workload.#SidecarContainersTrait,
		tr_workload.#InitContainersTrait,
	]

	spec: statelessWorkload: schemas.#StatelessWorkloadSchema
}
