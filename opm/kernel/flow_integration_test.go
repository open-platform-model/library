package kernel_test

import (
	"testing"
)

// TestFlow_WebApp_OnOpmPlatform is quarantined.
//
// See openspec/changes/remove-api-binding-dispatch design.md §D10:
// the re-synced apis/core/ schema is post-enhancement-0001 in
// #Platform.#registry (path-keyed #Subscription), #FQNType (SemVer), and
// #Module (#ctx instead of #defines). The consumer fixtures this test relies
// on — library/modules/opm_platform/platform.cue and
// library/testdata/modules/web_app/{module,components}.cue — predate that
// reshape on both import path (opmodel.dev/core/v1alpha2@v1) and
// registry / FQN shape.
//
// Re-enabling the flow-integration end-to-end is enhancement 0001's library
// slice. That slice rewrites the fixtures against #Subscription / #ctx /
// SemVer FQNs and republishes the catalog module at opmodel.dev/catalogs/opm
// (D23). Part B (this change) does mechanical cleanup only; folding design
// implementation in defeats reviewability.
func TestFlow_WebApp_OnOpmPlatform(t *testing.T) {
	t.Skip("quarantined — see openspec/changes/remove-api-binding-dispatch design.md D10; re-enabled by enhancement 0001 library slice")
}
