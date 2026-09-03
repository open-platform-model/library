// Second render fixture catalog, 0.2.0: 0.1.0's mirror transformer plus a
// SECOND provider for the first catalog's provider-fulfilled gateway
// contract. A platform carrying cat and this build supplies that key from
// two catalogs, which the in-build single-provider guard (0010 D32/D37)
// refuses naming the key and both registry keys.
package cat2

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#Catalog

metadata: {
	modulePath:  "testing.opmodel.dev/library-render/cat2@v0"
	version:     "0.2.0"
	description: "second single-build render fixture catalog"
}

_version: "0.2.0"
_tx:      "testing.opmodel.dev/library-render/cat2/transformers"

#transformers: {
	"\(_tx)/mirror-transformer@\(_version)": {
		metadata: {
			name:        "mirror-transformer"
			fqn:         "\(_tx)/mirror-transformer@\(_version)"
			description: "A second catalog's transformer for the container contract"
		}
		requiredResources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
		#transform: {
			#component: _
			output: {
				apiVersion: "v1"
				kind:       "Mirror"
				metadata: name: #component.#names.resourceName
				spec: image:    #component.spec.container.image
			}
		}
	}

	"\(_tx)/gateway-transformer@\(_version)": {
		metadata: {
			name:        "gateway-transformer"
			fqn:         "\(_tx)/gateway-transformer@\(_version)"
			description: "A second provider for the first catalog's provider-fulfilled gateway contract"
		}
		requiredResources: (cat.#GatewayResource.metadata.fqn): cat.#GatewayResource
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
