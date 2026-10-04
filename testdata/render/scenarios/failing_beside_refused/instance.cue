package failing_beside_refused

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "failing-beside-refused-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "failing_beside_refused"
		modulePath: "testing.opmodel.dev/library-render/scenarios/failing_beside_refused@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		crash: {
			#resources: (cat.#BrokenResource.metadata.fqn): cat.#BrokenResource
			spec: broken: note: "conflicts at application"
		}
		orphan: {
			#resources: (cat.#OrphanResource.metadata.fqn): cat.#OrphanResource
			spec: orphan: size: "10Gi"
		}
	}
}

values: {}
