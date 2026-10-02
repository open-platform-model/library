// Colliding-majors render fixture platform (the core its cue.mod pins, 0019:D5
// shape): maj 0.1.0 beside maj 1.4.0, two majors of one catalog listing the same
// container, expose and backup keys: core's #contracts reports the three keys
// as collisions with routable false, and the render refuses the platform
// before any object renders (maj 1.4.0's bridge would otherwise render a
// maj@v0 component twice).
// Consumed on-disk by Kernel.AcquirePlatformFromDir; never published; not
// discovered by the repo's CUE tasks (its catalog deps are served in-process).
package platform

import (
	c "opmodel.dev/core@v2"
	majv0 "testing.opmodel.dev/library-render/maj@v0"
	majv1 "testing.opmodel.dev/library-render/maj@v1"
)

c.#Platform

metadata: {
	name:        "render-fixture-collide"
	description: "Colliding-majors single-build render fixture platform"
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
}
