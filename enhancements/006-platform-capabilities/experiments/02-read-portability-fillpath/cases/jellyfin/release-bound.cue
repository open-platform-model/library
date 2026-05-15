package jellyfin

import (
	v1 "opmodel.dev/exp006-02/schemas:v1alpha2"
)

// Bound release — both #platform AND the matched #consumes writeback are
// supplied inline (simulating what the kernel does via FillPath at apply
// time). The component body's lexical `#consumes.required[fqn].spec.domain`
// reference re-resolves through the now-concrete value.
//
// CLAIMS PROVED HERE:
//   - D7 (read surface): components.app.env.AppHost.value concretizes to
//     "jellyfin.apps.example.com" via direct unification of the matched
//     spec into #module.#consumes.
//   - D13 (#-prefix exclusion): `cue export` on this file MUST NOT emit a
//     `#platform` field — definition fields are excluded from export.
//
// NOTE: this release fixture does the kernel's writeback inline. The Go
// harness under cmd/fillpath/ proves the same value flow happens when the
// writeback is performed externally via cue.Value.FillPath.
ReleaseBound: v1.#ModuleRelease & {
	metadata: {
		name:      "jellyfin-prod"
		namespace: "media"
	}
	// Simulates the kernel writeback: provider's spec is unified into
	// #module.#consumes.required[fqn] at this top-level position.
	#module: JellyfinModule & {
		#consumes: required: (v1.RouteFQN): v1.#Route & {
			spec: domain: "apps.example.com"
		}
	}
	#platform: ProdPlatform
	values: {}
}
