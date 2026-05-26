package core

// #ReleaseIdentity carries the deployment-scoped facts that compute per-component
// names and DNS variants. Set by #ModuleRelease and propagated into every
// #Component via the parent #Module's pattern constraint on #components.
//
// clusterDomain lives here (not buried inside a runtime context) so a single
// overridable value covers every #Component's FQDN derivation.
//
// Introduced by enhancement 0001 (D1, D3, D4).
#ReleaseIdentity: {
	name!:         #NameType
	namespace!:    #NameType
	uuid!:         #UUIDType
	clusterDomain: string | *"cluster.local"
}

// #ComponentNames is the shape of the per-component computed-names projection.
// The single source of truth lives on each #Component.#names; #Module.#ctx.components
// projects every component's #names into a map keyed by component id.
//
// Introduced by enhancement 0001 (D1, D2).
#ComponentNames: {
	resourceName!: #NameType
	dns: {
		short!: string // "<resourceName>"
		local!: string // "<resourceName>.<namespace>"
		fqdn!:  string // "<resourceName>.<namespace>.svc.<clusterDomain>"
	}
}
