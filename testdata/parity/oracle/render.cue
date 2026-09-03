// THE ORACLE. The render pipeline written as plain CUE unification, generic
// over its inputs. Lifted from enhancements/0019/experiments/01-purecue-render-flow
// (concluded 2026-08-19) and laid out in the same steps as the kernel's
// render glue (opm/internal/renderstage/render.cue.tmpl) so the two can be
// read side by side:
//
//     glue `match` (#Match)                    ->  `matched` / `pairs` (predicate rung only)
//     core's #TransformerContext projection    ->  `_contextFor` (0019 D12; the kernel supplies #runtimeName only)
//     glue `rendered`                          ->  `rendered`
//
// SCOPE OF `matched`. This is the predicate rung, enough to produce the right
// pairs for the fixtures; it is not a specification of matching (0019 D10
// owns that). The glue's reverse-index and always-unify rungs are absent: on
// the shipped fixtures no primitive body conflicts with a transformer's
// required copy, so the pair sets agree with no exemption (0019 D10); a
// fixture that did conflict would be refused by the kernel and fail the
// harness's pair-set comparison.
package oracle

#Render: {
	// Inputs, all definitions so `cue vet -c` checks concreteness of the
	// rendered output alone. Exposing the catalog map publicly would demand
	// concreteness of every unmatched transformer's optional templates.
	#instance: _ // a #ModuleInstance, entering by import

	#transformers: _ // a catalog's #transformers map, keyed by FQN

	#runtime: string // identity of the executing runtime; nothing in the artifacts can know it

	// The instance's components, verbatim. No finalization, no second value:
	// definition fields (#names, #resources, #traits, #instance) travel with
	// the value because that is what passing a value along means.
	_components: #instance.components

	// ---- Match (predicate rung; the glue's #Match) ----------------------
	matched: {
		for cid, comp in _components {
			(cid): {
				for tfqn, tf in #transformers {
					(tfqn): {
						_missingLabels: [
							if tf.requiredLabels != _|_ for k, v in tf.requiredLabels
							if (comp.matchLabels[k] & v) == _|_ {k},
						]
						_missingResources: [
							if tf.requiredResources != _|_ for fqn, _ in tf.requiredResources
							if comp.#resources[fqn] == _|_ {fqn},
						]
						_missingTraits: [
							if tf.requiredTraits != _|_ for fqn, _ in tf.requiredTraits
							if comp.#traits[fqn] == _|_ {fqn},
						]
						ok: len(_missingLabels) == 0 &&
							len(_missingResources) == 0 &&
							len(_missingTraits) == 0
					}
				}
			}
		}
	}

	pairs: [
		for cid, byTf in matched
		for tfqn, m in byTf
		if m.ok {{component: cid, transformer: tfqn}},
	]

	// ---- #context: the projection core performs (0019 D12) --------------
	_contextFor: {
		comp!: _
		out: {
			#moduleInstanceMetadata: {
				name:      #instance.metadata.name
				namespace: #instance.metadata.namespace
				fqn:       #instance.metadata.fqn
				uuid:      #instance.metadata.uuid
				version:   #instance.#moduleMetadata.version
				if #instance.metadata.labels != _|_ {labels: #instance.metadata.labels}
				if #instance.metadata.annotations != _|_ {annotations: #instance.metadata.annotations}
			}
			#componentMetadata: {
				name: comp.metadata.name
				if comp.metadata.labels != _|_ {labels: comp.metadata.labels}
				if comp.metadata.annotations != _|_ {annotations: comp.metadata.annotations}
			}
			#runtimeName: #runtime
		}
	}

	// ---- Execute: one unification per matched pair (the glue's `rendered`)
	// All three inputs core declares on #transform are supplied, whole:
	// #moduleInstance is the instance as imported, siblings included, the
	// same value the kernel fills since library-instance-fill (0019 D3, D11).
	rendered: {
		for p in pairs {
			"\(p.component) :: \(p.transformer)": (#transformers[p.transformer].#transform & {
				#moduleInstance: #instance
				#component:      _components[p.component]
				#context: (_contextFor & {comp: _components[p.component]}).out
			}).output
		}
	}
}
