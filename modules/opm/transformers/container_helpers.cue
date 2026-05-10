package transformers

import (
	"strconv"

	k8scorev1 "opmodel.dev/modules/opm/schemas/kubernetes/core/v1@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res "opmodel.dev/modules/opm/resources"
)

// #ToK8sContainer converts an OPM #ContainerSchema to a Kubernetes #Container.
// OPM uses struct-keyed env/ports/volumeMounts; Kubernetes expects lists.
//
// Env var dispatch:
//   value?            -> { name, value }
//   from?             -> { name, valueFrom: { secretKeyRef: ... } }
//   fieldRef?         -> { name, valueFrom: { fieldRef: ... } }
//   resourceFieldRef? -> { name, valueFrom: { resourceFieldRef: ... } }
//
// When #releasePrefix is set, it is prepended to the secretKeyRef name for
// auto-generated secrets (#SecretLiteral). Pre-existing K8s
// Secret references (#SecretK8sRef) are never prefixed.
//
// Usage:
//   (#ToK8sContainer & {"in": _container, #releasePrefix: "my-release"}).out
#ToK8sContainer: {
	X="in":          res.#ContainerSchema
	#releasePrefix?: string

	out: k8scorev1.#Container & {
		name:            X.name
		image:           X.image.reference
		imagePullPolicy: X.image.pullPolicy
		if X.command != _|_ {
			command: X.command
		}
		if X.args != _|_ {
			args: X.args
		}
		if X.ports != _|_ {
			ports: [for _, p in X.ports {
				name:          p.name
				containerPort: p.targetPort
				protocol:      p.protocol
				if p.hostIP != _|_ {hostIP: p.hostIP}
				if p.hostPort != _|_ {hostPort: p.hostPort}
			}]
		}

		// Env var dispatch: map OPM source types to K8s env entries
		if X.env != _|_ {
			env: [for _, e in X.env {
				// Literal value — inline string
				if e.value != _|_ {
					name:  e.name
					value: e.value
				}

				// Secret reference — dispatch by variant
				if e.from != _|_ {
					name: e.name
					// #SecretK8sRef: use the pre-existing K8s Secret name + key
					if e.from.secretName != _|_ {
						valueFrom: secretKeyRef: {
							name: e.from.secretName
							key:  e.from.remoteKey
						}
					}

					// #SecretLiteral: use $secretName / $dataKey.
					// When #releasePrefix is set, prepend it to the secret name so
					// that multiple releases of the same module can coexist in one
					// namespace without their auto-generated secrets colliding.
					if e.from.secretName == _|_ {
						if #releasePrefix != _|_ {
							valueFrom: secretKeyRef: {
								name: "\(#releasePrefix)-\(e.from.$secretName)"
								key:  e.from.$dataKey
							}
						}
						if #releasePrefix == _|_ {
							valueFrom: secretKeyRef: {
								name: e.from.$secretName
								key:  e.from.$dataKey
							}
						}
					}
				}

				// Downward API — pod/container metadata
				if e.fieldRef != _|_ {
					name: e.name
					valueFrom: fieldRef: {
						fieldPath: e.fieldRef.fieldPath
						if e.fieldRef.apiVersion != _|_ {
							apiVersion: e.fieldRef.apiVersion
						}
					}
				}

				// Container resource limits/requests
				if e.resourceFieldRef != _|_ {
					name: e.name
					valueFrom: resourceFieldRef: {
						resource: e.resourceFieldRef.resource
						if e.resourceFieldRef.containerName != _|_ {
							containerName: e.resourceFieldRef.containerName
						}
						if e.resourceFieldRef.divisor != _|_ {
							divisor: e.resourceFieldRef.divisor
						}
					}
				}
			}]
		}

		// Bulk injection from ConfigMaps/Secrets
		if X.envFrom != _|_ {
			envFrom: X.envFrom
		}

		if X.resources != _|_ {
			resources: {
				if X.resources.requests != _|_ {
					requests: {
						if X.resources.requests.cpu != _|_ {
							cpu: (schemas.#NormalizeCPU & {in: X.resources.requests.cpu}).out
						}
						if X.resources.requests.memory != _|_ {
							memory: (schemas.#NormalizeMemory & {in: X.resources.requests.memory}).out
						}
					}
				}

				if X.resources.limits != _|_ {
					limits: {
						if X.resources.limits.cpu != _|_ {
							cpu: (schemas.#NormalizeCPU & {in: X.resources.limits.cpu}).out
						}
						if X.resources.limits.memory != _|_ {
							memory: (schemas.#NormalizeMemory & {in: X.resources.limits.memory}).out
						}
					}
				}

				// GPU extended resource: emitted to both requests and limits.
				// Kubernetes requires extended resource requests == limits (no overcommit).
				if X.resources.gpu != _|_ {
					let _gpuVal = strconv.FormatInt(X.resources.gpu.count, 10)
					requests: {"\(X.resources.gpu.resource)": _gpuVal}
					limits: {"\(X.resources.gpu.resource)": _gpuVal}
				}
			}
		}

		// Container-level security context
		if X.securityContext != _|_ {
			let _sc = X.securityContext
			securityContext: {
				if _sc.privileged != _|_ {
					privileged: _sc.privileged
				}
				if _sc.runAsNonRoot != _|_ {
					runAsNonRoot: _sc.runAsNonRoot
				}
				if _sc.runAsUser != _|_ {
					runAsUser: _sc.runAsUser
				}
				if _sc.runAsGroup != _|_ {
					runAsGroup: _sc.runAsGroup
				}
				if _sc.readOnlyRootFilesystem != _|_ {
					readOnlyRootFilesystem: _sc.readOnlyRootFilesystem
				}
				if _sc.allowPrivilegeEscalation != _|_ {
					allowPrivilegeEscalation: _sc.allowPrivilegeEscalation
				}
				if _sc.capabilities != _|_ {
					capabilities: {
						if _sc.capabilities.add != _|_ {add: _sc.capabilities.add}
						if _sc.capabilities.drop != _|_ {drop: _sc.capabilities.drop}
					}
				}
			}
		}

		// Volume mounts: extract only K8s-valid fields (strip embedded volume source data)
		if X.volumeMounts != _|_ {
			volumeMounts: [for _, vm in X.volumeMounts {
				name:      vm.name
				mountPath: vm.mountPath
				if vm.subPath != _|_ {subPath: vm.subPath}
				if vm.readOnly == true {readOnly: vm.readOnly}
			}]
		}

		if X.startupProbe != _|_ {
			startupProbe: X.startupProbe
		}
		if X.livenessProbe != _|_ {
			livenessProbe: X.livenessProbe
		}
		if X.readinessProbe != _|_ {
			readinessProbe: X.readinessProbe
		}

		// Pre-stop lifecycle hook
		if X.preStopCommand != _|_ {
			lifecycle: preStop: exec: command: X.preStopCommand
		}
	}
}

// #ToK8sContainers converts a list of OPM containers to Kubernetes containers.
//
// Usage:
//   (#ToK8sContainers & {"in": _initContainers, #releasePrefix: "my-release"}).out
#ToK8sContainers: {
	X="in": [...res.#ContainerSchema]
	_prefix=#releasePrefix?: string
	out: [for c in X {
		(#ToK8sContainer & {"in": c, #releasePrefix: _prefix}).out
	}]
}

// #ToK8sVolumes converts OPM volumes map to Kubernetes volumes list.
// Handles all volume source types: emptyDir, persistentClaim, configMap, secret.
//
// For secret volumes, from is a #SecretSchema (carrying name, immutable, data).
// The K8s secret name is computed via #SecretImmutableName — identical to the
// name produced by #SecretTransformer — so mutable and immutable secrets both
// resolve correctly without any extra wiring in the component.
//
// Usage:
//   (#ToK8sVolumes & {"in": _component.spec.volumes, #releasePrefix: "my-release"}).out
#ToK8sVolumes: {
	X="in": [string]: res.#VolumeSchema
	_prefix=#releasePrefix?: string

	out: [for vName, vol in X {
		name: vol.name | *vName
		if vol.emptyDir != _|_ {
			emptyDir: {
				if vol.emptyDir.medium != _|_ if vol.emptyDir.medium == "memory" {
					medium: "Memory"
				}
				if vol.emptyDir.sizeLimit != _|_ {
					sizeLimit: vol.emptyDir.sizeLimit
				}
			}
		}
		if vol.persistentClaim != _|_ {
			persistentVolumeClaim: claimName: "\(_prefix)-\(vName)"
		}
		if vol.configMap != _|_ {
			// Compute the same K8s name the configmap-transformer will generate:
			//   {releasePrefix}-{configmap.name}[-{contenthash}]
			// #ImmutableName handles both mutable (stable name) and
			// immutable (content-hash suffix) ConfigMaps transparently.
			//
			// Note: `let _cmData` captures concrete field values before the
			// definition boundary — same reason #ImmutableName uses `let _d = data`
			// internally. Without this, CUE loses concrete entries through the
			// open [string]: string pattern.
			let _cmData = vol.configMap.data
			let _k8sName = (res.#ImmutableName & {
				baseName:  "\(_prefix)-\(vol.configMap.name)"
				data:      _cmData
				immutable: vol.configMap.immutable
			}).out
			configMap: name: _k8sName
		}
		if vol.secret != _|_ {
			secret: {
				// Compute the same K8s name the secret-transformer will generate:
				//   {releasePrefix}-{secret.name}[-{contenthash}]
				// #SecretImmutableName handles both mutable (stable name) and
				// immutable (content-hash suffix) secrets transparently.
				let _k8sName = (res.#SecretImmutableName & {
					baseName:  "\(_prefix)-\(vol.secret.from.name)"
					data:      vol.secret.from.data
					immutable: vol.secret.from.immutable
				}).out
				secretName: _k8sName
				if vol.secret.items != _|_ {
					items: [for item in vol.secret.items {
						key:  item.key
						path: item.path
						if item.mode != _|_ {
							mode: item.mode
						}
					}]
				}
				if vol.secret.defaultMode != _|_ {
					defaultMode: vol.secret.defaultMode
				}
				if vol.secret.optional != _|_ {
					optional: vol.secret.optional
				}
			}
		}
		if vol.hostPath != _|_ {
			hostPath: {
				path: vol.hostPath.path
				if vol.hostPath.type != _|_ {
					type: vol.hostPath.type
				}
			}
		}
		if vol.nfs != _|_ {
			nfs: {
				server: vol.nfs.server
				path:   vol.nfs.path
				if vol.nfs.readOnly != _|_ {
					readOnly: vol.nfs.readOnly
				}
			}
		}
	}]
}

