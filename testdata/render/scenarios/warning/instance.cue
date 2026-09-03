package warning

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "warning-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "warning"
		modulePath: "testing.opmodel.dev/library-render/scenarios/warning@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		web: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			#traits: (cat.#SidecarTrait.metadata.fqn):         cat.#SidecarTrait
			spec: {
				container: image: "nginx:1.27"
				sidecar: image:   "envoy:1.30"
			}
		}
	}
}

values: {}
