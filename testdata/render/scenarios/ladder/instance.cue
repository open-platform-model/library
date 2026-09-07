package ladder

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "ladder-demo"
	namespace: "default"
}

// The rung component demands ladder@v2, which no transformer implements. The
// platform implements the same base at v1alpha1, v1beta1 and v1, so the
// unresolved-demand row must list those three in ladder order (D34/D4), which
// is not their lexical order.
#module: {
	metadata: {
		name:       "ladder"
		modulePath: "testing.opmodel.dev/library-render/scenarios/ladder@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		rung: {
			#resources: (cat.#LadderV2Resource.metadata.fqn): cat.#LadderV2Resource
			spec: ladder: rung: "top"
		}
	}
}

values: {}
