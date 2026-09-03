// The shipped-catalog group: the web_app instance rendered by the oracle
// against the published catalogs/opm build this module's cue.mod pins. The
// kernel side of the same comparison lives in
// opm/kernel/parity_harness_test.go and renders the same instance through
// Kernel.Render against ../opm_platform, a separate module pinning the same
// catalog build.
//
// `cue vet -c ./shipped` from testdata/parity (with the workspace
// CUE_REGISTRY) checks the rendered output is fully concrete.
package shipped

import (
	catalog "opmodel.dev/catalogs/opm"
	inst "testing.opmodel.dev/library-parity/instance"
	"testing.opmodel.dev/library-parity/oracle"
)

oracle.#Render & {
	#instance:     inst
	#transformers: catalog.#transformers
	#runtime:      "parity"
}

// The catalog build this oracle resolved. The harness requires the build
// Render resolves for the platform's catalog import to be this one, so the
// two renderers are kept on one set of catalog bytes (0019 OQ3, executable).
catalogVersion: catalog.metadata.version
