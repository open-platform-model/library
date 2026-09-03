// Render fixture catalog for the single-build render tests
// (opm/kernel/render_test.go). Served to the kernel by the in-process
// registry (opm/internal/registrytest) from this directory; never published.
//
// One member per matching outcome the tests exercise: a healthy container /
// expose / config-maps trio, an orphan resource no transformer handles, a
// load-bearing backup trait, an advisory sidecar trait, a trait that states
// no optional posture, a narrowed required copy that plain unification
// disqualifies, and two sabotaged transformers (an output that conflicts, an
// output that never becomes concrete). Every fqn lives under testing.opmodel.dev/library-render so
// nothing here can collide with a published catalog.
package cat

import (
	c "opmodel.dev/core@v2"
)

c.#Catalog

metadata: {
	modulePath:  "testing.opmodel.dev/library-render/cat@v0"
	version:     "0.1.0"
	description: "single-build render fixture catalog"
}

_version: "0.1.0"
_res:     "testing.opmodel.dev/library-render/cat/resources"
_traits:  "testing.opmodel.dev/library-render/cat/traits"
_tx:      "testing.opmodel.dev/library-render/cat/transformers"

// ── Resources ───────────────────────────────────────────────────────

#ContainerResource: c.#Resource & {
	metadata: {
		name:           "container"
		modulePath:     "\(_res)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_res)/container@v1"
		description:    "A single container workload"
	}
	matchLabels: "render.test/workload": "stateless"
	spec: container: {
		image!: string
		port:   int | *8080
	}
}

#ConfigMapsResource: c.#Resource & {
	metadata: {
		name:           "config-maps"
		modulePath:     "\(_res)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_res)/config-maps@v1"
		description:    "Named config maps, one object each"
	}
	spec: configMaps: [string]: data: [string]: string
}

// No transformer requires or optionally consumes this: a demand for it is a
// rung-1 hard miss (empty bucket).
#OrphanResource: c.#Resource & {
	metadata: {
		name:           "orphan"
		modulePath:     "\(_res)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_res)/orphan@v1"
		description:    "A resource no transformer in this catalog handles"
	}
	spec: orphan: size!: string
}

// The same contract base at the next apiVersion, handled by
// orphan-v2-transformer: a demand for orphan@v1 is a hard miss whose
// same-base alternative is this key.
#OrphanV2Resource: c.#Resource & {
	metadata: {
		name:           "orphan"
		modulePath:     "\(_res)/v2"
		apiVersion:     "v2"
		catalogVersion: _version
		fqn:            "\(_res)/orphan@v2"
		description:    "The orphan contract at apiVersion v2"
	}
	spec: orphan: size!: string
}

// Required by narrow-transformer NARROWED to a name the demanding component
// never uses, so the always-unify rung disqualifies the only candidate.
#NarrowResource: c.#Resource & {
	metadata: {
		name:           "narrow"
		modulePath:     "\(_res)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_res)/narrow@v1"
		description:    "A resource whose only transformer narrows it"
	}
	spec: narrow: name!: string
}

#IncompleteResource: c.#Resource & {
	metadata: {
		name:           "incomplete"
		modulePath:     "\(_res)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_res)/incomplete@v1"
		description:    "Handled by a transformer whose output never becomes concrete"
	}
	spec: incomplete: note?: string
}

#BrokenResource: c.#Resource & {
	metadata: {
		name:           "broken"
		modulePath:     "\(_res)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_res)/broken@v1"
		description:    "Handled by a transformer whose output conflicts at application"
	}
	spec: broken: note?: string
}

// ── Traits ──────────────────────────────────────────────────────────

// Advisory posture; handled by service-transformer.
#ExposeTrait: c.#Trait & {
	metadata: {
		name:           "expose"
		modulePath:     "\(_traits)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_traits)/expose@v1"
		description:    "Expose the container through a Service"
	}
	optional: bool | *true
	spec: expose: port: int | *80
	appliesTo: [#ContainerResource]
}

// Advisory posture; no transformer handles it, so an attachment degrades to
// an unhandled-trait warning.
#SidecarTrait: c.#Trait & {
	metadata: {
		name:           "sidecar"
		modulePath:     "\(_traits)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_traits)/sidecar@v1"
		description:    "An advisory trait nothing here renders"
	}
	optional: bool | *true
	spec: sidecar: image?: string
	appliesTo: [#ContainerResource]
}

// Load-bearing posture; no transformer handles it, so an attachment is an
// unresolved trait demand.
#BackupTrait: c.#Trait & {
	metadata: {
		name:           "backup"
		modulePath:     "\(_traits)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_traits)/backup@v1"
		description:    "A load-bearing trait nothing here renders"
	}
	optional: bool | *false
	spec: backup: schedule?: string
	appliesTo: [#ContainerResource]
}

// NO optional posture stated: the fixture for the measured boundary where
// fail-closed arrives as an incomplete-value build error at core's own
// `optional: bool` rather than as a diagnostics row.
#UnstatedTrait: c.#Trait & {
	metadata: {
		name:           "unstated"
		modulePath:     "\(_traits)/v1"
		apiVersion:     "v1"
		catalogVersion: _version
		fqn:            "\(_traits)/unstated@v1"
		description:    "A trait attached without its catalog stating a posture"
	}
	spec: unstated: note?: string
	appliesTo: [#ContainerResource]
}

