package v1alpha2

// Concrete #Capability definitions used by the experiment fixtures.
//
// In a real catalog these would each live in their own package and be
// imported by both providers and consumers. Co-locating them here keeps the
// experiment to one CUE module.

// #Route — public app/route domain. A regex constraint on `domain` lets the
// schema-mismatch case (03) demonstrate that constraint violations surface
// as CUE bottoms localized at the field (006 D6 alternative branch).
#Route: #Capability & {
	metadata: {
		name:       "route"
		modulePath: "opmodel.dev/exp/caps/routing"
		version:    "v1"
	}
	spec: {
		domain: string & =~"^[a-z0-9.-]+\\.[a-z]+$"
	}
}

// #StorageClass — default storage class for the target cluster.
// Plain string field; used by the OQ6 inheritance case (06).
#StorageClass: #Capability & {
	metadata: {
		name:       "storage-class"
		modulePath: "opmodel.dev/exp/caps/storage"
		version:    "v1"
	}
	spec: {
		name: string
	}
}

// FQN constants for fixture readability.
RouteFQN:        "opmodel.dev/exp/caps/routing/route@v1"
StorageClassFQN: "opmodel.dev/exp/caps/storage/storage-class@v1"
