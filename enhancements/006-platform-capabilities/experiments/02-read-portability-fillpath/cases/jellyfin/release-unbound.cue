package jellyfin

import (
	v1 "opmodel.dev/exp006-02/schemas:v1alpha2"
)

// Unbound release — #platform deliberately NOT set. Simulates the artifact
// shape an end-user authors and ships before the runtime binds it.
//
// CLAIMS PROVED HERE:
//   - Portability: `cue vet -c` MUST fail with an actionable diagnostic
//     naming the missing route@v1 spec (or the unbound #platform). The
//     failure is the correct release-time signal that the artifact has not
//     yet been bound to a platform.
ReleaseUnbound: v1.#ModuleRelease & {
	metadata: {
		name:      "jellyfin-staging"
		namespace: "media"
	}
	#module: JellyfinModule
	// #platform deliberately omitted.
	values: {}
}
