@if(test)

// Positive case: a single #ComponentTransformer that requires a single resource
// FQN populates #matchers.resources with one candidate, and leaves
// #matchers._invalid empty (no duplicate-fulfiller conflicts). #Platform's
// _noMultiFulfiller hard-fail therefore evaluates to 0 and unifies cleanly.
package fixtures

import (
	core "opmodel.dev/core/v1alpha2@v1"
)

// _resource — single Resource definition published by _module.
_resource: {
	core.#Resource
	metadata: {
		name:       "thing"
		modulePath: "example.com/r"
		version:    "v1"
	}
	spec: thing: _
}

// _transformer — requires _resource by FQN. With only one transformer,
// the per-FQN candidate list has length 1 and _invalid is empty.
_transformer: {
	core.#ComponentTransformer
	metadata: {
		name:        "thing-transformer"
		modulePath:  "example.com/t"
		version:     "v0"
		description: "fixture transformer for matchers projection test"
	}
	requiredResources: "example.com/r/thing@v1": _resource
	#transform: {
		#runtimeName: string
		output: {}
	}
}

// _module — publishes _resource + _transformer through #defines, which is the
// hook #PlatformBase reads via #composedTransformers / #knownResources.
_module: {
	core.#Module
	metadata: {
		name:       "fixture-module"
		modulePath: "example.com/m"
		version:    "0.1.0"
	}
	#defines: {
		resources: "example.com/r/thing@v1":                _resource
		transformers: "example.com/t/thing-transformer@v0": _transformer
	}
}

input: {
	core.#Platform
	metadata: name: "fixture-platform"
	type: "kubernetes"
	#registry: "fixture-module": #module: _module
}

// expect — concrete equality target. Unifying input & expect under
// Validate(Concrete(true)) yields a fully-evaluated value with no bottoms if
// the matcher projection is correct. Single-FQN candidacy + empty _invalid +
// _noMultiFulfiller=0 is the contract under test.
expect: {
	#matchers: {
		resources: "example.com/r/thing@v1": [_]
		traits: {}
		_invalid: {
			resources: []
			traits: []
		}
		_noMultiFulfiller: 0
	}
}
