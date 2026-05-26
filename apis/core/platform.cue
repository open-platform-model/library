package core

// #SubscriptionFilter narrows the set of catalog builds a #Platform pulls
// from a subscribed registry path. All three fields are optional; when
// every field is absent the kernel selects the highest SemVer published
// for the path.
//
// Resolution order (D10):
//   1. `range` restricts the candidate set to SemVers satisfying the
//      constraint expression (e.g. ">=1.0.0 <2.0.0"). Parsed Go-side by
//      the kernel via Masterminds/semver (D11).
//   2. `allow` force-includes specific SemVers regardless of `range`.
//   3. `deny` force-excludes specific SemVers from the survivor set.
//
// Introduced by enhancement 0001 (D13).
#SubscriptionFilter: {
	range?: string // SemVer constraint expression
	allow?: [...#VersionType] // force-include specific versions
	deny?: [...#VersionType] // force-exclude specific versions
}

// #Subscription declares that a #Platform pulls primitives from a catalog
// published at a given CUE module path. The map key on #Platform.#registry
// carries the path; #Subscription carries the enable flag and the optional
// filter.
//
// One subscription per catalog path is enforced by CUE map semantics
// (D13). Multi-channel-per-path (e.g. RC + stable on the same platform)
// is not expressible at this stage; if needed later it lands as an
// additive extension that changes the key shape.
#Subscription: {
	enable:  bool | *true
	filter?: #SubscriptionFilter
}

// #Platform — path-keyed registry of catalog subscriptions plus
// kernel-filled materialization slots.
//
// Authors write #registry. The kernel's Materialize step (library-side)
// resolves every subscription's filter against the OCI registry, pulls
// the selected builds, indexes top-level #ComponentTransformer values
// into #composedTransformers, and computes a #matchers reverse index.
// The CUE-level #Platform value is therefore a spec; the kernel
// populates the materialization slots on a separate MaterializedPlatform
// twin (D14 — Materialize is explicit and caller-driven; the kernel
// holds no cache).
//
// Reshaped by enhancement 0001 (supersedes the prior Id-keyed
// #ModuleRegistration model). #knownResources / #knownTraits removed:
// primitives surface transitively via materialized transformers'
// required/optional maps.
#Platform: {
	kind: "Platform"

	metadata: {
		name!:        #NameType
		description?: string
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}

	// Informational. Future enhancement may enforce type-vs-transformer
	// compatibility; today it is an authored discriminator the matcher
	// does not consult (014 OQ2).
	type!: string

	// Path-keyed: map key is the catalog's CUE module path
	// (e.g. "opmodel.dev/catalogs/opm"). Exactly one subscription per
	// path — CUE map semantics enforce uniqueness (D13).
	#registry: [Path=#ModulePathType]: #Subscription

	// Kernel-filled after Materialize. Both optional because the
	// CUE-level #Platform value is a spec; the kernel populates these
	// on the materialized twin. Materialize is explicit and caller-driven
	// (D14 — no kernel cache).
	#composedTransformers?: #TransformerMap
	#matchers?: {
		resources: [#FQNType]: [...#ComponentTransformer]
		traits: [#FQNType]: [...#ComponentTransformer]
	}
}
