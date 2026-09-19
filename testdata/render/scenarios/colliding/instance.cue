// Two components whose rendered objects land on one apply identity: both
// carry the config-maps resource with the same map key, and the fixture
// catalog's configmap-transformer names each ConfigMap after the key alone,
// not after the component. The render succeeds — the kernel reads no
// Kubernetes identity — and only a caller of opm/helper/objectset learns
// that the second write would overwrite the first.
package colliding

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "colliding-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "colliding"
		modulePath: "testing.opmodel.dev/library-render/scenarios/colliding@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		primary: {
			#resources: (cat.#ConfigMapsResource.metadata.fqn): cat.#ConfigMapsResource
			spec: configMaps: app: data: MODE: "prod"
		}
		shadow: {
			#resources: (cat.#ConfigMapsResource.metadata.fqn): cat.#ConfigMapsResource
			spec: configMaps: app: data: MODE: "staging"
		}
	}
}

values: {}
