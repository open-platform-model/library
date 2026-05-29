// fixture.cue is the baseline synth-test fixture: a minimal #Module
// declaration that synth tests in opm/helper/synth/ and opm/kernel/
// load via cue/load.Instances + an in-memory overlay.
//
// Tests overlay-replace the file contents per case; this committed
// baseline exists so the package compiles standalone (useful for
// `cue eval` debugging and for the load.Instances("." …) entry point
// the helpers use). Keep the metadata fixed — derived UUIDs across
// existing synth tests assume these exact values.
package synthtest

import core "opmodel.dev/core@v0"

module: {
	core.#Module
	metadata: {
		name:       "demo"
		modulePath: "example.com/demo"
		version:    "0.1.0"
	}
	#components: {}
	#config:     {}
	debugValues: {}
}
