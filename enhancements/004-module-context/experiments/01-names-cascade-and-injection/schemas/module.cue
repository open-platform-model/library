package v1alpha2

// Trimmed #Module — drops the UUID derivation, #defines, debugValues, and
// label/annotation auto-injection from the production schema. Carries only
// the surface #ContextBuilder reads (metadata identity, #components map,
// #config) plus the 004 #ctx slot.
#Module: {
	apiVersion: #ApiVersion
	kind:       "Module"

	metadata: {
		name!:    #NameType
		version!: #VersionType
		fqn!:     #ModuleFQNType
		uuid!:    #UUIDType
	}

	#components: [Id=string]: #Component & {
		metadata: name: string | *Id
	}

	#config: _

	#ctx: #ModuleContext
}
