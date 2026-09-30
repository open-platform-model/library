// Disabled-definer render fixture platform (core 2.0.0-alpha.13, 0019:D5
// shape): cat 0.1.0 DISABLED beside cat2 0.2.0 and cat 1.0.0 enabled. cat
// lists the provider-fulfilled gateway contract but defines nothing while
// disabled; cat2 and cat 1.0.0 each still supply a transformer requiring it.
// Two registry entries are two providers whether or not an enabled catalog
// defines the key: the render refuses it, and the contract inventory reports
// it over-subscribed and the platform not routable, with no defining catalog.
// Consumed on-disk by Kernel.AcquirePlatformFromDir; never published; not
// discovered by the repo's CUE tasks (its catalog deps are served in-process).
package platform

import (
	c "opmodel.dev/core@v2"
	catv0 "testing.opmodel.dev/library-render/cat@v0"
	catv1 "testing.opmodel.dev/library-render/cat@v1"
	cat2 "testing.opmodel.dev/library-render/cat2@v0"
)

c.#Platform

metadata: {
	name:        "render-fixture-definer-disabled"
	description: "Three-entry single-build render fixture platform, defining catalog disabled"
}

type: "kubernetes"

#registry: {
	"testing.opmodel.dev/library-render/cat@v0": {
		enable:   false
		#catalog: catv0
	}
	"testing.opmodel.dev/library-render/cat2@v0": {
		enable:   true
		#catalog: cat2
	}
	"testing.opmodel.dev/library-render/cat@v1": {
		enable:   true
		#catalog: catv1
	}
}
