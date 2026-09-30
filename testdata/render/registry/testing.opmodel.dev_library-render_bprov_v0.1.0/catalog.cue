// Render fixture catalog bprov, major v0 (0.1.0): one backup transformer
// requiring maj@v0's provider-fulfilled backup trait and no contract of its
// own. Both bprov majors require the one backup key, so enabling both
// over-subscribes it (two registry entries, two providers).
package bprov

import (
	c "opmodel.dev/core@v2"
	m "testing.opmodel.dev/library-render/maj@v0"
)

c.#Catalog

metadata: {
	modulePath:  "testing.opmodel.dev/library-render/bprov@v0"
	version:     "0.1.0"
	description: "backup provider render fixture catalog"
}

_version: "0.1.0"
_tx:      "testing.opmodel.dev/library-render/bprov/transformers"

#transformers: "\(_tx)/backup-transformer@\(_version)": {
	metadata: {name: "backup-transformer", fqn: "\(_tx)/backup-transformer@\(_version)", description: "backup"}
	requiredTraits: (m.#BackupTrait.metadata.fqn): m.#BackupTrait
	#transform: {#component: _, output: {apiVersion: "backup/v1", kind: "Schedule", metadata: name: #component.#names.resourceName}}
}
