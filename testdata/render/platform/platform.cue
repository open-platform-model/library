// Render fixture platform in the D5 shape (core 2.0.0-alpha.7): the registry
// entry carries the fixture catalog by import, and which catalog build
// executes is stated once, in this module's cue.mod/module.cue. Consumed
// on-disk by Kernel.AcquirePlatformFromDir; never published. Not discovered
// by the repo's CUE tasks (its catalog dep is served in-process by the tests).
package platform

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#Platform

metadata: {
	name:        "render-fixture"
	description: "Single-build render fixture platform"
}

type: "kubernetes"

#registry: "testing.opmodel.dev/library-render/cat@v0": {
	enable:   true
	#catalog: cat
}
