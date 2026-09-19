package unlisted

import (
	c "opmodel.dev/core@v2"
)

c.#ModuleInstance

metadata: {
	name:      "unlisted-demo"
	namespace: "default"
}

// A contract no catalog on any fixture platform lists in its contract maps
// or implements with a transformer: authored inline under a path outside
// every served catalog. A demand for it is a hard miss whose row carries no
// defining catalog and no alternatives, so the refusal reads "no enabled
// catalog defines this contract" (0015:D18, the third arm).
#StrayResource: c.#Resource & {
	metadata: {
		name:           "stray"
		modulePath:     "testing.opmodel.dev/library-render/elsewhere/resources/v1"
		apiVersion:     "v1"
		catalogVersion: "0.0.1"
		fqn:            "testing.opmodel.dev/library-render/elsewhere/resources/stray@v1"
		description:    "A contract no catalog on the platform lists or implements"
	}
	spec: stray: note!: string
}

#module: {
	metadata: {
		name:       "unlisted"
		modulePath: "testing.opmodel.dev/library-render/scenarios/unlisted@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		stray: {
			#resources: (#StrayResource.metadata.fqn): #StrayResource
			spec: stray: note: "nobody lists me"
		}
	}
}

values: {}
