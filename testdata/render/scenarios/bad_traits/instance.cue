package bad_traits

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "bad-traits-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "bad_traits"
		modulePath: "testing.opmodel.dev/library-render/scenarios/bad_traits@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		web: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			// A top-level conflict: #traits exists but is bottom. The operator's
			// demand walk refuses it (Exists, then Fields errors); the glue's
			// presence guard would drop it.
			#traits: "not-a-trait-map"
			spec: container: image: "nginx:1.27"
		}
	}
}

values: {}
