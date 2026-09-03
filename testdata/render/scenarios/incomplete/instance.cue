package incomplete

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "incomplete-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "incomplete"
		modulePath: "testing.opmodel.dev/library-render/scenarios/incomplete@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		web: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			spec: container: image: "nginx:1.27"
		}
		hole: {
			#resources: (cat.#IncompleteResource.metadata.fqn): cat.#IncompleteResource
			spec: incomplete: note: "never concrete"
		}
	}
}

values: {}
