@if(test)

// Positive case: two #ComponentTransformers require the SAME resource FQN but
// differ in `requiredLabels` ({tier: "prod"} vs {tier: "dev"}). Per
// _predicateSignature in apis/core/v1alpha2/platform.cue:142-160, distinct
// labelParts produce distinct signatures, so #matchers._invalid stays empty
// even though both transformers appear in #matchers.resources for the same
// FQN. This proves the design claim at platform.cue:163-166: "Shared FQNs
// across candidates with *different* predicates are fine."
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

// Both transformers appear as candidates for the shared FQN; _invalid stays
// empty because their predicate signatures differ ("tier=prod;" vs "tier=dev;");
// _noMultiFulfiller therefore evaluates to 0 and unifies cleanly.
expect: {
	#matchers: {
		resources: "example.com/r/thing@v1": [_, _]
		traits: {}
		_invalid: {
			resources: []
			traits: []
		}
		_noMultiFulfiller: 0
	}
}
