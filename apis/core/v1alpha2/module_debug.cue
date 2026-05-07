package v1alpha2

import (
	cue_uuid "uuid"
)

// #ModuleDebug: Build/debug instantiation of a module.
//
// Mirrors #ModuleRelease structurally, with two deliberate differences:
//
//   1. No target namespace — debug is environment-agnostic. Identity is derived from (module.uuid, debug name) without an environment term.
//   2. Values are hardcoded to the module's bundled debugValues; consumers cannot override them. This unifies the value source for tooling like `opm mod build` and gives the kernel a single CUE-driven path for build-time validation, instead of branching in Go.
//
// Use #ModuleRelease for deployment to an environment; use #ModuleDebug for local build, validation, and rendering of a module against its own example values.
#ModuleDebug: {
	apiVersion: #ApiVersion
	kind:       "ModuleDebug"

	metadata: {
		name!: #NameType

		// Identity is tied to the module and the debug instance name. No namespace term — debug is not bound to an environment.
		uuid: #UUIDType & cue_uuid.SHA1(OPMNamespace, "\(#moduleMetadata.uuid):debug:\(name)")

		labels?: #LabelsAnnotationsType
		labels?: {if #moduleMetadata.labels != _|_ {#moduleMetadata.labels}}
		labels: {
			"module-debug.opmodel.dev/name": "\(name)"
			"module-debug.opmodel.dev/uuid": "\(uuid)"
		}

		annotations?: #LabelsAnnotationsType
		annotations?: {if #moduleMetadata.annotations != _|_ {#moduleMetadata.annotations}}
	}

	#module!:        #Module
	#moduleMetadata: #module.metadata

	// Value source is fixed: the module's own debugValues. No `values` field is exposed on #ModuleDebug; consumers express debug intent by authoring debugValues on the #Module itself.
	let unifiedModule = #module & {#config: #module.debugValues}

	// _autoSecrets discovers all #Secret instances from the resolved config. Behaviour matches #ModuleRelease so build output stays representative of release output.
	_autoSecrets: (#AutoSecrets & {#in: unifiedModule.#config}).out

	components: {
		for name, comp in unifiedModule.#components {
			(name): comp
		}
		if len(_autoSecrets) > 0 {
			"opm-secrets": (#OpmSecretsComponent & {#secrets: _autoSecrets}).out
		}
	}
}

#ModuleDebugMap: [string]: #ModuleDebug
