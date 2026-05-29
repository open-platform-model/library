package transformers

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

// ServiceAccountResourceTransformer converts standalone ServiceAccount resources
// to Kubernetes ServiceAccounts. Separate from the WorkloadIdentity-based
// #ServiceAccountTransformer which handles trait-attached identities.
#ServiceAccountResourceTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "\(id.ModulePath)/transformers"
		version:     id.Version
		name:        "serviceaccount-resource-transformer"
		description: "Converts standalone ServiceAccount resources to Kubernetes ServiceAccounts"

		labels: {
			"core.opmodel.dev/resource-category": "security"
			"core.opmodel.dev/resource-type":     "serviceaccount-resource"
		}
	}

	requiredLabels: {}

	// Required resources - ServiceAccount resource MUST be present
	requiredResources: {
		(res.#ServiceAccountResource.metadata.fqn): res.#ServiceAccountResource
	}

	optionalResources: {}
	requiredTraits: {}
	optionalTraits: {}

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_serviceAccount: #component.spec.serviceAccount

		output: (#ToK8sServiceAccount & {
			"in":    _serviceAccount
			context: #context
		}).out
	}
}

/////////////////////////////////////////////////////////////////
//// Test Data
/////////////////////////////////////////////////////////////////

_testSAResourceComponent: res.#ServiceAccount & {
	spec: serviceAccount: {
		name:           "ci-bot"
		automountToken: false
	}
}

_testSAResourceTransformer: (#ServiceAccountResourceTransformer.#transform & {
	#component: _testSAResourceComponent
	#context: {
		#moduleReleaseMetadata: {
			name:      "test-release"
			namespace: "ci"
			fqn:       "opmodel.dev/catalogs/opm/test-release@0.1.0"
			version:   "0.1.0"
			uuid:      "00000000-0000-0000-0000-000000000000"
		}
		#componentMetadata: {
			name: "ci-bot"
		}
		componentAnnotations: {}
	}
}).output
