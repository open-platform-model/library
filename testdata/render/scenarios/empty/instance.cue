package empty

import c "opmodel.dev/core@v2"

c.#ModuleInstance

metadata: {
	name:      "empty-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "empty"
		modulePath: "testing.opmodel.dev/library-render/scenarios/empty@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {}
}

values: {}
