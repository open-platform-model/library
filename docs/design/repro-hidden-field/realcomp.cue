// Pure-CUE test using the EXACT real finalized web component dumped from the
// kernel (OPM_DUMP_CUE). If `containersReal` exports concretely, the bug is
// NOT in CUE — it is in the Go value plumbing.
package hiddenbug

import (
	tf2 "opmodel.dev/catalogs/opm/transformers"
)

_realComponent: {
	kind: "Component"
	metadata: {
		name:         "web"
		resourceName: "web"
		labels: {
			"component.opmodel.dev/name":     "web"
			"core.opmodel.dev/workload-type": "stateless"
		}
	}
	spec: {
		initContainers: []
		restartPolicy: "Always"
		scaling: count: 2
		sidecarContainers: []
		statelessWorkload: {
			container: {
				name: "web"
				image: {repository: "nginx", tag: "1.27", digest: "", pullPolicy: "IfNotPresent", reference: "nginx:1.27"}
				ports: http: {name: "http", targetPort: 8080, protocol: "TCP"}
			}
			scaling: count: 2
			restartPolicy: "Always"
			updateStrategy: type: "RollingUpdate"
		}
		container: {
			name: "web"
			image: {repository: "nginx", tag: "1.27", digest: "", pullPolicy: "IfNotPresent", reference: "nginx:1.27"}
			ports: http: {name: "http", targetPort: 8080, protocol: "TCP"}
		}
		updateStrategy: type: "RollingUpdate"
		expose: {
			ports: http: {name: "http", targetPort: 8080, protocol: "TCP"}
			type: "ClusterIP"
		}
		httpRoute: {
			hostnames: ["web.example.test"]
			rules: [{
				backendPort: 8080
				matches: [{path: {type: "PathPrefix", value: "/"}}]
			}]
		}
	}
}

_appliedReal: tf2.#DeploymentTransformer.#transform & {
	#component: _realComponent
	#context: {
		#moduleInstanceMetadata: {name: "web-app-demo", namespace: "default", uuid: "11111111-2222-5333-8444-555555555555"}
		#componentMetadata: {name: "web", labels: {"component.opmodel.dev/name": "web", "core.opmodel.dev/workload-type": "stateless"}}
		#runtimeName: "opm-test"
	}
}

containersReal: _appliedReal.output.spec.template.spec.containers
