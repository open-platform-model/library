// Second render fixture catalog, for the multi-catalog platforms
// (testdata/render/platform_two, platform_oversubscribed). It implements
// contracts the first catalog declares, by import, so both catalogs carry
// byte-identical copies. 0.1.0 ships one transformer requiring the
// catalog-fulfilled container contract: many suppliers across catalogs are
// admitted for such a key, so a platform carrying both catalogs renders and
// every candidate participates in matching.
package cat2

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#Catalog

metadata: {
	modulePath:  "testing.opmodel.dev/library-render/cat2@v0"
	version:     "0.1.0"
	description: "second single-build render fixture catalog"
}

_version: "0.1.0"
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
}
