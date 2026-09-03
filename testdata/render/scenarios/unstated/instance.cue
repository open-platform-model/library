package unstated

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "unstated-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "unstated"
		modulePath: "testing.opmodel.dev/library-render/scenarios/unstated@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		web: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			#traits: (cat.#UnstatedTrait.metadata.fqn):        cat.#UnstatedTrait
			spec: {
				container: image: "nginx:1.27"
				unstated: note:   "no posture"
			}
		}
	}
}

values: {}
