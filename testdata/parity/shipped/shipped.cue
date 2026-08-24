// The shipped-catalog group: the web_app instance rendered by the oracle
// against the published catalogs/opm build the platform fixture subscribes
// to. The kernel side of the same comparison lives in
// opm/kernel/parity_harness_test.go.
//
// `cue vet -c ./shipped` from testdata/parity (with the workspace
// CUE_REGISTRY) checks the rendered output is fully concrete.
package shipped

import (
	catalog "opmodel.dev/catalogs/opm"
	inst "testing.opmodel.dev/library-parity/instance"
	plat "testing.opmodel.dev/library-parity/opm_platform"
	"testing.opmodel.dev/library-parity/oracle"
)

oracle.#Render & {
	#instance:     inst
	#transformers: catalog.#transformers
	#runtime:      "parity"
}

// The platform's authored subscription and the build cue.mod resolved must
// name the same catalog; a disagreement fails this package (0019 OQ3, made
// executable). The kernel materializes from the platform's scalar; the
// oracle imports from cue.mod, so this is what keeps the two on one build.
_versionsAgree: catalog.metadata.version & plat.#registry["opmodel.dev/catalogs/opm@v2"].version
