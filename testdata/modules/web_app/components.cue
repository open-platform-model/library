package web_app

import (
	tr "testing.opmodel.dev/catalogs/opm/traits"
	bp "testing.opmodel.dev/catalogs/opm/blueprints"
	res "testing.opmodel.dev/catalogs/opm/resources"
)

// One stateless web component. Attaches:
//   - Container resource     → satisfies DeploymentTransformer's required FQN
//   - Scaling trait          → optional for DeploymentTransformer
//   - RestartPolicy trait    → optional for DeploymentTransformer
//   - HttpRoute trait        → pairs with http-route-transformer
//   - Expose trait           → satisfies ServiceTransformer's required trait
//     FQN, so the component pairs deployment-transformer, service-transformer,
//     and http-route-transformer in a single match cycle
//   - StatelessWorkload blueprint → demonstrates Blueprint composition; its
//     spec.statelessWorkload field is satisfied alongside the direct primitives.
//     Blueprints import FLAT from …/blueprints (enhancement 0010 D42).
//
// The "core.opmodel.dev/workload-type": "stateless" label is what the
// DeploymentTransformer's requiredLabels matches against. It is set
// explicitly here because the matcher reads component metadata.labels until
// the library-match-labels slice flips the read to matchLabels (transitional
// invariant 2 of library-core-retarget).
#components: {
	web: {
		metadata: {
			name: "web"
			labels: "core.opmodel.dev/workload-type": "stateless"
		}
		bp.#StatelessWorkload
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