// ── Transformers ────────────────────────────────────────────────────
// Each re-declares the #transform slots it references: CUE resolves
// references lexically, so a slot core declares is not in scope here
// unless restated (0019 02-design.md, "The three inputs").

#transformers: {
	"\(_tx)/deployment-transformer@\(_version)": {
		metadata: {
			name:        "deployment-transformer"
			fqn:         "\(_tx)/deployment-transformer@\(_version)"
			description: "Renders a container component as a Deployment"
		}
		requiredLabels: "render.test/workload":               "stateless"
		requiredResources: (#ContainerResource.metadata.fqn): #ContainerResource
		optionalTraits: (#ExposeTrait.metadata.fqn):          #ExposeTrait
		#transform: {
			#moduleInstance: _
			#component:      _
			#context:        _
			output: {
				apiVersion: "apps/v1"
				kind:       "Deployment"
				metadata: {
					name:      #component.#names.resourceName
					namespace: #context.#moduleInstanceMetadata.namespace
					labels:    #context.labels
				}
				spec: {
					// Read through #moduleInstance (0019 D3) when the module
					// exposes a replica count; scenario modules with an empty
					// #config render one replica.
					replicas: *1 | int
					if #moduleInstance.values.replicas != _|_ {
						replicas: #moduleInstance.values.replicas
					}
					image: #component.spec.container.image
					port:  #component.spec.container.port
				}
			}
		}
	}

	"\(_tx)/service-transformer@\(_version)": {
		metadata: {
			name:        "service-transformer"
			fqn:         "\(_tx)/service-transformer@\(_version)"
			description: "Renders an exposed container component as a Service"
		}
		requiredResources: (#ContainerResource.metadata.fqn): #ContainerResource
		requiredTraits: (#ExposeTrait.metadata.fqn):          #ExposeTrait
		#transform: {
			#component: _
			#context:   _
			output: {
				apiVersion: "v1"
				kind:       "Service"
				metadata: {
					name:   #component.#names.dns.short
					labels: #context.labels
				}
				spec: {
					port:       #component.spec.expose.port
					targetPort: #component.spec.container.port
				}
			}
		}
	}

	"\(_tx)/configmap-transformer@\(_version)": {
		metadata: {
			name:        "configmap-transformer"
			fqn:         "\(_tx)/configmap-transformer@\(_version)"
			description: "Renders each named config map as a ConfigMap"
		}
		requiredResources: (#ConfigMapsResource.metadata.fqn): #ConfigMapsResource
		#transform: {
			#component: _
			#context:   _
			output: [
				for name, cm in #component.spec.configMaps {
					apiVersion: "v1"
					kind:       "ConfigMap"
					metadata: {
						"name": name
						labels: #context.labels
					}
					data: cm.data
				},
			]
		}
	}

	"\(_tx)/orphan-v2-transformer@\(_version)": {
		metadata: {
			name:        "orphan-v2-transformer"
			fqn:         "\(_tx)/orphan-v2-transformer@\(_version)"
			description: "Handles the orphan contract at apiVersion v2 only"
		}
		requiredResources: (#OrphanV2Resource.metadata.fqn): #OrphanV2Resource
		#transform: {
			#component: _
			output: {
				apiVersion: "v1"
				kind:       "OrphanStore"
				metadata: name: #component.#names.resourceName
			}
		}
	}

	"\(_tx)/narrow-transformer@\(_version)": {
		metadata: {
			name:        "narrow-transformer"
			fqn:         "\(_tx)/narrow-transformer@\(_version)"
			description: "Requires the narrow resource pinned to one name"
		}
		requiredResources: (#NarrowResource.metadata.fqn): #NarrowResource & {
			spec: narrow: name: "the-only-name-this-transformer-serves"
		}
		#transform: {
			#component: _
			output: {
				apiVersion: "v1"
				kind:       "Narrow"
				metadata: name: #component.#names.resourceName
			}
		}
	}

	"\(_tx)/incomplete-transformer@\(_version)": {
		metadata: {
			name:        "incomplete-transformer"
			fqn:         "\(_tx)/incomplete-transformer@\(_version)"
			description: "Output carries a declared field nothing ever fills"
		}
		requiredResources: (#IncompleteResource.metadata.fqn): #IncompleteResource
		#transform: output: {
			apiVersion: "v1"
			kind:       "Incomplete"
			metadata: name: string
		}
	}

	"\(_tx)/broken-transformer@\(_version)": {
		metadata: {
			name:        "broken-transformer"
			fqn:         "\(_tx)/broken-transformer@\(_version)"
			description: "Output conflicts with its own inputs at application time"
		}
		requiredResources: (#BrokenResource.metadata.fqn): #BrokenResource
		#transform: {
			#context: _
			output: {
				apiVersion: "v1"
				kind:       "Broken"
				// The conflict is against a CONCRETE input (the instance's
				// authored namespace); a defaulted disjunction such as the
				// component name would merely be narrowed by unification.
				metadata: {
					name:      "broken"
					namespace: #context.#moduleInstanceMetadata.namespace & "never-this-namespace"
				}
			}
		}
	}
}
