// Render fixture module app_maj0: a maj@v0 container component with the expose trait.
package app_maj0

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/maj@v0"
)

c.#Module

metadata: {
	name:        "app_maj0"
	modulePath:  "testing.opmodel.dev/library-render/app_maj0@v0"
	version:     "0.1.0"
	description: "colliding-majors render fixture module"
}

#config: {}

#components: web: {
	#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
	#traits: (cat.#ExposeTrait.metadata.fqn):          cat.#ExposeTrait
	spec: {container: image: "nginx:1", expose: port: 80}
}
