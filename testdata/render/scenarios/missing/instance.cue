package missing

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#ModuleInstance

metadata: {
	name:      "missing-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "missing"
		modulePath: "testing.opmodel.dev/library-render/scenarios/missing@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		orphan: {
			#resources: {
				(cat.#OrphanResource.metadata.fqn):     cat.#OrphanResource
				(cat.#ConfigMapsResource.metadata.fqn): cat.#ConfigMapsResource
			}
			#traits: (cat.#BackupTrait.metadata.fqn): cat.#BackupTrait
			spec: {
				orphan: size: "10Gi"
				configMaps: probe: data: KEY: "value"
				backup: schedule: "daily"
			}
		}
	}
}

values: {}
