// Two-major render fixture platform (core 2.0.0-alpha.12, 0019:D5 shape): cat
// 0.1.0 beside cat 1.0.0, two majors of one catalog each supplying a
// transformer requiring the provider-fulfilled gateway contract. Two registry
// entries are two providers: the render refuses the key, and the contract
// inventory reports it over-subscribed and the platform not routable.
// Consumed on-disk by Kernel.AcquirePlatformFromDir; never published; not
// discovered by the repo's CUE tasks (its catalog deps are served in-process).
package platform

import (
	c "opmodel.dev/core@v2"
	catv0 "testing.opmodel.dev/library-render/cat@v0"
	catv1 "testing.opmodel.dev/library-render/cat@v1"
)

c.#Platform

metadata: {
	name:        "render-fixture-two-majors"
	description: "Two-major single-build render fixture platform"
}

type: "kubernetes"

#registry: {
	"testing.opmodel.dev/library-render/cat@v0": {
		enable:   true
		#catalog: catv0
	}
	"testing.opmodel.dev/library-render/cat@v1": {
		enable:   true
		#catalog: catv1
	}
}
