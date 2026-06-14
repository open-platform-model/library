// Pure-CUE reproduction of the transformer output-local hidden field bug.
// No Go. The "glue" the kernel does in Go (FillPath #component / #context into
// #transform) is here expressed as plain CUE unification (&).
package hiddenbug

import (
	tf "opmodel.dev/catalogs/opm/transformers"
)

// Apply a concrete component + context to the real (buggy v0.5.2) deployment
// transformer's #transform, exactly as the kernel does — but via `&`.
_applied: tf.#DeploymentTransformer.#transform & {
	#component: {
		metadata: {
			name: "web"
			labels: "core.opmodel.dev/workload-type": "stateless"
		}
		spec: {
			container: {
				name: "web"
				image: {repository: "nginx", tag: "1.27", digest: ""}
			}
		}
	}
	#context: {
		#moduleReleaseMetadata: {
			name:      "web-app-demo"
			namespace: "default"
			uuid:      "11111111-2222-5333-8444-555555555555"
		}
		#componentMetadata: {
			name: "web"
		}
		#runtimeName: "opm-test"
	}
}

// The whole rendered Deployment.
output: _applied.output

// The specific field that goes non-concrete in the kernel.
containers: _applied.output.spec.template.spec.containers

// ---- materialize-style indirection, expressed in CUE ----
// The kernel's Materialize puts each transformer into a map keyed by FQN
// (#composedTransformers), then the executor looks it back out and fills
// #component/#context. Mimic that map round-trip purely in CUE.
_composed: {
	"opmodel.dev/catalogs/opm/transformers/deployment-transformer@0.5.2": tf.#DeploymentTransformer
}
_viaMap: _composed["opmodel.dev/catalogs/opm/transformers/deployment-transformer@0.5.2"].#transform & {
	#component: _applied.#component
	#context:  _applied.#context
}
containersViaMap: _viaMap.output.spec.template.spec.containers
