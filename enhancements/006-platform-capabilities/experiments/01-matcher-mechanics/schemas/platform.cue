package v1alpha2

// Trimmed #Platform — only the fields relevant to the matcher.
// #provides is the 006 addition under test.
#Platform: {
	apiVersion: #ApiVersion
	kind:       "Platform"

	metadata: {
		name!: #NameType
	}

	// Concrete capability instances this platform supplies.
	// Single source for capability values (006 D4). Per-platform variation is
	// handled by CUE unification of #Platform values (OQ6 — exercised in
	// cases/06-platform-inheritance.cue).
	//
	// Non-optional pattern field per D9 — empty map iterates cleanly.
	#provides: [FQN=#FQNType]: #Capability & {
		metadata: fqn: FQN
	}
}
