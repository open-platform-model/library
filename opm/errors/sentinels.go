package errors

import "errors"

// Sentinel errors every kernel acquisition and synthesis failure wraps via
// %w, so a frontend branches on the failure class with [errors.Is] rather
// than on message text.
//
// They live here, beside the typed render causes, because the packages that
// raise them (the kernel's internal loader and synthesizer) are internal:
// a sentinel declared there is unreachable for a consumer, and re-exporting
// it from opm/kernel would give one value two names. No other package under
// opm/ declares a sentinel of the same meaning.
var (
	// ErrInvalidPackage marks a structurally invalid package: the built
	// root value is not a struct, or the package load resolved other than
	// exactly one instance.
	ErrInvalidPackage = errors.New("invalid OPM package")

	// ErrWrongKind marks a package whose concrete kind does not match the
	// artifact the acquire verb was asked for.
	ErrWrongKind = errors.New("wrong OPM artifact kind")

	// ErrMissingRequiredField marks a package missing a required identity
	// field, or carrying it in non-concrete form. Concreteness is judged
	// before default finalization, so an identity field authored as a
	// defaulted disjunction is missing for this purpose.
	ErrMissingRequiredField = errors.New("missing required field")

	// ErrMissingModule marks an instance synthesis whose input carries no
	// Module, or one with no decoded modulePath/version identity to import
	// the module by.
	ErrMissingModule = errors.New("instance synthesis: Module is required")

	// ErrMissingName marks an instance synthesis whose input carries no
	// instance name.
	ErrMissingName = errors.New("instance synthesis: Name is required")

	// ErrMissingNamespace marks an instance synthesis whose input carries no
	// target namespace.
	ErrMissingNamespace = errors.New("instance synthesis: Namespace is required")

	// ErrMissingSource marks an instance synthesis whose Module carries no
	// staged source tree (HasSource reports false). Synthesis constructs the
	// instance INSIDE the module's own tree so the module's already-tidied
	// cue.mod/module.cue drives transitive resolution; it never fetches or
	// walks a tree of its own. Acquire a source-carrying module via
	// Kernel.AcquireModuleFromRegistry or Kernel.AcquireModuleFromDir.
	ErrMissingSource = errors.New("instance synthesis: Module has no staged source; acquire it via Kernel.AcquireModuleFromRegistry or Kernel.AcquireModuleFromDir")

	// ErrSchemaUnavailable marks a schema resolution that surfaces no core
	// release to derive the synthesized package's core import major from (a
	// bare-major loader whose load reports no version). A pinned loader never
	// produces it: the release is read off the pin without a load.
	ErrSchemaUnavailable = errors.New("instance synthesis: schema unavailable")
)
