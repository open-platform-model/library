package traits_only

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "traits-only-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "traits_only"
		modulePath: "testing.opmodel.dev/library-render/scenarios/traits_only@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		// An empty #resources map beside an attached trait: the demand is the
		// trait key alone.
		side: {
			#resources: {}
			#traits: (cat.#SidecarTrait.metadata.fqn): cat.#SidecarTrait
			spec: sidecar: image: "envoy:1.30"
		}
	}
}

values: {}
