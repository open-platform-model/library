package core

import (
	cue_uuid "uuid"
)

// #Module: The portable application blueprint created by developers and/or platform teams
#Module: {
	kind: "Module"

	metadata: {
		name!: #NameType // Example: "example-module"

		modulePath: metadata.modulePath                                 // Example: "example.com/modules"
		version:    metadata.version                                    // Example: "0.1.0"
		fqn:        #ModuleFQNType & "\(modulePath)/\(name):\(version)" // Example: "example.com/modules/example-module:0.1.0"

		// Unique identifier for the module, computed as a UUID v5 (SHA1) of the FQN using the OPM namespace UUID.
		uuid: #UUIDType & cue_uuid.SHA1(OPMNamespace, fqn)
		#definitionName: (#KebabToPascal & {"in": name}).out

		description?: string
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType

		labels: {
			// Standard labels for module identification
			"module.opmodel.dev/name":    "\(name)"
			"module.opmodel.dev/version": "\(version)"
			"module.opmodel.dev/uuid":    "\(uuid)"
		}
		annotations: {
			// Standard annotations for module metadata
			"module.opmodel.dev/default-namespace"?: string
		}
	}

	// Components defined in this module (developer-defined, required. May be added
	// to by the platform-team). The pattern constraint wires the module-level
	// release into every component so each component computes its own #names
	// from a shared release identity (enhancement 0001 D3).
	#components: [Id=#NameType]: #Component & {
		metadata: {
			name: string | *Id
			labels: "component.opmodel.dev/name": name
		}
		#release: #ctx.release
	}

	// Value schema - constraints and defaults.
	// Developers define the configuration contract and reference it in their components.
	// MUST be OpenAPIv3 compliant (no CUE templating - for/if statements)
	#config: _

	// debugValues: Example values for testing and debugging.
	// It is unified and validated in the runtime
	debugValues: _

	// Inline runtime context channel. Open at the top level (`...`) so future
	// enhancements can add `platform` / `environment` siblings without breaking
	// module bodies. Introduced by enhancement 0001 (D1).
	//
	// `release` is set by #ModuleRelease from its own metadata. `components`
	// is a pure CUE projection over every component's #names — components are
	// the single source of truth for their own identity; #ctx.components only
	// mirrors them (D2).
	#ctx: {
		release: #ReleaseIdentity

		components: {
			for id, c in #components {
				(id): c.#names
			}
		}

		...
	}
}

#ModuleMap: [string]: #Module