// Non-immutable secret: K8s name = {prefix}-{secret.name}, no hash suffix.
_testToK8sVolumesSecret: {
	in: {
		auth: {
			name: "auth"
			secret: {
				from: {
					name:      "zot-htpasswd"
					immutable: false
					data: {
						htpasswd: "admin:hashed"
					}
				}
				items: [{
					key:  "htpasswd"
					path: "auth/htpasswd"
					mode: 256
				}]
				defaultMode: 420
			}
		}
	}

	out: (#ToK8sVolumes & {
		"in":           in
		#releasePrefix: "registry"
	}).out

	out: [{
		name: "auth"
		secret: {
			secretName:  "registry-zot-htpasswd"
			defaultMode: 420
			items: [{
				key:  "htpasswd"
				path: "auth/htpasswd"
				mode: 256
			}]
		}
	}]
}

// Immutable secret: K8s name = {prefix}-{secret.name}-{contenthash}.
_testToK8sVolumesSecretImmutable: {
	in: {
		config: {
			name: "config"
			secret: {
				from: {
					name:      "my-config"
					immutable: true
					data: {
						"config.json": "hello"
					}
				}
				items: [{key: "config.json", path: "config.json"}]
			}
		}
	}

	out: (#ToK8sVolumes & {
		"in":           in
		#releasePrefix: "myapp-mycomponent"
	}).out

	out: [{
		name: "config"
		secret: {
			secretName: (res.#SecretImmutableName & {
				baseName: "myapp-mycomponent-my-config"
				data: {"config.json": "hello"}
				immutable: true
			}).out
			items: [{key: "config.json", path: "config.json"}]
		}
	}]
}

