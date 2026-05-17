package case02

import (
	v1 "opmodel.dev/exp004-01/schemas:v1alpha2"
)

// Hypothesis: setting metadata.resourceName on a #Component replaces the
// default resourceName and the override cascades through every dns.* variant (D13).

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
	metadata: {
		name:         "web"
		resourceName: "router"
	}
}

result: (v1.#ContextBuilder & {
	#release:    _release
	#module:     _module
	#components: _components
}).out

_assertResourceName: "router" & result.ctx.runtime.components.web.resourceName
_assertLocal:        "router" & result.ctx.runtime.components.web.dns.local
_assertNamespaced:   "router.myns" & result.ctx.runtime.components.web.dns.namespaced
_assertSvc:          "router.myns.svc" & result.ctx.runtime.components.web.dns.svc
_assertFqdn:         "router.myns.svc.cluster.local" & result.ctx.runtime.components.web.dns.fqdn
