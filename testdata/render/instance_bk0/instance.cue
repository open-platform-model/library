// On-disk render fixture instance of app_bk0, a module built on maj@v0,
// rendered against the colliding platforms. Consumed by
// Kernel.AcquireInstanceFromDir; never published.
package instance

import (
	c "opmodel.dev/core@v2"
	m "testing.opmodel.dev/library-render/app_bk0@v0"
)

c.#ModuleInstance

metadata: {
	name:      "probe-bk0"
	namespace: "default"
}

#module: m

values: {}
