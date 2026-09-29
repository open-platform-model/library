// Provider-fulfilled render fixture platform (core 2.0.0-alpha.10, 0019:D5
// shape): cat 0.1.0 beside providers 0.1.0, which lists two provider-fulfilled
// contracts nothing on this platform provides (snapshot, ledger) and one
// whose provider requires a label (archive). Consumed on-disk by
// Kernel.AcquirePlatformFromDir; never published; not discovered by the
// repo's CUE tasks (its catalog deps are served in-process).
package platform

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
	providers "testing.opmodel.dev/library-render/providers@v0"
)

c.#Platform

metadata: {
	name:        "render-fixture-providers"
	description: "Provider-fulfilled single-build render fixture platform"
}

type: "kubernetes"

#registry: {
	"testing.opmodel.dev/library-render/cat@v0": {
		enable:   true
		#catalog: cat
	}
	"testing.opmodel.dev/library-render/providers@v0": {
		enable:   true
		#catalog: providers
	}
}
