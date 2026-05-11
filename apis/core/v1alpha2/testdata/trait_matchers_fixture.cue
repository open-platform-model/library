@if(test)

// Positive case: symmetric to platform_matchers_fixture.cue but exercises
// the TRAITS branch of #Platform.#matchers. A single #ComponentTransformer
// requires one #Trait FQN. #matchers.traits["…@v1"] should contain one
// candidate; resources side stays empty. Catches edits to the resource
// projection that forget the trait twin.
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

_transformer: {
	core.#ComponentTransformer
	metadata: {
		name:        "scaling-transformer"
		modulePath:  "example.com/t"
		version:     "v0"
		description: "fixture transformer for trait-side matchers projection test"
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
		name:       "trait-fixture-module"
		modulePath: "example.com/m"
		version:    "0.1.0"
	}
	#defines: {
		traits: "example.com/tr/scaling@v1":              _trait
		transformers: "example.com/t/scaling-transformer@v0": _transformer
	}
}

input: {
	core.#Platform
	metadata: name: "trait-fixture-platform"
	type: "kubernetes"
	#registry: "trait-fixture-module": #module: _module
}

expect: {
	#matchers: {
		resources: {}
		traits: "example.com/tr/scaling@v1": [_]
	}
}
