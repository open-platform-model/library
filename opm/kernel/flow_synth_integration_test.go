package kernel_test

import (
	"testing"
)

// TestFlow_WebApp_SynthPath_OnOpmPlatform is quarantined.
//
// See openspec/changes/remove-api-binding-dispatch design.md §D10. Same
// rationale as TestFlow_WebApp_OnOpmPlatform in flow_integration_test.go:
// the consumer fixtures (library/modules/opm_platform and
// library/testdata/modules/web_app) predate enhancement 0001's schema
// reshape and are rewritten by 0001's library slice, not by Part B.
func TestFlow_WebApp_SynthPath_OnOpmPlatform(t *testing.T) {
	t.Skip("quarantined — see openspec/changes/remove-api-binding-dispatch design.md D10; re-enabled by enhancement 0001 library slice")
}
