// Package core defines the shared primitives of OPM-compiled artifacts.
// Compiled is the kernel's terminal output: the compile package emits
// *Compiled values carrying the rendered CUE value plus OPM provenance.
// Platform-native identity for a compiled artifact is the frontend's
// concern — each consumer wraps *Compiled in its own resource type.
package core

import (
	"cuelang.org/go/cue"
)

// Compiled is the terminal output of the OPM compile pipeline. It carries
// no platform-native fields — keeping platform vocabulary out of the
// kernel keeps it platform-neutral.
type Compiled struct {
	// Value is the CUE value produced by the transformer. Concrete and
	// fully evaluated — safe to encode directly to YAML or JSON.
	Value cue.Value

	// Instance is the name of the ModuleInstance that produced this resource.
	// Was: Release
	Instance string

	// Component is the source component name within the instance.
	Component string

	// Transformer is the FQN of the transformer that produced this resource.
	Transformer string
}
