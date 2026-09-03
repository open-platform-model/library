// Two-catalog render fixture platform (core 2.0.0-alpha.7, D5 shape): cat
// 0.1.0 beside cat2 0.1.0, one transformer per catalog requiring the catalog-fulfilled container contract: renders, every candidate matched.
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
