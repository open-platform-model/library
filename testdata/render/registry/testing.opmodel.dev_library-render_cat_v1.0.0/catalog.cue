// Render fixture catalog, the NEXT MAJOR of cat (1.0.0): one transformer
// requiring cat 0.1.0's provider-fulfilled gateway contract, and nothing
// else. Core stamps this catalog's transformers with the same major-free
// transformer module path as cat 0.1.0's, so a count keyed by that path sees
// one provider where a platform carrying both majors has two registry
// entries supplying the key. The platform's contract inventory
// (#contracts.providedBy) counts per registry entry (path plus major), so
// both majors enabled together over-subscribe the gateway contract, and the
// render refuses exactly when the inventory reads not routable. It requires
// no catalog-fulfilled contract, so comparability never confounds the count.
// Like cat2 it defines no contract of its own: its contract maps stay empty.
package cat

import (
	c "opmodel.dev/core@v2"
	cat0 "testing.opmodel.dev/library-render/cat@v0"
)

c.#Catalog

metadata: {
	modulePath:  "testing.opmodel.dev/library-render/cat@v1"
	version:     "1.0.0"
	description: "single-build render fixture catalog, next major"
}

_version: "1.0.0"
_tx:      "testing.opmodel.dev/library-render/cat/transformers"

#transformers: {
	"\(_tx)/gateway-transformer@\(_version)": {
		metadata: {
			name:        "gateway-transformer"
			fqn:         "\(_tx)/gateway-transformer@\(_version)"
			description: "The next major's provider for the gateway contract"
		}
		requiredResources: (cat0.#GatewayResource.metadata.fqn): cat0.#GatewayResource
		#transform: {
			#component: _
			output: {
				apiVersion: "v1"
				kind:       "Gateway"
				metadata: name: #component.#names.resourceName
				spec: host:     #component.spec.gateway.host
			}
		}
	}
}
