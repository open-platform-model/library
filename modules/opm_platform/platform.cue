// Default Kubernetes Platform fixture in the #CatalogEntry form (core
// 2.0.0-alpha.7, enhancement 0019 D5): the registry entry carries the
// consolidated opmodel.dev/catalogs/opm v4 line by IMPORT, and which build
// executes is stated once, in this module's cue.mod/module.cue. Core binds
// the entry key to the embedded catalog's metadata.modulePath, derives the
// entry's `version` from the catalog's stamped version and folds the
// enabled entries into #composedTransformers; the kernel reads none of that
// from Go. Kernel.AcquirePlatformFromDir stamps this directory as the
// platform's Source and Kernel.Render imports the package into the render
// build.
//
// Unpublished in-repo fixture: consumed on-disk by the kernel flow tests.
// It is not part of any publish path. The catalog pin is LOAD-BEARING
// (0010 D14): bumping it is an ordinary fixture update, any published tag in
// the key's major (`task cue:catalog:drift` verifies it exists on GHCR).
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
