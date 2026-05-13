package core

import (
	"cuelang.org/go/cue"
)

// Compiled is the raw output of the OPM compile pipeline before any
// platform wrapping. The compile package emits *Compiled values; adapters
// translate them to platform-specific Resource implementations that
// expose Identity().
//
// Compiled itself does NOT implement Resource — keeping the two apart
// stops library code from accidentally reading platform-native fields and
// keeps the kernel platform-neutral.
type Compiled struct {
	// Value is the CUE value produced by the transformer. Concrete and
	// fully evaluated — safe to encode directly to YAML or JSON.
	Value cue.Value

	// Release is the name of the ModuleRelease that produced this resource.
	Release string

	// Component is the source component name within the release.
	Component string

	// Transformer is the FQN of the transformer that produced this resource.
	Transformer string
}
