package schema

import "cuelang.org/go/cue"

// The CUE paths a consumer reads on an OPM artifact Package, where no
// artifact accessor gives the field. Every other path the kernel's Go code
// reads is internal to the library (opm/internal/corepath) and reached
// through an accessor: Instance.Components, Instance.Values,
// Instance.ConfigSchema, Module.ConfigSchema, Module.DebugValues,
// Platform.Contracts and Catalog.Provides. Each path's readers are:
//
//   - Metadata: metadata decoding of every artifact kind (including the
//     module metadata Instance.ModuleMetadata decodes), instance processing
//     and the registry loader's identity read.
//   - Module: Instance.ConfigSchema, Instance.ModuleMetadata and values
//     checking at acquire and synthesis.
//   - CatalogProvides: Catalog.Provides.
//
// Matching and execution read nothing by path from Go: the render build
// imports the instance and the platform as packages and the generated glue
// reads `components`, `#composedTransformers` and `#contracts` in CUE
// (0019:D9/D10).
//
// Definition fields (those starting with "#" in CUE) use cue.MakePath with
// cue.Def selectors; concrete fields use cue.ParsePath. The two forms are
// not interchangeable — definition paths constructed with ParsePath do not
// resolve on closed structs.
//
// Treat these variables as constants: assigning to one changes what every
// kernel in the process reads.
var (
	// Metadata is the `metadata` field at the root of every artifact.
	Metadata = cue.ParsePath("metadata")

	// Module is #ModuleInstance.#module, the instance's reference to its
	// source #Module.
	Module = cue.MakePath(cue.Def("module"))

	// CatalogProvides is #Catalog.provides: the provider-fulfilled
	// contracts the catalog implements, sorted and deduplicated, derived by
	// core since [ProvidesSince]. (*catalog.Catalog).Provides decodes it.
	CatalogProvides = cue.ParsePath("provides")
)

// ProvidedBySince is the first core release deriving
// #Platform.#contracts.providedBy, without the "v"
// prefix: the oldest core a platform module may pin for Kernel.Render and
// Platform.Contracts, named in their PlatformCoreTooOldError.
const ProvidedBySince = "2.0.0-alpha.12"

// ProvidesSince is the first core release deriving #Catalog.provides
// ([CatalogProvides]), without the "v" prefix. It decides which path
// (*catalog.Catalog).Provides takes for a catalog whose committed core pin
// is known, and tests use it. It is never a floor: a catalog built against
// an older core is answered through the deprecated fold, not refused.
const ProvidesSince = "2.0.0-beta.3"
