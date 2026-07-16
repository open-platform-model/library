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
	c "opmodel.dev/core@v1"
)

c.#Platform

metadata: {
	name:        "k8s-default"
	description: "Default Kubernetes Platform — subscribes to the opm core catalog"
}

type: "kubernetes"

// Path-keyed subscription: the map key is the catalog's CUE module path.
//
// The range pins the exact catalog version the web_app fixture pins in its
// cue.mod/module.cue. Both sides must name the same version: a resource FQN
// embeds the catalog's own version, so a component only pairs with a
// transformer drawn from the same catalog build. An unfiltered subscription
// resolves the highest *stable* version (v0.6.0, a core@v0 catalog) and would
// never reach this core@v1 pre-release. A `>=` range would not do either — it
// also admits the v1.0.0-dev.* tags, and every survivor is pulled and composed.
#registry: {
	"opmodel.dev/catalogs/opm": {
		enable: true
		filter: range: "1.0.0-alpha.1"
	}
}
