package schema

import "cuelang.org/go/cue"

// CUE paths the kernel's Go code reads or writes on an OPM artifact: metadata
// decoding, instance processing, the loaders' identity reads, the
// instance's components and #config accessors, the platform's on-demand
// contract inventory, the render's core floor and the catalog's on-demand
// provider-set derivation. This is the whole inventory. Matching and
// execution read nothing by path from Go: the render build imports the
// instance and the platform as packages and the generated glue reads
// `components`, `#composedTransformers` and `#contracts` in CUE
// (0019:D9/D10). A path with no reader is removed, not kept for a possible
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
	Components = cue.ParsePath("components")
	Values     = cue.ParsePath("values")
	Config     = cue.MakePath(cue.Def("config"))
	Module     = cue.MakePath(cue.Def("module")) // instance's reference to its source #Module

	// Platform. Contracts is #Platform.#contracts, the contract inventory
	// core derives from the enabled registry entries' contract maps and the
	// transformers' required demands (0015:D1, D2, D5, D18).
	// (*platform.Platform).Contracts decodes its eleven data fields on
	// demand; `defined` (member schemas, not data) is not decoded. Never
	// the loader gate, never platform construction.
	Contracts = cue.MakePath(cue.Def("contracts"))

	// ContractsProvidedBy is #Platform.#contracts.providedBy: every
	// provider-fulfilled contract FQN an enabled transformer requires, to
	// the sorted registry keys of the entries supplying it. It is the one
	// provider count: the render glue reads it in CUE, Contracts() decodes
	// it, and Kernel.Render checks, before staging, only that the
	// platform's Package carries it (a presence test, nothing more), so a
	// platform pinning a core older than [ProvidedBySince] is refused.
	ContractsProvidedBy = cue.MakePath(cue.Def("contracts"), cue.Str("providedBy"))

	// ContractsCollisions is #Platform.#contracts.collisions: every
	// contract FQN more than one enabled registry entry's catalog lists,
	// ascending. Such a key is folded into none of definedBy, requiredBy,
	// unfulfilled or comparable, and routable is false while any exists.
	// The render glue reads it and Contracts() decodes it, both guarded on
	// presence: a core without it cannot evaluate a colliding platform, so
	// absence means no collision ([CollisionsSince]).
	ContractsCollisions = cue.MakePath(cue.Def("contracts"), cue.Str("collisions"))

	// ContractsCollidingEntries is #Platform.#contracts.collidingEntries:
	// each [ContractsCollisions] key to the ascending registry keys (path
	// plus major) of the enabled entries whose catalogs list it.
	ContractsCollidingEntries = cue.MakePath(cue.Def("contracts"), cue.Str("collidingEntries"))

	// Catalog. Transformers is #Catalog.#transformers, the implementations
	// a catalog ships. RequiredResources, RequiredTraits and Fulfilment are
	// read RELATIVE to a transformer and to one of its demand entries, not
	// from an artifact root: the provider-fulfilled set a catalog implements
	// is the fold of every contract those two demand maps require whose
	// value carries fulfilment "provider". Their one reader is
	// (*catalog.Catalog).Provides, on demand.
	Transformers      = cue.MakePath(cue.Def("transformers"))
	RequiredResources = cue.ParsePath("requiredResources")
	RequiredTraits    = cue.ParsePath("requiredTraits")
	Fulfilment        = cue.ParsePath("fulfilment")

	// Module-internal field. DebugValues is a Module field — NOT a separate
	// kernel artifact. Frontends that want a debug overlay read it through
	// Module.DebugValues() and decide whether to layer it into the values
	// stack; the kernel never receives debugValues as a parameter.
	DebugValues = cue.ParsePath("debugValues")
)

// ProvidedBySince is the first core release deriving
// #Platform.#contracts.providedBy ([ContractsProvidedBy]), without the "v"
// prefix: the oldest core a platform module may pin for Kernel.Render and
// Platform.Contracts, named in their PlatformCoreTooOldError.
const ProvidedBySince = "2.0.0-alpha.12"

// CollisionsSince is the first core release reporting contract collisions
// (#Platform.#contracts.collisions and collidingEntries,
// [ContractsCollisions]), without the "v" prefix. It documents the report
// and is used by tests; it is never a floor. Every older core carrying
// #contracts fails to evaluate a platform whose enabled entries share a
// contract key (the definedBy fold conflicts), so a platform pinning a core
// between [ProvidedBySince] and this release decodes an absent report as no
// collision.
const CollisionsSince = "2.0.0-alpha.13"
