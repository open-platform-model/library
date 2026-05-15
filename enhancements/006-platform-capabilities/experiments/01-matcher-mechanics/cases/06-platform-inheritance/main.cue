package case06

import (
	v1 "opmodel.dev/exp006-01/schemas:v1alpha2"
)

// Hypothesis: per-platform variation via CUE unification of #Platform values
// (006 D4 / OQ6). #KindDev is built as `#KindBase & {#provides: {…}}`. A
// module consuming both `route@v1` and `storage-class@v1` resolves cleanly
// against #KindDev — base provisions inherited, dev provisions added.

// Base uses a default for metadata.name so derived platforms can override it
// with a concrete value via plain unification. Without `| *…` the literal
// would conflict — CUE definitions unify rather than override.
#KindBase: v1.#Platform & {
	metadata: name: *"kind-base" | _
	#provides: (v1.RouteFQN): v1.#Route & {
		spec: domain: "apps.example.com"
	}
}

#KindDev: #KindBase & {
	metadata: name: "kind-dev"
	#provides: (v1.StorageClassFQN): v1.#StorageClass & {
		spec: name: "fast-ssd"
	}
}

result: (v1.#ContextBuilder & {
	#platform: #KindDev
	#consumes: {
		required: {
			(v1.RouteFQN):        v1.#Route
			(v1.StorageClassFQN): v1.#StorageClass
		}
		optional: {}
	}
}).out

// Inherited from base.
_assertRoute: "apps.example.com" & result.consumes.required[v1.RouteFQN].spec.domain

// Added by dev.
_assertStorage: "fast-ssd" & result.consumes.required[v1.StorageClassFQN].spec.name

// Sanity: kind-dev's metadata.name is the dev override, not base's.
_assertPlatformName: "kind-dev" & #KindDev.metadata.name
