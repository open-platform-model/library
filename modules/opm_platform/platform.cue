// Default Kubernetes Platform fixture. Subscribes to the consolidated
// opmodel.dev/catalogs/opm v2 line via a #Subscription-shaped #registry. The
// kernel's Materialize step resolves the subscription against the registry,
// pulls the catalog build, and exposes the composed transformers / matcher
// index as native fields on the MaterializedPlatform (Transformers /
// Matchers) — it does NOT fill them onto this closed spec (ADR-003). This CUE
// value is the spec only.
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
	description: "Default Kubernetes Platform — subscribes to the consolidated catalogs/opm v2 line"
}

type: "kubernetes"

// Path-keyed subscription: the map key is the catalog's CUE module path,
// major suffix included (v2 #ModulePathType). The scalar `version` names the
// exact build to materialize (enhancement 0010 D37 — no filter, no range).
// Transitional invariant 1 (library-core-retarget): the pin MUST track the
// newest tag on the catalog's v2 line — the kernel's interim highest-stable
// resolution ignores the scalar and, on the major-scoped prerelease-only v2
// list, falls back to the highest alpha, which must be the same build the
// scalar names. library-acquire-and-subscription makes the pin load-bearing
// and removes this fragility.
#registry: {
	"opmodel.dev/catalogs/opm@v2": {
		enable:  true
		version: "2.0.0-alpha.3"
	}
}
