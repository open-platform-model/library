@if(test)

// Positive case: two #ComponentTransformers require the SAME resource FQN but
// differ in `requiredLabels` ({tier: "prod"} vs {tier: "dev"}). Both appear in
// #matchers.resources as candidates for the shared FQN — the runtime matcher
// is responsible for evaluating each candidate's predicate against the
// component at match time.
package fixtures

import (
	core "opmodel.dev/core/v1alpha2@v1"
)

_resource: {
	core.#Resource
	metadata: {
		name:       "thing"
		modulePath: "example.com/r"
		version:    "v1"
	}
	spec: thing: _
}

_transformerProd: {
	core.#ComponentTransformer
	metadata: {
		name:        "thing-prod"
		modulePath:  "example.com/t"
		version:     "v0"
		description: "applies to prod-tier components"
	}
	requiredLabels: tier: "prod"
	requiredResources: "example.com/r/thing@v1": _resource
	#transform: {
		#runtimeName: string
		output: {}
	}
}

_transformerDev: {
	core.#ComponentTransformer
	metadata: {
		name:        "thing-dev"
		modulePath:  "example.com/t"
		version:     "v0"
		description: "applies to dev-tier components"
	}
	requiredLabels: tier: "dev"
	requiredResources: "example.com/r/thing@v1": _resource
	#transform: {
		#runtimeName: string
		output: {}
	}
}

_module: {
	core.#Module
	metadata: {
		name:       "distinct-predicate-module"
		modulePath: "example.com/m"
		version:    "0.1.0"
	}
	#defines: {
		resources: "example.com/r/thing@v1": _resource
		transformers: {
			"example.com/t/thing-prod@v0": _transformerProd
			"example.com/t/thing-dev@v0":  _transformerDev
		}
	}
}

input: {
	core.#Platform
	metadata: name: "distinct-predicate-platform"
	type: "kubernetes"
	#registry: "distinct-predicate-module": #module: _module
}

// Both transformers appear as candidates for the shared FQN.
expect: {
	#matchers: {
		resources: "example.com/r/thing@v1": [_, _]
		traits: {}
	}
}
