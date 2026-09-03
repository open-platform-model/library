package mislabeled

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "mislabeled-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "mislabeled"
		modulePath: "testing.opmodel.dev/library-render/scenarios/mislabeled@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		// The component's matchLabels are the wholesale unification of its
		// primitives' (0010 D36): render.test/tier is the int 2 the
		// resource declares. Its only candidate requires 3 under the same
		// key, so the predicate rung refuses it and names the label.
		tiered: {
			#resources: (cat.#TieredResource.metadata.fqn): cat.#TieredResource
			spec: tiered: name: "gold"
		}
	}
}

values: {}
