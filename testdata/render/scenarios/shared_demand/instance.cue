package shared_demand

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "shared-demand-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "shared_demand"
		modulePath: "testing.opmodel.dev/library-render/scenarios/shared_demand@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		web: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			#traits: (cat.#ExposeTrait.metadata.fqn):          cat.#ExposeTrait
			spec: {
				container: image: "nginx:1.27"
				expose: port:     80
			}
		}
		// The same resource key as web, and no #traits at all.
		worker: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			spec: container: image: "busybox:1.36"
		}
		config: {
			#resources: (cat.#ConfigMapsResource.metadata.fqn): cat.#ConfigMapsResource
			spec: configMaps: app: data: MODE: "prod"
		}
	}
}

values: {}
