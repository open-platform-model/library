package v1alpha2

// Trimmed #Module — only the fields relevant to the matcher.
// #consumes is the 006 addition under test.
#Module: {
	apiVersion: #ApiVersion
	kind:       "Module"

	metadata: {
		name!: #NameType
	}

	// Stubbed — exp 01 isolates the matcher; we do not exercise components here.
	#components: [string]: _

	// Capabilities this module requires / optionally consumes.
	// Authored as declaration (FQN + schema); after #ContextBuilder matches,
	// the matched provider's spec is unified back into the entry (D7).
	// Non-optional pattern fields per D9 — empty map iterates cleanly.
	#consumes: {
		required: [FQN=#FQNType]: #Capability & {
			metadata: fqn: FQN
		}
		optional: [FQN=#FQNType]: #Capability & {
			metadata: fqn: FQN
		}
	}
}
