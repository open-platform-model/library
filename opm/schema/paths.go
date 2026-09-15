package schema

import "cuelang.org/go/cue"

// CUE paths the kernel's Go code reads or writes on an OPM artifact: metadata
// decoding, instance processing, the loaders' identity reads, the
// instance's components and #config accessors and the platform's on-demand
// contract inventory. This is the whole inventory. Matching and execution
// read nothing by path from Go: the render build imports the instance and
// the platform as packages and the generated glue reads `components` and
// `#composedTransformers` in CUE (enhancement 0019 D9/D10). A path with no
// reader is removed, not kept for a possible consumer.
//
// Definition fields (those starting with "#" in CUE) use cue.MakePath with
// cue.Def selectors; concrete fields use cue.ParsePath. The two forms are
// not interchangeable — definition paths constructed with ParsePath do not
// resolve on closed structs.
var (
	// Artifact root.
	Metadata = cue.ParsePath("metadata")

	// Module instance.
	Components = cue.ParsePath("components")
	Values     = cue.ParsePath("values")
	Config     = cue.MakePath(cue.Def("config"))
	Module     = cue.MakePath(cue.Def("module")) // instance's reference to its source #Module

	// Platform. Contracts is #Platform.#contracts, the contract inventory
	// core derives from the enabled catalogs' contract maps and the
	// transformers' required demands (enhancement 0015 D1, D2, D18). Its
	// one reader is (*platform.Platform).Contracts, on demand: never the
	// loader gate, never a kernel verb, never platform construction. The
	// six data fields under it are decoded; `defined` (member schemas, not
	// data) is not.
	Contracts = cue.MakePath(cue.Def("contracts"))

	// Catalog. Transformers is #Catalog.#transformers, the implementations
	// a catalog ships. RequiredResources, RequiredTraits and Fulfilment are
	// read RELATIVE to a transformer and to one of its demand entries, not
	// from an artifact root: the provider-fulfilled set a catalog implements
	// is the fold of every contract those two demand maps require whose
	// value carries fulfilment "provider" (ADR-009). Their one reader is
	// (*catalog.Catalog).Provides, on demand.
	Transformers      = cue.MakePath(cue.Def("transformers"))
	RequiredResources = cue.ParsePath("requiredResources")
	RequiredTraits    = cue.ParsePath("requiredTraits")
	Fulfilment        = cue.ParsePath("fulfilment")

	// Module-internal field. DebugValues is a Module field — NOT a separate
	// kernel artifact. Frontends that want a debug overlay read it from
	// Module.Package and decide whether to layer it into the values stack;
	// the kernel never receives debugValues as a parameter.
	DebugValues = cue.ParsePath("debugValues")
)
