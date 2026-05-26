// Default Kubernetes Platform fixture — registers the opm catalog Module so
// its Resources, Traits, and ComponentTransformers project into the
// computed views (#knownResources, #knownTraits, #composedTransformers,
// #matchers). The kernel reads these views during Match / Plan / Compile.
package opm_platform

import (
	p "opmodel.dev/core/v1alpha2@v1"
	opm_package "opmodel.dev/modules/opm"
)

p.#Platform

metadata: {
	name:        "k8s-default"
	description: "Default Kubernetes Platform — registers the opm catalog Module"
}

type: "kubernetes"

#registry: {
	opm: {
		#module: opm_package
		enabled: true
	}
}
