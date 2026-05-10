package resources

import (
	c "opmodel.dev/core/v1alpha2@v1"
)

/////////////////////////////////////////////////////////////////
//// CRDs Resource
/////////////////////////////////////////////////////////////////

#CRDsResource: c.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/resources"
		version:     "v1"
		name:        "crds"
		description: "One or more CustomResourceDefinitions to deploy to the cluster"
		labels: {
			"resource.opmodel.dev/category": "extension"
		}
	}

	spec: crds: [name=string]: #CRDSchema
}

#CRDs: c.#Component & {
	#resources: (#CRDsResource.metadata.fqn): #CRDsResource
}

/////////////////////////////////////////////////////////////////
//// CRD Schemas
/////////////////////////////////////////////////////////////////

// A single version entry in a CRD.
#CRDVersionSchema: {
	name!:    string
	served!:  bool
	storage!: bool
	schema?: {
		openAPIV3Schema: {...}
	}
	subresources?: {...}
	additionalPrinterColumns?: [...{...}]
}

// Kubernetes CustomResourceDefinition. Vendor operator CRDs alongside your module.
#CRDSchema: {
	group!: string
	names!: {
		kind!:     string
		plural!:   string
		singular?: string
		shortNames?: [...string]
		categories?: [...string]
	}
	scope!: "Namespaced" | "Cluster"
	versions!: [_, ...] & [...#CRDVersionSchema]
}

#CRDDefaults: #CRDSchema & {
	scope: "Namespaced"
}
