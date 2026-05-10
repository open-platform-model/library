@if(test)

// Negative case: two ComponentTransformers requiring the SAME resource FQN
// AND sharing identical predicate signatures (no requiredLabels, no
// requiredTraits — both produce signature ";"). Per
// apis/core/v1alpha2/platform.cue:142-160, identical signatures across
// candidates for one FQN populate _invalid.resources, which violates
// _noMultiFulfiller (the 0-unify gate at platform.cue:189-192) and surfaces a
// CUE bottom on `input`.
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

_transformerA: {
	core.#ComponentTransformer
	metadata: {
		name:        "alpha"
		modulePath:  "example.com/t"
		version:     "v0"
		description: "first conflicting transformer"
	}
	requiredResources: "example.com/r/thing@v1": _resource
	#transform: {
		#runtimeName: string
		output: {}
	}
}

_transformerB: {
	core.#ComponentTransformer
	metadata: {
		name:        "beta"
		modulePath:  "example.com/t"
		version:     "v0"
		description: "second conflicting transformer — same predicate signature"
	}
	requiredResources: "example.com/r/thing@v1": _resource
	#transform: {
		#runtimeName: string
		output: {}
	}
}

_module: {
	core.#Module
	metadata: {
		name:       "conflict-module"
		modulePath: "example.com/m"
		version:    "0.1.0"
	}
	#defines: {
		resources: "example.com/r/thing@v1": _resource
		transformers: {
			"example.com/t/alpha@v0": _transformerA
			"example.com/t/beta@v0":  _transformerB
		}
	}
}

input: {
	core.#Platform
	metadata: name: "conflict-platform"
	type: "kubernetes"
	#registry: "conflict-module": #module: _module
}
