package disqualified

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "disqualified-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "disqualified"
		modulePath: "testing.opmodel.dev/library-render/scenarios/disqualified@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		// The attachment narrows the primitive body to a name the
		// transformer's required copy excludes: a schema-level divergence
		// the always-unify rung disqualifies through plain `&`.
		narrow: {
			#resources: (cat.#NarrowResource.metadata.fqn): cat.#NarrowResource & {
				spec: narrow: name: "some-other-name"
			}
			spec: narrow: name: "some-other-name"
		}
	}
}

values: {}
