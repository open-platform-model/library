package schemas

/////////////////////////////////////////////////////////////////
//// Extension Schemas
/////////////////////////////////////////////////////////////////

// CRDVersionSchema defines a single version entry in a CRD.
#CRDVersionSchema: {
	// Version name (e.g., "v1", "v1alpha2")
	name!: string
	// Whether this version is served by the API server
	served!: bool
	// Whether this is the storage version (exactly one version must be true)
	storage!: bool
	// Optional OpenAPI v3 validation schema for this version
	schema?: {
		openAPIV3Schema: {...}
	}
	// Optional subresources (status, scale)
	subresources?: {...}
	// Optional additional printer columns
	additionalPrinterColumns?: [...{...}]
}

// CRDSchema defines a Kubernetes CustomResourceDefinition to be deployed to the cluster.
// Use this to vendor operator CRDs (e.g., Grafana, cert-manager) alongside your module.
#CRDSchema: {
	// API group for the custom resource (e.g., "grafana.integreatly.org")
	group!: string
	// Names configuration for the custom resource
	names!: {
		// PascalCase kind name (e.g., "Grafana")
		kind!: string
		// Lowercase plural name used in URLs (e.g., "grafanas")
		plural!: string
		// Optional lowercase singular name
		singular?: string
		// Optional short names (e.g., ["gr"])
		shortNames?: [...string]
		// Optional category membership (e.g., ["all"])
		categories?: [...string]
	}
	// Whether instances are Namespaced or Cluster-scoped
	scope!: *"Namespaced" | "Cluster"
	// List of versions for this CRD (at least one required)
	versions!: [_, ...] & [...#CRDVersionSchema]
}
