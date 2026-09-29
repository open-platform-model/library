package optional_unprovided

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
	providers "testing.opmodel.dev/library-render/providers@v0"
)

c.#ModuleInstance

metadata: {
	name:      "optional-unprovided-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "optional_unprovided"
		modulePath: "testing.opmodel.dev/library-render/scenarios/optional_unprovided@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		// A container the deployment transformer matches, plus the
		// provider-fulfilled snapshot trait nothing provides, made
		// effectively optional at the attachment site: an advisory
		// unhandled trait, never a skipped row.
		app: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			#traits: (providers.#SnapshotTrait.metadata.fqn): providers.#SnapshotTrait & {optional: true}
			spec: {
				container: image:   "nginx:1.27"
				snapshot: schedule: "hourly"
			}
		}
	}
}

values: {}
