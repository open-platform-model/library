package failing

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "failing-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "failing"
		modulePath: "testing.opmodel.dev/library-render/scenarios/failing@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		web: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			spec: container: image: "nginx:1.27"
		}
		crash: {
			#resources: (cat.#BrokenResource.metadata.fqn): cat.#BrokenResource
			spec: broken: note: "conflicts at application"
		}
	}
}

values: {}
