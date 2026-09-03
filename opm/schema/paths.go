package schema

import "cuelang.org/go/cue"

// CUE paths the kernel's Go code reads or writes on an OPM artifact: metadata
// decoding, instance processing and the module accessors. This is the whole
// inventory. Matching and execution read nothing by path from Go: the render
// build imports the instance and the platform as packages and the generated
// glue reads `components` and `#composedTransformers` in CUE (enhancement
// 0019 D9/D10). A path with no reader is removed, not kept for a possible
// consumer.
//
// Definition fields (those starting with "#" in CUE) use cue.MakePath with
// cue.Def selectors; concrete fields use cue.ParsePath. The two forms are
// not interchangeable — definition paths constructed with ParsePath do not
// resolve on closed structs.
var (
	// Artifact root.
	Metadata = cue.ParsePath("metadata")

	// Module instance.
	Components         = cue.ParsePath("components")
	Values             = cue.ParsePath("values")
	Config             = cue.MakePath(cue.Def("config"))
	Module             = cue.MakePath(cue.Def("module"))         // instance's reference to its source #Module
	ModuleMetadataPath = cue.MakePath(cue.Def("moduleMetadata")) // instance-side projection of #module.metadata. Suffixed -Path to avoid collision with the ModuleMetadata struct type.

	// Module-internal field. DebugValues is a Module field — NOT a separate
	// kernel artifact. Frontends that want a debug overlay read it from
	// Module.Package and decide whether to layer it into the values stack;
	// the kernel never receives debugValues as a parameter.
	DebugValues = cue.ParsePath("debugValues")
)
