// Two-catalog render fixture platform (the core its cue.mod pins, 0019:D5 shape): cat
// 0.1.0 beside cat2 0.2.0, two registry entries each supplying a transformer
// requiring the provider-fulfilled gateway contract: core's
// #contracts.providedBy counts both, the inventory reports the key
// over-subscribed and the render's single-provider guard, reading that count,
// refuses it.
// Consumed on-disk by Kernel.AcquirePlatformFromDir; never published; not
// discovered by the repo's CUE tasks (its catalog deps are served in-process).
package platform

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
	cat2 "testing.opmodel.dev/library-render/cat2@v0"
)

c.#Platform

metadata: {
	name:        "render-fixture-two"
	description: "Two-catalog single-build render fixture platform"
}

type: "kubernetes"

#registry: {
	"testing.opmodel.dev/library-render/cat@v0": {
		enable:   true
		#catalog: cat
	}
	"testing.opmodel.dev/library-render/cat2@v0": {
		enable:   true
		#catalog: cat2
	}
}
