package core

// #SecretsResourceFQN is the canonical FQN for the secrets resource.
// Must stay in sync with resources/config/secret.cue #SecretsResource.metadata.fqn.
#SecretsResourceFQN: "opmodel.dev/opm/resources/config/secrets@1.0.0"

// #OpmSecretsComponent builds the opm-secrets component from grouped secret data.
//
// Input:  #secrets — map of secretName -> (dataKey -> #Secret), i.e. #AutoSecrets output.
// Output: a component.#Component containing the opm-secrets resource.
#OpmSecretsComponent: {
	#secrets: {...}

	out: #Component & {
		metadata: {
			name: "opm-secrets"
			labels: {
				"component.opmodel.dev/name":    "opm-secrets"
				"resource.opmodel.dev/category": "config"
			}
			annotations: {
				"transformer.opmodel.dev/list-output": "true"
			}
		}

		// #resources provides the FQN key needed for transformer matching.
		// spec is auto-built by #Component from each resource's spec via _allFields.
		#resources: {
			(#SecretsResourceFQN): #Resource & {
				metadata: {
					modulePath: "opmodel.dev/opm/resources/config"
					version:    "1.0.0"
					name:       "secrets"
				}
				spec: secrets: {
					for secretName, _data in #secrets {
						(secretName): #SecretSchema & {
							name: secretName
							data: _data
						}
					}
				}
			}
		}
	}
}
