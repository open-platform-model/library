// Package corepath holds the CUE paths the kernel's Go code reads on an OPM
// artifact and that no consumer needs: a frontend reaches each of these
// fields through an artifact accessor (Instance.Components, Instance.Values,
// Instance.ConfigSchema, Module.ConfigSchema, Module.DebugValues,
// Platform.Contracts, Catalog.Provides). The three paths a consumer does
// read on an artifact Package (Metadata, Module, CatalogProvides) are
// exported from opm/schema, and each path is declared in exactly one of the
// two packages. Being internal, the variables here cannot be imported or
// reassigned from outside the library.
//
// Each path's readers are:
//
//   - Components: Instance.Components.
//   - Values: Instance.Values, the values check and conflict attribution of
//     an instance build, and the top-level `values:` unwrap of a file-backed
//     values source.
//   - Config: Module.ConfigSchema, Instance.ConfigSchema, and values checking
//     against #config at acquire and synthesis.
//   - DebugValues: Module.DebugValues.
//   - Contracts: the inventory decode platform.NewPlatformFromValue records,
//     which Platform.Contracts returns.
//   - ContractsProvidedBy: the core-floor presence test
//     platform.NewPlatformFromValue records, which Platform.CoreFloor
//     reports and Kernel.Render checks before staging.
//   - ContractsCollisions, ContractsCollidingEntries: in Go, tests only; they
//     document the collision report, whose fields the inventory decode reads
//     relative to [Contracts].
//   - Transformers: Catalog.Provides, on both of its paths, and the
//     deprecated provider-set fold inside it.
//   - RequiredResources, RequiredTraits, Fulfilment: the deprecated fold,
//     relative to a transformer and its demand entries.
//
// A path that no kernel code reads, by variable or relative to an inventory
// path, is removed, not kept for a possible consumer.
//
// Definition fields (those starting with "#" in CUE) use cue.MakePath with
// cue.Def selectors; concrete fields use cue.ParsePath. The two forms are
// not interchangeable: definition paths constructed with ParsePath do not
// resolve on closed structs.
package corepath

import "cuelang.org/go/cue"

var (
	// Module instance.
	Components = cue.ParsePath("components")
	Values     = cue.ParsePath("values")
	Config     = cue.MakePath(cue.Def("config"))

	// Platform. Contracts is #Platform.#contracts, the contract inventory
	// core derives from the enabled registry entries' contract maps and the
	// transformers' required demands (0015:D1, D2, D5, D18).
	// platform.NewPlatformFromValue decodes its eleven data fields once
	// and records them, and (*platform.Platform).Contracts returns the
	// record (a Platform the constructor did not build decodes them on its
	// first Contracts or CoreFloor call); `defined` (member schemas, not
	// data) is not decoded. Never the loader gate.
	Contracts = cue.MakePath(cue.Def("contracts"))

	// ContractsProvidedBy is #Platform.#contracts.providedBy: every
	// provider-fulfilled contract FQN an enabled transformer requires, to
	// the sorted registry keys of the entries supplying it. It is the one
	// provider count: the render glue reads it in CUE, and
	// platform.NewPlatformFromValue records whether the platform carries it
	// (a presence test, nothing more), the core floor
	// (*platform.Platform).CoreFloor reports and Kernel.Render checks before
	// staging, so a platform pinning a core older than schema.ProvidedBySince is
	// refused. The inventory decode reads the same field relative to
	// [Contracts].
	ContractsProvidedBy = cue.MakePath(cue.Def("contracts"), cue.Str("providedBy"))

	// ContractsCollisions is #Platform.#contracts.collisions: every
	// contract FQN more than one enabled registry entry's catalog lists,
	// ascending. Such a key is folded into none of definedBy, requiredBy,
	// unfulfilled or comparable, and routable is false while any exists.
	// The render glue reads it and Contracts() decodes it, both by name
	// relative to [Contracts] and guarded on presence: a core without it
	// cannot evaluate a colliding platform, so absence means no collision
	// (core 2.0.0-alpha.13 is the first release with the report). In Go only tests read this variable.
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
	// schema.ProvidesSince.
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
