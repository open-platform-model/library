// Default Kubernetes Platform fixture. Subscribes to the OPM core catalog via
// a #Subscription-shaped #registry (enhancement 0001). The kernel's
// Materialize step resolves the subscription against the registry, pulls the
// catalog build, and exposes the composed transformers / matcher index as
// native fields on the MaterializedPlatform (Transformers / Matchers) — it does
// NOT fill them onto this closed spec (ADR-003). This CUE value is the spec only.
//
// Unpublished in-repo fixture: consumed on-disk by the kernel flow tests
// (D-F). It is not part of any publish path.
package opm_platform

import (
	c "opmodel.dev/core@v0"
)

c.#Platform

metadata: {
	name:        "k8s-default"
	description: "Default Kubernetes Platform — subscribes to the opm core catalog"
}

type: "kubernetes"

// Path-keyed subscription: the map key is the catalog's CUE module path.
#registry: {
	"opmodel.dev/catalogs/opm": {
		enable: true
	}
}
