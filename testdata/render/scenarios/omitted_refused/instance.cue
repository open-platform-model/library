package omitted_refused

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
	providers "testing.opmodel.dev/library-render/providers@v0"
)

c.#ModuleInstance

metadata: {
	name:      "omitted-refused-demo"
	namespace: "default"
}

#module: {
	metadata: {
		name:       "omitted_refused"
		modulePath: "testing.opmodel.dev/library-render/scenarios/omitted_refused@v0"
		version:    "0.1.0"
	}
	#config: {}
	#components: {
		// The unprovided ledger resource omits this component. It also
		// attaches the advisory sidecar trait nothing handles (dropped
		// from the unhandled-trait table with the component), the
		// catalog-fulfilled backup trait and the label-gated archive
		// trait (both still refusing on the omitted component).
		ledger: {
			#resources: {
				(cat.#ContainerResource.metadata.fqn):    cat.#ContainerResource
				(providers.#LedgerResource.metadata.fqn): providers.#LedgerResource
			}
			#traits: {
				(cat.#SidecarTrait.metadata.fqn):       cat.#SidecarTrait
				(cat.#BackupTrait.metadata.fqn):        cat.#BackupTrait
				(providers.#ArchiveTrait.metadata.fqn): providers.#ArchiveTrait
			}
			spec: {
				container: image: "postgres:17"
				ledger: size:     "10Gi"
				sidecar: image:   "envoy:1.30"
				archive: target:  "s3"
			}
		}
		// A rendered sibling with the same advisory trait: it stays on
		// the unhandled-trait table.
		web: {
			#resources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
			#traits: (cat.#SidecarTrait.metadata.fqn):         cat.#SidecarTrait
			spec: {
				container: image: "nginx:1.27"
				sidecar: image:   "envoy:1.30"
			}
		}
	}
}

values: {}
