package schema

import "cuelang.org/go/cue"

// CUE paths the kernel's Go code reads on an OPM artifact. This is the whole
// inventory, and each path's readers are:
//
//   - Metadata: metadata decoding of every artifact kind (including the
//     module metadata Instance.ModuleMetadata decodes), instance processing
//     and the registry loader's identity read.
//   - Components: Instance.Components.
//   - Values: Instance.Values, the values check and conflict attribution of
//     an instance build, and the top-level `values:` unwrap of a file-backed
//     values source.
//   - Config: Module.ConfigSchema, Instance.ConfigSchema, and values checking
//     against #config at acquire and synthesis.
//   - Module: Instance.ConfigSchema, Instance.ModuleMetadata and values
//     checking at acquire and synthesis.
//   - DebugValues: Module.DebugValues.
//   - Contracts: Platform.Contracts.
//   - ContractsProvidedBy: the core-floor presence check Kernel.Render runs
//     before staging.
//   - ContractsCollisions, ContractsCollidingEntries: in Go, tests only; they
//     document the collision report, whose fields Platform.Contracts reads
//     relative to [Contracts].
//   - CatalogProvides: Catalog.Provides.
//   - Transformers: Catalog.Provides, on both of its paths, and the
//     deprecated provider-set fold inside it.
//   - RequiredResources, RequiredTraits, Fulfilment: the deprecated fold,
//     relative to a transformer and its demand entries.
//
// Matching and execution read nothing by path from Go: the render build
// imports the instance and the platform as packages and the generated glue
// reads `components`, `#composedTransformers` and `#contracts` in CUE
// (0019:D9/D10). A path that no kernel code reads, by variable or relative to
// an inventory path, is removed, not kept for a possible consumer.
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
	// The render glue reads it and Contracts() decodes it, both by name
	// relative to [Contracts] and guarded on presence: a core without it
	// cannot evaluate a colliding platform, so absence means no collision
	// ([CollisionsSince]). In Go only tests read this variable.
	ContractsCollisions = cue.MakePath(cue.Def("contracts"), cue.Str("collisions"))

	// ContractsCollidingEntries is #Platform.#contracts.collidingEntries:
	// each [ContractsCollisions] key to the ascending registry keys (path
	// plus major) of the enabled entries whose catalogs list it. As for
	// [ContractsCollisions], in Go only tests read this variable.
	ContractsCollidingEntries = cue.MakePath(cue.Def("contracts"), cue.Str("collidingEntries"))

	// Catalog. Transformers is #Catalog.#transformers, the implementations
	// a catalog ships. RequiredResources, RequiredTraits and Fulfilment are
	// read RELATIVE to a transformer and to one of its demand entries, not
	// from an artifact root: the provider-fulfilled set a catalog implements
	// is the fold of every contract those two demand maps require whose
	// value carries fulfilment "provider". (*catalog.Catalog).Provides
	// reads Transformers on both of its paths, to refuse an unevaluated
	// #transformers; the other three are read only by its deprecated fold,
	// on demand, for a catalog built against a core older than
	// [ProvidesSince].
	Transformers      = cue.MakePath(cue.Def("transformers"))
	RequiredResources = cue.ParsePath("requiredResources")
	RequiredTraits    = cue.ParsePath("requiredTraits")
	Fulfilment        = cue.ParsePath("fulfilment")

	// CatalogProvides is #Catalog.provides: the provider-fulfilled
	// contracts the catalog implements, sorted and deduplicated, derived by
	// core since [ProvidesSince]. (*catalog.Catalog).Provides decodes it.
	CatalogProvides = cue.ParsePath("provides")

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

// ProvidesSince is the first core release deriving #Catalog.provides
// ([CatalogProvides]), without the "v" prefix. It decides which path
// (*catalog.Catalog).Provides takes for a catalog whose committed core pin
// is known, and tests use it. It is never a floor: a catalog built against
// an older core is answered through the deprecated fold, not refused.
const ProvidesSince = "2.0.0-beta.3"
