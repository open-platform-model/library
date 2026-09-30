// Render fixture module app_bk0: a maj@v0 container component with the load-bearing backup trait.
package app_bk0

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/maj@v0"
)

c.#Module

metadata: {
	name:        "app_bk0"
	modulePath:  "testing.opmodel.dev/library-render/app_bk0@v0"
	version:     "0.1.0"
	description: "colliding-majors render fixture module"
}

#config: {}

#components: web: {
	#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
	#traits: (cat.#BackupTrait.metadata.fqn):          cat.#BackupTrait
	spec: {container: image: "nginx:1", backup: schedule: "@daily"}
}
