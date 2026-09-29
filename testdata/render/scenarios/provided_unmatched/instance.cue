package provided_unmatched

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
	providers "testing.opmodel.dev/library-render/providers@v0"
)

c.#ModuleInstance

metadata: {
	name:      "provided-unmatched-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "provided_unmatched"
		modulePath: "testing.opmodel.dev/library-render/scenarios/provided_unmatched@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		// The archive trait's one provider requires the label
		// render.test/archive: "on", which this component never carries:
		// a provider exists and does not match.
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
