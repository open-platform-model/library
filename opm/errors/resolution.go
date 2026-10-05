package errors

// ResolutionKind says which kind of author-defect resolution failure a
// [*ResolutionError] reports.
type ResolutionKind int

const (
	// ResolutionOther is the zero value: a ResolutionError built without a
	// kind. [Classify] never builds it.
	ResolutionOther ResolutionKind = iota

	// ResolutionImportUnprovided is an imported package that no module of
	// the build provides. CUE reports three causes the same way, and this
	// kind covers all three: an import of a package missing from the main
	// module's own path, an import of a module the main module does not
	// declare, and an import of a package missing from a declared dependency
	// that was fetched. `cue mod tidy` says the same against a registry that
	// lists no version of the imported module. None of them is transient:
	// the library cannot tell a late publish from a mistyped import here, so
	// it never retries one.
	ResolutionImportUnprovided

	// ResolutionImportAmbiguous is an imported package that more than one
	// module of the build provides, such as the main module holding a
	// directory at the path a declared dependency also provides.
	ResolutionImportAmbiguous

	// ResolutionModuleFileInvalid is a dependency whose module file does not
	// parse, met while the module graph is expanded or while a directly
	// imported dependency is read. The module file is a published one or
	// the one in a local replacement directory (cue.mod/local-module.cue
	// replaceWith); the error names the replaced coordinate either way. No
	// retry cures it: a published version is immutable, so the import moves
	// to another version, and a replacement directory's file is fixed in
	// place.
	ResolutionModuleFileInvalid
)

// String returns the kind's name: "other", "import unprovided", "import
// ambiguous" or "module file invalid".
func (k ResolutionKind) String() string {
	switch k {
	case ResolutionImportUnprovided:
		return "import unprovided"
	case ResolutionImportAmbiguous:
		return "import ambiguous"
	case ResolutionModuleFileInvalid:
		return "module file invalid"
	default:
		return "other"
	}
}

// ResolutionError is an author-defect resolution failure: the build's
// imports do not resolve, and no failed registry interaction caused it. It
// is typed so a caller tells it apart from a fetch failure ([*FetchError])
// and from other author defects without reading message text
// (0021:D8:R12). The library builds it through [Classify] at every site
// where such a failure leaves it, inside the site's existing wrap.
//
// It is never transient: errors.Is(err, [ErrTransient]) is false through
// it, and it is not a FetchError, so a frontend that retries every fetch
// failure does not retry it. Its message is the cause's message, unchanged,
// and it unwraps to the cause, so errors.Is and errors.As on the cause (a
// CUE error list included) keep matching through it.
type ResolutionError struct {
	// Kind says which kind of failure this is.
	Kind ResolutionKind

	// Err is the cause, unchanged.
	Err error
}

// Error returns the cause's message unchanged, or the kind's name when Err
// is nil (a caller bug that never panics).
func (e *ResolutionError) Error() string {
	if e.Err == nil {
		return "dependency resolution failed: " + e.Kind.String()
	}
	return e.Err.Error()
}

// Unwrap returns the cause.
func (e *ResolutionError) Unwrap() error { return e.Err }
