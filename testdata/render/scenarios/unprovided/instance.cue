package unprovided

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
	providers "testing.opmodel.dev/library-render/providers@v0"
)

c.#ModuleInstance

metadata: {
	name:      "unprovided-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "unprovided"
		modulePath: "testing.opmodel.dev/library-render/scenarios/unprovided@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		// A container the deployment transformer matches, plus a
		// load-bearing provider-fulfilled trait nothing provides.
		app: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			#traits: (providers.#SnapshotTrait.metadata.fqn):  providers.#SnapshotTrait
			spec: {
				container: image:   "nginx:1.27"
				snapshot: schedule: "hourly"
			}
		}
		// A container the deployment transformer matches, plus a
		// provider-fulfilled resource nothing provides, plus the same
		// unprovided trait.
		ledger: {
			#resources: {
				(cat.#ContainerResource.metadata.fqn):    cat.#ContainerResource
				(providers.#LedgerResource.metadata.fqn): providers.#LedgerResource
			}
			#traits: (providers.#SnapshotTrait.metadata.fqn): providers.#SnapshotTrait
			spec: {
				container: image:   "postgres:17"
				ledger: size:       "10Gi"
				snapshot: schedule: "daily"
			}
		}
		// Fully satisfied sibling.
		healthy: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			spec: container: image: "busybox:1.37"
		}
	}
}

values: {}
