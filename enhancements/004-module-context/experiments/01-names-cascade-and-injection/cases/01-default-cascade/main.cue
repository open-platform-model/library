package case01

import (
	v1 "opmodel.dev/exp004-01/schemas:v1alpha2"
)

// Hypothesis: with no metadata.resourceName override, `resourceName` defaults to
// "{release}-{component}" and all four dns.* variants cascade from it (D10).

_release: {
	name:      "myrelease"
	namespace: "myns"
	uuid:      "00000000-0000-0000-0000-000000000001"
}

_module: {
	name:    "myapp"
	version: "1.0.0"
	fqn:     "example.com/modules/myapp:1.0.0"
	uuid:    "00000000-0000-0000-0000-0000000000a1"
}

_components: web: v1.#Component & {
	metadata: name: "web"
}

result: (v1.#ContextBuilder & {
	#release:    _release
	#module:     _module
	#components: _components
}).out

// Equality unification — CUE bottoms out if the computed value differs.
_assertResourceName: "myrelease-web" & result.ctx.runtime.components.web.resourceName
_assertLocal:        "myrelease-web" & result.ctx.runtime.components.web.dns.local
_assertNamespaced:   "myrelease-web.myns" & result.ctx.runtime.components.web.dns.namespaced
_assertSvc:          "myrelease-web.myns.svc" & result.ctx.runtime.components.web.dns.svc
_assertFqdn:         "myrelease-web.myns.svc.cluster.local" & result.ctx.runtime.components.web.dns.fqdn
