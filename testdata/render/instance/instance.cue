// On-disk render fixture instance: the module enters by import (from the
// in-process registry) and the values are concrete. Consumed by
// Kernel.AcquireInstanceFromDir and imported as a package by the render
// build; never published.
package instance

import (
	c "opmodel.dev/core@v2"
	webapp "testing.opmodel.dev/library-render/web_app@v0"
)

c.#ModuleInstance

metadata: {
	name:      "web-demo"
	namespace: "default"
}

#module: webapp

values: {
	image:    "nginx:1.27"
	replicas: 2
}