// Immutable configMap: K8s name = {configmap.name}-{contenthash}.
_testToK8sVolumesConfigMapImmutable: {
	in: {
		config: {
			name: "config"
			configMap: {
				name:      "wolf-config-toml"
				immutable: true
				data: {
					"wolf.toml": "[wolf]\nenabled = true"
				}
			}
		}
	}

	out: (#ToK8sVolumes & {
		"in":           in
		#releasePrefix: "wolf-release-wolf"
	}).out

	out: [{
		name: "config"
		configMap: name: (res.#ImmutableName & {
			baseName: "wolf-release-wolf-wolf-config-toml"
			data: {"wolf.toml": "[wolf]\nenabled = true"}
			immutable: true
		}).out
	}]
}

_testToK8sContainer: {
	// Example input container
	in: {
		name: "example-container"
		image: {
			repository: "example-image"
			tag:        "latest"
			digest:     ""
		}
		command: ["/bin/example"]
		args: ["--example-arg"]
		ports: {
			http: {
				name:       "http"
				targetPort: 8080
				protocol:   "TCP"
			}
		}
		env: {
			EXAMPLE_ENV_VAR: {
				name:  "EXAMPLE_ENV_VAR"
				value: "example-value"
			}
		}
		resources: {
			requests: {
				cpu:    "100m"
				memory: "128Mi"
			}
			limits: {
				cpu:    "200m"
				memory: "256Mi"
			}
		}
		volumeMounts: {
			exampleVolumeMount: {
				name:      "example-volume"
				mountPath: "/data/example"
			}
		}
	}

	out: (#ToK8sContainer & {"in": in}).out
}

_testToK8sContainers: {
	// Example list of input containers
	in: [
		{
			name: "example-container-1"
			image: {
				repository: "example-image-1"
				tag:        "latest"
				digest:     ""
			}
		},
		{
			name: "example-container-2"
			image: {
				repository: "example-image-2"
				tag:        "latest"
				digest:     ""
			}
		},
	]

	out: (#ToK8sContainers & {"in": in}).out
}

// Test: preStopCommand produces lifecycle.preStop.exec.command
_testToK8sContainerPreStop: {
	in: {
		name: "graceful"
		image: {
			repository: "app"
			tag:        "v1"
			digest:     ""
		}
		preStopCommand: ["/bin/sh", "-c", "sleep 5"]
	}

	out: (#ToK8sContainer & {"in": in}).out

	out: {
		name:  "graceful"
		image: "app:v1"
		lifecycle: preStop: exec: command: ["/bin/sh", "-c", "sleep 5"]
	}
}
