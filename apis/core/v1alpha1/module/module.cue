package module

import (
	cue_uuid "uuid"
	t "opmodel.dev/core/v1alpha1/types@v1"
	component "opmodel.dev/core/v1alpha1/component@v1"
	policy "opmodel.dev/core/v1alpha1/policy@v1"
)

// #Module: The portable application blueprint created by developers and/or platform teams
#Module: {
	apiVersion: "opmodel.dev/core/v1alpha1"
	kind:       "Module"

	metadata: {
		modulePath!: t.#ModulePathType                                     // Example: "example.com/modules"
		name!:       t.#NameType                                           // Example: "example-module"
		version!:    t.#VersionType                                        // Example: "0.1.0"
		fqn:         t.#ModuleFQNType & "\(modulePath)/\(name):\(version)" // Example: "example.com/modules/example-module:0.1.0"

		// Unique identifier for the module, computed as a UUID v5 (SHA1) of the FQN using the OPM namespace UUID.
		uuid: t.#UUIDType & cue_uuid.SHA1(t.OPMNamespace, fqn)
		#definitionName: (t.#KebabToPascal & {"in": name}).out

		defaultNamespace?: string
		description?:      string
		labels?:           t.#LabelsAnnotationsType
		annotations?:      t.#LabelsAnnotationsType

		labels: {
			// Standard labels for module identification
			"module.opmodel.dev/name":    "\(name)"
			"module.opmodel.dev/version": "\(version)"
			"module.opmodel.dev/uuid":    "\(uuid)"
		}
	}

	// Components defined in this module (developer-defined, required. May be added to by the platform-team)
	#components: [Id=string]: component.#Component & {
		metadata: {
			name: string | *Id
			labels: "component.opmodel.dev/name": name
		}
	}

	// List of all components in this module
	// Useful for policies that want to apply to all components
	// #allComponents: [for _, c in #components {c}]

	// Module-level policies (developer-defined, optional. May be added to by the platform-team)
	#policies?: [Id=string]: policy.#Policy

	// Value schema - constraints and defaults.
	// Developers define the configuration contract and reference it in their components.
	// MUST be OpenAPIv3 compliant (no CUE templating - for/if statements)
	#config: _

	// debugValues: Example values for testing and debugging.
	// It is unified and validated in the runtime
	debugValues: _
}

#ModuleMap: [string]: #Module
