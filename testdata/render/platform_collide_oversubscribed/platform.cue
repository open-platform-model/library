// Colliding-majors render fixture platform (core 2.0.0-beta.1, 0019:D5
// shape): platform_collide plus bprov 0.1.0 and bprov 1.0.0, both requiring
// the colliding backup trait: the inventory reports the three collisions and
// the backup key over-subscribed together, and the render refuses with the
// collision cause first, then the over-subscription.
// Consumed on-disk by Kernel.AcquirePlatformFromDir; never published; not
// discovered by the repo's CUE tasks (its catalog deps are served in-process).
package platform

import (
	c "opmodel.dev/core@v2"
	majv0 "testing.opmodel.dev/library-render/maj@v0"
	majv1 "testing.opmodel.dev/library-render/maj@v1"
	bprov0 "testing.opmodel.dev/library-render/bprov@v0"
	bprov1 "testing.opmodel.dev/library-render/bprov@v1"
)

c.#Platform

metadata: {
	name:        "render-fixture-collide-oversubscribed"
	description: "Colliding-majors render fixture platform with two backup providers"
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
	"testing.opmodel.dev/library-render/bprov@v1": {
		enable:   true
		#catalog: bprov1
	}
}
