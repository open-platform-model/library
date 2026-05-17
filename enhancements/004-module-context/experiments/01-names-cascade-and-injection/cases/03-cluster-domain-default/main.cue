package case03

import (
	v1 "opmodel.dev/exp004-01/schemas:v1alpha2"
)

// Hypothesis: #ComponentNames._clusterDomain self-defaults to "cluster.local";
// 004 carries no override path (D36 — the override moved to enhancement 006 with
// #Environment / #Platform). dns.fqdn always ends ".svc.cluster.local".
//
// No override fixture is provided — that's the point: there is no surface
// here to override it from. If a future revision reintroduces an override
// path, this case file will need to be updated to exercise it.

_release: {
	name:      "myrelease"
	namespace: "media"
	uuid:      "00000000-0000-0000-0000-000000000001"
}

_module: {
	name:    "jellyfin"
	version: "1.0.0"
	fqn:     "example.com/modules/jellyfin:1.0.0"
	uuid:    "00000000-0000-0000-0000-0000000000a1"
}

_components: jellyfin: v1.#Component & {
	metadata: name: "jellyfin"
}

result: (v1.#ContextBuilder & {
	#release:    _release
	#module:     _module
	#components: _components
}).out

_assertFqdn: "myrelease-jellyfin.media.svc.cluster.local" & result.ctx.runtime.components.jellyfin.dns.fqdn
