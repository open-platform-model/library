package extension

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
)

#CRDsResource: c.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/resources/extension"
		version:     "v1"
		name:        "crds"
		description: "One or more CustomResourceDefinitions to deploy to the cluster"
		labels: {
			"resource.opmodel.dev/category": "extension"
		}
	}

	spec: crds: [name=string]: schemas.#CRDSchema
}

#CRDs: c.#Component & {
	#resources: (#CRDsResource.metadata.fqn): #CRDsResource
}
