package case04

import (
	v1 "opmodel.dev/exp004-01/schemas:v1alpha2"
)

// Hypothesis: after #ModuleRelease unifies the builder's injections back into
// #module.#components (where #Component.#names: #ComponentNames applies the
// schema), each component's `#names` is value-equal to its corresponding
// entry in `#ctx.runtime.components` (D32 — lock-step). Two components are
// used so per-component identity, not a single global value, is exercised.
//
// The lock-step holds by construction: #ContextBuilder threads
// `_componentNames[compName]` into both `ctx.runtime.components` and
// `injections.<compName>.#names`. This case asserts the construction holds
// end-to-end through the #ModuleRelease pipeline.

_module: v1.#Module & {
	metadata: {
		name:    "myapp"
		version: "1.0.0"
		fqn:     "example.com/modules/myapp:1.0.0"
		uuid:    "00000000-0000-0000-0000-0000000000a1"
	}
	#components: {
		web: {metadata: name: "web"}
		api: {metadata: {
			name:         "api"
			resourceName: "api-router"
		}}
	}
}

release: v1.#ModuleRelease & {
	metadata: {
		name:      "myrelease"
		namespace: "myns"
		uuid:      "00000000-0000-0000-0000-000000000001"
	}
	#module: _module
	values: {}
}

// Unification — CUE bottoms out if the per-component injection ever drifts
// from the ctx.runtime.components entry.
_lockstepWeb: release.components.web.#names & release.ctx.runtime.components.web
_lockstepApi: release.components.api.#names & release.ctx.runtime.components.api

// Spot-check overridden + default components cascade identically through
// both surfaces.
_assertWebFqdnNames: "myrelease-web.myns.svc.cluster.local" & release.components.web.#names.dns.fqdn
_assertWebFqdnCtx:   "myrelease-web.myns.svc.cluster.local" & release.ctx.runtime.components.web.dns.fqdn
_assertApiFqdnNames: "api-router.myns.svc.cluster.local" & release.components.api.#names.dns.fqdn
_assertApiFqdnCtx:   "api-router.myns.svc.cluster.local" & release.ctx.runtime.components.api.dns.fqdn
