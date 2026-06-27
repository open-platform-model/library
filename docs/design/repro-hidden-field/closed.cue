// Does wrapping the transformer in the REAL closed core schema
// (#TransformerMap / #Platform), the way the kernel's Materialize does,
// reproduce the bug in PURE CUE? If these still render concretely, the
// closedness interaction is Go-FillPath-specific, not a CUE-level effect.
package hiddenbug

import (
	tf3 "opmodel.dev/catalogs/opm/transformers"
	c "opmodel.dev/core@v1"
)

_fqn: "opmodel.dev/catalogs/opm/transformers/deployment-transformer@0.5.2"

// (1) Constrain the composed map with the closed #TransformerMap pattern,
// exactly like #composedTransformers?: #TransformerMap.
_composedClosed: c.#TransformerMap & {
	(_fqn): tf3.#DeploymentTransformer
}
_txClosed: _composedClosed[_fqn].#transform & {
	#component: _applied.#component
	#context:  _applied.#context
}
containersClosedMap: _txClosed.output.spec.template.spec.containers

// (2) Go all the way: fill #composedTransformers on a real closed c.#Platform.
_platform: c.#Platform & {
	metadata: name: "kubernetes"
	type: "kubernetes"
	#registry: "opmodel.dev/catalogs/opm": {enable: true}
	#composedTransformers: {
		(_fqn): tf3.#DeploymentTransformer
	}
}
_txViaPlatform: _platform.#composedTransformers[_fqn].#transform & {
	#component: _applied.#component
	#context:  _applied.#context
}
containersViaPlatform: _txViaPlatform.output.spec.template.spec.containers
