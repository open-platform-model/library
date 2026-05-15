package jellyfin

import (
	v1 "opmodel.dev/exp006-02/schemas:v1alpha2"
)

// Concrete #Platform supplying the route capability. Used by both the
// CUE-side bound release and the Go FillPath harness.
ProdPlatform: v1.#Platform & {
	metadata: name: "prod"
	#provides: (v1.RouteFQN): v1.#Route & {
		spec: domain: "apps.example.com"
	}
}
