// Render fixture module: a stateless web component plus a config-maps
// component, authored against the render fixture catalog. Served by the
// in-process registry so the kernel can acquire it with source and
// synthesize an overlay-mode instance from it (Kernel.SynthesizeInstance),
// and imported by the on-disk instance fixture (testdata/render/instance).
package web_app

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"

	id "testing.opmodel.dev/library-render/web_app/identity"
)

c.#Module

// modulePath and version come from the identity subpackage through an
// unqualified self-import (the shape opm module init writes): the overlay
// render path must resolve it through the render module's default-major
// marker for the replaced module.
metadata: {
	name:        "web_app"
	modulePath:  id.ModulePath
	version:     id.Version
	description: "Single-build render fixture module"
}

#config: {
	image:    string | *"nginx:1.27"
	replicas: int | *2
}

#components: {
	web: {
		#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
		#traits: (cat.#ExposeTrait.metadata.fqn):          cat.#ExposeTrait
		spec: {
			container: image: #config.image
			expose: port:     80
		}
	}
	config: {
		#resources: (cat.#ConfigMapsResource.metadata.fqn): cat.#ConfigMapsResource
		spec: configMaps: app: data: MODE: "prod"
	}
}
