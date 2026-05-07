package v1alpha2

// #SecretsResourceFQN is the canonical FQN for the secrets resource.
// Must stay in sync with resources/config/secret.cue #SecretsResource.metadata.fqn.
#SecretsResourceFQN: "opmodel.dev/opm/resources/config/secrets@v1"

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
		#resources: {
			(#SecretsResourceFQN): {
				spec: {
					secrets: {
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

		// spec mirrors #resources[SecretsResourceFQN].spec for direct access
		// by transformers (matches the shape of #SecretsResource.spec).
		spec: close({
			secrets: {
				for secretName, _data in #secrets {
					(secretName): #SecretSchema & {
						name: secretName
						data: _data
					}
				}
			}
		})
	}
}
