@if(test)

// Negative case: symmetric to multi_fulfiller_fixture.cue but the conflict is
// on requiredTraits. Two transformers both require the same trait FQN AND
// share identical predicate signatures (no requiredLabels, no
// requiredResources). _invalid.traits becomes non-empty and _noMultiFulfiller
// gates with conflicting 0 / N. Catches an edit that keeps the resource-side
// _invalid logic but breaks the trait-side projection.
package fixtures

import (
	core "opmodel.dev/core/v1alpha2@v1"
)

_resource: {
	core.#Resource
	metadata: {
		name:       "anchor"
		modulePath: "example.com/r"
		version:    "v1"
	}
	spec: anchor: _
}

_trait: {
	core.#Trait
	metadata: {
		name:       "scaling"
		modulePath: "example.com/tr"
		version:    "v1"
	}
	spec: scaling: _
	appliesTo: [_resource]
}

_transformerAlpha: {
	core.#ComponentTransformer
	metadata: {
		name:        "scaling-alpha"
		modulePath:  "example.com/t"
		version:     "v0"
		description: "first conflicting trait-fulfiller"
	}
	requiredTraits: "example.com/tr/scaling@v1": _trait
	#transform: {
		#runtimeName: string
		output: {}
	}
}

_transformerBeta: {
	core.#ComponentTransformer
	metadata: {
		name:        "scaling-beta"
		modulePath:  "example.com/t"
		version:     "v0"
		description: "second conflicting trait-fulfiller — same predicate signature"
	}
	requiredTraits: "example.com/tr/scaling@v1": _trait
	#transform: {
		#runtimeName: string
		output: {}
	}
}

_module: {
	core.#Module
	metadata: {
		name:       "trait-conflict-module"
		modulePath: "example.com/m"
		version:    "0.1.0"
	}
	#defines: {
		traits: "example.com/tr/scaling@v1": _trait
		transformers: {
			"example.com/t/scaling-alpha@v0": _transformerAlpha
			"example.com/t/scaling-beta@v0":  _transformerBeta
		}
	}
}

input: {
	core.#Platform
	metadata: name: "trait-conflict-platform"
	type: "kubernetes"
	#registry: "trait-conflict-module": #module: _module
}
