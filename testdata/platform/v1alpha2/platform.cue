// Minimal #Platform fixture for slice-08 tests. Mirrors catalog enhancement
// 014's #Platform shape: identity (apiVersion, kind, metadata, type) plus
// the four computed views over an empty #registry. Concrete-only — no
// imports — so tests can load it without CUE_REGISTRY configuration.
package platform

apiVersion: "opmodel.dev/v1alpha2"
kind:       "Platform"

metadata: {
	name:        "test-platform"
	description: "Fixture used by pkg/platform and pkg/helper/loader/file tests"
	labels: env:   "dev"
	annotations: owner: "library-tests"
}

type: "kubernetes"

#registry: {}

// The four computed views are declared concretely so binding-path tests can
// assert each path resolves to an existing value. Real platforms compute
// these from #registry; this fixture has no registered Modules so every
// view is empty.
#knownResources:       {}
#knownTraits:          {}
#composedTransformers: {}
#matchers: {
	resources: {}
	traits:    {}
}
