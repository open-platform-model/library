@if(test)

// Positive case: pin a #Module's name/modulePath/version and assert that
// metadata.fqn matches the format string at module.cue:17 and metadata.uuid
// matches the deterministic UUIDv5 hash computed from
// (OPMNamespace, fqn) at module.cue:20. The harness asserts both via
// assertField/assertValue (Go-side decode + equality).
//
// IMPORTANT: the pinned UUID is the canonical drift sentinel for the
// OPMNamespace constant in apis/core/v1alpha2/types.cue:50. If
// OPMNamespace changes, every #Module on every platform gets a new uuid
// and every label/annotation that stamps the uuid changes too. To
// regenerate the value, run:
//   cue eval -t test ./testdata/module_uuid_fixture.cue --expression input.metadata.uuid
// and paste the result into the schemaCases table row.
package fixtures

import (
	core "opmodel.dev/core/v1alpha2@v1"
)

input: {
	core.#Module
	metadata: {
		name:       "demo"
		modulePath: "example.com/demo"
		version:    "0.1.0"
	}
}
