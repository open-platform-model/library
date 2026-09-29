// Provider-fulfilled render fixture catalog for the skip-unprovided tests
// (opm/kernel/render_skip_test.go, opm/internal/renderstage/skip_test.go).
// Served by the in-process registry (opm/internal/registrytest) from this
// directory; never published.
//
// Three provider-fulfilled contracts, each listed in the contract maps so a
// platform embedding this catalog names it as the defining catalog:
//
//   - the snapshot trait and the ledger resource ship no transformer, the
//     shape of a catalog that lists `backup` and leaves it to a provider.
//     On a platform carrying no provider they are unprovided.
//   - the archive trait is required by archive-transformer here, which also
//     requires a label, so a component attaching it without the label has a
//     provider that exists and does not match.
//
// Kept apart from cat so that cat stays a fulfilled catalog: listing an
// unprovided contract makes the platform's contract inventory unfulfilled.
package providers

import (
	c "opmodel.dev/core@v2"
	cat "testing.opmodel.dev/library-render/cat@v0"
)

c.#Catalog

metadata: {
	modulePath:  "testing.opmodel.dev/library-render/providers@v0"
	version:     "0.1.0"
	description: "provider-fulfilled single-build render fixture catalog"
}

_version: "0.1.0"
_res:     "testing.opmodel.dev/library-render/providers/resources"
_traits:  "testing.opmodel.dev/library-render/providers/traits"
_tx:      "testing.opmodel.dev/library-render/providers/transformers"

// ── Resources ───────────────────────────────────────────────────────

// Provider-fulfilled and required by no transformer: a demand for it is
// unprovided.
#LedgerResource: c.#Resource & {
	metadata: {
		name:           "ledger"
		modulePath:     "\(_res)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_res)/ledger@v1"
		description:    "A provider-fulfilled resource nothing here provides"
	}
	fulfilment: "provider"
	spec: ledger: size!: string
}

// ── Traits ──────────────────────────────────────────────────────────

// Load-bearing, provider-fulfilled and required by no transformer: an
// attachment is an unprovided trait demand.
#SnapshotTrait: c.#Trait & {
	metadata: {
		name:           "snapshot"
		modulePath:     "\(_traits)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_traits)/snapshot@v1"
		description:    "A provider-fulfilled trait nothing here provides"
	}
	fulfilment: "provider"
	optional:   bool | *false
	spec: snapshot: schedule?: string
	appliesTo: [cat.#ContainerResource]
}

// Load-bearing and provider-fulfilled, with one provider here
// (archive-transformer) that requires a label a component may not carry.
#ArchiveTrait: c.#Trait & {
	metadata: {
		name:           "archive"
		modulePath:     "\(_traits)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_traits)/archive@v1"
		description:    "A provider-fulfilled trait whose one provider requires a label"
	}
	fulfilment: "provider"
	optional:   bool | *false
	spec: archive: target?: string
	appliesTo: [cat.#ContainerResource]
}

// ── Contract maps ───────────────────────────────────────────────────

#resources: {
	(#LedgerResource.metadata.fqn): #LedgerResource
}

#traits: {
	(#SnapshotTrait.metadata.fqn): #SnapshotTrait
	(#ArchiveTrait.metadata.fqn):  #ArchiveTrait
}

// ── Transformers ────────────────────────────────────────────────────

#transformers: {
	"\(_tx)/archive-transformer@\(_version)": {
		metadata: {
			name:        "archive-transformer"
			fqn:         "\(_tx)/archive-transformer@\(_version)"
			description: "The one provider for the archive contract, gated on a label"
		}
		requiredLabels: "render.test/archive":                    "on"
		requiredResources: (cat.#ContainerResource.metadata.fqn): cat.#ContainerResource
		requiredTraits: (#ArchiveTrait.metadata.fqn):             #ArchiveTrait
		#transform: {
			#component: _
			output: {
				apiVersion: "v1"
				kind:       "Archive"
				metadata: name: #component.#names.resourceName
			}
		}
	}
}
