package errors

import "errors"

// ErrTransient marks a registry failure that the same request may get past
// later with nothing changed: the registry could not be reached (no HTTP
// response, which includes an expired deadline), or it answered with a 5xx
// status. It is network-level only. An absent module, a refused credential,
// a 429 answer and every error the library does not recognise as a fetch
// failure are not transient, and neither is a context cancellation.
//
// Match it with errors.Is; a [*FetchError] in the chain answers for it. The
// caller owns retries, and a cache that memoized the failure keeps it: a
// schema load error held by an opm/schema Cache keeps its classification, so
// a retry needs a fresh Cache (a fresh Kernel). Source: 0021:D8:R12.
var ErrTransient = errors.New("transient registry failure")

// FetchKind says which kind of registry fetch failure a [*FetchError]
// reports. An author-defect resolution failure is a [*ResolutionError] and
// has a [ResolutionKind] instead.
type FetchKind int

const (
	// FetchOther is a failed registry interaction of no narrower kind: a
	// registry answer other than 401, 403 and 404 (a 429 or a 5xx, whose
	// code is in [FetchError.Status]), or a fetch whose cause the library
	// does not recognise, such as a published archive that does not unzip.
	FetchOther FetchKind = iota

	// FetchNotFound is an absent module, version or package the registry was
	// asked for: it does not hold the tag, or the version a path@version
	// load names does not provide the package. CUE's registry client reports
	// a 403 answer to a tag lookup as not found too, so that case is
	// FetchNotFound. An import no module of the build provides (an
	// undeclared dependency, or a package missing from the module's own
	// path or from a declared dependency) is an author defect, a
	// [*ResolutionError] of kind [ResolutionImportUnprovided], not a
	// FetchError.
	FetchNotFound

	// FetchUnauthorized is a registry that refused the credentials (401),
	// or refused access (403) where the 403 answer reaches the library
	// (see FetchNotFound for the tag lookup).
	FetchUnauthorized

	// FetchUnreachable is a request that got no HTTP response: a refused
	// connection, a DNS or TLS failure, a timeout or an expired deadline.
	FetchUnreachable
)

// String returns the kind's name: "other", "not found", "unauthorized" or
// "unreachable".
func (k FetchKind) String() string {
	switch k {
	case FetchNotFound:
		return "not found"
	case FetchUnauthorized:
		return "unauthorized"
	case FetchUnreachable:
		return "unreachable"
	default:
		return "other"
	}
}

// FetchError is a failed registry interaction, during a fetch or during
// dependency resolution, typed so a caller branches on it without reading
// message text (0021:D8:R12). An author-defect resolution failure, which no
// registry interaction caused, is a [*ResolutionError] instead. The library
// builds both through [Classify] at every site where such a failure leaves
// it, inside the site's existing wrap.
//
// Its message is the cause's message, unchanged, and it unwraps to the
// cause, so errors.Is and errors.As on the cause (a CUE error list
// included) keep matching through it.
type FetchError struct {
	// Kind says which kind of failure this is.
	Kind FetchKind

	// Coordinate is the "path@vX.Y.Z" the failing fetch asked for, when the
	// failing site knows it, and "" otherwise. A dependency resolution
	// failure leaves it empty: it may concern any module in the graph, and
	// the cause's text names it.
	Coordinate string

	// Status is the HTTP status the registry answered with, or 0 when there
	// was no answer or the status is not known.
	Status int

	// Err is the cause, unchanged.
	Err error
}

// Error returns the cause's message unchanged, or the kind's name when Err
// is nil (a caller bug that never panics).
func (e *FetchError) Error() string {
	if e.Err == nil {
		return "registry fetch failed: " + e.Kind.String()
	}
	return e.Err.Error()
}

// Unwrap returns the cause.
func (e *FetchError) Unwrap() error { return e.Err }

// Is reports whether target is [ErrTransient] and the failure is transient.
func (e *FetchError) Is(target error) bool {
	return target == ErrTransient && e.Transient()
}

// Transient reports whether the failure is network-level: the registry was
// unreachable, or answered with a 5xx status. It is the answer
// errors.Is(err, ErrTransient) gives.
func (e *FetchError) Transient() bool {
	return e.Kind == FetchUnreachable || e.Status >= 500
}
