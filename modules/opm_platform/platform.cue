// Default Kubernetes Platform fixture. Subscribes to the library's fixture
// catalog via a #Subscription-shaped #registry. The kernel's Materialize step
// resolves the subscription against the registry, pulls the catalog build,
// and exposes the composed transformers / matcher index as native fields on
// the MaterializedPlatform (Transformers / Matchers) — it does NOT fill them
// onto this closed spec (ADR-003). This CUE value is the spec only.
//
// Unpublished in-repo fixture: consumed on-disk by the kernel flow tests
// (D-F). It is not part of any publish path.
package opm_platform

import (
	c "opmodel.dev/core@v2"
)

c.#Platform

metadata: {
	name:        "k8s-default"
	description: "Default Kubernetes Platform — subscribes to the library's fixture catalog"
}

type: "kubernetes"

// Path-keyed subscription: the map key is the catalog's CUE module path,
// major suffix included (v2 #ModulePathType). The scalar `version` names the
// exact build to materialize (enhancement 0010 D37 — no filter, no range);
// the fixture catalog publishes exactly this one version (library-core-retarget
// transitional invariant 1), so the kernel's interim highest-stable resolution
// selects the same build the scalar names.
#registry: {
	"testing.opmodel.dev/catalogs/opm@v1": {
		enable:  true
		version: "1.0.0"
	}
}
