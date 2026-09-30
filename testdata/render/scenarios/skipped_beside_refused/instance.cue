package skipped_beside_refused

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
	providers "testing.opmodel.dev/library-render/providers@v0"
)

c.#ModuleInstance

metadata: {
	name:      "skipped-beside-refused-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "skipped_beside_refused"
		modulePath: "testing.opmodel.dev/library-render/scenarios/skipped_beside_refused@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		// The load-bearing snapshot trait nothing provides: skippable.
		app: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			#traits: (providers.#SnapshotTrait.metadata.fqn):  providers.#SnapshotTrait
			spec: {
				container: image:   "nginx:1.27"
				snapshot: schedule: "hourly"
			}
		}
		// The archive trait without the label its one provider requires:
		// a provider exists and does not match, a standing refusal.
		vault: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			#traits: (providers.#ArchiveTrait.metadata.fqn):   providers.#ArchiveTrait
			spec: {
				container: image: "vault:1.18"
				archive: target:  "s3"
			}
		}
	}
}

values: {}
