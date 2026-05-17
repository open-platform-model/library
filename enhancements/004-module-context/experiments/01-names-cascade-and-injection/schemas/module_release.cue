package v1alpha2

// Trimmed #ModuleRelease implementing the 3-step flow from 03-schema.md:
//   1. Unify values into #config so dynamic #components materialise.
//   2. Feed the post-config component map to #ContextBuilder.
//   3. Unify the builder's outputs back into the module.
//
// The production schema includes a UUID derivation, auto-secrets injection,
// and label propagation — none of which exercise the context-builder
// mechanics. The fields below are the minimum needed for cases that
// exercise the full release pipeline (exp 02). Exp 01 typically invokes
// #ContextBuilder directly without going through #ModuleRelease.
#ModuleRelease: {
	apiVersion: #ApiVersion
	kind:       "ModuleRelease"

	metadata: {
		name!:      #NameType
		namespace!: string
		uuid!:      #UUIDType
	}

	#module!: #Module
	values:   _

	let _withConfig = #module & {#config: values}
	let _moduleMetadata = _withConfig.metadata

	let _builderOut = (#ContextBuilder & {
		#release: {
			name:      metadata.name
			namespace: metadata.namespace
			uuid:      metadata.uuid
		}
		#module: {
			name:    _moduleMetadata.name
			version: _moduleMetadata.version
			fqn:     _moduleMetadata.fqn
			uuid:    _moduleMetadata.uuid
		}
		#components: _withConfig.#components
	}).out

	let unifiedModule = _withConfig & {
		#ctx:        _builderOut.ctx
		#components: _builderOut.injections
	}

	// Surfaces the resolved values for assertions.
	ctx: _builderOut.ctx
	components: {
		for name, comp in unifiedModule.#components {(name): comp}
	}
}
