package v1alpha2

// Copied from enhancement 004 / 03-schema.md (post-slim). Identity-only;
// no #platform / #environment inputs. Per-component _componentNames are
// computed once and threaded into both ctx.runtime.components and
// per-component injections (so #Component.#names ==
// #ctx.runtime.components[<key>] by construction — D32).

#ContextBuilder: {
	#release: {
		name:      #NameType
		namespace: string
		uuid:      #UUIDType
	}
	#module: {
		name:    #NameType
		version: #VersionType
		fqn:     #ModuleFQNType
		uuid:    #UUIDType
	}
	#components: [string]: _

	let _componentNames = {
		for compName, comp in #components {
			(compName): {
				_releaseName: #release.name
				_namespace:   #release.namespace
				_compName:    compName
				if comp.metadata.resourceName != _|_ {
					resourceName: comp.metadata.resourceName
				}
			}
		}
	}

	out: {
		ctx: #ModuleContext & {
			runtime: #RuntimeContext & {
				release:    #release
				module:     #module
				components: _componentNames
			}
		}

		injections: {
			for compName, _ in #components {
				(compName): #names: _componentNames[compName]
			}
		}
	}
}
