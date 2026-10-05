package unify_order

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "unify-order-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "unify_order"
		modulePath: "testing.opmodel.dev/library-render/scenarios/unify_order@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		// Three always-unify refusals on one component, with the candidates
		// walked in a different order from the platform's transformers: the
		// narrow resource comes first, so narrow-transformer is the first
		// candidate, yet it sits after deployment-transformer in the
		// catalog. The container resource is authored as a plain struct
		// whose port is a string, so the transformers' required
		// #ContainerResource copy conflicts with it.
		twice: {
			#resources: {
				(cat.#NarrowResource.metadata.fqn): cat.#NarrowResource & {
					spec: narrow: name: "some-other-name"
				}
				(cat.#ContainerResource.metadata.fqn): {
					metadata: cat.#ContainerResource.metadata
					matchLabels: "render.test/workload": "stateless"
					spec: container: {
						image: "nginx"
						port:  "eighty"
					}
				}
			}
			spec: {
				narrow: name: "some-other-name"
				container: {
					image: "nginx"
					port:  "eighty"
				}
			}
		}
	}
}

values: {}
