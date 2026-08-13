package web_app

import (
	tr "opmodel.dev/catalogs/opm/traits/v1beta1"
	bp "opmodel.dev/catalogs/opm/blueprints/v1beta1"
	res "opmodel.dev/catalogs/opm/resources/v1beta1"
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
//     Imports name the apiVersion level (…/blueprints/v1beta1, 0010 D49).
//
// Matching reads the component's derived matchLabels (0010 D36) — the
// StatelessWorkload blueprint's matchLabels carry the
// "core.opmodel.dev/workload-type": "stateless" key the
// DeploymentTransformer's requiredLabels selects on. The explicit
// metadata.labels duplicate below is DESCRIPTIVE and stays: render reads
// (e.g. an hpa-style transformer's workload-type lookup off the component)
// consume metadata.labels through the transformer context, which D36 keeps
// on the descriptive field.
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
				// The optional rollingUpdate substruct is deliberately
				// omitted (legal per #UpdateStrategySchema): since catalog
				// 2.0.0-alpha.3 the deployment-transformer guards the
				// dereference, so this exercises the schema-legal omission
				// end-to-end through the flow tests.
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
