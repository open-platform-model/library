package web_app

import (
	tr "opmodel.dev/modules/opm/traits@v1"
	bp_workload "opmodel.dev/modules/opm/blueprints/workload@v1"
	res "opmodel.dev/modules/opm/resources@v1"
)

// One stateless web component. Attaches:
//   - Container resource     → satisfies DeploymentTransformer's required FQN
//   - Scaling trait          → optional for DeploymentTransformer
//   - RestartPolicy trait    → optional for DeploymentTransformer
//   - HttpRoute trait        → unmatched in this fixture (no opm HTTPRoute
//     transformer registered) — surfaces the unhandled-trait warning path
//   - Expose trait           → satisfies ServiceTransformer's required trait
//     FQN, so the component pairs both deployment-transformer and
//     service-transformer in a single match cycle
//   - StatelessWorkloadBlueprint → demonstrates Blueprint composition; its
//     spec.statelessWorkload field is satisfied alongside the direct primitives
//
// The "core.opmodel.dev/workload-type": "stateless" label is what the
// DeploymentTransformer's requiredLabels matches against. It is set
// explicitly here so the matcher selects deployment-transformer over the
// other workload transformers (statefulset / daemonset / job / cronjob).
#components: {
	web: {
		metadata: {
			name: "web"
			labels: "core.opmodel.dev/workload-type": "stateless"
		}
		bp_workload.#StatelessWorkload
		tr.#HttpRoute
		tr.#Expose

		spec: {

			statelessWorkload: {
				container: {
					name:  "web"
					image: #config.image
					ports: http: {
						name:       "http"
						targetPort: #config.port
					}
				}
				scaling: {count: #config.replicas}
				restartPolicy: "Always"
				updateStrategy: type: "RollingUpdate"
			}

			expose: {
				type: "ClusterIP"
				ports: http: {
					name:       "http"
					targetPort: #config.port
				}
			}

			httpRoute: {
				hostnames: #config.hostnames
				rules: [{
					backendPort: #config.port
					matches: [{path: {type: "PathPrefix", value: "/"}}]
				}]
			}
		}
	}

	// Second component: a config-only attachment carrying two ConfigMap
	// entries. Pairs with configmap-transformer and exercises the
	// list-output path — two Compiled items per (component, transformer).
	config: {
		metadata: name: "config"
		res.#ConfigMaps

		spec: configMaps: {
			"app-config": {
				immutable: false
				data: {
					"LOG_LEVEL":      "info"
					"FEATURE_FLAG_A": "on"
				}
			}
			"feature-flags": {
				immutable: false
				data: {
					"FLAG_B": "off"
					"FLAG_C": "on"
				}
			}
		}
	}
}
