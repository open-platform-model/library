// COPY of modules/opm_platform/platform.cue for the render-parity harness.
// Kept byte-identical to its source apart from this header; both pin the
// same published core and catalog builds. Edit the source, then refresh this
// copy.
//
// It is a NESTED module (its own cue.mod) rather than a package of the
// parity module: Kernel.Render brings the instance and the platform into the
// render build as two directory replacements, and one module path cannot be
// replaced with two directories, so the platform and the instance
// (../instance, inside the parity module) must be distinct modules. The
// oracle build (../shipped) imports the catalog directly and never this
// platform; the harness asserts that the build Render resolves for the
// catalog is the build the oracle imported. Consumed on-disk by
// Kernel.AcquirePlatformFromDir; never published.
package opm_platform

import (
	c "opmodel.dev/core@v2"
	opm "opmodel.dev/catalogs/opm@v4"
)

c.#Platform

metadata: {
	name:        "k8s-default"
	description: "Default Kubernetes Platform — imports the consolidated catalogs/opm v4 line"
}

type: "kubernetes"

// Path-keyed: the key is the catalog's module path, major suffix included
// (v2 #ModulePathType), bound by core into the embedded catalog's
// metadata.modulePath; the build that executes is the one cue.mod resolves
// for that path.
#registry: "opmodel.dev/catalogs/opm@v4": {
	enable:   true
	#catalog: opm
}
