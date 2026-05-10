@if(test)

// Negative case: two #Module values register on one #Platform, both publishing
// `#defines.resources["example.com/r/thing@v1"]` with conflicting concrete
// bodies (different metadata.description). The #knownResources projection in
// apis/core/v1alpha2/platform.cue:60-69 unifies the two definitions on the
// shared FQN; CUE bottoms on the field whose values disagree. This is the
// surface that catches accidental cross-module FQN reuse (D3 in the spec).
package fixtures

import (
	core "opmodel.dev/core/v1alpha2@v1"
)

_resourceA: {
	core.#Resource
	metadata: {
		name:        "thing"
		modulePath:  "example.com/r"
		version:     "v1"
		description: "version published by module-a"
	}
	spec: thing: _
}

_resourceB: {
	core.#Resource
	metadata: {
		name:        "thing"
		modulePath:  "example.com/r"
		version:     "v1"
		description: "version published by module-b — conflicts with module-a"
	}
	spec: thing: _
}

_moduleA: {
	core.#Module
	metadata: {
		name:       "module-a"
		modulePath: "example.com/m"
		version:    "0.1.0"
	}
	#defines: resources: "example.com/r/thing@v1": _resourceA
}

_moduleB: {
	core.#Module
	metadata: {
		name:       "module-b"
		modulePath: "example.com/m"
		version:    "0.1.0"
	}
	#defines: resources: "example.com/r/thing@v1": _resourceB
}

input: {
	core.#Platform
	metadata: name: "collision-platform"
	type: "kubernetes"
	#registry: {
		"module-a": #module: _moduleA
		"module-b": #module: _moduleB
	}
}
