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
// exact build to materialize (enhancement 0010 D14/D37 — no filter, no
// range). The pin is LOAD-BEARING: the kernel pulls exactly this build, so
// bumping it is an ordinary fixture update — pick any published tag in the
// key's major (`task cue:catalog:drift` verifies it exists on GHCR).
#registry: {
	"opmodel.dev/catalogs/opm@v2": {
		enable:  true
		version: "2.0.0-alpha.3"
	}
}
