package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"list"
	"strings"

	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
)

/////////////////////////////////////////////////////////////////
//// Secrets Resource
/////////////////////////////////////////////////////////////////

#SecretsResource: c.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/resources"
		version:     "v1"
		name:        "secrets"
		description: "A Secret definition for sensitive configuration"
		labels: {
			"resource.opmodel.dev/category": "config"
		}
	}

	spec: secrets: [secretName=string]: #SecretSchema & {name: string | *secretName}
}

#Secrets: c.#Component & {
	#resources: (#SecretsResource.metadata.fqn): #SecretsResource
}

/////////////////////////////////////////////////////////////////
//// Secret Contract Type
//// #Secret is the contract type that module authors place on
//// sensitive fields. The $opm discriminator enables auto-discovery
//// via CUE comprehensions.
/////////////////////////////////////////////////////////////////

#Secret: #SecretLiteral | #SecretK8sRef

#SecretType: {
	$opm:          "secret"
	$secretName!:  schemas.#NameType
	$dataKey!:     string
	$description?: string
}

// User provides the actual value. The transformer creates a K8s Secret with this data entry.
#SecretLiteral: {
	#SecretType
	value!: string
}

// References a pre-existing K8s Secret. OPM emits no resource — only wires
// secretKeyRef in env vars.
#SecretK8sRef: {
	#SecretType
	secretName!: string
	remoteKey!:  string
}

/////////////////////////////////////////////////////////////////
//// Secret Schema
/////////////////////////////////////////////////////////////////

// `data` holds either #Secret entries (auto-discovered via #AutoSecrets) or plain strings.
// `name` is auto-populated from the map key in the resource spec.
#SecretSchema: {
	name!:     string
	type:      "Opaque" | "kubernetes.io/service-account-token" | "kubernetes.io/dockercfg" | "kubernetes.io/dockerconfigjson" | "kubernetes.io/basic-auth" | "kubernetes.io/ssh-auth" | "kubernetes.io/tls" | "bootstrap.kubernetes.io/token"
	immutable: bool
	data: [string]: #Secret | string
}

#SecretDefaults: #SecretSchema & {
	type:      "Opaque"
	immutable: false
}

/////////////////////////////////////////////////////////////////
//// Content Hash Helpers
////
//// Regular fields (not #-prefixed) carry concrete values through
//// unification chains; definition fields lose them when forwarded.
/////////////////////////////////////////////////////////////////

// Deterministic 10-character hex hash of a string data map. Used by
// ConfigMapTransformer and as a building block for #SecretContentHash.
#ContentHash: {
	data: [string]: string

	let _keys = [for k, _ in data {k}]
	let _sorted = list.SortStrings(_keys)
	let _pairs = [for _, k in _sorted {"\(k)=\(data[k])"}]
	let _concat = strings.Join(_pairs, "\n")

	out: hex.Encode(sha256.Sum256(_concat)[:5])
}

// Normalize #Secret entries and plain strings to a string map, then hash.
//   string          -> key=<value>
//   #SecretLiteral  -> key=<value>
//   #SecretK8sRef   -> key=k8sref:<secretName>:<remoteKey>
#SecretContentHash: {
	data: [string]: #Secret | string

	let _normalized = {
		for k, v in data {
			if (v & string) != _|_ {
				"\(k)": v
			}
			if (v & string) == _|_ if (v & #SecretLiteral) != _|_ {
				"\(k)": v.value
			}
			if (v & string) == _|_ if (v & #SecretK8sRef) != _|_ {
				"\(k)": "k8sref:\(v.secretName):\(v.remoteKey)"
			}
		}
	}

	out: (#ContentHash & {data: _normalized}).out
}

// K8s resource name for a ConfigMap. Appends content-hash suffix when immutable.
// `let _d = data` captures concrete entries — without it CUE only forwards
// the [string]:string pattern.
#ImmutableName: {
	baseName: string
	data: [string]: string
	immutable: bool | *false

	let _d = data
	_hash: (#ContentHash & {data: _d}).out

	if immutable {
		out: "\(baseName)-\(_hash)"
	}
	if !immutable {
		out: baseName
	}
}

