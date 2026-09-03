// The parity platform in the #CatalogEntry form (core 2.0.0-alpha.7, 0019
// D5): the registry entry carries the published catalogs/opm build by
// import, and which build executes is stated once, in this module's
// cue.mod/module.cue. It is the twin of ../opm_platform (the subscription
// form on the alpha.6 pin) and names the SAME catalog build, so the
// old-versus-new proof (opm/kernel/parity_cutover_test.go, openspec change
// library-render-cutover PR 1) compares Kernel.Compile and Kernel.Render on
// one set of catalog bytes. It becomes opm_platform when the old path is
// deleted (PR 2).
//
// Its own cue.mod makes it a nested module: the parity module pins alpha.6
// for the old path, this one pins alpha.7 for Render. Consumed on-disk by
// Kernel.AcquirePlatformFromDir; never published; not discovered by the
// repo's CUE tasks (CUE_MODULE_GLOBS names testdata/parity, whose ./... stops
// at this cue.mod).
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

// Path-keyed: the key is the catalog's module path, bound by core into the
// embedded catalog's metadata.modulePath; the build that executes is the one
// cue.mod resolves for that path (0019 D5, 0010 D14).
#registry: "opmodel.dev/catalogs/opm@v4": {
	enable:   true
	#catalog: opm
}
