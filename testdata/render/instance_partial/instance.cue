// On-disk render fixture instance whose values leave `replicas` unset (the
// module defaults it): the extra-values acquire path
// (Kernel.AcquireInstanceFromDir with WithValues) layers caller-supplied
// values onto this package in one build, and the layered instance is then
// imported as a package by the render build. Never published.
package instance

import (
	c "opmodel.dev/core@v2"
	webapp "testing.opmodel.dev/library-render/web_app@v0"
)

c.#ModuleInstance

metadata: {
	name:      "web-partial"
	namespace: "default"
}

#module: webapp

values: {
	image: "nginx:1.27"
}