// K8s resource name for a Secret. Appends content-hash suffix when immutable.
#SecretImmutableName: {
	baseName: string
	data: [string]: #Secret | string
	immutable: bool | *false

	let _d = data
	_hash: (#SecretContentHash & {data: _d}).out

	if immutable {
		out: "\(baseName)-\(_hash)"
	}
	if !immutable {
		out: baseName
	}
}

/////////////////////////////////////////////////////////////////
//// Secret Discovery Pipeline
/////////////////////////////////////////////////////////////////

// Group a flat map of discovered secrets by $secretName, keyed by $dataKey.
// Mirrors the K8s Secret resource layout.
#GroupSecrets: {
	X=#in: {...}
	out: {
		for _k, v in X {
			(v.$secretName): (v.$dataKey): v
		}
	}
}

// Discover all #Secret instances from a resolved config and group by
// $secretName/$dataKey in one step. Output ready as spec.secrets data.
#AutoSecrets: {
	X=#in: {...}
	let _discovered = (#DiscoverSecrets & {#in: X}).out
	out: (#GroupSecrets & {#in: _discovered}).out
}

// Walk a resolved config (up to 10 levels) and collect all fields whose
// value is a #Secret. Detection: presence of the $opm discriminator field.
// Result is keyed by path (e.g., "dbUser", "database/password").
#DiscoverSecrets: {
	X=#in: {...}
	out: {
		// Level 1
		for k1, v1 in X
		if (v1.$opm != _|_) {
			(k1): v1
		}

		// Level 2
		for k1, v1 in X
		if (v1.$opm == _|_)
		if ((v1 & {...}) != _|_) {
			for k2, v2 in v1
			if (v2.$opm != _|_) {
				("\(k1)/\(k2)"): v2
			}
		}

		// Level 3
		for k1, v1 in X
		if (v1.$opm == _|_)
		if ((v1 & {...}) != _|_) {
			for k2, v2 in v1
			if (v2.$opm == _|_)
			if ((v2 & {...}) != _|_) {
				for k3, v3 in v2
				if (v3.$opm != _|_) {
					("\(k1)/\(k2)/\(k3)"): v3
				}
			}
		}

		// Level 4
		for k1, v1 in X
		if (v1.$opm == _|_)
		if ((v1 & {...}) != _|_) {
			for k2, v2 in v1
			if (v2.$opm == _|_)
			if ((v2 & {...}) != _|_) {
				for k3, v3 in v2
				if (v3.$opm == _|_)
				if ((v3 & {...}) != _|_) {
					for k4, v4 in v3
					if (v4.$opm != _|_) {
						("\(k1)/\(k2)/\(k3)/\(k4)"): v4
					}
				}
			}
		}

		// Level 5
		for k1, v1 in X
		if (v1.$opm == _|_)
		if ((v1 & {...}) != _|_) {
			for k2, v2 in v1
			if (v2.$opm == _|_)
			if ((v2 & {...}) != _|_) {
				for k3, v3 in v2
				if (v3.$opm == _|_)
				if ((v3 & {...}) != _|_) {
					for k4, v4 in v3
					if (v4.$opm == _|_)
					if ((v4 & {...}) != _|_) {
						for k5, v5 in v4
						if (v5.$opm != _|_) {
							("\(k1)/\(k2)/\(k3)/\(k4)/\(k5)"): v5
						}
					}
				}
			}
		}

		// Level 6
		for k1, v1 in X
		if (v1.$opm == _|_)
		if ((v1 & {...}) != _|_) {
			for k2, v2 in v1
			if (v2.$opm == _|_)
			if ((v2 & {...}) != _|_) {
				for k3, v3 in v2
				if (v3.$opm == _|_)
				if ((v3 & {...}) != _|_) {
					for k4, v4 in v3
					if (v4.$opm == _|_)
					if ((v4 & {...}) != _|_) {
						for k5, v5 in v4
						if (v5.$opm == _|_)
						if ((v5 & {...}) != _|_) {
							for k6, v6 in v5
							if (v6.$opm != _|_) {
								("\(k1)/\(k2)/\(k3)/\(k4)/\(k5)/\(k6)"): v6
							}
						}
					}
				}
			}
		}

		// Level 7
		for k1, v1 in X
		if (v1.$opm == _|_)
		if ((v1 & {...}) != _|_) {
			for k2, v2 in v1
			if (v2.$opm == _|_)
			if ((v2 & {...}) != _|_) {
				for k3, v3 in v2
				if (v3.$opm == _|_)
				if ((v3 & {...}) != _|_) {
					for k4, v4 in v3
					if (v4.$opm == _|_)
					if ((v4 & {...}) != _|_) {
						for k5, v5 in v4
						if (v5.$opm == _|_)
						if ((v5 & {...}) != _|_) {
							for k6, v6 in v5
							if (v6.$opm == _|_)
							if ((v6 & {...}) != _|_) {
								for k7, v7 in v6
								if (v7.$opm != _|_) {
									("\(k1)/\(k2)/\(k3)/\(k4)/\(k5)/\(k6)/\(k7)"): v7
								}
							}
						}
					}
				}
			}
		}

		// Level 8
		for k1, v1 in X
		if (v1.$opm == _|_)
		if ((v1 & {...}) != _|_) {
			for k2, v2 in v1
			if (v2.$opm == _|_)
			if ((v2 & {...}) != _|_) {
				for k3, v3 in v2
				if (v3.$opm == _|_)
				if ((v3 & {...}) != _|_) {
					for k4, v4 in v3
					if (v4.$opm == _|_)
					if ((v4 & {...}) != _|_) {
						for k5, v5 in v4
						if (v5.$opm == _|_)
						if ((v5 & {...}) != _|_) {
							for k6, v6 in v5
							if (v6.$opm == _|_)
							if ((v6 & {...}) != _|_) {
								for k7, v7 in v6
								if (v7.$opm == _|_)
								if ((v7 & {...}) != _|_) {
									for k8, v8 in v7
									if (v8.$opm != _|_) {
										("\(k1)/\(k2)/\(k3)/\(k4)/\(k5)/\(k6)/\(k7)/\(k8)"): v8
									}
								}
							}
						}
					}
				}
			}
		}

		// Level 9
		for k1, v1 in X
		if (v1.$opm == _|_)
		if ((v1 & {...}) != _|_) {
			for k2, v2 in v1
			if (v2.$opm == _|_)
			if ((v2 & {...}) != _|_) {
				for k3, v3 in v2
				if (v3.$opm == _|_)
				if ((v3 & {...}) != _|_) {
					for k4, v4 in v3
					if (v4.$opm == _|_)
					if ((v4 & {...}) != _|_) {
						for k5, v5 in v4
						if (v5.$opm == _|_)
						if ((v5 & {...}) != _|_) {
							for k6, v6 in v5
							if (v6.$opm == _|_)
							if ((v6 & {...}) != _|_) {
								for k7, v7 in v6
								if (v7.$opm == _|_)
								if ((v7 & {...}) != _|_) {
									for k8, v8 in v7
									if (v8.$opm == _|_)
									if ((v8 & {...}) != _|_) {
										for k9, v9 in v8
										if (v9.$opm != _|_) {
											("\(k1)/\(k2)/\(k3)/\(k4)/\(k5)/\(k6)/\(k7)/\(k8)/\(k9)"): v9
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// Level 10
		for k1, v1 in X
		if (v1.$opm == _|_)
		if ((v1 & {...}) != _|_) {
			for k2, v2 in v1
			if (v2.$opm == _|_)
			if ((v2 & {...}) != _|_) {
				for k3, v3 in v2
				if (v3.$opm == _|_)
				if ((v3 & {...}) != _|_) {
					for k4, v4 in v3
					if (v4.$opm == _|_)
					if ((v4 & {...}) != _|_) {
						for k5, v5 in v4
						if (v5.$opm == _|_)
						if ((v5 & {...}) != _|_) {
							for k6, v6 in v5
							if (v6.$opm == _|_)
							if ((v6 & {...}) != _|_) {
								for k7, v7 in v6
								if (v7.$opm == _|_)
								if ((v7 & {...}) != _|_) {
									for k8, v8 in v7
									if (v8.$opm == _|_)
									if ((v8 & {...}) != _|_) {
										for k9, v9 in v8
										if (v9.$opm == _|_)
										if ((v9 & {...}) != _|_) {
											for k10, v10 in v9
											if (v10.$opm != _|_) {
												("\(k1)/\(k2)/\(k3)/\(k4)/\(k5)/\(k6)/\(k7)/\(k8)/\(k9)/\(k10)"): v10
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}
