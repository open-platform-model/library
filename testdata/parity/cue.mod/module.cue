// Parity fixture module for the render-parity harness
// (opm/kernel/parity_harness_test.go; enhancement 0019 D1/D4/D14).
//
// One module so the oracle can import the instance, the platform and the
// catalog without a registry round-trip, while the kernel loads the same
// packages from the same tree. Pinned to the exact published builds the
// library flow tests use, so both renderers evaluate the same bytes.
//
// Never published: testing.opmodel.dev is the fixture domain (workspace
// registry policy), and intra-module imports need only a valid identity.
module: "testing.opmodel.dev/library-parity@v0"
language: {
	version: "v0.17.0"
}
deps: {
	"opmodel.dev/core@v2": {
		v: "v2.0.0-alpha.4"
	}
	"opmodel.dev/catalogs/opm@v2": {
		v: "v2.0.0-alpha.3"
	}
}
