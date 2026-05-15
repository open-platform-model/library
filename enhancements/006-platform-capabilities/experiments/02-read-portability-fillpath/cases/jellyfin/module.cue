package jellyfin

import (
	v1 "opmodel.dev/exp006-02/schemas:v1alpha2"
)

// Jellyfin module — declares it requires the route capability and reads the
// resolved domain inside a component body via string interpolation.
//
// This is the 006/01-problem.md motivating example. The "after" picture from
// 006/02-design.md §"Read surface — #consumes itself" puts the interpolation
// straight on `#consumes.required[fqn].spec.domain` rather than on a
// `#ctx.platform.appDomain` open-struct read.
JellyfinModule: v1.#Module & {
	metadata: name: "jellyfin"

	#consumes: {
		required: (v1.RouteFQN): v1.#Route
		optional: {}
	}

	#components: app: {
		// A trimmed env-var shape — enough to host the interpolation we are
		// validating. The real #Component shape is richer; that's not what
		// this experiment is probing.
		env: AppHost: {
			name:  "JELLYFIN_AppHost"
			value: "jellyfin.\(#consumes.required[v1.RouteFQN].spec.domain)"
		}
	}
}
