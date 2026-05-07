package v1alpha2

import (
	cue_uuid "uuid"
)

// #ModuleRelease: The concrete deployment instance
// Contains: Reference to Module, values, target namespace
// Users/deployment systems create this to deploy a specific version
#ModuleRelease: {
	apiVersion: #ApiVersion
	kind:       "ModuleRelease"

	metadata: {
		name!:      #NameType
		namespace!: string // Required for releases (target environment)

		// Generate a stable UUID for this release based on the module's UUID, name, and namespace
		uuid: #UUIDType & cue_uuid.SHA1(OPMNamespace, "\(#moduleMetadata.uuid):\(name):\(namespace)")

		labels?: #LabelsAnnotationsType
		labels?: {if #moduleMetadata.labels != _|_ {#moduleMetadata.labels}}
		labels: {
			// Standard labels for module release identification
			"module-release.opmodel.dev/name": "\(name)"
			"module-release.opmodel.dev/uuid": "\(uuid)"
		}

		annotations?: #LabelsAnnotationsType
		annotations?: {if #moduleMetadata.annotations != _|_ {#moduleMetadata.annotations}}

	}

	// Reference to the Module to deploy
	#module!:        #Module
	#moduleMetadata: #module.metadata

	let unifiedModule = #module & {#config: values}

	// _autoSecrets discovers all #Secret instances from the resolved config (internal).
	// When non-empty, an opm-secrets component is automatically added to components.
	_autoSecrets: (#AutoSecrets & {#in: unifiedModule.#config}).out

	// components includes all user-defined components plus the auto-generated
	// opm-secrets component when the module config contains #Secret fields.
	//
	// Uses a comprehension to merge user components with the conditional opm-secrets entry.
	// The opm-secrets component uses explicit component.#Component typing;
	// see core/helpers/autosecrets.cue.
	components: {
		for name, comp in unifiedModule.#components {
			(name): comp
		}
		if len(_autoSecrets) > 0 {
			"opm-secrets": (#OpmSecretsComponent & {#secrets: _autoSecrets}).out
		}
	}

	// Concrete values (everything closed/concrete)
	// Must satisfy the #config from #module
	values: _
}

#ModuleReleaseMap: [string]: #ModuleRelease
