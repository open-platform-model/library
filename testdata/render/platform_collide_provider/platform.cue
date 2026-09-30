// Colliding-majors render fixture platform (core 2.0.0-alpha.13, 0019:D5
// shape): platform_collide plus bprov 0.1.0, whose transformer requires the
// colliding provider-fulfilled backup trait: providedBy names bprov@v0 for the
// key, while definedBy, requiredBy and unfulfilled leave it out (the stated
// blind spot), so fulfilled and discriminated read true and routable false.
// Consumed on-disk by Kernel.AcquirePlatformFromDir; never published; not
// discovered by the repo's CUE tasks (its catalog deps are served in-process).
package platform

import (
	c "opmodel.dev/core@v2"
	majv0 "testing.opmodel.dev/library-render/maj@v0"
	majv1 "testing.opmodel.dev/library-render/maj@v1"
	bprov0 "testing.opmodel.dev/library-render/bprov@v0"
)

c.#Platform

metadata: {
	name:        "render-fixture-collide-provider"
	description: "Colliding-majors render fixture platform with one backup provider"
}

type: "kubernetes"

#registry: {
	"testing.opmodel.dev/library-render/maj@v0": {
		enable:   true
		#catalog: majv0
	}
	"testing.opmodel.dev/library-render/maj@v1": {
		enable:   true
		#catalog: majv1
	}
	"testing.opmodel.dev/library-render/bprov@v0": {
		enable:   true
		#catalog: bprov0
	}
}
