@if(test)

// Positive case: a #ModuleRegistration whose `enabled: false` MUST suppress
// every projection of its primitives — none of its #defines surface in
// #knownResources, #knownTraits, or #composedTransformers (guards at
// platform.cue:62, 73, 84). The disabled module carries a small payload
// (two transformers + a resource) to make the suppression observable; a
// second, enabled module supplies a benign transformer so the resulting
// platform is non-trivial.
package fixtures

import (
	core "opmodel.dev/core/v1alpha2@v1"
)

// ── Disabled module: suppressed payload ──────────────────────────────────
_disabledResource: {
	core.#Resource
	metadata: {
		name:       "ghost"
		modulePath: "example.com/r"
		version:    "v1"
	}
	spec: ghost: _
}

_disabledTransformerA: {
	core.#ComponentTransformer
	metadata: {
		name:        "ghost-alpha"
		modulePath:  "example.com/t"
		version:     "v0"
		description: "would conflict with ghost-beta on example.com/r/ghost@v1"
	}
	requiredResources: "example.com/r/ghost@v1": _disabledResource
	#transform: {
		#runtimeName: string
		output: {}
	}
}

_disabledTransformerB: {
	core.#ComponentTransformer
	metadata: {
		name:        "ghost-beta"
		modulePath:  "example.com/t"
		version:     "v0"
		description: "second transformer in the disabled module"
	}
	requiredResources: "example.com/r/ghost@v1": _disabledResource
	#transform: {
		#runtimeName: string
		output: {}
	}
}

_disabledModule: {
	core.#Module
	metadata: {
		name:       "disabled-module"
		modulePath: "example.com/m"
		version:    "0.1.0"
	}
	#defines: {
		resources: "example.com/r/ghost@v1": _disabledResource
		transformers: {
			"example.com/t/ghost-alpha@v0": _disabledTransformerA
			"example.com/t/ghost-beta@v0":  _disabledTransformerB
		}
	}
}

// ── Enabled module: benign single transformer ────────────────────────────
_visibleResource: {
	core.#Resource
	metadata: {
		name:       "visible"
		modulePath: "example.com/r"
		version:    "v1"
	}
	spec: visible: _
}

_visibleTransformer: {
	core.#ComponentTransformer
	metadata: {
		name:        "visible-transformer"
		modulePath:  "example.com/t"
		version:     "v0"
		description: "the only transformer that should appear in the platform projections"
	}
	requiredResources: "example.com/r/visible@v1": _visibleResource
	#transform: {
		#runtimeName: string
		output: {}
	}
}

_visibleModule: {
	core.#Module
	metadata: {
		name:       "visible-module"
		modulePath: "example.com/m"
		version:    "0.1.0"
	}
	#defines: {
		resources: "example.com/r/visible@v1":             _visibleResource
		transformers: "example.com/t/visible-transformer@v0": _visibleTransformer
	}
}

input: {
	core.#Platform
	metadata: name: "suppress-platform"
	type: "kubernetes"
	#registry: {
		"disabled-module": {
			#module: _disabledModule
			enabled: false
		}
		"visible-module": #module: _visibleModule
	}
}

// The disabled module's ghost resource and ghost-{alpha,beta} transformers
// MUST NOT appear in any projection. Only the enabled module's primitives
// surface.
expect: {
	#matchers: {
		resources: "example.com/r/visible@v1": [_]
		traits: {}
	}
}
