package case04

import (
	v1 "opmodel.dev/exp004-02/schemas:v1alpha2"
)

// Hypothesis (D35): inside an inline-literal `v1.#Module & {...}` fixture,
// references to `#ctx` and `#names` are resolved by LEXICAL scope of the
// case file — not by the post-unification structure of the value. The case
// file is in `package case04`, which does NOT export `#ctx` / `#names`, so
// the inline literal must declare `#ctx: _` and `#names: _` at the same
// nesting level as the reference for the identifier lookup to succeed.
//
// Real modules live in the same package as `#Module` (where `#ctx:
// #ModuleContext` and `#Component.#names: #ComponentNames` are declared at
// package level) and therefore see those identifiers without any inline
// declaration.

_module: v1.#Module & {
	metadata: {
		name:    "myapp"
		version: "1.0.0"
		fqn:     "example.com/modules/myapp:1.0.0"
		uuid:    "00000000-0000-0000-0000-0000000000a1"
	}
	// Bring `#ctx` into the inline literal's lexical scope so nested
	// components can reference it without writing `_module.#ctx....`.
	#ctx: _
	#components: web: {
		metadata: name: "web"
		// Bring `#names` into the component literal's lexical scope.
		#names: _

		// Component body reads its own injected DNS name through the
		// package-lexical identifier — the same form a real module file
		// would use.
		spec: url: "http://\(#names.dns.fqdn):8080"
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

// After the release flow injects #names, the component-level reference
// concretizes to the expected URL.
_assertUrl: "http://myrelease-web.myns.svc.cluster.local:8080" & release.components.web.spec.url
