// Package identity is the single source of the fixture catalog's module path
// and version. Committed with the REAL values (enhancement 0010 D5 — no
// publish-time stamping) and import-free, so every leaf package can source
// ModulePath/Version without a circular import.
package identity

// ModulePath is the catalog's complete CUE module path, major suffix included
// — byte-identical to cue.mod's `module:` field (enhancement 0010 D1).
ModulePath: "testing.opmodel.dev/catalogs/opm@v1"

// Version is the single build this fixture catalog ever publishes.
// Exactly one version per subscription is transitional invariant 1 of the
// library-core-retarget change: the kernel still resolves highestStable until
// the subscription-collapse slice reads the scalar `version:`, and with one
// published version the two answers coincide.
Version: "1.0.0"

// RegistryPath is the major-free OCI repository path.
RegistryPath: "testing.opmodel.dev/catalogs/opm"

// kindPrefix mirrors core's #IdentityPackage.kindPrefix — exactly one prefix
// per kind, no grouping segment beneath any of them (enhancement 0010 D42).
kindPrefix: {
	resources:    RegistryPath + "/resources"
	traits:       RegistryPath + "/traits"
	blueprints:   RegistryPath + "/blueprints"
	transformers: RegistryPath + "/transformers"
}
