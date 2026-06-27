package schema

import "cuelang.org/go/cue"

// CUE paths the kernel, matcher, helpers, and renderer use to read or write
// fields on an OPM artifact. Every consumer that previously called
// binding.Paths() now references these vars directly.
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

	// Provider.
	Transformers = cue.ParsePath("#transformers")

	// Platform. Five paths point at #registry and the four CUE-computed views
	// over it. KnownResources / KnownTraits are retained in the inventory
	// even though enhancement 0001's #Platform reshape drops the fields —
	// downstream consumers may still query them via LookupPath, and a missing
	// field yields a non-existent cue.Value rather than a hard error.
	Registry             = cue.ParsePath("#registry")
	KnownResources       = cue.ParsePath("#knownResources")
	KnownTraits          = cue.ParsePath("#knownTraits")
	ComposedTransformers = cue.ParsePath("#composedTransformers")
	Matchers             = cue.ParsePath("#matchers")
	MatchersResources    = cue.ParsePath("#matchers.resources")
	MatchersTraits       = cue.ParsePath("#matchers.traits")

	// Transformer body and matching predicates.
	Transform                    = cue.ParsePath("#transform")
	TransformerRequiredLabels    = cue.ParsePath("requiredLabels")
	TransformerRequiredResources = cue.ParsePath("requiredResources")
	TransformerRequiredTraits    = cue.ParsePath("requiredTraits")
	TransformerOptionalTraits    = cue.ParsePath("optionalTraits")

	// Inside #transform.
	Component = cue.ParsePath("#component")
	Context   = cue.MakePath(cue.Def("context"))
	Output    = cue.ParsePath("output")

	// Sub-paths of #context filled per (instance, component, transformer) pair.
	// Was: ContextModuleReleaseMetadata
	ContextModuleInstanceMetadata = cue.MakePath(cue.Def("context"), cue.Def("moduleInstanceMetadata"))
	ContextComponentMetadata      = cue.MakePath(cue.Def("context"), cue.Def("componentMetadata"))
	ContextRuntimeName            = cue.MakePath(cue.Def("context"), cue.Def("runtimeName"))

	// Component sub-paths.
	//
	// Blueprints are deliberately omitted: a Blueprint is a composition
	// template — its composedResources / composedTraits unify into a
	// Component's spec at CUE-evaluation time (see component.cue _allFields).
	// By the time the renderer sees a Component, blueprints are already
	// merged. Walking #blueprints separately would double-count.
	MetadataLabels      = cue.ParsePath("metadata.labels")
	MetadataAnnotations = cue.ParsePath("metadata.annotations")
	MetadataFQN         = cue.ParsePath("metadata.fqn")
	ComponentResources  = cue.MakePath(cue.Def("resources"))
	ComponentTraits     = cue.MakePath(cue.Def("traits"))
)
